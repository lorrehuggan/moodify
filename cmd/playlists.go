package cmd

import (
	"context"
	"fmt"

	"github.com/lorrehuggan/moodify/internal/auth"
	"github.com/spf13/cobra"
	"github.com/zmb3/spotify/v2"
)

var (
	showPublic    bool
	showPrivate   bool
	showAll       bool
	playlistLimit int
)

func init() {
	playlistsCmd := &cobra.Command{
		Use:   "playlists",
		Short: "View and manage your Spotify playlists",
		Long: `View your Spotify playlists and get information about them.
Use this command to see playlists you've created or follow.`,
		RunE: runPlaylists,
	}

	playlistsCmd.Flags().BoolVar(&showPublic, "public", false, "Show only public playlists")
	playlistsCmd.Flags().BoolVar(&showPrivate, "private", false, "Show only private playlists")
	playlistsCmd.Flags().BoolVar(&showAll, "all", false, "Show all playlists (including followed ones)")
	playlistsCmd.Flags().IntVarP(&playlistLimit, "limit", "n", 20, "Number of playlists to show (max 50)")

	rootCmd.AddCommand(playlistsCmd)
}

func runPlaylists(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Check if user is authenticated
	if !auth.QuickCheck() {
		fmt.Println("🔐 Authentication required!")
		fmt.Println("Run this command to get started: moodify login")
		return fmt.Errorf("not authenticated - run 'moodify login' first")
	}

	// Get authenticated Spotify client
	config := &auth.Config{
		ClientID:    auth.GetClientIDFromEnv(),
		RedirectURI: "http://127.0.0.1:8808/callback",
		Port:        "8808",
		Scopes: []string{
			"user-top-read",
			"playlist-modify-private",
			"user-read-private",
			"playlist-read-private",
		},
	}

	client, err := auth.GetAuthenticatedClient(ctx, config)
	if err != nil {
		fmt.Println("❌ Token expired or invalid. Please re-authenticate:")
		fmt.Println("   moodify login")
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Get current user info
	user, err := client.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	fmt.Printf("🎵 Playlists for %s\n", user.DisplayName)
	fmt.Println("══════════════════════════════════")
	fmt.Println()

	// Validate limit
	if playlistLimit > 50 {
		playlistLimit = 50
	}
	if playlistLimit < 1 {
		playlistLimit = 20
	}

	// Get user's playlists
	playlists, err := client.CurrentUsersPlaylists(ctx,
		spotify.Limit(playlistLimit))
	if err != nil {
		return fmt.Errorf("failed to get playlists: %w", err)
	}

	if len(playlists.Playlists) == 0 {
		fmt.Println("📭 No playlists found")
		fmt.Println("Create your first playlist by searching and using --save:")
		fmt.Println("   moodify search happy songs --save \"My Happy Playlist\"")
		return nil
	}

	// Filter playlists based on flags
	filteredPlaylists := make([]spotify.SimplePlaylist, 0)
	for _, playlist := range playlists.Playlists {
		// Apply visibility filters
		if showPublic && !playlist.IsPublic {
			continue
		}
		if showPrivate && playlist.IsPublic {
			continue
		}

		// Apply ownership filter (if not showing all)
		if !showAll && playlist.Owner.ID != user.ID {
			continue
		}

		filteredPlaylists = append(filteredPlaylists, playlist)
	}

	if len(filteredPlaylists) == 0 {
		fmt.Println("📭 No playlists match your filters")
		return nil
	}

	// Display playlists
	for i, playlist := range filteredPlaylists {
		// Determine ownership and visibility
		ownership := "👤 Yours"
		if playlist.Owner.ID != user.ID {
			ownership = fmt.Sprintf("👥 By %s", playlist.Owner.DisplayName)
		}

		visibility := "🔒 Private"
		if playlist.IsPublic {
			visibility = "🌍 Public"
		}

		// Format description
		description := playlist.Description
		if description == "" {
			description = "No description"
		}
		if len(description) > 80 {
			description = description[:77] + "..."
		}

		// Track count
		trackCount := fmt.Sprintf("%d tracks", playlist.Tracks.Total)

		fmt.Printf("%2d. %s\n", i+1, playlist.Name)
		fmt.Printf("    %s • %s • %s\n", ownership, visibility, trackCount)
		fmt.Printf("    %s\n", description)
		if playlist.ExternalURLs["spotify"] != "" {
			fmt.Printf("    🔗 %s\n", playlist.ExternalURLs["spotify"])
		}
		fmt.Println()
	}

	// Show summary
	totalShown := len(filteredPlaylists)
	totalAvailable := len(playlists.Playlists)

	if totalShown == totalAvailable {
		fmt.Printf("📊 Showing all %d playlists\n", totalShown)
	} else {
		fmt.Printf("📊 Showing %d of %d playlists", totalShown, totalAvailable)
		if showPublic || showPrivate || !showAll {
			fmt.Print(" (filtered)")
		}
		fmt.Println()
	}

	// Show helpful tips
	fmt.Println()
	fmt.Println("💡 Tips:")
	fmt.Println("   • Use --public or --private to filter by visibility")
	fmt.Println("   • Use --all to include playlists you follow")
	fmt.Println("   • Create new playlists: moodify search <query> --save \"Playlist Name\"")

	return nil
}
