# 🖥️ Scenario-to-Desktop

Transform Vrooli scenarios into professional native desktop applications using Electron and other modern frameworks. Built as part of the Vrooli AI intelligence platform, this scenario provides a complete pipeline for converting web-based AI scenarios into standalone desktop software.

## 🎯 Overview

scenario-to-desktop is a **permanent intelligence capability** that enables any Vrooli scenario to become a professional desktop application. Unlike simple web wrappers, this system generates truly native desktop experiences with OS integration, offline capability, and professional distribution channels.

### Core Value Proposition

- **🚀 Instant Desktop Apps**: Convert any scenario to desktop in minutes, not months
- **💼 Professional Quality**: Code signing, auto-updates, native menus, system integration
- **🌍 Cross-Platform**: Windows, macOS, and Linux from a single generation
- **⚡ Multiple Frameworks**: Electron (primary), Tauri, Neutralino support
- **🎨 Template Variety**: Basic, Advanced, Multi-Window, and Kiosk mode applications
- **🛠️ Complete Toolchain**: Generation, building, testing, packaging, and distribution
- **📊 Scenario Inventory**: NEW - View all scenarios and their desktop deployment status
- **📁 Standardized Structure**: NEW - All desktop apps go to `platforms/electron/` for consistency

## 🏗️ Architecture

```
scenario-to-desktop/
├── 📄 PRD.md                          # Product Requirements Document
├── 📄 README.md                       # This documentation
├── ⚙️  .vrooli/service.json           # Service configuration
├── 🔧 api/                           # Go API server (dynamic port 15000-19999)
├── 💻 cli/                           # Command-line interface
├── 🌐 ui/                            # Web management interface (dynamic port 35000-39999)
├── 🎨 templates/                     # Desktop app templates
│   ├── vanilla/                     # Base Electron templates
│   ├── advanced/                    # Specialized template configurations
│   └── build-tools/                 # Template generation system
├── 🤖 prompts/                       # AI agent prompts for creation/debugging
└── 🔄 initialization/               # N8n workflows and automation
```

## 🚀 Quick Start

### 1. Installation

```bash
# Install the CLI
cd scenarios/scenario-to-desktop/cli
./install.sh

# Start the API server
cd ../api
make run

# Start the web UI (optional)
cd ../ui
npm install && npm start
```

### 2. Discover Scenarios

```bash
# View all scenarios and their desktop status
# Open the web UI at http://localhost:<UI_PORT>
# Click "Scenario Inventory" tab to see:
# - Which scenarios have desktop versions
# - Which are built and ready to distribute
# - One-click generation for scenarios without desktop support
```

### 3. Generate Your First Desktop App

**🚀 NEW: One-Click Quick Generate** - The system now automatically detects scenario configuration!

```bash
# Method 1: Using the Web UI (Easiest - Recommended!)
# 1. Open http://localhost:<UI_PORT>
# 2. Go to "Scenario Inventory" tab
# 3. Find your scenario and click "Generate Desktop"
# 4. Watch real-time build progress
# 5. Desktop app is auto-configured and ready!

# The system automatically:
# - Reads scenario's .vrooli/service.json for metadata
# - Detects UI port, API port, display name, version
# - Validates that ui/dist/ is built
# - Copies UI files to electron wrapper
# - Generates fully-configured desktop app

# Method 2: Using the API directly
curl -X POST http://localhost:<API_PORT>/api/v1/desktop/generate/quick \
  -H "Content-Type: application/json" \
  -d '{"scenario_name": "picker-wheel", "template_type": "basic"}'

# Method 3: Using the CLI (manual config)
scenario-to-desktop generate picker-wheel

# The desktop wrapper will be created at:
# scenarios/picker-wheel/platforms/electron/

# Advanced generation with custom options
scenario-to-desktop generate picker-wheel \
  --framework electron \
  --template advanced \
  --platforms win,mac,linux
```

### 4. Build Desktop Packages

**🚀 NEW: One-Click Building from UI**

