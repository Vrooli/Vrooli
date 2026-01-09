# Version Compatibility Matrix

> Reference for supported versions of tools and dependencies used in deployment workflows.

## Core Tools

| Tool | Minimum | Tested With | Required For | Notes |
|------|---------|-------------|--------------|-------|
| **Node.js** | 18.0.0 | 20.11.0, 22.0.0 | UI builds, Electron | LTS versions recommended |
| **Go** | 1.21 | 1.22.0, 1.23.0 | API, CLI, runtime builds | CGO_ENABLED=0 for cross-compilation |
| **pnpm** | 8.0.0 | 9.0.0 | All JS package management | npm/yarn not supported |

## Electron & Packaging

| Tool | Minimum | Tested With | Notes |
|------|---------|-------------|-------|
| **Electron** | 28.0.0 | 33.0.0 | Main process + Chromium |
| **electron-builder** | 24.0.0 | 26.0.0 | Installer generation |
| **electron-updater** | 6.0.0 | 6.3.0 | Auto-update functionality |

## Installer Outputs

| Platform | Format | Tool Version | Notes |
|----------|--------|--------------|-------|
| Windows | MSI | electron-builder 24+ | NSIS-based, supports silent install |
| Windows | EXE | electron-builder 24+ | Legacy fallback |
| macOS | PKG | electron-builder 24+ | Requires signing for distribution |
| macOS | DMG | electron-builder 24+ | Legacy fallback |
| Linux | AppImage | electron-builder 24+ | Portable, no install required |
| Linux | DEB | electron-builder 24+ | Debian/Ubuntu package manager |

## Bundle Runtime

| Component | Language | Minimum | Notes |
|-----------|----------|---------|-------|
| Runtime supervisor | Go | 1.21 | Static binary, no runtime deps |
| Runtime control API | Go | 1.21 | HTTP server on loopback |
| Manifest schema | JSON | v0.1 | Current desktop schema version |

## Database Compatibility

| Database | Tier 1 | Tier 2 (Desktop) | Tier 3 (Mobile) | Notes |
|----------|--------|------------------|-----------------|-------|
| PostgreSQL 14+ | Yes | Swap required | Swap required | Default for local dev |
| PostgreSQL 15+ | Yes | Swap required | Swap required | Recommended |
| SQLite 3.35+ | Via swap | Yes | Yes | Bundle-safe alternative |

## Infrastructure

| Service | Version | Tier 1 | Tier 2+ | Notes |
|---------|---------|--------|---------|-------|
| Redis 7+ | 7.0+ | Optional | Swap required | In-process cache for bundles |
| Ollama | Latest | Optional | Swap to models | Heavy; requires swap for desktop |
| Browserless | 2.0+ | Optional | Swap to Playwright | Chromium automation |

## Cross-Platform Build Requirements

### Building Windows Installers on Linux/macOS

| Tool | Version | Purpose |
|------|---------|---------|
| Wine | 8.0+ | Windows binary execution |
| mono | 6.0+ | .NET framework emulation |

### Building macOS Installers on Linux/Windows

| Requirement | Notes |
|-------------|-------|
| macOS machine or VM | Code signing requires macOS |
| Apple Developer account | For notarization |

### Building Linux Installers on Windows/macOS

| Tool | Version | Purpose |
|------|---------|---------|
| Docker | 20.0+ | Linux environment emulation |
| WSL2 (Windows) | - | Alternative to Docker |

## CLI Compatibility

| CLI | API Version | Notes |
|-----|-------------|-------|
| deployment-manager 1.0.x | v1 | Current stable |
| deployment-manager 1.1.x | v1 | Current development |

## API Version History

| API Version | Status | Introduced | Notes |
|-------------|--------|------------|-------|
| v1 | Current | Dec 2024 | All current endpoints |

## Manifest Schema Versions

| Schema | Target | Status | Notes |
|--------|--------|--------|-------|
| v0.1 | desktop | Current | Full service/secret/port support |
| v0.2 | desktop | Planned | Migration improvements |
| v0.1 | mobile | Planned | iOS/Android support |
| v0.1 | saas | Planned | Cloud deployment support |

---

## Checking Your Versions

```bash
# Node.js
node --version

# Go
go version

# pnpm
pnpm --version

# Electron (in project)
cd scenarios/<scenario>/platforms/electron
cat package.json | grep '"electron"'

# deployment-manager CLI
deployment-manager --version
```

## Known Compatibility Issues

### Node.js 21.x

Node.js 21.x (non-LTS) may have issues with certain native modules. Use LTS versions (18.x, 20.x, 22.x) for production builds.

### Go 1.20 and Earlier

Go 1.20 lacks some features used in the runtime supervisor. Upgrade to 1.21+ for full compatibility.

### electron-builder < 24

Older versions lack proper MSI support and have known issues with Apple Silicon (arm64) builds.

### SQLite < 3.35

Versions before 3.35 lack RETURNING clause support, which some migrations rely on.

---

**Related**: [Packaging Matrix](packaging-matrix.md) | [Desktop Workflow](../workflows/desktop-deployment.md)
