package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

const (
	// Production Client ID - pre-registered with multiple ports for zero-setup UX
	// This allows users to start using Moodify immediately without creating their own Spotify app
	DefaultClientID = "e16f7d194de8467882f2f198eec1729f"

	// Default configuration
	DefaultPort        = "8808"
	DefaultRedirectURI = "http://127.0.0.1:8808/callback"

	// File names
	TokenFileName = "token.json"
	ConfigDirName = "moodify"
)

var (
	// Common ports to try, in order of preference
	// All of these are registered in the Spotify app dashboard
	CommonPorts = []string{
		"8808", // Default
		"8080", // Alternative
		"3000", // Common dev port
		"8000", // HTTP alternative
		"9000", // High port
	}
)

// Config holds authentication configuration
type Config struct {
	ClientID    string
	RedirectURI string
	Port        string
	Scopes      []string
}

// TokenStore represents stored authentication tokens
type TokenStore struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ClientID:    DefaultClientID,
		RedirectURI: DefaultRedirectURI,
		Port:        DefaultPort,
		Scopes: []string{
			spotifyauth.ScopeUserTopRead,
			spotifyauth.ScopePlaylistModifyPrivate,
			spotifyauth.ScopeUserReadPrivate,
		},
	}
}

// ConfigWithClientID returns a config with the specified client ID
func ConfigWithClientID(clientID string) *Config {
	config := DefaultConfig()
	config.ClientID = clientID
	return config
}

// getConfigDir returns the user's configuration directory
func getConfigDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	configDir := filepath.Join(usr.HomeDir, ".config", ConfigDirName)
	return configDir, nil
}

// getTokenPath returns the path to the token file
func getTokenPath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, TokenFileName), nil
}

// generateCodeVerifier generates a random code verifier for PKCE
func generateCodeVerifier() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// generateCodeChallenge generates a code challenge from a verifier
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// generateState generates a random state parameter
func generateState() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// saveToken saves a token to disk with secure permissions
func saveToken(token *oauth2.Token) error {
	tokenPath, err := getTokenPath()
	if err != nil {
		return err
	}

	tokenStore := &TokenStore{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}

	data, err := json.MarshalIndent(tokenStore, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Write with secure permissions (readable/writable only by owner)
	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// loadToken loads a token from disk
func loadToken() (*oauth2.Token, error) {
	tokenPath, err := getTokenPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no token found, please run login first")
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var tokenStore TokenStore
	if err := json.Unmarshal(data, &tokenStore); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  tokenStore.AccessToken,
		RefreshToken: tokenStore.RefreshToken,
		TokenType:    tokenStore.TokenType,
		Expiry:       tokenStore.Expiry,
	}, nil
}

// deleteToken removes the stored token file
func deleteToken() error {
	tokenPath, err := getTokenPath()
	if err != nil {
		return err
	}

	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token file: %w", err)
	}

	return nil
}

// openBrowser attempts to open the given URL in the user's browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch {
	case isCommandAvailable("xdg-open"): // Linux
		cmd = "xdg-open"
		args = []string{url}
	case isCommandAvailable("open"): // macOS
		cmd = "open"
		args = []string{url}
	case isCommandAvailable("cmd"): // Windows
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("no browser opener found")
	}

	return execCommand(cmd, args...)
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(name string) bool {
	_, err := execLookPath(name)
	return err == nil
}