```bash
# Method 1: Using the Web UI (Easiest!)
# 1. Open http://localhost:<UI_PORT>
# 2. Go to "Scenario Inventory" tab
# 3. Find scenario with "Desktop" badge
# 4. Click "Build Desktop App" button
# 5. Watch real-time build progress
# 6. Download buttons appear when ready!

# Method 2: Using the API directly
curl -X POST http://localhost:<API_PORT>/api/v1/desktop/build/picker-wheel \
  -H "Content-Type: application/json" \
  -d '{"platforms": ["win"]}'

# Method 3: Manual build in desktop wrapper
cd scenarios/picker-wheel/platforms/electron
npm install
npm run build
npm run dist:win    # Build Windows executable
```

**Build Process**:
1. Installs npm dependencies in electron wrapper
2. Compiles TypeScript (main.ts, preload.ts)
3. Packages with electron-builder for target platform(s)
4. Creates installers in `dist-electron/`:
   - Windows: `<name>-Setup-<version>.exe`
   - macOS: `<name>-<version>.dmg`
   - Linux: `<name>-<version>.AppImage`, `<name>-<version>.deb`

### 5. Download and Test on Windows

**🪟 Testing from Your Windows Laptop**

```bash
# Step 1: Access scenario-to-desktop through app-monitor proxy
1. Connect to your secure tunnel
2. Navigate to scenario-to-desktop UI
3. Go to Scenario Inventory

# Step 2: Download the Windows executable
1. Find scenario with "Built" badge
2. See 🪟 Windows download button
3. Click to download .exe file
4. Save to your Windows machine

# Step 3: Install and test
1. Run the downloaded Setup.exe
2. Windows SmartScreen: Click "More info" → "Run anyway"
3. Follow installation wizard
4. Launch the installed desktop app
5. Verify all functionality works
```

**Download URL Pattern**:
```
http://localhost:<API_PORT>/api/v1/desktop/download/<scenario-name>/<platform>

Examples:
- Windows: /api/v1/desktop/download/picker-wheel/win
- macOS:   /api/v1/desktop/download/picker-wheel/mac
- Linux:   /api/v1/desktop/download/picker-wheel/linux
```

### 6. Development and Testing

```bash
# Development mode (local testing on server)
cd scenarios/picker-wheel/platforms/electron
npm run dev              # Launch with DevTools

# Build for distribution
npm run dist             # Current platform only
npm run dist:win         # Windows executable
npm run dist:mac         # macOS DMG
npm run dist:linux       # Linux AppImage
npm run dist:all         # All platforms
```

## 💼 Use Cases & Examples

### Simple Utilities
**Template: Basic** | **Framework: Electron**
- `picker-wheel` → Random selection tool
- `qr-code-generator` → QR code creator
- `palette-gen` → Color palette generator
- `notes` → Simple note-taking app

### Professional Tools  
**Template: Advanced** | **Framework: Electron**
- `system-monitor` → System monitoring dashboard
- `document-manager` → Document management system
- `research-assistant` → AI research tool
- `personal-digital-twin` → AI assistant application

### Complex Workflows
**Template: Multi-Window** | **Framework: Electron**
- `agent-dashboard` → Multi-agent management interface
- `mind-maps` → Mind mapping with multiple canvases
- `brand-manager` → Brand management with multiple views
- `campaign-content-studio` → Content creation workspace

### Kiosk & Embedded
**Template: Kiosk** | **Framework: Electron**
- Information displays for conferences/retail
- Point-of-sale systems
- Interactive museum exhibits
- Industrial control panels

## 🎨 Templates Deep Dive

### Basic Template
Perfect for simple utilities and tools:
- ✅ Native menus and keyboard shortcuts
- ✅ Auto-updater integration
- ✅ File operations (save/open dialogs)
- ✅ System notifications
- ✅ Single window interface
- 🎯 **Use for**: Utilities, calculators, simple productivity tools

### Advanced Template  
Full-featured professional applications:
- ✅ Everything from Basic template
- ✅ System tray integration
- ✅ Global keyboard shortcuts
- ✅ Rich context menus
- ✅ Background operation
- ✅ Advanced OS integration
- 🎯 **Use for**: System tools, professional software, background services

