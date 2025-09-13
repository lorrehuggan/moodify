package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"strings"

	"github.com/lorrehuggan/moodify/internal/ai"
	"github.com/lorrehuggan/moodify/internal/auth"
	spotifyx "github.com/lorrehuggan/moodify/internal/spotify"
	"github.com/spf13/cobra"
	"github.com/zmb3/spotify/v2"
)

var limit int
var market string
var saveToPlaylist string
var makePublic bool
var verbose bool

func init() {
	searchCmd := &cobra.Command{
		Use:   "search <free text query>",
		Short: "Search Spotify using natural language",
		Long: `Search Spotify using natural language descriptions of mood, genre, and era.

The app supports two parsing modes:
• 🤖 AI-powered (when OPENAI_API_KEY is set): Uses GPT-4o-mini for sophisticated understanding
  of complex musical descriptions like "melancholic indie with dreamy reverb"
• 📝 Basic keyword matching (default): Works well for simple queries like "happy pop music"

Examples:
  moodify search happy energetic workout songs
  moodify search chill lofi study music
  moodify search sad 90s alternative rock
  moodify search aggressive metal for gym
  moodify search nostalgic dreamy shoegaze  # AI mode understands this better

Use --verbose to see which parsing mode is active and view parsed attributes.`,
		Args: cobra.MinimumNArgs(1),
		RunE: runSearch,
	}
	searchCmd.Flags().IntVarP(&limit, "limit", "n", 15, "Number of tracks to return (1-100)")
	searchCmd.Flags().StringVar(&market, "market", "US", "ISO market code (e.g., US, GB)")
	searchCmd.Flags().StringVar(&saveToPlaylist, "save", "", "Save results to a new playlist with this name")
	searchCmd.Flags().BoolVar(&makePublic, "public", false, "Make the saved playlist public (default: private)")
	searchCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed processing information including AI parsing details")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")
	ctx := context.Background()

	// 1) Check if user is authenticated
	if !auth.QuickCheck() {
		fmt.Println("🔐 Authentication required!")
		fmt.Println("Run this command to get started: moodify login")
		fmt.Println()
		return fmt.Errorf("not authenticated - run 'moodify login' first")
	}

	// 2) Get authenticated Spotify client
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
		fmt.Println("❌ Token expired or invalid. Please re-authenticate:")
		fmt.Println("   moodify login")
		return fmt.Errorf("authentication failed: %w", err)
	}

	// 3) Parse natural language → filters
	if verbose {
		fmt.Printf("🎯 Analyzing query: %q\n", query)
	}

	// Check if OpenAI is available and notify user
	openaiEnabled := os.Getenv("OPENAI_API_KEY") != ""
	if openaiEnabled {
		fmt.Println("🤖 Using AI-powered query parsing (OpenAI GPT-4o-mini)")
		if verbose {
			fmt.Println("   This provides enhanced understanding of mood, genre, and musical attributes")
		}
	} else {
		fmt.Println("📝 Using basic keyword parsing")
		if verbose {
			fmt.Println("   For smarter results, set OPENAI_API_KEY environment variable")
		}
	}

	filters, err := ai.ParseQuery(ctx, query)
	if err != nil {
		if openaiEnabled {
			fmt.Printf("⚠️  AI parsing failed, falling back to basic parsing\n")
			if verbose {
				fmt.Printf("   Error: %v\n", err)
			}
		}
		log.Printf("AI parse failed, falling back to simple parser: %v", err)
		filters = ai.SimpleParse(query)
	}

	if verbose {
		fmt.Printf("🎼 Parsed filters:\n")
		if len(filters.Genres) > 0 {
			fmt.Printf("   Genres: %v\n", filters.Genres)
		}
		if filters.MinEnergy > 0 || filters.MaxEnergy < 1.0 {
			fmt.Printf("   Energy: %.2f - %.2f\n", filters.MinEnergy, filters.MaxEnergy)
		}
		if filters.MinValence > 0 || filters.MaxValence < 1.0 {
			fmt.Printf("   Mood (valence): %.2f - %.2f\n", filters.MinValence, filters.MaxValence)
		}
		if filters.MinDanceability > 0 || filters.MaxDanceability < 1.0 {
			fmt.Printf("   Danceability: %.2f - %.2f\n", filters.MinDanceability, filters.MaxDanceability)
		}
		if filters.YearStart > 0 || filters.YearEnd > 0 {
			fmt.Printf("   Year range: %d - %d\n", filters.YearStart, filters.YearEnd)
		}
		fmt.Println()
	}

	// 3) Build recommendation seeds + tuneable attributes
	seeds := spotify.Seeds{
		// Smart defaults: prefer up to 5 total across artists/genres/tracks
		Genres: filters.Genres, // []string
	}

	// Validate and clean up genres - remove any that might be invalid
	validGenres := validateGenres(seeds.Genres)
	seeds.Genres = validGenres

	// If no valid genres from parsing, seed by user's top artists as a nice fallback:
	if len(seeds.Genres) == 0 {
		top, err := client.CurrentUsersTopArtists(ctx, spotify.Limit(3))
		if err == nil && len(top.Artists) > 0 {
			for i, a := range top.Artists {
				if i >= 2 { // Limit to 2 artist seeds to leave room for genres if needed
					break
				}
				seeds.Artists = append(seeds.Artists, a.ID)
			}
		}
	}

	// Ensure we have at least one seed - use popular genres as last resort
	totalSeeds := len(seeds.Genres) + len(seeds.Artists) + len(seeds.Tracks)
	if totalSeeds == 0 {
		seeds.Genres = []string{"pop"}
	}

	// Ensure we don't exceed Spotify's limit of 5 seeds total
	if totalSeeds > 5 {
		if len(seeds.Genres) > 3 {
			seeds.Genres = seeds.Genres[:3]
		}
		if len(seeds.Artists) > 2 {
			seeds.Artists = seeds.Artists[:2]
		}
	}

	// Year/era constraint via seed query trick:
	// Spotify recs don't accept year directly; we'll post-filter if provided.
	hasYearFilter := filters.YearStart > 0 || filters.YearEnd > 0

	// 4) Try recommendations API first, fall back to search if it fails
	recs, err := spotifyx.GetRecommendationsWithFilters(ctx, client, seeds,
		filters.MinDanceability, filters.MaxDanceability,
		filters.MinEnergy, filters.MaxEnergy,
		filters.MinValence, filters.MaxValence,
		filters.MinTempo, filters.MaxTempo,
		filters.MinPopularity, filters.MaxPopularity,
		limit, market)

	var tracks []spotify.SimpleTrack

	if err != nil {
		// Fallback to search-based approach
		searchResults, searchErr := searchBasedFallback(ctx, client, query, filters, limit)
		if searchErr != nil {
			return fmt.Errorf("music discovery failed - please try a different search or try again later")
		}
		tracks = searchResults
	} else {
		tracks = recs.Tracks
	}

	// Optional: post-filter by release year if user mentioned an era (if not already done in fallback)
	if hasYearFilter {
		filtered := make([]spotify.SimpleTrack, 0, len(tracks))
		for _, t := range tracks {
			yr := spotifyx.ParseYear(t.Album.ReleaseDate)
			if (filters.YearStart == 0 || yr >= filters.YearStart) &&
				(filters.YearEnd == 0 || yr <= filters.YearEnd) {
				filtered = append(filtered, t)
			}
		}
		tracks = filtered
	}

	// 5) Print results
	if len(tracks) == 0 {
		fmt.Println("No tracks matched your vibe. Try loosening the query.")
		return nil
	}

	fmt.Printf("\n🎧 Results for: %q  (%d tracks)\n\n", query, len(tracks))
	for i, t := range tracks {
		artist := "Unknown"
		if len(t.Artists) > 0 {
			artist = t.Artists[0].Name
		}
		year := spotifyx.ParseYear(t.Album.ReleaseDate)
		fmt.Printf("%2d. %s — %s  (%d)\n    %s\n",
			i+1, t.Name, artist, year, t.ExternalURLs["spotify"])
	}

	// Save to playlist if requested
	if saveToPlaylist != "" {
		fmt.Printf("\n💾 Saving to playlist: %s\n", saveToPlaylist)
		if err := createPlaylistFromTracks(ctx, client, tracks, saveToPlaylist, makePublic); err != nil {
			fmt.Printf("❌ Failed to create playlist: %v\n", err)
		} else {
			visibility := "private"
			if makePublic {
				visibility = "public"
			}
			fmt.Printf("✅ Created %s playlist '%s' with %d tracks!\n", visibility, saveToPlaylist, len(tracks))
		}
	}

	return nil
}

