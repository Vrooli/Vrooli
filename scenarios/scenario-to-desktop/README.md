# 🖥️ Scenario-to-Desktop

> **Two modes available:**
> - **Thin Client** (UI only) - Bundles UI; connects to running Tier 1 server
> - **Bundled App** (offline) - Full offline package with UI + API + runtime. Use `deployment-manager deploy-desktop` for the automated pipeline.

Transform Vrooli scenarios into desktop applications. scenario-to-desktop generates Electron wrappers that can either connect to a remote Vrooli server (thin client) or run completely offline with bundled services (bundled mode).

## Bundled Desktop Apps (Recommended)

For complete offline desktop applications, use the **deployment-manager** orchestration:

```bash
# Create a deployment profile
deployment-manager profile create my-profile my-scenario --tier 2

# Build everything (binaries, Electron wrapper, installers)
deployment-manager deploy-desktop --profile my-profile
```

This handles:
- Bundle manifest generation with dependency swaps
- Cross-compilation of API binaries for all platforms
- Electron wrapper generation
- Platform installers (Windows/macOS/Linux)
- Runtime supervisor bundling

See [Hello Desktop Tutorial](../deployment-manager/docs/tutorials/hello-desktop-walkthrough.md) for a complete walkthrough.

## Thin Client Mode (Connect to Server)

For UI-only builds that connect to a running Vrooli server:

**Limitations:**
- **UI-only bundles** – copies `ui/dist` assets into the Electron wrapper
- **Requires server** – API and resources must run elsewhere
- **Secrets on server** – auth/session traffic forwards to running scenario
- **No offline mode** – requires network connection to server

## Thin-Client Workflow (connect to your Vrooli server)

> **New:** The scenario-to-desktop UI now generates wrappers, runs `npm install/build/dist` for Windows/macOS/Linux, and ingests telemetry directly. The manual steps below are still useful for debugging or when you need to script the process yourself.

Thin clients are just remote controls for the full Vrooli stack that is already running your scenario. Until bundling lands, desktop builds must follow this routine:

1. **Confirm `vrooli` exists on the host running the scenario.** Run `vrooli --version`. If missing, install it and run `./scripts/manage.sh setup --yes yes` once.
2. **Start the scenario** with `vrooli scenario start <name>` (or `make start`). Wait until `vrooli scenario status <name>` reports healthy.
3. **Expose the scenario.**
   - LAN: use `http://hostname:${UI_PORT}/`.
   - Remote/mobile: proxy through `app-monitor` + Cloudflare and copy the exact proxy URL (for example `https://app-monitor.<domain>/apps/<scenario>/proxy/`).
4. **Point the desktop wrapper at that proxy URL.** The generator UI and CLI now capture a single `proxy_url`, show detected suggestions, and ship a "Test connection" button so you can confirm it responds before building. Keep `DEPLOYMENT_MODE=external-server` so telemetry and deployment-manager know the UI/API still live on your server.
5. **Distribute, collect telemetry, and clean up.** Ship the installer, ask testers for their `deployment-telemetry.jsonl`, upload it through the scenario-to-desktop UI (or with `scenario-to-desktop telemetry collect` if you prefer the CLI), then stop the remote scenario with `vrooli scenario stop <name>` when you’re done.

Deployment-manager will eventually automate these steps (detecting/installing `vrooli`, starting/stopping scenarios, and swapping dependencies), but documenting the pipeline now keeps Tier 2 expectations realistic.

> **Server bootstrap is opt-in.** Keep `auto_manage_vrooli` off (default) if your desktop build should connect to the remote Vrooli server you already host. Set `auto_manage_vrooli: true` only when you expect the desktop user to run the scenario locally and have (or be willing to install) the `vrooli` CLI.

### Guided build experience in Scenario Inventory

The Scenario Inventory dashboard mirrors the workflow above so nobody has to remember the checklist:

