# Desktop Application Templates

This directory contains the universal Electron templates and build tools for generating desktop applications from Vrooli scenarios.

## Directory Structure

```
templates/
├── vanilla/                          # Base Electron template files
│   ├── main.ts                       # Electron main process
│   ├── preload.ts                    # Secure IPC bridge
│   ├── splash.html                   # Splash screen
│   ├── package.json.template         # Desktop package.json template
│   ├── tsconfig.json                 # TypeScript configuration
│   └── README.md                     # Generated app README
│
├── advanced/                         # Template configurations
│   ├── basic-app.json               # Basic desktop app config
│   ├── advanced-app.json            # Advanced features config
│   ├── multi-window.json            # Multi-window app config
│   └── kiosk-mode.json              # Kiosk/fullscreen config
│
├── build-tools/                      # Template generation system
│   ├── template-generator.ts        # Main generator script
│   ├── config-validator.ts          # Configuration validator
│   ├── package.json                 # Build tools dependencies
│   └── tsconfig.json                # Build tools TS config
│
├── service-lifecycle-addition.json   # Lifecycle template for scenarios
└── README.md                         # This file
```

## Template Types

### Basic Template
**Use for**: Simple utilities, tools, single-purpose apps

**Features**:
- Single window interface
- Native menus and keyboard shortcuts
- File operations (open/save dialogs)
- System notifications
- Auto-updater integration

**Best for**: picker-wheel, qr-code-generator, palette-gen, notes

### Advanced Template
**Use for**: Professional applications with system integration

**Features**:
- All Basic template features
- System tray integration
- Global keyboard shortcuts
- Rich context menus
- Background operation
- Advanced OS integration

**Best for**: system-monitor, document-manager, research-assistant

### Multi-Window Template
**Use for**: Complex applications needing multiple views

**Features**:
- All Advanced template features
- Multiple window management
- Inter-window communication
- Floating tool panels
- Window state persistence
- Advanced workflow support

**Best for**: agent-dashboard, mind-maps, campaign-content-studio

### Kiosk Template
**Use for**: Full-screen dedicated hardware deployments

**Features**:
- Full-screen lock mode
- Security hardening
- Remote monitoring
- Auto-restart capabilities
- Screensaver integration
- Unattended operation

**Best for**: Public displays, point-of-sale, industrial controls

## How It Works

### 1. Template Selection
The API or CLI receives a generation request with:
- Scenario name
- Template type (basic/advanced/multi_window/kiosk)
- Configuration options

### 2. File Generation
The template-generator.ts:
1. Loads the selected template configuration from `advanced/`
2. Copies base files from `vanilla/`
3. Performs variable substitution (`{{APP_NAME}}`, etc.)
4. Generates platform-specific assets (icons, splash screens)
5. Creates package.json with correct dependencies
6. Outputs to `<scenario>/platforms/electron/`

### 3. Variable Substitution
Templates use Mustache-style variables:
- `{{APP_NAME}}` - Scenario name (e.g., "picker-wheel")
- `{{APP_DISPLAY_NAME}}` - Human-readable name
- `{{VERSION}}` - App version
- `{{SERVER_TYPE}}` - How to load the UI (static/node/external)
- `{{SERVER_PORT}}` - Development server port
- `{{SCENARIO_DIST_PATH}}` - Path to built UI files

### 4. Standard Output Location
All desktop apps are generated to:
```
scenarios/<scenario-name>/platforms/electron/
```

This keeps deployments organized and makes it easy to add future platforms (iOS, Android, etc.)

## Adding New Templates

1. **Create template config** in `advanced/`:
```json
{
  "name": "my-template",
  "description": "Template description",
  "features": {
    "splash": true,
    "systemTray": false,
    "multiWindow": false
  },
  "window": {
    "width": 1200,
    "height": 800
  }
}
```

2. **Update template-generator.ts** to handle new features

3. **Add template metadata** to API's template list

4. **Test generation**:
```bash
scenario-to-desktop generate test-scenario --template my-template
```

## Development

### Building the Generator
```bash
cd build-tools
npm install
npm run build
```

### Testing Template Generation
```bash
# Generate test app
node build-tools/dist/template-generator.js \
  --config test-config.json

# Verify output
ls -la /tmp/test-desktop-app/
```

### Modifying Templates
1. Edit files in `vanilla/`
2. Update template configs in `advanced/`
3. Rebuild generator: `cd build-tools && npm run build`
4. Test with a real scenario

## Integration with Scenarios

After generating a desktop app, scenarios can add lifecycle hooks:

```json
// Add to .vrooli/service.json
{
  "lifecycle": {
    "build-desktop": {
      "steps": [
        { "run": "cd platforms/electron && npm install" },
        { "run": "cd platforms/electron && npm run dist" }
      ]
    }
  }
}
```

See `service-lifecycle-addition.json` for complete template.

## Security Considerations

### Sandboxing
- Renderer processes run with `nodeIntegration: false`
- `contextIsolation: true` enabled by default
- IPC communication through secure preload script

### Input Validation
- All user-provided paths validated in generator
- Path traversal protection in template-generator.ts
- Whitelisted template types in API

### Code Signing
Templates include signing configuration for:
- macOS code signing and notarization
- Windows Authenticode signing
- Linux AppImage signing (optional)

## Resources

- **Electron Documentation**: https://www.electronjs.org/docs
- **Electron Builder**: https://www.electron.build/
- **Desktop Integration Guide**: ../docs/desktop-integration-guide.md
- **Main README**: ../README.md

## Troubleshooting

### Template generation fails
- Check that `vanilla/` files are intact
- Verify template config JSON is valid
- Ensure output path is writable

### Icons not generating
- Verify `assets/` directory exists in output
- Check that icon source files are valid PNG
- Ensure imagemagick or sharp is installed

### Variable substitution not working
- Check template config has all required variables
- Verify variable names match exactly (case-sensitive)
- Look for unclosed `{{}}` brackets in template files

---

**Maintained by**: Vrooli Platform Team
**Last Updated**: 2025-11-14