### Multi-Window Template
Complex applications with multiple interfaces:
- ✅ Everything from Advanced template
- ✅ Multiple window management
- ✅ Inter-window communication
- ✅ Floating tool panels
- ✅ Window state persistence
- ✅ Advanced workflow support
- 🎯 **Use for**: IDEs, dashboards, design tools, complex workflows

### Kiosk Template
Full-screen applications for dedicated hardware:
- ✅ Full-screen lock mode
- ✅ Security hardening
- ✅ Remote monitoring
- ✅ Auto-restart capabilities
- ✅ Screensaver integration
- ✅ Unattended operation
- 🎯 **Use for**: Public displays, point-of-sale, industrial controls

## 🛠️ Development Workflow

### 1. Template Generation
The system analyzes your scenario and generates:
- **Electron main process** (`main.ts`) - App lifecycle and window management
- **Preload script** (`preload.ts`) - Secure renderer-main communication
- **Splash screen** (`splash.html`) - Professional startup experience
- **Package configuration** (`package.json`) - Dependencies and build setup
- **TypeScript config** (`tsconfig.json`) - Compilation settings

### 2. Server Integration
Desktop apps integrate with scenarios through multiple patterns:
- **Node.js Server**: Fork existing Express/Fastify servers
- **Static Files**: Load pre-built SPA applications
- **External API**: Connect to cloud/remote services
- **Executable**: Bundle and manage compiled backends (Go, Rust, Python)

### 3. Build Pipeline
Automated cross-platform building:
```bash
npm run build      # Compile TypeScript
npm run dist       # Package for distribution
npm run dist:all   # Build for all platforms
```

### 4. Testing & Validation
Comprehensive testing suite:
- Package structure validation
- Dependency verification
- UI screenshot testing (via Browserless)
- Platform compatibility checks
- Performance profiling

### 5. Distribution
Professional deployment options:
- **App Stores**: Microsoft Store, Mac App Store, Snap Store
- **Direct Download**: Standalone installers
- **Enterprise**: MSI/PKG packages with silent install
- **Auto-updates**: Seamless version management

## 📁 Standardized File Structure

All desktop applications are generated to a consistent location:

```
scenarios/<scenario-name>/
├── api/                    # Go API server
├── cli/                    # Command-line interface
├── ui/                     # React web application
│   └── dist/              # Built web app (required for desktop)
└── platforms/              # Deployment targets
    └── electron/           # Desktop wrapper (generated)
        ├── main.ts        # Electron main process
        ├── preload.ts     # Secure IPC bridge
        ├── splash.html    # Splash screen
        ├── package.json   # Desktop dependencies
        ├── tsconfig.json  # TypeScript config
        ├── assets/        # Platform icons
        ├── dist/          # Compiled TypeScript
        └── dist-electron/ # Built packages
```

**Why this structure?**
- ✅ Predictable location for all desktop versions
- ✅ Easy to check "does this scenario have desktop?"
- ✅ Won't clutter scenario root when adding iOS/Android later
- ✅ Separates deployment concerns from source code
- ✅ CI/CD can easily find and build desktop versions

## 🌐 API Reference

### REST Endpoints

#### System Status
```http
GET /api/v1/health                       # Health check
GET /api/v1/status                       # System information
GET /api/v1/templates                    # Available templates
GET /api/v1/scenarios/desktop-status     # NEW: All scenarios and desktop status
```

#### Desktop Operations
```http
POST /api/v1/desktop/generate                      # Generate desktop app (manual config)
POST /api/v1/desktop/generate/quick                # 🆕 Quick generate with auto-detection
GET  /api/v1/desktop/status/{id}                   # Build/generation status
POST /api/v1/desktop/build/{scenario_name}         # 🆕 Build desktop packages
GET  /api/v1/desktop/download/{scenario}/{platform} # 🆕 Download built package
POST /api/v1/desktop/build                         # Build project (legacy)
POST /api/v1/desktop/test                          # Test functionality
POST /api/v1/desktop/package                       # Package for distribution
```