1. **Connect to your Tier 1 scenario** — paste the Cloudflare/app-monitor URL once and the generator reuses it for every teammate. A quick “Test connection” button confirms the proxy responds before you regenerate the wrapper.
2. **Build installers** — pick Windows/macOS/Linux chips and click “Build selected installers.” The service runs `npm install`, `npm run build`, and `npm run dist` for those platforms while streaming status per platform.
3. **Download + share telemetry** — previously built installers stay visible even after a refresh, and the telemetry panel now lists the exact `%APPDATA%` / `~/Library/Application Support/` / `~/.config/` paths so you can upload `deployment-telemetry.jsonl` without leaving the UI.

That UI-first loop eliminates the “refresh and rebuild” trap, keeps non-Electron experts oriented, and gives deployment-manager consistent telemetry for every desktop experiment.

### Optional: Let the desktop app start the scenario locally

Set `auto_manage_vrooli: true` when calling `POST /api/v1/desktop/generate` (or when editing the generated config JSON) to let Electron stand up the scenario automatically:

```json
{
  "app_name": "picker-wheel",
  "template_type": "universal",
  "auto_manage_vrooli": true,
  "deployment_mode": "external-server"
}
```

With this flag enabled the wrapper:

1. Looks for the `vrooli` CLI, prompting for a path when it cannot auto-detect it.
2. Runs `vrooli setup --yes yes --skip-sudo yes` once per machine.
3. Executes `vrooli scenario start <name>` when your desktop app launches and `vrooli scenario stop <name>` when it exits (if the wrapper started it).

Leave the flag off to deliver the traditional thin client that simply targets whatever URL you bake into `SERVER_PATH`/`API_ENDPOINT`.

### Deployment Telemetry

Every generated Electron wrapper records lifecycle events (`app_start`, `dependency_unreachable`, `local_vrooli_start_failed`, etc.) to `deployment-telemetry.jsonl` inside the OS-specific user data directory (`%APPDATA%/<App Name>/` on Windows, `~/Library/Application Support/<App Name>/` on macOS, and `~/.config/<App Name>/` on Linux). Use the new CLI helper to ingest those logs:

```bash
scenario-to-desktop telemetry collect \
  --scenario picker-wheel \
  --file "$HOME/Library/Application Support/Picker Wheel/deployment-telemetry.jsonl"
```

The API stores the events under `.vrooli/deployment/telemetry/picker-wheel.jsonl`, giving deployment-manager and scenario-dependency-analyzer a single source of truth for how thin clients behave in the wild.

### Installer outputs and updater channels

**Default installer formats** (optimized for cross-platform builds on Linux):

| Platform | Format | Extension | Notes |
|----------|--------|-----------|-------|
| **Linux** | AppImage | `.AppImage` | Portable, runs on any distro |
| **Linux** | DEB | `.deb` | Debian/Ubuntu package |
| **Windows** | NSIS | `Setup.exe` | Standard installer, works via Wine |
| **macOS** | ZIP | `.zip` | Contains `.app` bundle, user drags to Applications |

> **Why these formats?** NSIS and ZIP can be built on Linux via Wine, enabling single-machine cross-platform builds. For DMG/PKG/MSI installers, use macOS/Windows CI runners. See [Cross-Platform Builds Guide](docs/CROSS_PLATFORM_BUILDS.md) for details.

- **Channel intent**: Auto-update hooks remain off by default. When you wire a publish target, stick to three channels (`dev`, `beta`, `stable`) and publish per-platform artifacts with signatures; the runtime/Electron wrapper should only enable updates when a channel URL and signing material are configured.
- **Bundled mode impact**: Offline bundles will initially rely on manual installer refreshes; differential updates stay on the roadmap. Until then, treat each installer as a full reinstall and keep telemetry enabled so deployment-manager can flag upgrade pain.

### Generator UI Upgrades

