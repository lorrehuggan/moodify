# Moodify 🎵

**Zero-setup music discovery CLI for Spotify** - just clone, build, and start finding music!

A natural language Spotify search tool that understands your mood, vibe, and era preferences using AI-powered query parsing and Spotify's recommendation engine.

## ✨ Zero Setup Required

No Spotify app registration, no API keys, no configuration - just works!

## Features

- 🚀 **Zero Setup**: Works immediately without any configuration
- 🔐 **Secure Authentication**: Uses Spotify's Authorization Code Flow with PKCE
- 🎯 **Smart Port Detection**: Automatically finds available ports for authentication
- 🤖 **AI-Powered Search**: Natural language processing for mood and genre understanding
- 🎵 **Smart Recommendations**: Leverages Spotify's recommendation API
- 💾 **Token Management**: Automatic token refresh and secure local storage
- 🌍 **Cross-Platform**: Works on macOS, Linux, and Windows

## Quick Start

### 30-Second Setup

```bash
# 1. Clone and build
git clone https://github.com/lorrehuggan/moodify.git
cd moodify
go build

# 2. Login (opens browser automatically)
./moodify login

# 3. Start discovering music!
./moodify search happy energetic workout songs
./moodify search chill 90s alternative rock
./moodify search sad indie songs for rainy days
```

**That's it!** No configuration files, no API keys, no Spotify app setup required.

## Usage

### Authentication Commands

#### Login (Zero Setup!)
```bash
# Just run this - it handles everything automatically
./moodify login

# Advanced: Use specific port if needed
./moodify login --port 9000

# Advanced: Use your own Spotify app
./moodify login --client-id "your_client_id_here"
```

#### Check Status
```bash
# See authentication status and configuration
./moodify status
```

#### Logout
```bash
./moodify logout
```

### Search Examples

```bash
# Mood-based search
./moodify search chill lofi study music
./moodify search energetic workout songs
./moodify search sad melancholy indie

# Era-based search
./moodify search 90s alternative rock
./moodify search 2000s pop hits

# Genre-specific
./moodify search jazz fusion instrumental
./moodify search deep house electronic

# Combined preferences
./moodify search happy 80s dance music
./moodify search relaxing ambient for sleep

# Advanced options
./moodify search --limit 25 --market GB indie rock
```

### Search Options

- `--limit, -n`: Number of tracks to return (1-100, default: 15)
- `--market`: ISO market code for regional results (default: US)

## Configuration

### Zero Configuration Mode (Default)

Moodify works immediately without any setup using a shared Spotify application. Your authentication is still completely private and secure.

### Advanced Configuration (Optional)

#### Using Your Own Spotify App
For power users who want their own Spotify app:

```bash
# Optional: Use your own Spotify Client ID
export SPOTIFY_CLIENT_ID="your_client_id_here"

# Optional: Run the setup wizard
./moodify setup
```

#### Enable AI-Powered Query Parsing (Highly Recommended!)
Moodify works great out-of-the-box, but adding OpenAI makes it **significantly smarter**:

```bash
# Get your API key from https://platform.openai.com/api-keys
export OPENAI_API_KEY="sk-your-openai-api-key-here"
```

**🤖 With OpenAI enabled:**
- Much better understanding of complex mood descriptions
- Smarter genre detection and audio feature mapping
- More accurate results for phrases like "nostalgic dreamy shoegaze" or "aggressive workout metal"

**📝 Without OpenAI (default):**
- Uses basic keyword matching
- Still works well for simple queries like "chill jazz" or "happy pop"
- Completely free and private

**💡 How to get an OpenAI API key:**
1. Sign up at https://platform.openai.com/
2. Add a payment method (pay-per-use, typically $0.01-0.10 per search)
3. Create an API key in your dashboard
4. Export it: `export OPENAI_API_KEY="your_key_here"`

The app will automatically detect and use OpenAI when available, and clearly indicate when AI processing is being used.

### File Locations

- **Config Directory**: `~/.config/moodify/`
- **Token Storage**: `~/.config/moodify/token.json`

## How It Works

### 1. Zero-Setup Authentication