#### Quick Generate (NEW!)
Auto-detects scenario configuration and generates desktop app:

```http
POST /api/v1/desktop/generate/quick

Request:
{
  "scenario_name": "picker-wheel",
  "template_type": "basic"  // optional, defaults to "basic"
}

Response:
{
  "build_id": "uuid",
  "status": "building",
  "scenario_name": "picker-wheel",
  "desktop_path": ".../scenarios/picker-wheel/platforms/electron",
  "detected_metadata": {
    "name": "picker-wheel",
    "display_name": "Picker Wheel",
    "description": "Random selection wheel application",
    "version": "1.0.0",
    "has_ui": true,
    "ui_dist_path": ".../scenarios/picker-wheel/ui/dist",
    "api_port": 15000,
    "ui_port": 35000
  },
  "status_url": "/api/v1/desktop/status/{build_id}"
}
```

**Auto-Detection Features:**
- Reads `.vrooli/service.json` for metadata
- Reads `ui/package.json` for additional info
- Validates `ui/dist/` exists and is built
- Detects if scenario has API
- Sets sensible defaults for all config
- Copies UI files automatically

#### Build Desktop App (NEW!)
Build executable packages for a scenario that has a desktop wrapper:

```http
POST /api/v1/desktop/build/{scenario_name}

Request Body (optional):
{
  "platforms": ["win"],  // optional: win, mac, linux (defaults to all)
  "clean": false         // optional: clean before building
}

Response:
{
  "build_id": "uuid",
  "status": "building",
  "scenario": "picker-wheel",
  "desktop_path": ".../scenarios/picker-wheel/platforms/electron",
  "platforms": ["win"],
  "status_url": "/api/v1/desktop/status/{build_id}"
}
```

**Build Process**:
1. Runs `npm install` to install dependencies
2. Runs `npm run build` to compile TypeScript
3. Runs `npm run dist:win` (or dist:mac, dist:linux) to package
4. Creates installers in `dist-electron/` directory
5. Typical build time: 3-8 minutes depending on platforms

**Note**: Building Windows executables on Linux requires wine, which is installed automatically by electron-builder.

#### Download Built Package (NEW!)
Download the built executable for a specific platform:

```http
GET /api/v1/desktop/download/{scenario_name}/{platform}

Parameters:
- scenario_name: Name of the scenario (e.g., "picker-wheel")
- platform: One of: "win", "mac", "linux"

Response:
- Content-Type: application/x-msdownload (for .exe)
- Content-Type: application/x-apple-diskimage (for .dmg)
- Content-Type: application/x-executable (for .AppImage)
- Content-Disposition: attachment; filename=<installer-file>

File Downloads:
- Windows: picker-wheel-Setup-1.0.0.exe
- macOS: picker-wheel-1.0.0.dmg
- Linux: picker-wheel-1.0.0.AppImage
```

**Example Usage**:
```bash
# Download Windows installer
curl -O http://localhost:${API_PORT}/api/v1/desktop/download/picker-wheel/win

# Or open in browser to trigger download
open http://localhost:${API_PORT}/api/v1/desktop/download/picker-wheel/win
```

#### Scenario Discovery (NEW)
```http
GET /api/v1/scenarios/desktop-status

Response:
{
  "scenarios": [
    {
      "name": "picker-wheel",
      "display_name": "picker-wheel-desktop",
      "has_desktop": true,
      "desktop_path": ".../platforms/electron",
      "version": "1.0.0",
      "platforms": ["win", "mac", "linux"],
      "built": true,
      "package_size": 47185920,
      "last_modified": "2025-11-14 15:30:00"
    }
  ],
  "stats": {
    "total": 130,
    "with_desktop": 5,
    "built": 3,
    "web_only": 125
  }
}
```