- **Deployment intent picker** highlights Thin Client (ready today) vs Cloud API / Bundled (stubs). Selecting anything other than Thin Client surfaces a "coming soon" warning so builders don’t think offline bundles ship yet.
- **Server strategy select** lets you choose between external, static, embedded Node, or executable launches. External remains the golden path; the others stay available for experiments.
- **Proxy connection group** captures the Cloudflare/app-monitor URL right inside the Web UI, shows detected hints, and lets you test the proxy before building. You can also opt-in to have the desktop wrapper run `vrooli setup/start/stop` per build.
- **Scenario inventory button** now expands into the same mini wizard, so "Generate Desktop" can’t happen without forcing a conscious deployment choice.

## 📚 Documentation
- Docs manifest for UI tab: `docs/manifest.json`
- Start here: `docs/QUICKSTART.md` and `docs/deployment-modes.md`
- **Cross-platform builds: `docs/CROSS_PLATFORM_BUILDS.md`** - Build formats, Wine setup, CI/CD recommendations
- Builds and troubleshooting: `docs/build-and-packaging.md`, `docs/DEBUGGING_WINDOWS.md`, `docs/WINE_INSTALLATION.md`
- Feature cookbook: `docs/desktop-integration-guide.md`
- Telemetry/ops: `docs/telemetry.md`

## 🎯 Overview

scenario-to-desktop is a **permanent intelligence capability** for packaging scenarios. In v1 it focuses on rapid thin clients with native menus and distribution scaffolding. Offline capability, auto-updates, and bundled resources are roadmap items coordinated through the deployment hub.

### Core Value Proposition

- **🚀 Instant Desktop Apps**: Convert any scenario to desktop in minutes, not months (as a thin client today)
- **💼 Professional Quality**: Native menus, OS integration, and future support for code signing/auto-updates
- **🌍 Cross-Platform**: Windows, macOS, and Linux from a single generation
- **⚡ Frameworks**: Electron thin clients today; other frameworks are future stubs
- **🎨 Template Variety**: Basic, Advanced, Multi-Window, and Kiosk mode applications
- **🛠️ Complete Toolchain**: Generation, building, testing, packaging, and (future) distribution automation
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

1) **Start the scenario with lifecycle**  
```bash
cd scenarios/scenario-to-desktop
make start        # preferred; or: vrooli scenario start scenario-to-desktop
```

2) **Open the UI → Scenario Inventory**  
Paste the LAN or Cloudflare/app-monitor proxy URL for the target scenario, keep `Deployment Mode = Thin Client`, pick platforms (Win/macOS/Linux), and click **Generate Desktop**.

3) **Download installers + telemetry**  
Built artifacts stay listed after refresh; collect `deployment-telemetry.jsonl` via the UI or `scenario-to-desktop telemetry collect`.

4) **Stop when done**  
```bash
make stop   # or: vrooli scenario stop scenario-to-desktop
```

CLI-only/manual config example:
```bash
scenario-to-desktop generate <scenario> \
  --deployment-mode external-server \
  --server-type external \
  --server-url https://app-monitor.<domain>/apps/<scenario>/proxy/ \
  --api-url https://app-monitor.<domain>/apps/<scenario>/proxy/api/ \
  --auto-manage-vrooli=false
```

Full walkthroughs: see `docs/QUICKSTART.md` and `docs/deployment-modes.md`.
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
npm run dist:win    # Build Windows MSI installer
```

**Build Process**:
1. Installs npm dependencies in electron wrapper
2. Compiles TypeScript (main.ts, preload.ts)
3. Packages with electron-builder for target platform(s)
4. Creates installers in `dist-electron/`:
   - Windows: `<name>-<version>.msi`
   - macOS: `<name>-<version>.pkg`
   - Linux: `<name>-<version>.AppImage`, `<name>-<version>.deb`

### 5. Windows builds (pointer)

- For Windows builds on Linux, follow `docs/WINE_INSTALLATION.md` for a no-sudo Wine setup.
- For runtime/build troubleshooting on Windows, see `docs/DEBUGGING_WINDOWS.md`.

### 6. Development and Testing

```bash
# Development mode (local testing on server)
cd scenarios/picker-wheel/platforms/electron
npm run dev              # Launch with DevTools

