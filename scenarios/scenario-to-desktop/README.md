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

## 🏗️ Architecture

```
scenario-to-desktop/
├── 📄 PRD.md                          # Product Requirements Document
├── 📄 README.md                       # This documentation
├── ⚙️  .vrooli/service.json           # Service configuration
├── 🔧 api/                           # Go API server (port 3202)
├── 💻 cli/                           # Command-line interface
├── 🌐 ui/                            # Web management interface (port 3203)
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

### 2. Generate Your First Desktop App

```bash
# Generate desktop app for picker-wheel scenario
scenario-to-desktop generate picker-wheel

# Advanced generation with options
scenario-to-desktop generate picker-wheel \
  --framework electron \
  --template advanced \
  --platforms win,mac,linux \
  --output ./picker-wheel-desktop
```

### 3. Build and Test

```bash
# Navigate to generated app
cd ./picker-wheel-desktop

# Development mode
npm run dev

# Build for distribution
npm run dist
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

## 🌐 API Reference

### REST Endpoints

#### System Status
```http
GET /api/v1/health          # Health check
GET /api/v1/status          # System information
GET /api/v1/templates       # Available templates
```

#### Desktop Operations
```http
POST /api/v1/desktop/generate      # Generate desktop app
GET  /api/v1/desktop/status/{id}   # Build status
POST /api/v1/desktop/build         # Build project  
POST /api/v1/desktop/test          # Test functionality
POST /api/v1/desktop/package       # Package for distribution
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
  "output_path": "./desktop-app",
  "features": {
    "splash": true,
    "autoUpdater": true,
    "systemTray": false
  }
}
```

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

Access the web management interface at `http://localhost:3203`:

- **🎛️ Generation Dashboard**: Visual template selection and configuration
- **📊 Build Monitoring**: Real-time build status and logs
- **📋 Template Browser**: Explore available templates and features
- **📈 System Statistics**: Build success rates and usage metrics

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
PORT=3202                    # API server port
API_BASE_URL=http://localhost:3202

# UI Configuration  
UI_PORT=3203                # Web interface port
NODE_ENV=production         # Environment mode

# Build Configuration
DESKTOP_BUILD_TIMEOUT=600000    # Build timeout (ms)
BROWSERLESS_URL=http://localhost:3000  # Testing service
```

### Service Configuration (`.vrooli/service.json`)
```json
{
  "name": "scenario-to-desktop",
  "version": "1.0.0",
  "services": {
    "api": { "enabled": true, "port": 3202 },
    "cli": { "enabled": true, "binary": "scenario-to-desktop" },
    "ui": { "enabled": true, "port": 3203 }
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
# Check API health
curl http://localhost:3202/api/v1/health

# Verify service status
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
- **API Reference**: http://localhost:3202/api/v1/status
- **Web Interface**: http://localhost:3203
- **GitHub Issues**: https://github.com/vrooli/vrooli/issues
- **Community**: https://discord.gg/vrooli

---

**Built with ❤️ by the [Vrooli Platform](https://vrooli.com)**

*scenario-to-desktop is part of Vrooli's recursive intelligence system, where every capability built becomes a permanent tool for building even more advanced capabilities. Each desktop app generated contributes to the ever-expanding intelligence of the platform.*

**Version**: 1.0.0 | **Status**: Production Ready | **License**: MIT