Moodify uses a pre-configured shared Spotify application for instant setup:

1. **Smart Port Detection**: Automatically finds an available port (8808, 8080, 3000, etc.)
2. **Secure PKCE Flow**: Uses Spotify's Authorization Code Flow with PKCE
3. **Browser Integration**: Opens your browser automatically (with fallback for headless systems)
4. **Token Management**: Securely stores and auto-refreshes your personal tokens
5. **Privacy First**: Your tokens never leave your machine

### 2. Natural Language Processing

The app processes your search query using:

- **🤖 AI Enhancement** (when `OPENAI_API_KEY` is set): GPT-4o-mini powered query analysis that understands complex mood descriptions, musical nuances, and context
- **📝 Fallback Parser** (default): Keyword-based mood and genre detection using curated patterns
- **🎵 Audio Feature Mapping**: Converts natural language to Spotify's tuneable attributes (danceability, energy, valence, tempo, etc.)

**The app will always tell you which parsing method it's using** and automatically falls back to keyword parsing if AI fails.

### 3. Smart Recommendations

- Uses Spotify's recommendation engine with calculated audio features
- Supports genre seeds, artist seeds, and audio attribute ranges
- Post-filters results by era/year when specified
- Balances popular and discoverable tracks

## Security & Privacy

- ✅ **No passwords handled**: Uses OAuth2 flow only
- ✅ **No client secrets**: PKCE eliminates need for secrets
- ✅ **Secure token storage**: Tokens stored with 600 permissions
- ✅ **Automatic token refresh**: Handles token expiry transparently
- ✅ **Local-only storage**: No data sent to third parties

## Troubleshooting

### Port Issues
```bash
# Try a specific port
./moodify login --port 9999

# Check what's happening
./moodify status
```

### Browser Won't Open
The app automatically displays the authorization URL to copy/paste manually.

### Authentication Errors
1. Check status: `./moodify status`
2. Try logout and login: `./moodify logout && ./moodify login`
3. For persistent issues, try custom setup: `./moodify setup`

### OpenAI Issues
```bash
# Check if OpenAI is detected
./moodify status

# Test with a complex query to see the difference
./moodify search melancholic indie with dreamy reverb
```

**Common OpenAI problems:**
- **Invalid API key**: Check your key at https://platform.openai.com/api-keys
- **Billing required**: OpenAI requires a payment method for API access
- **Rate limits**: Wait a moment and try again
- **Network issues**: The app automatically falls back to basic parsing

### No Results Found
- Try broader search terms
- Check your market setting
- Increase limit: `./moodify search --limit 50 your query`
- If using complex descriptions without OpenAI, try simpler keywords

## Development

### Project Structure

```
moodify/
├── cmd/                    # CLI commands
│   ├── login.go           # Authentication command
│   ├── logout.go          # Logout command
│   ├── search.go          # Search command
│   └── root.go            # Root command setup
├── internal/
│   ├── auth/              # PKCE authentication
│   ├── ai/                # Query parsing & AI integration
│   └── spotify/           # Spotify API wrapper
├── main.go                # Application entry point
└── go.mod                 # Dependencies
```

### Building from Source

```bash
git clone https://github.com/lorrehuggan/moodify.git
cd moodify
go mod tidy
go build
```

### Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Spotify Web API SDK](https://github.com/zmb3/spotify) - Spotify API client
- [OpenAI Go SDK](https://github.com/sashabaranov/go-openai) - AI query processing

## Why Zero Setup Works

- **Shared App**: Uses a pre-registered Spotify application with multiple redirect URIs
- **PKCE Security**: No client secrets involved - your authentication is still private
- **Smart Fallbacks**: Tries multiple ports and provides guidance if issues occur
- **Optional Advanced Setup**: Power users can still configure their own Spotify app

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

- 📝 [Create an issue](https://github.com/lorrehuggan/moodify/issues)
- 💬 [Discussions](https://github.com/lorrehuggan/moodify/discussions)
- 📚 [Spotify Web API Documentation](https://developer.spotify.com/documentation/web-api/)

---

**🎵 Start discovering music in 30 seconds: `moodify login`**