// validateGenres filters out potentially invalid genre names
func validateGenres(genres []string) []string {
	// Known good Spotify recommendation genres (a subset of commonly used ones)
	validGenres := map[string]bool{
		"acoustic": true, "afrobeat": true, "alt-rock": true, "alternative": true,
		"ambient": true, "blues": true, "bossanova": true, "brazil": true,
		"breakbeat": true, "british": true, "chill": true, "classical": true,
		"club": true, "country": true, "dance": true, "dancehall": true,
		"deep-house": true, "disco": true, "drum-and-bass": true, "dub": true,
		"dubstep": true, "edm": true, "electronic": true, "folk": true,
		"funk": true, "garage": true, "gospel": true, "groove": true,
		"hip-hop": true, "house": true, "indie": true, "indie-pop": true,
		"jazz": true, "latin": true, "metal": true, "pop": true,
		"punk": true, "r-n-b": true, "reggae": true, "rock": true,
		"soul": true, "techno": true, "trance": true, "world-music": true,
	}

	var result []string
	for _, genre := range genres {
		if validGenres[strings.ToLower(genre)] {
			result = append(result, strings.ToLower(genre))
		}
	}

	return result
}

// createPlaylistFromTracks creates a new Spotify playlist with the given tracks
func createPlaylistFromTracks(ctx context.Context, client *spotify.Client, tracks []spotify.SimpleTrack, name string, public bool) error {
	// Get current user
	user, err := client.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Create playlist description
	description := fmt.Sprintf("Generated by Moodify - %d tracks discovered through natural language search", len(tracks))

	// Create playlist
	playlist, err := client.CreatePlaylistForUser(ctx, user.ID, name, description, public, false)
	if err != nil {
		return fmt.Errorf("failed to create playlist: %w", err)
	}

	// Convert SimpleTrack to track IDs for playlist addition
	var trackIDs []spotify.ID
	for _, track := range tracks {
		// Extract ID from URI (format: spotify:track:ID)
		uriParts := strings.Split(string(track.URI), ":")
		if len(uriParts) >= 3 {
			trackIDs = append(trackIDs, spotify.ID(uriParts[2]))
		}
	}

	if len(trackIDs) == 0 {
		return fmt.Errorf("no valid track IDs found")
	}

	// Add tracks to playlist (Spotify API limits to 100 tracks per request)
	const batchSize = 100
	for i := 0; i < len(trackIDs); i += batchSize {
		end := i + batchSize
		if end > len(trackIDs) {
			end = len(trackIDs)
		}

		batch := trackIDs[i:end]
		_, err = client.AddTracksToPlaylist(ctx, playlist.ID, batch...)
		if err != nil {
			return fmt.Errorf("failed to add tracks to playlist (batch %d-%d): %w", i+1, end, err)
		}
	}

	return nil
}

