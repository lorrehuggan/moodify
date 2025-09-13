package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lorrehuggan/moodify/internal/ai"
	"github.com/spf13/cobra"
)

func init() {
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test OpenAI API key functionality",
		Long: `Test your OpenAI API key to verify AI-powered query parsing is working.

This command will:
• Check if OPENAI_API_KEY environment variable is set
• Test the OpenAI connection with a sample query
• Show you the difference between AI and basic parsing
• Help diagnose any OpenAI-related issues

Use this to verify your OpenAI setup before running searches.`,
		RunE: runTest,
	}

	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	fmt.Println("🧪 Testing OpenAI Integration")
	fmt.Println("═════════════════════════════")
	fmt.Println()

	// Check if API key is set
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ No OpenAI API key found")
		fmt.Println("   Set your API key: export OPENAI_API_KEY=\"sk-your-key-here\"")
		fmt.Println("   Get one at: https://platform.openai.com/api-keys")
		fmt.Println()
		fmt.Println("📝 Testing basic parsing instead...")
		testBasicParsing()
		return nil
	}

	fmt.Println("✅ OpenAI API key detected")
	fmt.Printf("   Key: %s...%s\n", apiKey[:7], apiKey[len(apiKey)-4:])
	fmt.Println()

	// Test OpenAI connection with a sample query
	fmt.Println("🤖 Testing AI parsing...")
	testQuery := "melancholic indie rock with dreamy reverb from the 2000s"
	fmt.Printf("   Sample query: \"%s\"\n", testQuery)
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	start := time.Now()
	aiFilters, err := ai.ParseQuery(ctx, testQuery)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ AI parsing failed: %v\n", err)
		fmt.Printf("   Response time: %v\n", duration)
		fmt.Println()
		fmt.Println("🔍 Possible issues:")
		fmt.Println("   • Invalid API key - check https://platform.openai.com/api-keys")
		fmt.Println("   • Billing not set up - OpenAI requires payment method")
		fmt.Println("   • Rate limit exceeded - wait a moment and try again")
		fmt.Println("   • Network connectivity issues")
		fmt.Println()
		fmt.Println("📝 Falling back to basic parsing...")
		testBasicParsing()
		return nil
	}

	fmt.Printf("✅ AI parsing successful! (took %v)\n", duration)
	printFilters("AI", aiFilters)

	// Compare with basic parsing
	fmt.Println("📝 Comparing with basic parsing...")
	basicFilters := ai.SimpleParse(testQuery)
	printFilters("Basic", basicFilters)

	// Show the difference
	fmt.Println("🔍 Key Differences:")
	showFilterDifferences(aiFilters, basicFilters)

	fmt.Println()
	fmt.Println("🎉 OpenAI integration is working perfectly!")
	fmt.Println("   Your searches will use AI-powered parsing for better results.")

	return nil
}

func testBasicParsing() {
	fmt.Println("📝 Testing basic parsing...")
	testQuery := "chill indie rock from the 90s"
	fmt.Printf("   Sample query: \"%s\"\n", testQuery)
	fmt.Println()

	basicFilters := ai.SimpleParse(testQuery)
	printFilters("Basic", basicFilters)

	fmt.Println("💡 To enable smarter AI parsing:")
	fmt.Println("   1. Get an API key: https://platform.openai.com/api-keys")
	fmt.Println("   2. Set up billing (typically $0.01-0.10 per search)")
	fmt.Println("   3. Export key: export OPENAI_API_KEY=\"your_key_here\"")
}

func printFilters(mode string, filters ai.Filters) {
	fmt.Printf("   %s Results:\n", mode)
	if len(filters.Genres) > 0 {
		fmt.Printf("     Genres: %v\n", filters.Genres)
	} else {
		fmt.Printf("     Genres: (none detected)\n")
	}

	if filters.MinEnergy > 0 || filters.MaxEnergy < 1.0 {
		fmt.Printf("     Energy: %.2f - %.2f\n", filters.MinEnergy, filters.MaxEnergy)
	}

	if filters.MinValence > 0 || filters.MaxValence < 1.0 {
		fmt.Printf("     Mood: %.2f - %.2f\n", filters.MinValence, filters.MaxValence)
	}

	if filters.MinDanceability > 0 || filters.MaxDanceability < 1.0 {
		fmt.Printf("     Danceability: %.2f - %.2f\n", filters.MinDanceability, filters.MaxDanceability)
	}

	if filters.YearStart > 0 || filters.YearEnd > 0 {
		fmt.Printf("     Years: %d - %d\n", filters.YearStart, filters.YearEnd)
	}

	if filters.MinTempo > 0 && filters.MaxTempo > 0 {
		fmt.Printf("     Tempo: %.0f - %.0f BPM\n", filters.MinTempo, filters.MaxTempo)
	}

	fmt.Println()
}

func showFilterDifferences(ai, basic ai.Filters) {
	differences := []string{}

	// Compare genres
	if len(ai.Genres) != len(basic.Genres) {
		differences = append(differences, fmt.Sprintf("   • Genres: AI found %d, Basic found %d", len(ai.Genres), len(basic.Genres)))
	}

	// Compare energy
	if ai.MinEnergy != basic.MinEnergy || ai.MaxEnergy != basic.MaxEnergy {
		differences = append(differences, "   • Energy ranges differ")
	}

	// Compare valence (mood)
	if ai.MinValence != basic.MinValence || ai.MaxValence != basic.MaxValence {
		differences = append(differences, "   • Mood interpretation differs")
	}

	// Compare years
	if ai.YearStart != basic.YearStart || ai.YearEnd != basic.YearEnd {
		differences = append(differences, "   • Era detection differs")
	}

	if len(differences) == 0 {
		fmt.Println("   • Results are similar for this query")
	} else {
		for _, diff := range differences {
			fmt.Println(diff)
		}
	}
}