// Login performs the PKCE authentication flow
func Login(ctx context.Context, config *Config) error {
	// Generate PKCE parameters
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)
	state := generateState()

	// Create authenticator
	auth := spotifyauth.New(
		spotifyauth.WithClientID(config.ClientID),
		spotifyauth.WithRedirectURL(config.RedirectURI),
		spotifyauth.WithScopes(config.Scopes...),
	)

	// Build authorization URL with PKCE parameters
	authURL := auth.AuthURL(state) +
		"&code_challenge=" + url.QueryEscape(codeChallenge) +
		"&code_challenge_method=S256"

	// Start callback server
	tokenChan := make(chan *oauth2.Token, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: mux,
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Check state parameter
		if r.FormValue("state") != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errChan <- fmt.Errorf("invalid state parameter")
			return
		}

		// Check for authorization error
		if authError := r.FormValue("error"); authError != "" {
			errorDesc := r.FormValue("error_description")
			http.Error(w, fmt.Sprintf("Authorization error: %s - %s", authError, errorDesc), http.StatusBadRequest)
			errChan <- fmt.Errorf("authorization error: %s - %s", authError, errorDesc)
			return
		}

		// Get authorization code
		code := r.FormValue("code")
		if code == "" {
			http.Error(w, "No authorization code received", http.StatusBadRequest)
			errChan <- fmt.Errorf("no authorization code received")
			return
		}

		// Exchange code for token with PKCE
		token, err := exchangeCodeForToken(ctx, config, code, codeVerifier)
		if err != nil {
			http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
			errChan <- fmt.Errorf("failed to exchange code for token: %w", err)
			return
		}

		// Success response
		fmt.Fprintln(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; margin-top: 50px; }
        .success { color: #4CAF50; font-size: 24px; }
        .message { margin-top: 20px; color: #666; }
    </style>
</head>
<body>
    <div class="success">✓ Authentication Successful!</div>
    <div class="message">You can now close this tab and return to your terminal.</div>
</body>
</html>`)

		tokenChan <- token
	})

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Try to open browser
	fmt.Printf("Opening browser for Spotify authentication...\n")
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically. Please visit this URL:\n\n%s\n\n", authURL)
	}

	// Wait for token or error
	select {
	case token := <-tokenChan:
		server.Shutdown(ctx)

		// Save token
		if err := saveToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Println("✓ Successfully authenticated and saved credentials!")
		return nil

	case err := <-errChan:
		server.Shutdown(ctx)
		return err

	case <-ctx.Done():
		server.Shutdown(ctx)
		return fmt.Errorf("authentication cancelled")
	}
}

// exchangeCodeForToken exchanges an authorization code for an access token using PKCE
func exchangeCodeForToken(_ context.Context, config *Config, code, codeVerifier string) (*oauth2.Token, error) {
	// Prepare token exchange request
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {config.ClientID},
		"code":          {code},
		"redirect_uri":  {config.RedirectURI},
		"code_verifier": {codeVerifier},
	}

	// Make token request
	resp, err := http.PostForm("https://accounts.spotify.com/api/token", data)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Create token
	token := &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return token, nil
}

// Logout removes stored credentials
func Logout() error {
	if err := deleteToken(); err != nil {
		return err
	}
	fmt.Println("✓ Successfully logged out!")
	return nil
}

// GetAuthenticatedClient returns an authenticated Spotify client
func GetAuthenticatedClient(ctx context.Context, config *Config) (*spotify.Client, error) {
	token, err := loadToken()
	if err != nil {
		return nil, err
	}

	// Check if token needs refresh
	if token.Expiry.Before(time.Now().Add(5 * time.Minute)) {
		log.Println("Token expired or expiring soon, refreshing...")
		refreshedToken, err := refreshToken(ctx, config, token.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		// Save refreshed token
		if err := saveToken(refreshedToken); err != nil {
			log.Printf("Warning: failed to save refreshed token: %v", err)
		}

		token = refreshedToken
	}

	// Create OAuth2 config for token source
	oauthConfig := &oauth2.Config{
		ClientID: config.ClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
		RedirectURL: config.RedirectURI,
		Scopes:      config.Scopes,
	}

	// Create HTTP client with token
	httpClient := oauthConfig.Client(ctx, token)

	// Create Spotify client
	client := spotify.New(httpClient)

	return client, nil
}

// refreshToken refreshes an expired access token
func refreshToken(_ context.Context, config *Config, refreshToken string) (*oauth2.Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {config.ClientID},
	}

	resp, err := http.PostForm("https://accounts.spotify.com/api/token", data)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh request failed with status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	// Use existing refresh token if new one not provided
	if tokenResp.RefreshToken == "" {
		tokenResp.RefreshToken = refreshToken
	}

	token := &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return token, nil
}

// GetClientIDFromEnv returns the client ID from environment or default
func GetClientIDFromEnv() string {
	if clientID := os.Getenv("SPOTIFY_CLIENT_ID"); clientID != "" {
		return clientID
	}
	return DefaultClientID
}

// Helper functions for command execution
var (
	execCommand  = execCommandImpl
	execLookPath = exec.LookPath
)

func execCommandImpl(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	return cmd.Run()
}

// SmartLogin performs intelligent login with automatic port detection
func SmartLogin(ctx context.Context) error {
	fmt.Println("🎵 Starting Moodify authentication...")
	fmt.Println("🔍 Finding available port...")

	// Try each port until we find one that works
	for i, port := range CommonPorts {
		fmt.Printf("   Trying port %s... ", port)

		if !isPortAvailable(port) {
			fmt.Println("❌ in use")
			continue
		}

		fmt.Println("✅ available")

		config := &Config{
			ClientID:    getSmartClientID(),
			RedirectURI: fmt.Sprintf("http://127.0.0.1:%s/callback", port),
			Port:        port,
			Scopes: []string{
				"user-top-read",
				"playlist-modify-private",
				"user-read-private",
			},
		}

		fmt.Printf("🔗 Using redirect URI: %s\n", config.RedirectURI)
		fmt.Printf("🆔 Using Client ID: %s...%s\n", config.ClientID[:8], config.ClientID[len(config.ClientID)-4:])
		fmt.Println()

		// Attempt login with this configuration
		if err := Login(ctx, config); err != nil {
			fmt.Printf("❌ Authentication failed on port %s: %v\n", port, err)

			// If this was the last port, show help
			if i == len(CommonPorts)-1 {
				return showFallbackHelp(err)
			}

			fmt.Println("🔄 Trying next port...")
			continue
		}

		// Success!
		return nil
	}

	return fmt.Errorf("all ports exhausted")
}

// getSmartClientID returns the best available client ID
func getSmartClientID() string {
	// Priority: Environment variable > Production shared ID
	if envClientID := os.Getenv("SPOTIFY_CLIENT_ID"); envClientID != "" {
		return envClientID
	}

	return DefaultClientID
}

// isPortAvailable checks if a port is available for listening
func isPortAvailable(port string) bool {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// showFallbackHelp provides guidance when automatic login fails
func showFallbackHelp(lastError error) error {
	fmt.Println()
	fmt.Println("😔 Automatic setup didn't work. Here's how to fix it:")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Println("🔧 Option 1: Try a specific port")
	fmt.Println("   moodify login --port 9999")
	fmt.Println()

	fmt.Println("🔧 Option 2: Use your own Spotify app (2 minutes)")
	fmt.Println("   1. Go to: https://developer.spotify.com/dashboard")
	fmt.Println("   2. Create app with redirect URI: http://127.0.0.1:8808/callback")
	fmt.Println("   3. Copy your Client ID")
	fmt.Println("   4. Run: export SPOTIFY_CLIENT_ID=your_client_id_here")
	fmt.Println("   5. Run: moodify login")
	fmt.Println()

	fmt.Println("🔧 Option 3: Manual port specification")
	fmt.Println("   moodify login --client-id YOUR_ID --port 8808")
	fmt.Println()

	fmt.Printf("💡 Last error: %v\n", lastError)
	fmt.Println()
	fmt.Println("Need help? Visit: https://github.com/lorrehuggan/moodify#troubleshooting")

	return fmt.Errorf("authentication setup required")
}

// QuickCheck verifies if user is already authenticated
func QuickCheck() bool {
	token, err := loadToken()
	if err != nil {
		return false
	}

	// Check if token is valid (not expired)
	return token.Expiry.After(time.Now().Add(1 * time.Minute))
}

// LoadTokenForStatus returns token info for status display (exported version)
func LoadTokenForStatus() (*oauth2.Token, error) {
	return loadToken()
}

// GetConfigDirForStatus returns config directory for status display (exported version)
func GetConfigDirForStatus() (string, error) {
	return getConfigDir()
}

// GetTokenPathForStatus returns token path for status display (exported version)
func GetTokenPathForStatus() (string, error) {
	return getTokenPath()

}
