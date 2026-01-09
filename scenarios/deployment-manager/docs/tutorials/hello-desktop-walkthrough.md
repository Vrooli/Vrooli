# Hello Desktop Walkthrough

> **Step-by-step guide**: Deploy a scenario as a bundled desktop application.

This tutorial walks through the complete desktop deployment pipeline using `hello-desktop` as the reference scenario. By the end, you'll have distributable installers (`.exe`, `.dmg`/`.zip`, `.AppImage`/`.deb`) for Windows, macOS, and Linux.

## Prerequisites

- Vrooli environment configured (`vrooli info` works)
- Node.js 18+
- Go 1.21+ (for API binary compilation)
- deployment-manager running (`make start` in `scenarios/deployment-manager`)
- scenario-to-desktop running (`make start` in `scenarios/scenario-to-desktop`)

## Overview

The desktop deployment pipeline has 7 phases:

```
┌─────────────────────────────────────────────────────────────────┐
│  1. Create Profile  →  2. Validate  →  3. Assemble Manifest     │
│                                                                 │
│  4. Export Manifest  →  5. Build Binaries  →  6. Gen Electron   │
│                                                                 │
│  7. Build Installers  →  Distributable packages (.exe/.dmg/.deb)│
└─────────────────────────────────────────────────────────────────┘
```

## Step 1: Understand the Scenario Structure

First, examine the hello-desktop scenario:

```bash
cd $VROOLI_ROOT/scenarios/hello-desktop

# See the structure
tree -I 'node_modules|dist*|bin' .
```

Output:
```
.
├── api/
│   ├── main.go          # Go API with /health and /api/greet
│   └── go.mod
├── ui/
│   ├── index.html       # Static UI
│   ├── styles.css
│   └── app.js
├── platforms/
│   └── electron/        # Generated desktop wrapper (after step 6)
├── .vrooli/
│   └── service.json     # Lifecycle config with desktop metadata
├── bundle.json          # Bundle manifest (after step 3-4)
├── Makefile
└── README.md
```

### Key Files

**`api/main.go`** - Minimal Go API:
```go
func main() {
    port := os.Getenv("API_PORT")
    if port == "" { port = "8080" }

    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/api/greet", greetHandler)
    http.ListenAndServe(":"+port, nil)
}
```

**`.vrooli/service.json`** - Desktop deployment metadata:
```json
{
  "scenario": "hello-desktop",
  "services": {
    "api": {
      "build": {
        "type": "go",
        "source_dir": "api",
        "entry_point": ".",
        "output_pattern": "bin/{{platform}}/hello-desktop-api{{ext}}"
      }
    }
  }
}
```

## Step 2: Create a Deployment Profile

A profile captures tier-specific configuration (secrets, swaps, settings):

```bash
# Create a desktop (tier 2) profile
deployment-manager profile create hello-profile hello-desktop --tier 2
```

Output:
```
✓ Profile created: hello-profile
  Scenario: hello-desktop
  Tier:     2 (desktop)
  ID:       hello-profile
```

### View Profile Details

```bash
deployment-manager profile get hello-profile
```

Output:
```json
{
  "id": "hello-profile",
  "scenario": "hello-desktop",
  "tier": 2,
  "tier_name": "desktop",
  "created_at": "2025-01-15T10:30:00Z",
  "swaps": [],
  "secrets": {
    "configured": [],
    "required": []
  }
}
```

> **Note**: hello-desktop has no external dependencies, so there are no swaps (postgres→sqlite, redis→in-process) or secrets to configure.

## Step 3: Validate the Profile

Check that the scenario is ready for desktop deployment:

```bash
deployment-manager profile validate hello-profile
```

Output:
```
✓ Profile Validation: hello-profile

  Compatibility Checks:
    ✓ No blocking dependencies
    ✓ UI bundle exists
    ✓ API has build configuration
    ✓ No required secrets missing

  Fitness Score: 95/100
  Status: Ready for desktop deployment
```

