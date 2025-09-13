package cmd

import (
	"github.com/lorrehuggan/moodify/internal/auth"
	"github.com/spf13/cobra"
)

func init() {
	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove stored Spotify credentials",
		Long: `Logout from Spotify by removing stored authentication tokens.
After logging out, you will need to run 'login' again before using
commands that require Spotify authentication.`,
		RunE: runLogout,
	}

	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) error {
	return auth.Logout()
}
