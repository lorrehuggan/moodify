package ai

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type Filters struct {
	Genres                           []string
	MinDanceability, MaxDanceability float64
	MinEnergy, MaxEnergy             float64
	MinValence, MaxValence           float64
	MinTempo, MaxTempo               float64
	MinPopularity, MaxPopularity     int
	YearStart, YearEnd               int
}

// ParseQuery: AI-powered parser (falls back to SimpleParse if no API key)
func ParseQuery(ctx context.Context, q string) (Filters, error) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		return SimpleParse(q), nil
	}
	c := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	sys := `You convert a music vibe prompt into strict JSON of tuneable attributes for Spotify Recommendations.
Return ONLY JSON with fields:
genres (array of lowercase strings, max 3),
min_danceability, max_danceability (0..1),
min_energy, max_energy (0..1),
min_valence, max_valence (0..1),
min_tempo, max_tempo (BPM, realistic 60..180),
min_popularity, max_popularity (0..100),
year_start, year_end (integers or 0).
Prefer broad ranges if uncertain.`
	user := "Prompt: " + q

	resp, err := c.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: sys},
			{Role: "user", Content: user},
		},
		Temperature: 0.2,
	})
	if err != nil || len(resp.Choices) == 0 {
		// Return the error so calling code can show appropriate fallback message
		return SimpleParse(q), err
	}

	jsonText := strings.TrimSpace(resp.Choices[0].Message.Content)
	// Very small and safe JSON "parse" without importing a full struct unmarshaller
	// (keep snippet short). In your real code: define a struct and json.Unmarshal.
	f := SimpleParse(q) // start with defaults; then patch in any fields we find

	// genres
	if arr := captureArray(jsonText, `(?i)"genres"\s*:\s*\[(.*?)\]`); len(arr) > 0 {
		f.Genres = arr
	}
	f.MinDanceability = captureFloat(jsonText, `"min_danceability"\s*:\s*([0-9\.]+)`, f.MinDanceability)
	f.MaxDanceability = captureFloat(jsonText, `"max_danceability"\s*:\s*([0-9\.]+)`, f.MaxDanceability)
	f.MinEnergy = captureFloat(jsonText, `"min_energy"\s*:\s*([0-9\.]+)`, f.MinEnergy)
	f.MaxEnergy = captureFloat(jsonText, `"max_energy"\s*:\s*([0-9\.]+)`, f.MaxEnergy)
	f.MinValence = captureFloat(jsonText, `"min_valence"\s*:\s*([0-9\.]+)`, f.MinValence)
	f.MaxValence = captureFloat(jsonText, `"max_valence"\s*:\s*([0-9\.]+)`, f.MaxValence)
	f.MinTempo = captureFloat(jsonText, `"min_tempo"\s*:\s*([0-9\.]+)`, f.MinTempo)
	f.MaxTempo = captureFloat(jsonText, `"max_tempo"\s*:\s*([0-9\.]+)`, f.MaxTempo)
	f.MinPopularity = captureInt(jsonText, `"min_popularity"\s*:\s*([0-9]+)`, f.MinPopularity)
	f.MaxPopularity = captureInt(jsonText, `"max_popularity"\s*:\s*([0-9]+)`, f.MaxPopularity)
	f.YearStart = captureInt(jsonText, `"year_start"\s*:\s*([0-9]+)`, f.YearStart)
	f.YearEnd = captureInt(jsonText, `"year_end"\s*:\s*([0-9]+)`, f.YearEnd)

	return f, nil
}

// SimpleParse: tiny heuristic fallback
func SimpleParse(q string) Filters {
	q = strings.ToLower(q)
	f := Filters{
		Genres:        []string{},
		MinPopularity: 20, MaxPopularity: 100,
		MinDanceability: 0.0, MaxDanceability: 1.0,
		MinEnergy: 0.0, MaxEnergy: 1.0,
		MinValence: 0.0, MaxValence: 1.0,
		MinTempo: 0.0, MaxTempo: 0.0,
		YearStart: 0, YearEnd: 0,
	}

	// rough keywords
	if strings.Contains(q, "chill") || strings.Contains(q, "lofi") {
		f.Genres = append(f.Genres, "chill", "ambient")
		f.MaxEnergy, f.MaxDanceability = 0.6, 0.7
	}
	if strings.Contains(q, "workout") || strings.Contains(q, "running") || strings.Contains(q, "gym") {
		f.MinEnergy, f.MinDanceability = 0.7, 0.7
		f.MinTempo, f.MaxTempo = 120, 180
	}
	if strings.Contains(q, "happy") || strings.Contains(q, "uplifting") || strings.Contains(q, "feel good") {
		f.MinValence = 0.6
	}
	if strings.Contains(q, "sad") || strings.Contains(q, "melancholy") {
		f.MaxValence = 0.5
	}
	// era
	if strings.Contains(q, "90s") || strings.Contains(q, "1990s") {
		f.YearStart, f.YearEnd = 1990, 1999
	}
	if strings.Contains(q, "2000s") {
		f.YearStart, f.YearEnd = 2000, 2009
	}
	// common genres (using valid Spotify recommendation genres)
	genreMap := map[string]string{
		"indie":         "indie",
		"pop":           "pop",
		"rock":          "rock",
		"jazz":          "jazz",
		"house":         "house",
		"techno":        "techno",
		"classical":     "classical",
		"ambient":       "ambient",
		"electronic":    "electronic",
		"hip-hop":       "hip-hop",
		"hip hop":       "hip-hop",
		"country":       "country",
		"folk":          "folk",
		"blues":         "blues",
		"reggae":        "reggae",
		"metal":         "metal",
		"punk":          "punk",
		"alternative":   "alternative",
		"r&b":           "r-n-b",
		"rnb":           "r-n-b",
		"soul":          "soul",
		"funk":          "funk",
		"disco":         "disco",
		"dance":         "dance",
		"edm":           "edm",
		"dubstep":       "dubstep",
		"drum-and-bass": "drum-and-bass",
		"dnb":           "drum-and-bass",
		"trance":        "trance",
		"garage":        "garage",
		"ska":           "ska",
		"gospel":        "gospel",
		"latin":         "latin",
		"world":         "world-music",
	}

	for keyword, genre := range genreMap {
		if strings.Contains(q, keyword) {
			f.Genres = append(f.Genres, genre)
		}
	}
	// dedupe & cap at 3
	seen := map[string]bool{}
	out := make([]string, 0, 3)
	for _, g := range f.Genres {
		g = strings.TrimSpace(g)
		if g == "" || seen[g] {
			continue
		}
		seen[g] = true
		out = append(out, g)
		if len(out) == 3 {
			break
		}
	}
	f.Genres = out
	return f
}

// --- tiny helpers for extracting JSON-ish bits without full JSON parsing ---

func captureArray(s, pattern string) []string {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return nil
	}
	inner := m[1]
	items := regexp.MustCompile(`"([^"]+)"`).FindAllStringSubmatch(inner, -1)
	out := []string{}
	for _, it := range items {
		out = append(out, strings.ToLower(strings.TrimSpace(it[1])))
	}
	return out
}

func captureFloat(s, pattern string, def float64) float64 {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return def
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return def
	}
	return v
}

func captureInt(s, pattern string, def int) int {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return def
	}
	v, err := strconv.Atoi(m[1])
	if err != nil {
		return def
	}
	return v
}
