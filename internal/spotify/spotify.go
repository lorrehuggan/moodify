package spotify

import (
	"context"
	"strconv"

	"github.com/zmb3/spotify/v2"
)

// EnsureClient returns a Spotify client. This is now a simple wrapper
// that expects the client to already be authenticated.
func EnsureClient(client *spotify.Client) *spotify.Client {
	return client
}

// ParseYear extracts year from Spotify date format
func ParseYear(releaseDate string) int {
	// Spotify release_date may be "YYYY", "YYYY-MM-DD", or "YYYY-MM"
	if len(releaseDate) >= 4 {
		if year, err := strconv.Atoi(releaseDate[:4]); err == nil {
			return year
		}
	}
	return 0
}

// GetRecommendationsWithFilters is a convenience wrapper for getting recommendations
// with audio feature filters
func GetRecommendationsWithFilters(ctx context.Context, client *spotify.Client, seeds spotify.Seeds,
	minDanceability, maxDanceability float64,
	minEnergy, maxEnergy float64,
	minValence, maxValence float64,
	minTempo, maxTempo float64,
	minPopularity, maxPopularity int,
	limit int, market string) (*spotify.Recommendations, error) {

	opts := spotify.NewTrackAttributes()

	if minDanceability > 0 {
		opts = opts.MinDanceability(minDanceability)
	}
	if maxDanceability > 0 {
		opts = opts.MaxDanceability(maxDanceability)
	}
	if minEnergy > 0 {
		opts = opts.MinEnergy(minEnergy)
	}
	if maxEnergy > 0 {
		opts = opts.MaxEnergy(maxEnergy)
	}
	if minValence > 0 {
		opts = opts.MinValence(minValence)
	}
	if maxValence > 0 {
		opts = opts.MaxValence(maxValence)
	}
	if minTempo > 0 {
		opts = opts.MinTempo(minTempo)
	}
	if maxTempo > 0 {
		opts = opts.MaxTempo(maxTempo)
	}
	if minPopularity > 0 {
		opts = opts.MinPopularity(minPopularity)
	}
	if maxPopularity > 0 {
		opts = opts.MaxPopularity(maxPopularity)
	}

	return client.GetRecommendations(ctx, seeds, opts,
		spotify.Limit(limit), spotify.Market(market))
}