### Example Generation Request
```json
{
  "app_name": "picker-wheel",
  "app_display_name": "Picker Wheel Desktop",
  "app_description": "Random selection wheel application",
  "version": "1.0.0",
  "author": "Your Name",
  "framework": "electron",
  "template_type": "basic",
  "platforms": ["win", "mac", "linux"],
  "output_path": "",
  "features": {
    "splash": true,
    "autoUpdater": true,
    "systemTray": false
  }
}
```

**Note**: Leave `output_path` empty to use the standard location `scenarios/<app_name>/platforms/electron/`. This is the recommended approach for consistency.

## 💻 CLI Commands

### Core Commands
```bash
scenario-to-desktop help                    # Show help
scenario-to-desktop version                 # Show version
scenario-to-desktop status                  # System status
scenario-to-desktop templates               # List templates
```

### Generation & Building
```bash
scenario-to-desktop generate <scenario>     # Generate desktop app
scenario-to-desktop build <path>            # Build application
scenario-to-desktop test <path>             # Test functionality  
scenario-to-desktop package <path>          # Package for distribution
```

### Advanced Options
```bash
--framework electron|tauri|neutralino       # Choose framework
--template basic|advanced|multi_window|kiosk # Choose template
--platforms win,mac,linux                   # Target platforms
--output ./path                            # Output directory
--config config.json                       # Use config file
```

## 🌍 Web Interface

Access the web management interface via the dynamically allocated UI port. Check the port with `vrooli scenario status scenario-to-desktop`:

- **🎛️ Generation Dashboard**: Visual template selection and configuration
- **📊 Build Monitoring**: Real-time build status and logs
- **📋 Template Browser**: Explore available templates and features
- **📈 System Statistics**: Build success rates and usage metrics

**Example**: If UI_PORT is allocated as 39689, access at `http://localhost:39689`

## 🔄 Integration & Automation

### N8n Workflow
Automated desktop build pipeline via `initialization/n8n/desktop-build-automation.json`:
1. Validates build requests
2. Generates applications using templates
3. Installs dependencies and builds TypeScript
4. Packages for target platforms
5. Performs UI testing via Browserless
6. Sends completion notifications
7. Handles error cases gracefully

### Cross-Scenario Integration
scenario-to-desktop enhances these scenarios:
- **system-monitor** → Native desktop system monitoring
- **document-manager** → Desktop file management with native integration
- **personal-digital-twin** → Offline-capable AI assistant
- **research-assistant** → Desktop research tool with file access
- **agent-dashboard** → Multi-window agent management interface

## 🔧 Configuration

### Environment Variables
```bash
# API Configuration
API_PORT=${API_PORT}              # API server port (allocated from range 15000-19999)
API_BASE_URL=http://localhost:${API_PORT}

# UI Configuration
UI_PORT=${UI_PORT}                # Web interface port (allocated from range 35000-39999)
NODE_ENV=production               # Environment mode

# Build Configuration
DESKTOP_BUILD_TIMEOUT=600000      # Build timeout (ms)
BROWSERLESS_URL=http://localhost:3000  # Testing service (if browserless resource enabled)
```

### Service Configuration (`.vrooli/service.json`)
```json
{
  "name": "scenario-to-desktop",
  "version": "1.0.0",
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999",
      "description": "Desktop build API server port"
    },
    "ui": {
      "env_var": "UI_PORT",
      "range": "35000-39999",
      "description": "Desktop build UI server port"
    }
  }
}
```

## 🧪 Testing

### Running Tests
```bash
# API tests
cd api && make test

# Template validation
cd templates/build-tools && npm test

# CLI integration tests  
cd cli && ./test.sh

# End-to-end testing
scenario-to-desktop test ./test-app --headless
```

### Test Coverage
- ✅ Template generation validation
- ✅ Cross-platform build testing
- ✅ API endpoint validation
- ✅ CLI command testing
- ✅ UI functionality testing
- ✅ Desktop app integration testing

## 🔒 Security

### Template Security
- Context isolation enabled by default
- Node integration disabled in renderer
- Strict Content Security Policy
- IPC channel validation
- Input sanitization

### Distribution Security
- Code signing support (requires certificates)
- Automated security scanning
- Update verification
- Sandbox mode support
- Permission minimization

