# 🌿 Game Dialog Generator - Jungle Adventure 🎮

> AI-powered character dialog generation for video games with jungle platformer aesthetics

[![Jungle Theme](https://img.shields.io/badge/Theme-Jungle%20Platformer-green)](https://github.com/vrooli)
[![Game Development](https://img.shields.io/badge/Category-Game%20Development-orange)](https://github.com/vrooli)
[![AI Powered](https://img.shields.io/badge/AI-Ollama%20Integration-blue)](https://github.com/vrooli)

## 🎯 Overview

The Game Dialog Generator transforms game character development by providing AI-powered dialog generation with personality consistency. Designed with a vibrant jungle platformer aesthetic, this tool helps game developers create memorable characters with authentic, context-aware dialog.

### ✨ Key Features

- **🐒 Character AI**: Create characters with detailed personality traits and speech patterns
- **💬 Dynamic Dialog**: Generate contextual dialog that maintains character consistency
- **🎭 Emotion Modeling**: Characters respond appropriately to different emotional states
- **🎮 Game Integration**: Export dialog for Unity, Unreal Engine, and other platforms
- **🌿 Jungle Theme**: Immersive jungle platformer aesthetic throughout the interface
- **⚡ Real-time & Batch**: Support both dynamic gameplay and traditional dialog scripting

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- Node.js 16+
- PostgreSQL
- Qdrant vector database
- Ollama with llama3.2 model

### Installation

1. **Clone or navigate to the scenario:**
   ```bash
   cd scenarios/game-dialog-generator
   ```

2. **Run the setup:**
   ```bash
   ./scripts/manage.sh setup --yes yes
   ```

3. **Start the jungle adventure:**
   ```bash
   vrooli scenario run game-dialog-generator
   ```

4. **Access the interface:**
   - 🌿 **Jungle UI**: http://localhost:3200
   - 🔧 **API**: http://localhost:8080
   - 💻 **CLI**: `game-dialog-generator --help`

## 🎮 Usage Examples

### Creating Your First Jungle Character

```bash
# Interactive character creation
game-dialog-generator character-create "Kiko the Monkey" --interactive

# Or with CLI parameters
game-dialog-generator character-create "Luna the Owl" \
  --personality-file characters/luna.json
```

### Generating Adventure Dialog

```bash
# Generate context-aware dialog
game-dialog-generator dialog-generate \
  <character-id> \
  "A peaceful jungle clearing at dawn" \
  --emotion hopeful \
  --voice
```

### Web Interface Features

1. **Character Management** - Visual character creation with personality sliders
2. **Dialog Studio** - Real-time dialog generation with jungle-themed interface
3. **Project Organization** - Manage game projects and export settings
4. **Adventure Theme** - Animated jungle backgrounds with floating leaves

## 🏗️ Architecture

### Core Components

```
🌿 Game Dialog Generator
├── 🔧 Go API Server (main.go)
│   ├── Character management
│   ├── Dialog generation engine
│   ├── Project organization
│   └── Ollama/Qdrant integration
├── 🎮 Jungle-themed Web UI
│   ├── Interactive character creation
│   ├── Real-time dialog generation
│   └── Game project management
├── 💻 CLI Wrapper (bash)
│   ├── Character commands
│   ├── Dialog generation
│   └── Project management
└── 🗄️ Data Layer
    ├── PostgreSQL (characters, projects, dialog)
    ├── Qdrant (character embeddings)
    └── Sample jungle characters
```

### Resource Integration

- **PostgreSQL**: Persistent character and project data
- **Qdrant**: Character personality and scene context embeddings
- **Ollama**: Local LLM for dialog generation (llama3.2, nomic-embed-text)
- **Whisper**: Optional voice synthesis for character audio

## 🎭 Character System

### Personality Modeling

Characters are defined with:
- **Personality Traits**: Brave, humorous, loyal, etc. (0.0-1.0 scale)
- **Background Story**: Character history and motivations
- **Speech Patterns**: Vocabulary, tone, catchphrases
- **Voice Profile**: Pitch, speed, accent parameters
- **Relationships**: Dynamic connections with other characters

### Sample Jungle Characters

The scenario includes ready-to-use characters:
- **🐒 Kiko the Brave Monkey**: Fearless protagonist
- **🦉 Luna the Wise Owl**: Ancient mentor
- **🦏 Rocco the Gruff Rhino**: Tough ally
- **🐆 Zara the Sneaky Jaguar**: Mysterious anti-hero
- **🦜 Pip the Cheerful Toucan**: Comic relief
- **🐍 Dr. Venom the Snake**: Primary antagonist

## 🔌 API Reference

### Core Endpoints

#### Character Management
```http
POST /api/v1/characters
GET /api/v1/characters
GET /api/v1/characters/{id}
GET /api/v1/characters/{id}/personality
```

#### Dialog Generation
```http
POST /api/v1/dialog/generate
POST /api/v1/dialog/batch
```

#### Project Management
```http
POST /api/v1/projects
GET /api/v1/projects
```

### Example API Usage

```javascript
// Create a jungle character
const character = await fetch('/api/v1/characters', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    name: "Adventure Monkey",
    personality_traits: {
      brave: 0.9,
      humorous: 0.7,
      loyal: 0.8
    },
    background_story: "A fearless jungle explorer",
    voice_profile: {
      pitch: "medium-high",
      accent: "playful"
    }
  })
});

// Generate contextual dialog
const dialog = await fetch('/api/v1/dialog/generate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    character_id: character.character_id,
    scene_context: "Discovering a hidden temple",
    emotion_state: "excited"
  })
});
```

## 🎨 Jungle Platformer Theme

### Visual Design
- **Color Palette**: Rich greens, earth tones, bright accents
- **Animations**: Parallax jungle backgrounds, floating leaves
- **Typography**: Adventure-game inspired fonts
- **UI Elements**: Organic shapes, vine decorations

### Character Aesthetics
- **Animal Characters**: Emoji-based character avatars
- **Personality Cards**: Game-style character selection interface
- **Dialog Bubbles**: Leaf-shaped speech bubbles
- **Adventure Terminology**: Jungle-themed messages and labels

## 🧪 Testing & Validation

### Running Tests
```bash
# Run all scenario tests
vrooli scenario test game-dialog-generator

# Test specific components
vrooli scenario test game-dialog-generator --structure
vrooli scenario test game-dialog-generator --integration
```

### Validation Criteria
- ✅ Character consistency scoring > 80%
- ✅ Dialog generation < 2s response time
- ✅ All jungle theme elements present
- ✅ API endpoints functional
- ✅ CLI commands executable
- ✅ UI loads with proper theming

## 🔧 Configuration

### Environment Variables
```bash
# Server ports
API_PORT=8080
UI_PORT=3200

# Database connections
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=game_dialog_generator

# AI services
OLLAMA_URL=http://localhost:11434
QDRANT_URL=http://localhost:6333

# Optional voice synthesis
WHISPER_URL=http://localhost:9000
```

### Service Configuration
The `.vrooli/service.json` file defines:
- Resource dependencies and health checks
- Lifecycle management (setup, develop, test, stop)
- Port allocation and service discovery
- Jungle theme metadata

## 🎯 Game Engine Integration

### Supported Export Formats
- **Unity**: JSON dialog trees with character metadata
- **Unreal Engine**: BP-compatible data structures
- **Godot**: Resource files for dialog systems
- **Custom JSON**: Flexible format for any engine

### Integration Example (Unity)
```csharp
// Load generated character dialog
var dialogData = DialogGenerator.LoadCharacterDialog("character-id");
foreach (var line in dialogData.DialogLines) {
    Debug.Log($"{line.CharacterName}: {line.Content}");
    audioSource.PlayClipAtPoint(line.AudioClip, transform.position);
}
```

## 🌟 Advanced Features

### Character Relationships
Model dynamic relationships between characters that influence dialog:
- Friendship levels affect supportive dialog
- Rivalries create tension in interactions
- Mentor relationships enable teaching moments

### Emotion State Tracking
Characters maintain emotional states that influence dialog generation:
- Recent events affect character mood
- Personality traits modify emotional responses
- Scene context provides emotional cues

### Voice Synthesis
Optional character-specific voice generation:
- Personality-based voice parameters
- Consistent character audio across dialog
- Export audio files for game integration

## 🛠️ Development

### Project Structure
```
game-dialog-generator/
├── 📋 PRD.md                    # Product requirements
├── 🔧 api/                      # Go API server
│   ├── main.go                  # Main server logic
│   └── go.mod                   # Go dependencies
├── 💻 cli/                      # Command-line interface
│   ├── game-dialog-generator    # Main CLI script
│   └── install.sh              # CLI installation
├── 🎮 ui/                       # Web interface
│   ├── index.html              # Jungle-themed SPA
│   ├── server.js               # Node.js server
│   └── package.json            # UI dependencies
├── 🗄️ initialization/           # Database and data setup
│   ├── storage/postgres/        # Database schema
│   └── data/                   # Sample characters
├── 🧪 tests/                    # Test scenarios
└── 📖 docs/                     # Additional documentation
```

### Contributing

1. Follow the jungle platformer theme consistently
2. Maintain character consistency scoring above 80%
3. Add tests for new features
4. Update documentation with examples

## 🚀 Deployment

### Local Development
```bash
# Start all services
vrooli scenario run game-dialog-generator

# Development with hot reload
cd ui && npm run dev
```

### Production Deployment
The scenario supports containerized deployment with:
- Docker Compose for local production
- Kubernetes Helm charts
- Cloud provider templates (AWS, GCP, Azure)

## 🎮 Use Cases

### Indie Game Development
- Rapid character dialog prototyping
- Consistent character voice development
- Dynamic dialog for interactive narratives

### Game Studios
- Large-scale character dialog generation
- Voice acting script preparation
- Character consistency validation

### Interactive Fiction
- Branching dialog tree creation
- Character relationship modeling
- Narrative consistency checking

## 🌿 Jungle Adventure Continues...

The Game Dialog Generator brings the spirit of classic jungle platformers to modern game development. Create memorable characters, generate engaging dialog, and build the next great adventure game!

### Next Steps
1. 🐒 Create your first jungle character
2. 💬 Generate some adventure dialog
3. 🎮 Export to your favorite game engine
4. 🌟 Share your jungle adventure with the world!

---

**🌿 Ready to swing into action? Let's create some unforgettable game characters! 🎮**

For more information, see:
- [📋 Product Requirements (PRD.md)](./PRD.md)
- [🧪 Phased Tests (test/run-tests.sh)](./test/run-tests.sh)
- [🔧 API Documentation](./docs/api.md)
- [💻 CLI Reference](./docs/cli.md)

*Part of the Vrooli AI ecosystem - Building the future of intelligent automation* 🚀
