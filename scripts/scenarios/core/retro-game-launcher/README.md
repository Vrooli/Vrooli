# 🎮 Retro Game Launcher

**AI-Powered Retro Game Creation Platform with Custom React UI**

A stunning example of Vrooli's scenario-based approach featuring a completely custom retro-themed user interface, Go API backend, and full CLI suite. This scenario demonstrates how Vrooli scenarios can have their own unique visual designs and architectures beyond the standard Windmill-based templates.

## ✨ Features

### 🎨 **Custom Retro UI**
- **Single-file React application** with Tailwind CSS
- **Neon cyberpunk aesthetic** with CRT scanlines, glitch effects, and retro animations
- **Fully responsive design** optimized for both desktop and mobile
- **Three main sections**: Browse Games, AI Game Generator, Featured Games
- **Real-time search** and filtering capabilities
- **Game modal** with detailed information and play tracking

### 🚀 **Go API Backend**
- **RESTful API** built with Gorilla Mux for high performance
- **PostgreSQL integration** for game metadata and user data
- **AI game generation** endpoints (integrates with n8n workflows)
- **Search and discovery** with full-text search capabilities
- **Health monitoring** with service status checks
- **CORS enabled** for seamless frontend integration

### 🖥️ **Command Line Interface**
- **Beautiful retro-styled CLI** with ANSI color support and ASCII art
- **Complete game management**: list, search, create, play, and view games
- **AI game generation** from command line prompts
- **Template suggestions** for quick game creation
- **Service status monitoring** and health checks
- **BATS test suite** for CLI validation

### 🧠 **AI Integration**
- **n8n workflow integration** for AI-powered game generation
- **Ollama model support** (CodeLlama, Llama3.2) for code generation
- **Multiple game engines**: JavaScript, PICO-8, TIC-80
- **Template-driven prompts** for consistent game creation

## 🏗️ Architecture

This scenario showcases Vrooli's flexible architecture approach:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   React UI      │    │   Go API        │    │   CLI Tool      │
│   Port 3000     │────│   Port 8080     │────│   Global        │
│   Tailwind CSS  │    │   REST API      │    │   Command       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                  ┌─────────────────────────────┐
                  │      Infrastructure         │
                  │  ┌─────────┐ ┌─────────┐    │
                  │  │PostgreSQL│ │   n8n   │    │
                  │  │Database │ │Workflows│    │
                  │  └─────────┘ └─────────┘    │
                  │  ┌─────────┐ ┌─────────┐    │
                  │  │ Ollama  │ │ Qdrant  │    │
                  │  │AI Models│ │ Vector  │    │
                  │  └─────────┘ └─────────┘    │
                  └─────────────────────────────┘
```

## 🚀 Quick Start

### 1. Setup
```bash
./scripts/manage.sh setup
```

### 2. Start Services
```bash
./scripts/manage.sh develop
```

### 3. Access the Application
- **Web UI**: http://localhost:3000 (Custom React app)
- **API**: http://localhost:8080 (Go backend)
- **CLI**: `retro-game-launcher --help`

## 🎯 Usage Examples

### Web Interface
1. Open http://localhost:3000
2. Browse existing games or click "Generate Game"
3. Enter a prompt like: "Create a simple platformer with a blue character"
4. Select engine (JavaScript, PICO-8, TIC-80)
5. Click "Generate Game" and wait for AI creation

### Command Line
```bash
# Show available games
retro-game-launcher list

# Search for games
retro-game-launcher search "platformer"

# Generate a new game
retro-game-launcher generate "Create a space shooter with neon graphics"

# Get prompt templates
retro-game-launcher templates

# Check system status
retro-game-launcher status
```

### API Direct Access
```bash
# Health check
curl http://localhost:8080/health

# List games
curl http://localhost:8080/api/games

# Search games
curl "http://localhost:8080/api/search/games?q=space"

