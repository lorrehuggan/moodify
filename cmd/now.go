package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/lorrehuggan/moodify/internal/auth"
	"github.com/spf13/cobra"
)

var showExtendedInfo bool

func init() {
	nowCmd := &cobra.Command{
		Use:   "now",
		Short: "Show what's currently playing on Spotify",
		Long: `Display information about the currently playing track on your Spotify account.
Shows track name, artist, album, progress, and playback controls information.`,
		RunE: runNow,
	}

	nowCmd.Flags().BoolVarP(&showExtendedInfo, "extended", "e", false, "Show extended track information (audio features)")

	rootCmd.AddCommand(nowCmd)
}

func runNow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Check authentication
	if !auth.QuickCheck() {
		fmt.Println("🔐 Authentication required!")
		fmt.Println("Run: moodify login")
		return fmt.Errorf("not authenticated")
	}

	// Get authenticated client with playback scopes
	config := &auth.Config{
		ClientID:    auth.GetClientIDFromEnv(),
		RedirectURI: "http://127.0.0.1:8808/callback",
		Port:        "8808",
		Scopes: []string{
			"user-read-currently-playing",
			"user-read-playback-state",
			"user-read-private",
		},
	}

	client, err := auth.GetAuthenticatedClient(ctx, config)
	if err != nil {
		fmt.Println("❌ Authentication failed. Run: moodify login")
		return err
	}

	// Get currently playing track
	currently, err := client.PlayerCurrentlyPlaying(ctx)
	if err != nil {
		return fmt.Errorf("failed to get currently playing track: %w", err)
	}

	if currently == nil || currently.Item == nil {
		fmt.Println("🎵 Nothing is currently playing")
		fmt.Println()
		fmt.Println("💡 Tips:")
		fmt.Println("   • Start playing music in Spotify")
		fmt.Println("   • Make sure Spotify is active on a device")
		fmt.Println("   • Try: moodify search <query> to find something to play")
		return nil
	}

	track := currently.Item
	fmt.Println("🎵 Now Playing")
	fmt.Println("═══════════════")
	fmt.Println()

	// Basic track info
	fmt.Printf("🎤 Track: %s\n", track.Name)

	// Artist(s)
	if len(track.Artists) > 0 {
		if len(track.Artists) == 1 {
			fmt.Printf("👤 Artist: %s\n", track.Artists[0].Name)
		} else {
			fmt.Print("👥 Artists: ")
			for i, artist := range track.Artists {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(artist.Name)
			}
			fmt.Println()
		}
	}

	// Album info
	fmt.Printf("💿 Album: %s", track.Album.Name)
	if track.Album.ReleaseDate != "" {
		if len(track.Album.ReleaseDate) >= 4 {
			fmt.Printf(" (%s)", track.Album.ReleaseDate[:4])
		}
	}
	fmt.Println()

	// Progress and duration
	if track.Duration > 0 {
		progress := time.Duration(currently.Progress) * time.Millisecond
		duration := time.Duration(track.Duration) * time.Millisecond

		fmt.Printf("⏰ Progress: %s / %s",
			formatPlaybackDuration(progress),
			formatPlaybackDuration(duration))

		// Progress bar
		if duration > 0 {
			percentage := float64(currently.Progress) / float64(track.Duration) * 100
			fmt.Printf(" (%.1f%%)", percentage)

			// Visual progress bar
			barLength := 30
			filled := int(percentage / 100 * float64(barLength))
			fmt.Print("\n    ")
			for i := 0; i < barLength; i++ {
				if i < filled {
					fmt.Print("█")
				} else {
					fmt.Print("░")
				}
			}
		}
		fmt.Println()
	}

	// Playback state
	playState := "⏸️  Paused"
	if currently.Playing {
		playState = "▶️  Playing"
	}
	fmt.Printf("🔄 Status: %s\n", playState)

	// Device info (if available)
	playerState, err := client.PlayerState(ctx)
	if err == nil && playerState != nil {
		fmt.Printf("📱 Device: %s (%s)\n", playerState.Device.Name, playerState.Device.Type)

		if playerState.ShuffleState {
			fmt.Print("🔀 Shuffle: On  ")
		} else {
			fmt.Print("🔀 Shuffle: Off  ")
		}

		switch playerState.RepeatState {
		case "track":
			fmt.Println("🔂 Repeat: Track")
		case "context":
			fmt.Println("🔁 Repeat: Context")
		default:
			fmt.Println("🔁 Repeat: Off")
		}

		if playerState.Device.Volume > 0 {
			fmt.Printf("🔊 Volume: %d%%\n", playerState.Device.Volume)
		}
	}

	// Spotify link
	if track.ExternalURLs["spotify"] != "" {
		fmt.Printf("🔗 Spotify: %s\n", track.ExternalURLs["spotify"])
	}

	// Extended info (audio features)
	if showExtendedInfo {
		fmt.Println()
		fmt.Println("🎛️  Audio Features")
		fmt.Println("═══════════════════")

		features, err := client.GetAudioFeatures(ctx, track.ID)
		if err == nil && len(features) > 0 && features[0] != nil {
			feature := features[0]

			fmt.Printf("🎵 Key: %s\n", getMusicalKey(int(feature.Key)))
			fmt.Printf("🎶 Tempo: %.0f BPM\n", feature.Tempo)
			fmt.Printf("⚡ Energy: %.1f/1.0\n", feature.Energy)
			fmt.Printf("💃 Danceability: %.1f/1.0\n", feature.Danceability)
			fmt.Printf("😊 Valence: %.1f/1.0\n", feature.Valence)
			fmt.Printf("🔊 Loudness: %.1f dB\n", feature.Loudness)

			if feature.Speechiness > 0.66 {
				fmt.Println("🎤 Type: Mostly speech")
			} else if feature.Speechiness > 0.33 {
				fmt.Println("🎤 Type: Music with speech")
			} else {
				fmt.Println("🎤 Type: Music")
			}
		} else {
			fmt.Println("Unable to get audio features for this track")
		}
	}

	fmt.Println()
	fmt.Println("💡 Tips:")
	fmt.Println("   • Use --extended (-e) for audio feature analysis")
	fmt.Println("   • Find similar music: moodify search <artist or genre>")
	if !currently.Playing {
		fmt.Println("   • Resume playback in your Spotify app")
	}

	return nil
}

func formatPlaybackDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func getMusicalKey(key int) string {
	keys := []string{"C", "C#/Db", "D", "D#/Eb", "E", "F", "F#/Gb", "G", "G#/Ab", "A", "A#/Bb", "B"}
	if key >= 0 && key < len(keys) {
		return keys[key]
	}
	return "Unknown"
}