## Step 4: Run the Full Pipeline (Dry Run First)

Preview what will happen:

```bash
deployment-manager deploy-desktop --profile hello-profile --dry-run
```

Output:
```
✓ Desktop Deployment: dry-run
  Profile:  hello-profile
  Scenario: hello-desktop

Steps:
  ✓ Load profile - loaded profile for scenario hello-desktop
  ✓ Validate profile - profile validation passed
  ⊘ Assemble manifest - dry run - would assemble manifest
  ⊘ Export manifest - dry run - would write manifest
  ⊘ Build binaries - dry run - would build 1 service(s)
  ⊘ Generate desktop wrapper - dry run - would generate Electron wrapper
  ⊘ Build platform installers - dry run - would build MSI/PKG/AppImage
```

## Step 5: Execute the Full Pipeline

Run the actual deployment:

```bash
deployment-manager deploy-desktop --profile hello-profile
```

This takes 3-10 minutes depending on your machine. Output:
```
✓ Desktop Deployment: success
  Profile:  hello-profile
  Scenario: hello-desktop
  Duration: 4m32s

Steps:
  ✓ Load profile - loaded profile for scenario hello-desktop
  ✓ Validate profile - profile validation passed
  ✓ Assemble manifest - assembled manifest with 0 swaps
  ✓ Export manifest - wrote manifest to /home/user/Vrooli/scenarios/hello-desktop/bundle.json
  ✓ Build binaries - built 1 service(s) for 5 platform(s)
  ✓ Generate desktop wrapper - generated Electron wrapper at .../platforms/electron
  ✓ Build platform installers - built installers for 3 platform(s)

Manifest: /home/user/Vrooli/scenarios/hello-desktop/bundle.json
Binaries: 5/5 succeeded
Desktop Wrapper: /home/user/Vrooli/scenarios/hello-desktop/platforms/electron

Installers:
  win:     .../dist-electron/Hello Desktop Setup 1.0.0.exe
  mac:     .../dist-electron/Hello Desktop-1.0.0-mac.zip
  linux:   .../dist-electron/Hello Desktop-1.0.0.AppImage

Next steps:
  $ Test on your platform by running the installer
  $ Upload to distribution channels (GitHub Releases, etc.)
```

## Step 6: Understand What Was Generated

### Bundle Manifest (`bundle.json`)

```bash
cat scenarios/hello-desktop/bundle.json
```

```json
{
  "schema_version": "v0.1",
  "target": "desktop",
  "app": {
    "name": "Hello Desktop",
    "version": "1.0.0"
  },
  "ipc": {
    "mode": "loopback-http",
    "host": "127.0.0.1",
    "port": 39200
  },
  "services": [
    {
      "id": "hello-desktop-api",
      "type": "api-binary",
      "binaries": {
        "linux-x64": { "path": "bin/linux-x64/hello-desktop-api" },
        "darwin-arm64": { "path": "bin/darwin-arm64/hello-desktop-api" },
        "win-x64": { "path": "bin/win-x64/hello-desktop-api.exe" }
      },
      "health": {
        "type": "http",
        "path": "/health",
        "port_name": "api"
      }
    },
    {
      "id": "ui",
      "type": "ui-bundle",
      "binaries": {
        "linux-x64": { "path": "ui/dist/index.html" }
      }
    }
  ]
}
```

### Compiled Binaries

```bash
ls -la scenarios/hello-desktop/platforms/electron/bundle/bin/
```

```
darwin-arm64/hello-desktop-api   8.4M
darwin-x64/hello-desktop-api     8.4M
linux-arm64/hello-desktop-api    8.4M
linux-x64/hello-desktop-api      8.4M
win-x64/hello-desktop-api.exe    8.5M
```

### Runtime Supervisor

```bash
ls -la scenarios/hello-desktop/platforms/electron/bundle/runtime/
```

```
linux/runtime       10M
linux/runtimectl    8.7M
mac/runtime         10M
mac/runtimectl      8.7M
win/runtime.exe     10M
win/runtimectl.exe  8.7M
```