## 📊 Monitoring & Analytics

### Build Metrics
- Build success/failure rates
- Average build times
- Template usage statistics
- Platform distribution
- Error frequency analysis

### Performance Monitoring
- Desktop app startup times
- Memory usage patterns
- Resource utilization
- User engagement metrics
- Update adoption rates

## 🚨 Troubleshooting

### Common Issues

**Build Failures**
```bash
# Check Node.js version
node --version  # Requires 18+

# Verify dependencies
npm install

# Check build tools
which electron-builder
```

**Template Issues**
```bash
# Validate template syntax  
scenario-to-desktop templates

# Test template generation
scenario-to-desktop generate test-app --output /tmp/test
```

**API Connection**
```bash
# Check service status and find allocated ports
vrooli scenario status scenario-to-desktop

# Check API health (replace ${API_PORT} with actual allocated port)
curl http://localhost:${API_PORT}/api/v1/health

# Or use the CLI status command
scenario-to-desktop status
```

### Debug Mode
```bash
# Enable verbose logging
scenario-to-desktop generate my-app --verbose

# API debug mode
cd api && DEBUG=* make run

# Template debug
export DEBUG_TEMPLATES=true
```

## 🔮 Roadmap

### v1.1 - Enhanced Frameworks
- Complete Tauri integration
- Neutralino template support
- Flutter Desktop exploration
- Performance optimizations

### v1.2 - Advanced Features
- Plugin architecture
- Custom template creation
- Visual template builder
- Advanced debugging tools

### v1.3 - Enterprise Features
- Fleet management dashboard
- Enterprise security policies
- Bulk deployment tools
- Analytics and reporting

## 🤝 Contributing

### Development Setup
```bash
# Clone and setup
git clone <repo>
cd scenarios/scenario-to-desktop

# Install CLI
./cli/install.sh

# Start API server
cd api && make run

# Start UI server  
cd ui && npm install && npm start
```

### Adding Templates
1. Create template configuration in `templates/advanced/`
2. Update template generation logic
3. Add template tests
4. Update documentation

### Code Style
- Go: `gofmt` and `go vet`
- TypeScript: `prettier` and `eslint`
- Shell: `shellcheck`
- Markdown: `markdownlint`

## 📚 Related Documentation

- [PRD.md](./PRD.md) - Comprehensive product requirements
- [Templates README](./templates/README.md) - Template system details
- [API Documentation](./api/README.md) - REST API reference
- [CLI Reference](./cli/README.md) - Command-line usage
- [Build Tools](./templates/build-tools/README.md) - Generation system

## 💡 Examples Gallery

### Generated Desktop Apps
- **Picker Wheel Desktop** - Random selection with native animations
- **QR Generator Pro** - QR code creation with file export
- **System Monitor Plus** - Real-time system monitoring dashboard
- **Mind Map Studio** - Multi-window mind mapping application

### Template Showcases
- **Basic**: Simple, clean interfaces for utilities
- **Advanced**: Rich system integration for professional tools
- **Multi-Window**: Complex workflows with multiple panels
- **Kiosk**: Full-screen applications for dedicated hardware

## 🔗 Links

- **Homepage**: https://vrooli.com/scenarios/scenario-to-desktop
- **Documentation**: https://docs.vrooli.com/scenarios/scenario-to-desktop
- **API Reference**: Check allocated port via `vrooli scenario status scenario-to-desktop`
- **Web Interface**: Check allocated port via `vrooli scenario status scenario-to-desktop`
- **GitHub Issues**: https://github.com/vrooli/vrooli/issues
- **Community**: https://discord.gg/vrooli

---

**Built with ❤️ by the [Vrooli Platform](https://vrooli.com)**

*scenario-to-desktop is part of Vrooli's recursive intelligence system, where every capability built becomes a permanent tool for building even more advanced capabilities. Each desktop app generated contributes to the ever-expanding intelligence of the platform.*

**Version**: 1.0.0 | **Status**: Production Ready | **License**: MIT