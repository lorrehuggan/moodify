package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/lorrehuggan/moodify/internal/auth"
	spotifyx "github.com/lorrehuggan/moodify/internal/spotify"
	"github.com/spf13/cobra"
	"github.com/zmb3/spotify/v2"
)

var (
	discoverGenre      string
	discoverDecade     string
	discoverMood       string
	discoverEnergy     string
	discoverLimit      int
	discoverPopularity string
)

func init() {
	discoverCmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover new music based on various criteria",
		Long: `Explore and discover new music using Spotify's recommendation engine.
Find tracks based on genres, decades, moods, energy levels, and popularity.
Perfect for finding music you've never heard before!`,
		RunE: runDiscover,
	}

	discoverCmd.Flags().StringVarP(&discoverGenre, "genre", "g", "", "Specific genre (e.g., indie, jazz, electronic)")
	discoverCmd.Flags().StringVarP(&discoverDecade, "decade", "d", "", "Music decade (e.g., 80s, 90s, 2000s, 2010s)")
	discoverCmd.Flags().StringVarP(&discoverMood, "mood", "m", "", "Mood (happy, sad, energetic, chill, angry, romantic)")
	discoverCmd.Flags().StringVarP(&discoverEnergy, "energy", "e", "", "Energy level (low, medium, high)")
	discoverCmd.Flags().StringVarP(&discoverPopularity, "popularity", "p", "", "Popularity (mainstream, underground, balanced)")
	discoverCmd.Flags().IntVarP(&discoverLimit, "limit", "n", 20, "Number of tracks to discover (1-50)")

	rootCmd.AddCommand(discoverCmd)
}

func runDiscover(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Check authentication
	if !auth.QuickCheck() {
		fmt.Println("🔐 Authentication required!")
		fmt.Println("Run: moodify login")
		return fmt.Errorf("not authenticated")
	}

	// Get authenticated client
	config := &auth.Config{
		ClientID:    auth.GetClientIDFromEnv(),
		RedirectURI: "http://127.0.0.1:8808/callback",
		Port:        "8808",
		Scopes: []string{
			"user-top-read",
			"playlist-modify-private",
			"user-read-private",
		},
	}

	client, err := auth.GetAuthenticatedClient(ctx, config)
	if err != nil {
		fmt.Println("❌ Authentication failed. Run: moodify login")
		return err
	}

	// Validate limit
	if discoverLimit > 50 {
		discoverLimit = 50
	}
	if discoverLimit < 1 {
		discoverLimit = 20
	}

	fmt.Println("🔍 Music Discovery Engine")
	fmt.Println("═════════════════════════")
	fmt.Println()

	// If no specific criteria provided, do random discovery
	if discoverGenre == "" && discoverDecade == "" && discoverMood == "" && discoverEnergy == "" && discoverPopularity == "" {
		return runRandomDiscovery(ctx, client)
	}

	// Build recommendation parameters
	seeds, trackAttribs, yearStart, yearEnd := buildDiscoveryParameters(ctx, client)

	// Get recommendations
	recs, err := client.GetRecommendations(ctx, seeds, trackAttribs,
		spotify.Limit(discoverLimit), spotify.Market("US"))
	if err != nil {
		return fmt.Errorf("failed to get recommendations: %w", err)
	}

	tracks := recs.Tracks

	// Filter by year if decade specified
	if yearStart > 0 || yearEnd > 0 {
		filtered := make([]spotify.SimpleTrack, 0, len(tracks))
		for _, track := range tracks {
			year := spotifyx.ParseYear(track.Album.ReleaseDate)
			if (yearStart == 0 || year >= yearStart) && (yearEnd == 0 || year <= yearEnd) {
				filtered = append(filtered, track)
			}
		}
		tracks = filtered
	}

	if len(tracks) == 0 {
		fmt.Println("😔 No tracks found matching your criteria.")
		fmt.Println("Try broadening your search parameters.")
		return nil
	}

	// Display results
	fmt.Printf("🎵 Discovered %d tracks", len(tracks))
	if discoverGenre != "" {
		fmt.Printf(" in %s", discoverGenre)
	}
	if discoverDecade != "" {
		fmt.Printf(" from the %s", discoverDecade)
	}
	if discoverMood != "" {
		fmt.Printf(" with %s vibes", discoverMood)
	}
	fmt.Println()
	fmt.Println()

	for i, track := range tracks {
		artist := "Unknown Artist"
		if len(track.Artists) > 0 {
			artist = track.Artists[0].Name
		}
		year := spotifyx.ParseYear(track.Album.ReleaseDate)

		fmt.Printf("%2d. %s — %s", i+1, track.Name, artist)
		if year > 0 {
			fmt.Printf(" (%d)", year)
		}
		fmt.Printf("\n    Album: %s\n", track.Album.Name)
		if track.ExternalURLs["spotify"] != "" {
			fmt.Printf("    🔗 %s\n", track.ExternalURLs["spotify"])
		}
		fmt.Println()
	}

	// Show discovery tips
	fmt.Println("💡 Discovery Tips:")
	fmt.Println("   • Like what you hear? Save to playlist: --save \"My Discoveries\"")
	fmt.Println("   • Try different combinations of --genre, --mood, --energy")
	fmt.Println("   • Use --popularity underground to find hidden gems")
	fmt.Println("   • Explore decades: --decade 80s, 90s, 2000s, 2010s")

	return nil
}