# Generate game
curl -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Create a retro snake game", "engine": "javascript"}'
```

## 🎨 Design Philosophy

This scenario demonstrates **visual excellence** as a core Vrooli principle:

### Retro Cyberpunk Aesthetic
- **Neon color palette**: Cyan, magenta, yellow, green
- **Typography**: Orbitron font for that futuristic feel
- **Effects**: Glitch animations, CRT scanlines, glow effects
- **Layout**: Grid-based design with retro spacing

### Responsive & Accessible
- **Mobile-first design** with desktop enhancements
- **High contrast** neon colors for visibility
- **Keyboard navigation** support throughout
- **Screen reader friendly** with proper ARIA labels

### Performance Optimized
- **Single-file React app** for minimal load times
- **Tailwind CSS** for optimal styling performance
- **Go backend** for lightning-fast API responses
- **Efficient database queries** with proper indexing

## 🧪 Testing

Run the complete test suite:

```bash
./test.sh
```

This will test:
- ✅ Go API compilation and functionality
- ✅ CLI script execution and commands
- ✅ React UI structure and dependencies
- ✅ BATS test suite for CLI validation

## 📁 File Structure

```
retro-game-launcher/
├── .vrooli/
│   └── service.json           # Lifecycle configuration
├── api/                       # Go API backend
│   ├── main.go               # API server implementation
│   ├── go.mod                # Go dependencies
│   └── go.sum                # Go dependency locks
├── cli/                       # Command line interface
│   ├── retro-game-launcher   # Main CLI script
│   ├── install-cli.sh        # CLI installation script
│   └── retro-game-launcher.bats # BATS test suite
├── ui/                        # React frontend
│   ├── public/
│   │   └── index.html        # HTML template
│   ├── src/
│   │   ├── App.js            # Main React application
│   │   ├── index.js          # React entry point
│   │   └── index.css         # Tailwind + custom styles
│   ├── package.json          # NPM dependencies
│   ├── tailwind.config.js    # Tailwind configuration
│   └── postcss.config.js     # PostCSS configuration
├── initialization/            # Database and workflow setup
│   ├── automation/n8n/       # n8n workflow definitions
│   └── storage/postgres/     # Database schema and seeds
├── deployment/
│   └── startup.sh            # Legacy startup (deprecated)
├── test.sh                   # Main test script
└── README.md                 # This file
```

## 🔧 Development

### Prerequisites
- Go 1.21+
- Node.js 18+
- PostgreSQL 13+
- Ollama with models
- n8n for AI workflows

### Environment Variables
- `RETRO_API_URL`: API base URL (default: http://localhost:8080)
- `SERVICE_PORT`: API port (auto-allocated)
- `UI_PORT`: React dev server port (auto-allocated)
- `POSTGRES_URL`: Database connection string
- `N8N_BASE_URL`: n8n instance URL
- `OLLAMA_URL`: Ollama API URL

### Custom Styling
The UI uses a custom Tailwind configuration with:
- **Neon colors**: `neon-cyan`, `neon-pink`, `neon-green`, etc.
- **Retro backgrounds**: `retro-dark`, `retro-purple`, `retro-blue`
- **Custom animations**: `pulse-neon`, `glow`, `scan`, `flicker`
- **Special components**: `.neon-button`, `.game-card`, `.glitch-text`

## 🎯 Scenario Conversion Benefits

This transformation from Windmill-based to custom UI demonstrates:

1. **Visual Freedom**: Complete control over design and user experience
2. **Performance**: Optimized single-file React app loads faster
3. **Technology Choice**: Use any tech stack (Go + React vs. Python + Windmill)
4. **Scalability**: Direct API access enables mobile apps, integrations
5. **Maintenance**: Standard web development practices and tooling

## 🚀 Future Enhancements

- **Real-time multiplayer** gaming support
- **Game IDE integration** with live coding
- **Social features** (user profiles, game sharing, ratings)
- **Advanced AI features** (image generation, sound synthesis)
- **Mobile app** using the same Go API
- **Game publishing** to retro console formats

---

**This scenario exemplifies Vrooli's vision**: scenarios aren't just templates, they're complete business applications with unlimited customization potential. The retro game launcher proves that with Vrooli, you can build anything from simple automations to complex, visually stunning applications.