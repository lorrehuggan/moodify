package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/lorrehuggan/moodify/internal/auth"
	"github.com/spf13/cobra"
)

func init() {
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Optional setup wizard for custom Spotify app configuration",
		Long: `Interactive setup wizard for advanced users who want to use their own Spotify app
instead of the shared one. This is completely optional - most users can just run 'moodify login'.

Reasons to use your own app:
- You want full control over the authentication flow
- You're developing/modifying Moodify
- You need different scopes or settings

For most users: just run 'moodify login' instead!`,
		RunE: runSetup,
	}

	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("🔧 Moodify Advanced Setup")
	fmt.Println("═════════════════════════════")
	fmt.Println()
	fmt.Println("⚠️  NOTICE: Most users don't need this!")
	fmt.Println("   Moodify works out-of-the-box with 'moodify login'")
	fmt.Println("   Only use this if you need a custom Spotify app setup.")
	fmt.Println()

	if !askYesNo("Are you sure you want to configure a custom Spotify app?") {
		fmt.Println("✅ Perfect! Just run 'moodify login' to get started.")
		return nil
	}

	fmt.Println()

	// Check if already set up
	currentClientID := auth.GetClientIDFromEnv()
	if currentClientID != auth.DefaultClientID && currentClientID != "" {
		fmt.Println("✅ You already have a custom Client ID configured!")
		fmt.Printf("   Current Client ID: %s\n", maskClientID(currentClientID))
		fmt.Println()
		if !askYesNo("Do you want to reconfigure with a new Client ID?") {
			fmt.Println("Setup cancelled. Your current configuration is unchanged.")
			return nil
		}
		fmt.Println()
	}

	// Step 1: Explain what we're doing
	fmt.Println("📋 What this setup will do:")
	fmt.Println("   1. Guide you through creating a Spotify app")
	fmt.Println("   2. Get your custom Client ID")
	fmt.Println("   3. Save it to your environment configuration")
	fmt.Println()

	if !askYesNo("Ready to continue?") {
		fmt.Println("Setup cancelled.")
		return nil
	}

	fmt.Println()

	// Step 2: Guide through Spotify app creation
	fmt.Println("🌐 Step 1: Create Your Spotify App")
	fmt.Println("══════════════════════════════════════")
	fmt.Println()
	fmt.Println("1. Open: https://developer.spotify.com/dashboard")
	fmt.Println("2. Log in with your Spotify account")
	fmt.Println("3. Click 'Create app'")
	fmt.Println("4. Fill in the form:")
	fmt.Println("   • App name: 'My Personal Moodify'")
	fmt.Println("   • App description: 'Personal music discovery CLI'")
	fmt.Println("   • Website: (leave blank or use GitHub repo)")
	fmt.Println("   • Redirect URIs: Add ALL of these:")
	fmt.Println("     - http://127.0.0.1:8808/callback")
	fmt.Println("     - http://127.0.0.1:8080/callback")
	fmt.Println("     - http://127.0.0.1:3000/callback")
	fmt.Println("     - http://127.0.0.1:8000/callback")
	fmt.Println("     - http://127.0.0.1:9000/callback")
	fmt.Println("   • API/SDKs: Check 'Web API'")
	fmt.Println("5. Click 'Save'")
	fmt.Println()

	if !askYesNo("Have you created the Spotify app with all redirect URIs?") {
		fmt.Println("❌ Please create the app first, then run 'moodify setup' again.")
		return nil
	}

	fmt.Println()

	// Step 3: Get Client ID
	fmt.Println("🔑 Step 2: Get Your Client ID")
	fmt.Println("════════════════════════════════")
	fmt.Println()
	fmt.Println("1. On your app's dashboard page, find 'Client ID'")
	fmt.Println("2. Copy the Client ID (32-character hex string)")
	fmt.Println("3. Paste it below")
	fmt.Println()

	// Get Client ID from user
	clientID := ""
	for {
		fmt.Print("Paste your Client ID: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		clientID = strings.TrimSpace(input)

		if clientID == "" {
			fmt.Println("❌ Client ID cannot be empty.")
			continue
		}

		if isValidClientID(clientID) {
			break
		}

		fmt.Println("❌ That doesn't look like a valid Spotify Client ID.")
		fmt.Println("   It should be 32 characters of letters and numbers.")
		fmt.Println("   Example: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6")
		fmt.Println()

		if !askYesNo("Try again?") {
			fmt.Println("Setup cancelled.")
			return nil
		}
	}

	fmt.Println()

	// Step 4: Save configuration
	fmt.Println("💾 Step 3: Save Configuration")
	fmt.Println("═════════════════════════════")

	// Try to save to shell config files
	saved := false
	configPaths := getShellConfigPaths()

	for _, configPath := range configPaths {
		if fileExists(configPath) {
			if askYesNo(fmt.Sprintf("Add to %s?", configPath)) {
				if err := appendToShellConfig(configPath, clientID); err != nil {
					fmt.Printf("⚠️  Failed to write to %s: %v\n", configPath, err)
				} else {
					fmt.Printf("✅ Added to %s\n", configPath)
					saved = true
					break
				}
			}
		}
	}

	// Fallback: create a source-able config file
	if !saved {
		if err := saveClientIDToConfigFile(clientID); err != nil {
			fmt.Printf("⚠️  Could not create config file: %v\n", err)
			fmt.Println("   You can set the environment variable manually:")
			fmt.Printf("   export SPOTIFY_CLIENT_ID=%s\n", clientID)
		} else {
			fmt.Println("✅ Saved to ~/.moodify_config")
			fmt.Println("   Run: source ~/.moodify_config")
		}
	}

	fmt.Println()
	fmt.Println("🎉 Custom Setup Complete!")
	fmt.Println("═════════════════════════════")
	fmt.Println("Your Moodify now uses your custom Spotify app!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Restart your terminal (or run: source ~/.bashrc)")
	fmt.Println("2. Run: moodify login")
	fmt.Println("3. Try: moodify search happy energetic songs")
	fmt.Println()
	fmt.Println("🔒 Your Client ID is safe to share - it's like a username.")

	return nil
}

func askYesNo(question string) bool {
	fmt.Printf("%s (y/n): ", question)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes"
}

func isValidClientID(clientID string) bool {
	// Spotify Client IDs are 32-character hex strings
	matched, _ := regexp.MatchString(`^[a-f0-9]{32}$`, strings.ToLower(clientID))
	return matched
}

func maskClientID(clientID string) string {
	if len(clientID) < 8 {
		return clientID
	}
	return clientID[:4] + "..." + clientID[len(clientID)-4:]
}

func getShellConfigPaths() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}

	return []string{
		homeDir + "/.bashrc",
		homeDir + "/.zshrc",
		homeDir + "/.bash_profile",
		homeDir + "/.profile",
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func appendToShellConfig(configPath, clientID string) error {
	content := fmt.Sprintf("\n# Moodify Spotify Configuration\nexport SPOTIFY_CLIENT_ID=%s\n", clientID)

	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

func saveClientIDToConfigFile(clientID string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configFile := homeDir + "/.moodify_config"

	content := fmt.Sprintf(`# Moodify Configuration
# Generated by 'moodify setup'
export SPOTIFY_CLIENT_ID=%s

# To use this configuration:
# source ~/.moodify_config
#
# Or add this line to your ~/.bashrc or ~/.zshrc:
# source ~/.moodify_config
`, clientID)

	return os.WriteFile(configFile, []byte(content), 0644)
}