### Electron Wrapper

```bash
ls scenarios/hello-desktop/platforms/electron/
```

```
assets/          # Icons, license
bundle/          # Binaries + manifest
dist/            # Compiled TypeScript
dist-electron/   # Installers
node_modules/
package.json
scripts/         # Build helpers (Wine setup, etc.)
src/             # Electron main.ts
tsconfig.json
```

### Final Installers

```bash
ls -la scenarios/hello-desktop/platforms/electron/dist-electron/
```

```
Hello Desktop Setup 1.0.0.exe          75M   # Windows NSIS installer
Hello Desktop-1.0.0-mac.zip           112M   # macOS app bundle
Hello Desktop-1.0.0-arm64-mac.zip     108M   # macOS Apple Silicon
Hello Desktop-1.0.0.AppImage           98M   # Linux portable
hello-desktop_1.0.0_amd64.deb          72M   # Debian package
latest.yml                                   # Auto-update manifest
latest-linux.yml
latest-mac.yml
```

## Step 7: Test the Desktop App

### On Linux

```bash
chmod +x "Hello Desktop-1.0.0.AppImage"
./Hello\ Desktop-1.0.0.AppImage
```

### On macOS

```bash
unzip "Hello Desktop-1.0.0-mac.zip"
open "Hello Desktop.app"
```

### On Windows

Double-click `Hello Desktop Setup 1.0.0.exe` and follow the installer.

## What Happens at Runtime

When the bundled desktop app launches:

1. **Electron starts** and reads `bundle/bundle.json`
2. **Runtime supervisor launches** from `bundle/runtime/{platform}/runtime`
3. **Runtime starts services** defined in the manifest (API binary)
4. **Electron waits** for health checks to pass on `127.0.0.1:39200`
5. **UI loads** in the Electron window, connecting to the local API
6. **App is ready** - fully offline, self-contained

## Partial Pipeline Runs

### Skip Installer Builds (Just Generate Wrapper)

```bash
deployment-manager deploy-desktop --profile hello-profile --skip-installers
```

### Skip Everything Except Manifest

```bash
deployment-manager deploy-desktop --profile hello-profile --skip-build --skip-packaging
```

### Build for Specific Platforms

```bash
deployment-manager deploy-desktop --profile hello-profile --platforms win,linux
```

## Troubleshooting

### "scenario-to-desktop not available"

Ensure the scenario-to-desktop API is running:
```bash
cd $VROOLI_ROOT/scenarios/scenario-to-desktop
make start
make logs  # Check for errors
```

### "UI bundle not found"

Build the UI first:
```bash
cd $VROOLI_ROOT/scenarios/hello-desktop/ui
# For hello-desktop, the UI is static HTML - just ensure it exists
ls index.html
```

### "Wine not installed" (Windows builds on Linux)

The build scripts auto-install Wine, but you can do it manually:
```bash
cd $VROOLI_ROOT/scenarios/hello-desktop/platforms/electron
node scripts/setup-wine.js
```

### Binary compilation fails

Check Go is installed and in PATH:
```bash
go version  # Should be 1.21+
```

## Next Steps

- **Add auto-updates**: Configure `publish` in `package.json` to enable electron-updater
- **Code signing**: See [Code Signing Guide](../guides/code-signing.md)
- **Custom branding**: Edit `assets/icon.png` and rebuild
- **Add your scenario**: Copy hello-desktop's structure for your own scenario

## Related Documentation

- [Desktop Deployment Guide](../DESKTOP-DEPLOYMENT-GUIDE.md) - Detailed reference
- [Bundle Manifest Schema](../guides/bundle-manifest-schema.md) - Full schema docs
- [Cross-Platform Builds](../../scenario-to-desktop/docs/CROSS_PLATFORM_BUILDS.md) - Installer formats
- [Deployment Commands](../cli/deployment-commands.md) - CLI reference
