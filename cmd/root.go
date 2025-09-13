package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "moodify",
	Short: "Zero-setup music discovery CLI for Spotify",
	Long: `🎵 Zero-setup music discovery CLI for Spotify

Find music by describing your mood, vibe, or era in natural language.
No configuration required - just run 'moodify login' and start discovering!

Examples:
  moodify login                              # One-time setup (opens browser)
  moodify search happy energetic workout     # Find upbeat gym music
  moodify search chill 90s alternative       # Discover laid-back 90s rock
  moodify search sad indie for rainy days    # Perfect melancholy playlist

Get started in 30 seconds: no API keys, no Spotify app setup required!`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// child commands added in other files' init()
}