func runRandomDiscovery(ctx context.Context, client *spotify.Client) error {
	fmt.Println("🎲 Random Music Discovery")
	fmt.Println("No criteria specified - discovering based on your music taste!")
	fmt.Println()

	// Get user's top genres from their top artists
	topArtists, err := client.CurrentUsersTopArtists(ctx, spotify.Limit(5))
	if err != nil {
		// Fallback to popular genres if we can't get user's top artists
		return runGenreBasedDiscovery(ctx, client)
	}

	if len(topArtists.Artists) == 0 {
		return runGenreBasedDiscovery(ctx, client)
	}

	// Use user's top artists as seeds
	seeds := spotify.Seeds{}
	for i, artist := range topArtists.Artists {
		if i >= 3 { // Limit to 3 artist seeds
			break
		}
		seeds.Artists = append(seeds.Artists, artist.ID)
	}

	// Add some randomness to attributes
	rand.Seed(time.Now().UnixNano())
	attrs := spotify.NewTrackAttributes().
		MinPopularity(20).
		MaxPopularity(80)

	// Randomly adjust some attributes for discovery
	if rand.Float32() > 0.5 {
		attrs = attrs.MinEnergy(0.4).MaxEnergy(1.0)
	}
	if rand.Float32() > 0.5 {
		attrs = attrs.MinValence(0.3).MaxValence(0.9)
	}

	recs, err := client.GetRecommendations(ctx, seeds, attrs,
		spotify.Limit(discoverLimit), spotify.Market("US"))
	if err != nil {
		return fmt.Errorf("failed to get personalized recommendations: %w", err)
	}

	fmt.Printf("🎵 Found %d personalized discoveries based on your taste:\n\n", len(recs.Tracks))

	for i, track := range recs.Tracks {
		artist := "Unknown Artist"
		if len(track.Artists) > 0 {
			artist = track.Artists[0].Name
		}
		year := spotifyx.ParseYear(track.Album.ReleaseDate)

		fmt.Printf("%2d. %s — %s", i+1, track.Name, artist)
		if year > 0 {
			fmt.Printf(" (%d)", year)
		}
		fmt.Printf("\n    🔗 %s\n\n", track.ExternalURLs["spotify"])
	}

	return nil
}