# Build for distribution
npm run dist             # Current platform only
npm run dist:win         # Windows MSI installer
npm run dist:mac         # macOS PKG installer
npm run dist:linux       # Linux AppImage
npm run dist:all         # All platforms
```

## 🔌 Connecting Desktop Builds to Running Scenarios

scenario-to-desktop currently assumes you already have the scenario running on a Vrooli server. Before handing a binary to anyone, make sure:

1. The target scenario is running inside your existing environment (e.g., `vrooli scenario start picker-wheel`).
2. The scenario's UI/API are reachable from the desktop machine (either via the app-monitor + Cloudflare tunnel or an SSH tunnel you expose yourself).
3. Any required resources (Postgres, Ollama, Redis, etc.) are online in that environment.

If a tester double-clicks the installer while the remote scenario is down, the desktop wrapper will error after its 30-second server check timeout.

### APP_CONFIG Reference

Every generated desktop wrapper writes an `APP_CONFIG` block near the top of `platforms/electron/src/main.ts` (copied from [`templates/vanilla/main.ts`](templates/vanilla/main.ts)). The most important fields today:

```ts
const APP_CONFIG = {
  APP_DISPLAY_NAME: "Picker Wheel",
  APP_URL: "https://picker-wheel.vrooli.dev",   // Used in menus/help
  SERVER_TYPE: "external",                      // 'external' | 'static' | 'node' | 'executable'
  DEPLOYMENT_MODE: "external-server",           // external-server | cloud-api | bundled
  SERVER_PATH: "https://app-monitor.vrooli.dev/apps/picker-wheel/", // Remote UI/API base URL
  SERVER_PORT: 48001,                            // Used when SERVER_TYPE !== 'external'
  API_ENDPOINT: "https://app-monitor.vrooli.dev/api/picker-wheel" // Optional convenience
};
```

- `SERVER_TYPE="external"` is the default and is the only fully-supported mode right now; it tells the wrapper to open whatever URL you put in `SERVER_PATH`.
- `SERVER_TYPE="static" | "node" | "executable"` are scaffolding hooks for the future deployment-manager flow. They exist so we can eventually bake UI bundles, Go APIs, or compiled binaries directly into the desktop package, but they are not wired up yet.
- `DEPLOYMENT_MODE` expresses intent (`external-server`, `cloud-api`, or `bundled`). Thin clients stay on `external-server` so telemetry and deployment-manager know the UI/API still live on your Vrooli server.
- `SERVER_PATH` doubles as either the remote URL (`external`) or the relative path inside the packaged app (`static`/`node`/`executable`).
- `API_ENDPOINT` is surfaced to preload scripts/helper menus so you can expose “open API docs” buttons; it does **not** magically proxy API calls—your UI code must still call the correct backend URL.

Whenever you regenerate a desktop wrapper these values are refreshed from `.vrooli/service.json`. If you need to point an existing build to a new server endpoint (e.g., staging vs production), edit `platforms/electron/src/main.ts` before rebuilding.

### Thin-Client Telemetry

Every generated Electron app now logs newline-delimited JSON telemetry to the user data directory (e.g., `%APPDATA%/Picker Wheel/deployment-telemetry.jsonl` on Windows, `~/Library/Application Support/Picker Wheel/deployment-telemetry.jsonl` on macOS, `~/.config/Picker Wheel/deployment-telemetry.jsonl` on Linux). Events include `app_start`, `external_server_mode`, `server_ready`, `dependency_unreachable`, `app_ready`, `startup_error`, and `app_shutdown`, all tagged with `DEPLOYMENT_MODE`. deployment-manager and scenario-dependency-analyzer can consume this file to see how often thin clients fail to reach their servers.

### Vrooli Server Automation Settings

- `AUTO_MANAGE_TIER1` → `true` enables CLI automation. When enabled (and `DEPLOYMENT_MODE === "external-server"`) the Electron app will locate the `vrooli` binary, run `vrooli setup --yes yes --skip-sudo yes`, start the scenario, and stop it on exit.
- `SCENARIO_NAME` → used to determine which `vrooli scenario start/stop` commands to run. Defaults to the scenario's `service.name`.
- `VROOLI_BINARY_PATH` → optional override when `vrooli` is not on `PATH`. If left empty, the desktop wrapper will search the system and, failing that, prompt the user to select the CLI.

### Making your Vrooli server reachable

Most teams expose their Vrooli instance via the `app-monitor` scenario and a Cloudflare tunnel:

```bash
# On the Vrooli server
vrooli scenario start app-monitor