// searchBasedFallback implements music discovery using Spotify's search API when recommendations fail
func searchBasedFallback(ctx context.Context, client *spotify.Client, originalQuery string, filters ai.Filters, limit int) ([]spotify.SimpleTrack, error) {
	// Build search query based on parsed filters
	searchQuery := buildSearchQuery(originalQuery, filters)

	// Search for tracks
	results, err := client.Search(ctx, searchQuery, spotify.SearchTypeTrack, spotify.Limit(limit*2)) // Get more results to filter
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(results.Tracks.Tracks) == 0 {
		return nil, fmt.Errorf("no tracks found")
	}

	// Convert FullTrack to SimpleTrack and apply filters
	var tracks []spotify.SimpleTrack
	for _, fullTrack := range results.Tracks.Tracks {
		// Convert to SimpleTrack
		simpleTrack := spotify.SimpleTrack{
			Artists: make([]spotify.SimpleArtist, len(fullTrack.Artists)),
			Album: spotify.SimpleAlbum{
				Name:        fullTrack.Album.Name,
				ReleaseDate: fullTrack.Album.ReleaseDate,
				ID:          fullTrack.Album.ID,
			},
			ExternalURLs: fullTrack.ExternalURLs,
			ID:           fullTrack.ID,
			Name:         fullTrack.Name,
			URI:          fullTrack.URI,
		}

		// Convert artists
		for i, artist := range fullTrack.Artists {
			simpleTrack.Artists[i] = spotify.SimpleArtist{
				ID:   artist.ID,
				Name: artist.Name,
			}
		}

		// Apply year filter if specified
		if filters.YearStart > 0 || filters.YearEnd > 0 {
			year := spotifyx.ParseYear(simpleTrack.Album.ReleaseDate)
			if (filters.YearStart > 0 && year < filters.YearStart) ||
				(filters.YearEnd > 0 && year > filters.YearEnd) {
				continue // Skip tracks outside year range
			}
		}

		tracks = append(tracks, simpleTrack)
		if len(tracks) >= limit {
			break // We have enough tracks
		}
	}

	return tracks, nil
}

// buildSearchQuery creates a search string from the original query and parsed filters
func buildSearchQuery(originalQuery string, filters ai.Filters) string {
	query := originalQuery

	// Add genre information to search if available
	if len(filters.Genres) > 0 {
		// Add the first genre to the search query
		query += " genre:" + filters.Genres[0]
	}

	// Add year range if specified
	if filters.YearStart > 0 && filters.YearEnd > 0 {
		query += fmt.Sprintf(" year:%d-%d", filters.YearStart, filters.YearEnd)
	} else if filters.YearStart > 0 {
		query += fmt.Sprintf(" year:%d-2024", filters.YearStart)
	} else if filters.YearEnd > 0 {
		query += fmt.Sprintf(" year:1950-%d", filters.YearEnd)
	}

	return query
}
