# Hello Desktop

> **Minimal scenario for validating the desktop deployment pipeline.**

This scenario has no external dependencies (no postgres, redis, ollama, etc.) and serves as the reference implementation for desktop deployment. It demonstrates the complete bundled desktop workflow with working installers.

**For a complete step-by-step walkthrough, see: [Hello Desktop Tutorial](../deployment-manager/docs/tutorials/hello-desktop-walkthrough.md)**

## Purpose

- Validate the full `deploy-desktop` workflow end-to-end
- Serve as a template for desktop-ready scenarios
- Test binary cross-compilation (Go → 5 platforms)
- Test Electron wrapper generation
- Test installer builds (Windows NSIS, macOS ZIP, Linux AppImage/DEB)

## Structure

```
hello-desktop/
├── api/
│   ├── main.go          # Minimal Go API (health + greet endpoints)
│   └── go.mod
├── ui/
│   ├── index.html       # Single-page vanilla HTML/CSS/JS UI
│   └── dist/            # Built UI bundle
├── platforms/
│   └── electron/        # Generated Electron wrapper
│       ├── bundle/      # Bundled binaries + runtime
│       ├── dist-electron/  # Built installers (.exe, .AppImage, etc.)
│       └── src/main.ts  # Electron main process
├── .vrooli/
│   └── service.json     # v2.0 lifecycle config with desktop metadata
├── bundle.json          # Bundle manifest
├── Makefile
└── README.md
```

## Quick Start

```bash
# Build and start locally (Tier 1)
make build
make start

# Check status
make status

# Test the API
curl http://localhost:$(vrooli scenario port hello-desktop API_PORT)/health
```

## Desktop Deployment

```bash
# Full deployment pipeline
deployment-manager profile create hello-profile hello-desktop --tier 2
deployment-manager deploy-desktop --profile hello-profile

# Or dry-run first
deployment-manager deploy-desktop --profile hello-profile --dry-run
```

### Pre-built Installers

After running `deploy-desktop`, find installers at:

```
platforms/electron/dist-electron/
├── Hello Desktop Setup 1.0.0.exe     # Windows
├── Hello Desktop-1.0.0-mac.zip       # macOS (x64 + arm64)
├── Hello Desktop-1.0.0.AppImage      # Linux portable
└── hello-desktop_1.0.0_amd64.deb     # Debian/Ubuntu
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check (JSON) |
| `/api/greet?name=X` | GET | Returns greeting message |

## Why This Scenario Exists

1. **No external dependencies** - Can deploy without postgres, redis, etc.
2. **Simple build** - Single Go binary, static HTML UI
3. **Fast iteration** - Quick builds for testing pipeline changes
4. **Reference implementation** - Copy this structure for new desktop-ready scenarios
5. **Validated pipeline** - Full end-to-end build has been tested with working installers

## Related Documentation

- [Hello Desktop Tutorial](../deployment-manager/docs/tutorials/hello-desktop-walkthrough.md) - Step-by-step walkthrough
- [deploy-desktop CLI](../deployment-manager/docs/cli/deployment-commands.md#deploy-desktop) - Command reference
- [Bundle Manifest Schema](../deployment-manager/docs/guides/bundle-manifest-schema.md) - bundle.json format