# Find the public URL (Cloudflare subdomain)
vrooli scenario status app-monitor

# Use that URL in APP_CONFIG.SERVER_PATH, e.g.
SERVER_PATH: "https://<org>-app-monitor.trycloudflare.com/apps/picker-wheel/"
```

For ad-hoc testing you can forward a local port instead:

```bash
# From your laptop, tunnel app-monitor over SSH
ssh -L 4444:localhost:37842 user@server

# Set SERVER_PATH to http://localhost:4444/apps/picker-wheel/
```

Until deployment-manager lands, **desktop builds = glorified browsers** that rely on one of these reachability patterns.


## 💼 Use Cases & Examples

### Simple Utilities
**Template: Universal (Default)** | **Framework: Electron**
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

### Template Selection Guide

**🎯 Which template should I use?**

- **Universal (Default)**: Use for 95% of scenarios - works for any web app
- **Advanced**: Only if you need system tray or global shortcuts
- **Multi-Window**: Only if you need multiple separate windows (IDEs, dashboards)
- **Kiosk**: Only for dedicated hardware/public displays

**When in doubt, use Universal!** The system auto-detects your scenario configuration and applies the universal template, which works perfectly for most use cases.

### Universal Template (Default)
The universal wrapper that works for any scenario:
- ✅ Native menus and keyboard shortcuts
- ✅ Auto-updater integration
- ✅ File operations (save/open dialogs)
- ✅ System notifications
- ✅ Professional splash screen
- ✅ Single window interface
- ✅ Clean, minimal design
- 🎯 **Use for**: ANY scenario that needs desktop deployment
- 📊 **Examples**: picker-wheel, qr-code-generator, palette-gen, nutrition-tracker

### Advanced Template
Full-featured professional applications:
- ✅ Everything from Universal template
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

**Why `platforms/` folder?**
- ✅ Predictable location for all deployment types
- ✅ Easy to check "does this scenario have desktop?"
- ✅ Won't clutter scenario root when adding more platforms
- ✅ Separates deployment concerns from source code
- ✅ CI/CD can easily find and build platform versions
- ✅ Future-proof for iOS, Android, browser extensions, etc.

**Future Platform Organization** (when implemented):
```
scenarios/<scenario-name>/
└── platforms/
    ├── electron/     # Desktop (Windows, macOS, Linux)
    ├── ios/          # iOS mobile app (future)
    ├── android/      # Android mobile app (future)
    └── chrome-ext/   # Browser extensions (future)
```

All platform-specific wrappers live under `platforms/` to keep the scenario root clean and organized.

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

**Note**: Building Windows MSI installers on Linux still requires wine, which is installed automatically by electron-builder.

#### Download Built Package (NEW!)
Download the built executable for a specific platform:

```http
GET /api/v1/desktop/download/{scenario_name}/{platform}

Parameters:
- scenario_name: Name of the scenario (e.g., "picker-wheel")
- platform: One of: "win", "mac", "linux"

Response:
- Content-Type: application/x-msi (for .msi)
- Content-Type: application/octet-stream (for .pkg)
- Content-Type: application/x-executable (for .AppImage)
- Content-Disposition: attachment; filename=<installer-file>

File Downloads:
- Windows: picker-wheel-1.0.0.msi
- macOS: picker-wheel-1.0.0.pkg
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
- **[Desktop Wrapper Guide](./templates/DESKTOP_WRAPPER_GUIDE.md) - Universal wrapper principles and patterns** ⭐ **NEW**
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

**Version**: 1.0.0 | **Status**: Thin-client only (bundled/cloud modes stubbed) | **License**: MIT
