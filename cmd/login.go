package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/lorrehuggan/moodify/internal/auth"
	"github.com/spf13/cobra"
)

var (
	clientID string
	port     string
)

func init() {
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Spotify using OAuth2 PKCE flow",
		Long: `Login to Spotify using the secure Authorization Code Flow with PKCE.
This will open your browser to authorize the application and store your
credentials securely in your local config directory.

The application will never see your Spotify password and only requests
the minimum necessary permissions.`,
		RunE: runLogin,
	}

	loginCmd.Flags().StringVar(&clientID, "client-id", "", "Spotify Client ID (overrides environment variable)")
	loginCmd.Flags().StringVar(&port, "port", auth.DefaultPort, "Port for the callback server")

	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// If user specified custom client ID or port, use manual configuration
	if clientID != "" || port != auth.DefaultPort {
		return runManualLogin(ctx, cmd, args)
	}

	// Use smart login that handles everything automatically
	if err := auth.SmartLogin(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Println("\n🎉 You're ready to use Moodify!")
	fmt.Println("Try: moodify search happy upbeat songs")
	return nil
}

// runManualLogin handles login with user-specified parameters
func runManualLogin(ctx context.Context, cmd *cobra.Command, args []string) error {
	// Determine client ID from flag, environment, or default
	var finalClientID string
	if clientID != "" {
		finalClientID = clientID
	} else {
		finalClientID = auth.GetClientIDFromEnv()
	}

	// Create auth config
	config := &auth.Config{
		ClientID:    finalClientID,
		RedirectURI: fmt.Sprintf("http://127.0.0.1:%s/callback", port),
		Port:        port,
		Scopes: []string{
			"user-top-read",
			"playlist-modify-private",
			"user-read-private",
		},
	}

	// Check if port is available
	if err := checkPortAvailable(port); err != nil {
		return fmt.Errorf("port %s is not available: %w\nTry using --port flag to specify a different port", port, err)
	}

	fmt.Printf("🎵 Starting Spotify authentication...\n")
	fmt.Printf("📱 Redirect URI: %s\n", config.RedirectURI)
	fmt.Printf("🔐 Client ID: %s\n", config.ClientID)
	fmt.Printf("📝 Scopes: %v\n\n", config.Scopes)

	// Perform login
	if err := auth.Login(ctx, config); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Println("\n🎉 You're now ready to use Moodify!")
	fmt.Println("Try: moodify search happy upbeat songs")

	return nil
}

// checkPortAvailable checks if a port is available for listening
func checkPortAvailable(port string) error {
	// This is a simple check - in a real implementation you might want to
	// actually try binding to the port temporarily
	if port == "" {
		return fmt.Errorf("port cannot be empty")
	}
	return nil
}