func runGenreBasedDiscovery(ctx context.Context, client *spotify.Client) error {
	// Fallback: use popular genres
	popularGenres := []string{"pop", "rock", "indie", "electronic", "hip-hop", "jazz", "classical"}
	rand.Seed(time.Now().UnixNano())

	selectedGenres := make([]string, 0, 3)
	for i := 0; i < 3 && i < len(popularGenres); i++ {
		idx := rand.Intn(len(popularGenres))
		selectedGenres = append(selectedGenres, popularGenres[idx])
	}

	seeds := spotify.Seeds{Genres: selectedGenres}
	attrs := spotify.NewTrackAttributes().MinPopularity(20).MaxPopularity(80)

	recs, err := client.GetRecommendations(ctx, seeds, attrs,
		spotify.Limit(discoverLimit), spotify.Market("US"))
	if err != nil {
		return fmt.Errorf("failed to get genre-based recommendations: %w", err)
	}

	fmt.Printf("🎵 Found %d tracks from genres: %v\n\n", len(recs.Tracks), selectedGenres)

	for i, track := range recs.Tracks {
		artist := "Unknown Artist"
		if len(track.Artists) > 0 {
			artist = track.Artists[0].Name
		}

		fmt.Printf("%2d. %s — %s\n", i+1, track.Name, artist)
		fmt.Printf("    🔗 %s\n\n", track.ExternalURLs["spotify"])
	}

	return nil
}

func buildDiscoveryParameters(ctx context.Context, client *spotify.Client) (spotify.Seeds, *spotify.TrackAttributes, int, int) {
	seeds := spotify.Seeds{}
	attrs := spotify.NewTrackAttributes()
	var yearStart, yearEnd int

	// Handle genre
	if discoverGenre != "" {
		seeds.Genres = append(seeds.Genres, discoverGenre)
	}

	// Handle decade
	if discoverDecade != "" {
		switch discoverDecade {
		case "60s", "1960s":
			yearStart, yearEnd = 1960, 1969
		case "70s", "1970s":
			yearStart, yearEnd = 1970, 1979
		case "80s", "1980s":
			yearStart, yearEnd = 1980, 1989
		case "90s", "1990s":
			yearStart, yearEnd = 1990, 1999
		case "2000s":
			yearStart, yearEnd = 2000, 2009
		case "2010s":
			yearStart, yearEnd = 2010, 2019
		case "2020s":
			yearStart, yearEnd = 2020, 2029
		}
	}

	// Handle mood
	switch discoverMood {
	case "happy", "joyful", "uplifting":
		attrs = attrs.MinValence(0.7).MinEnergy(0.5)
	case "sad", "melancholy", "depressing":
		attrs = attrs.MaxValence(0.4).MaxEnergy(0.6)
	case "energetic", "pumped", "exciting":
		attrs = attrs.MinEnergy(0.7).MinDanceability(0.6)
	case "chill", "relaxed", "calm":
		attrs = attrs.MaxEnergy(0.5).MinValence(0.3)
	case "angry", "aggressive", "intense":
		attrs = attrs.MinEnergy(0.8).MaxValence(0.4)
	case "romantic", "love", "intimate":
		attrs = attrs.MinValence(0.5).MaxEnergy(0.7).MinDanceability(0.3)
	}

	// Handle energy
	switch discoverEnergy {
	case "low":
		attrs = attrs.MaxEnergy(0.4)
	case "medium":
		attrs = attrs.MinEnergy(0.4).MaxEnergy(0.7)
	case "high":
		attrs = attrs.MinEnergy(0.7)
	}

	// Handle popularity
	switch discoverPopularity {
	case "mainstream", "popular":
		attrs = attrs.MinPopularity(70)
	case "underground", "obscure":
		attrs = attrs.MaxPopularity(30)
	case "balanced":
		attrs = attrs.MinPopularity(20).MaxPopularity(80)
	default:
		attrs = attrs.MinPopularity(10).MaxPopularity(90)
	}

	return seeds, attrs, yearStart, yearEnd
}
