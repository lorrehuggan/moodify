package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/lorrehuggan/moodify/internal/auth"
	"github.com/spf13/cobra"
)

func init() {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check authentication status and configuration",
		Long: `Display current authentication status, token expiry, and configuration details.
Use this to verify your setup and troubleshoot authentication issues.`,
		RunE: runStatus,
	}

	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Moodify Status")
	fmt.Println("═════════════════")
	fmt.Println()

	// Check client ID configuration
	clientID := auth.GetClientIDFromEnv()
	fmt.Println("📱 Configuration:")
	if clientID == auth.DefaultClientID {
		fmt.Println("   Client ID: Using shared Moodify app (zero-setup mode)")
		fmt.Println("   Setup: ✅ No setup required")
	} else if clientID != "" {
		fmt.Printf("   Client ID: %s (custom)\n", clientID[:4]+"..."+clientID[len(clientID)-4:])
		fmt.Println("   Setup: ✅ Using custom Spotify app")
	} else {
		fmt.Println("   Client ID: ❌ Not configured")
		fmt.Println("   Setup: ❌ Run 'moodify login' to get started")
	}

	// Check OpenAI configuration
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey != "" {
		fmt.Println("   OpenAI: ✅ AI-powered query parsing enabled")
	} else {
		fmt.Println("   OpenAI: ➖ Using basic keyword parsing (set OPENAI_API_KEY for AI enhancement)")
	}
	fmt.Println()

	// Check authentication status
	fmt.Println("🔐 Authentication:")
	if auth.QuickCheck() {
		fmt.Println("   Status: ✅ Authenticated and ready")

		// Try to get token details
		if token, err := auth.LoadTokenForStatus(); err == nil {
			timeUntilExpiry := time.Until(token.Expiry)
			if timeUntilExpiry > 0 {
				fmt.Printf("   Token expires: %s (%s from now)\n",
					token.Expiry.Format("2006-01-02 15:04:05"),
					formatDuration(timeUntilExpiry))
			} else {
				fmt.Println("   Token expires: ⚠️  Expired (will auto-refresh on next use)")
			}
		}
	} else {
		fmt.Println("   Status: ❌ Not authenticated")
		fmt.Println("   Action: Run 'moodify login' to authenticate")
	}
	fmt.Println()

	// Check config directory
	fmt.Println("📁 Storage:")
	if configDir, err := auth.GetConfigDirForStatus(); err == nil {
		fmt.Printf("   Config directory: %s\n", configDir)

		if tokenPath, err := auth.GetTokenPathForStatus(); err == nil {
			if _, err := os.Stat(tokenPath); err == nil {
				fmt.Printf("   Token file: %s ✅\n", tokenPath)
			} else {
				fmt.Printf("   Token file: %s ❌ (not found)\n", tokenPath)
			}
		}
	}
	fmt.Println()

	// Show available commands based on status
	fmt.Println("💡 Available Actions:")
	if !auth.QuickCheck() {
		fmt.Println("   • moodify login     - Authenticate with Spotify")
		fmt.Println("   • moodify setup     - Configure custom Spotify app (optional)")
	} else {
		fmt.Println("   • moodify search    - Search for music")
		fmt.Println("   • moodify logout    - Remove stored credentials")
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	} else {
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
}
