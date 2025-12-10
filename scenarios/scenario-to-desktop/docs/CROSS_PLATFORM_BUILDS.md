# Cross-Platform Desktop Builds

This guide explains how to build Electron desktop applications for Windows, macOS, and Linux from a single machine.

## Build Compatibility Matrix

The following table shows which installer formats can be built on which host operating system:

| Target Platform | Format | Build on Linux | Build on macOS | Build on Windows | Notes |
|-----------------|--------|----------------|----------------|------------------|-------|
| **Linux** | AppImage | Native | Via Docker | Via Docker/WSL | Portable, no install needed |
| **Linux** | DEB | Native | Via Docker | Via Docker/WSL | Debian/Ubuntu package |
| **Linux** | RPM | Native | Via Docker | Via Docker/WSL | Fedora/RHEL package |
| **Windows** | NSIS (.exe) | Via Wine | Native | Native | Standard installer, recommended |
| **Windows** | MSI | Via Wine (unstable) | Native | Native | Enterprise installer, Wine issues |
| **Windows** | Portable | Via Wine | Native | Native | No install, single .exe |
| **macOS** | ZIP | Native | Native | Via Docker | Contains .app bundle |
| **macOS** | DMG | macOS only | Native | macOS only | Disk image with drag-to-install |
| **macOS** | PKG | macOS only | Native | macOS only | Installer package |

### Recommended Formats for Cross-Platform Builds on Linux

| Platform | Recommended Format | Why |
|----------|-------------------|-----|
| Linux | AppImage + DEB | Native builds, wide compatibility |
| Windows | NSIS (.exe) | Works reliably via Wine |
| macOS | ZIP | Works on Linux, contains full .app bundle |

## Installer Format Deep Dive

### Linux Formats

#### AppImage
- **Extension**: `.AppImage`
- **Install method**: Download, make executable (`chmod +x`), run
- **Pros**: No installation, runs on any distro, self-contained
- **Cons**: Larger file size, no system integration by default
- **Best for**: Portable apps, users who don't want to install

```bash
# User runs:
chmod +x MyApp-1.0.0.AppImage
./MyApp-1.0.0.AppImage
```

#### DEB (Debian Package)
- **Extension**: `.deb`
- **Install method**: `sudo dpkg -i` or double-click in GUI
- **Pros**: System integration, desktop shortcuts, auto-updates via apt
- **Cons**: Debian/Ubuntu only
- **Best for**: Debian, Ubuntu, Linux Mint users

```bash
# User runs:
sudo dpkg -i myapp_1.0.0_amd64.deb
# Or with dependency resolution:
sudo apt install ./myapp_1.0.0_amd64.deb
```

#### RPM (Red Hat Package)
- **Extension**: `.rpm`
- **Install method**: `sudo rpm -i` or `sudo dnf install`
- **Pros**: System integration for RHEL-based distros
- **Cons**: Fedora/RHEL/CentOS only
- **Best for**: Enterprise Linux, Fedora users

```bash
# User runs:
sudo dnf install myapp-1.0.0.x86_64.rpm
```

### Windows Formats

#### NSIS (.exe Installer)
- **Extension**: `Setup.exe` or just `.exe`
- **Install method**: Double-click, follow wizard
- **Pros**: Widely recognized, customizable, works via Wine on Linux
- **Cons**: Some enterprise environments prefer MSI
- **Best for**: Consumer apps, general distribution

**Build on Linux:**
```bash
npm run dist:win  # Automatically sets up Wine if needed
```

#### MSI (Windows Installer)
- **Extension**: `.msi`
- **Install method**: Double-click or `msiexec /i`
- **Pros**: Enterprise-friendly, Group Policy deployment, silent install
- **Cons**: WiX toolset has Wine compatibility issues
- **Best for**: Enterprise deployment (build on Windows/CI)

**Why MSI fails on Linux:**
WiX 4.x's `light.exe` crashes under Wine's wow64 mode with COM exceptions. If you need MSI, use Windows CI runners.

#### Portable
- **Extension**: `.exe` (single file)
- **Install method**: Just run it
- **Pros**: No installation, can run from USB drive
- **Cons**: No shortcuts, no uninstaller
- **Best for**: USB distribution, testing

### macOS Formats

#### ZIP (with .app bundle)
- **Extension**: `.zip` containing `MyApp.app`
- **Install method**: Extract, drag to Applications
- **Pros**: Can be built on Linux, simple distribution
- **Cons**: No fancy installer UI, manual drag-to-install
- **Best for**: Cross-platform builds, developer distribution

**Build on Linux:**
```bash
npm run dist:mac  # Creates .zip with .app inside
```

#### DMG (Disk Image)
- **Extension**: `.dmg`
- **Install method**: Mount, drag app to Applications folder
- **Pros**: Professional appearance, customizable background
- **Cons**: Requires `hdiutil` (macOS only)
- **Best for**: Polished consumer releases

**Why DMG fails on Linux:**
DMG creation requires `hdiutil`, a macOS-only tool.

#### PKG (Installer Package)
- **Extension**: `.pkg`
- **Install method**: Double-click, follow wizard
- **Pros**: Can run scripts, professional installer flow
- **Cons**: Requires `productbuild` (macOS only)
- **Best for**: Enterprise deployment, apps needing post-install setup

**Why PKG fails on Linux:**
PKG creation requires `productbuild`, a macOS-only tool.

## Setting Up Cross-Platform Builds

### Prerequisites

#### For Linux Builds (Native)
```bash
# Debian/Ubuntu
sudo apt-get install rpm  # If building RPM
```

#### For Windows Builds on Linux (Wine)
Wine is automatically installed by the build scripts:
```bash
npm run dist:win  # Runs setup-wine.js automatically
```

Or install manually:
```bash
# Using AppImage (no sudo required)
node scripts/setup-wine.js

# Or via Flatpak
flatpak --user install flathub org.winehq.Wine
```

#### For macOS Builds on Linux
No additional setup needed for ZIP target. DMG/PKG require macOS.

### Build Commands

```bash
# Build for current platform
npm run dist

# Build for specific platforms
npm run dist:linux   # AppImage + DEB
npm run dist:win     # NSIS installer (via Wine on Linux)
npm run dist:mac     # ZIP with .app bundle

# Build for all platforms
npm run dist:all     # Linux + Windows + macOS

# Debug builds (unpacked, no installer)
npm run dist:win:debug  # Creates win-unpacked/ folder
npm run pack            # Current platform, unpacked
```

### Changing Target Formats

Edit `package.json` build configuration:

```json
{
  "build": {
    "win": {
      "target": [
        { "target": "nsis", "arch": ["x64"] }
      ]
    },
    "mac": {
      "target": [
        { "target": "zip", "arch": ["x64", "arm64"] }
      ]
    },
    "linux": {
      "target": [
        { "target": "AppImage", "arch": ["x64"] },
        { "target": "deb", "arch": ["x64"] }
      ]
    }
  }
}
```

**Available targets:**
- **Windows**: `nsis`, `nsis-web`, `msi`, `portable`, `appx`, `squirrel`
- **macOS**: `zip`, `dmg`, `pkg`, `mas` (Mac App Store)
- **Linux**: `AppImage`, `deb`, `rpm`, `snap`, `pacman`, `freebsd`

## CI/CD Recommendations

For production releases, use platform-native CI runners:

```yaml
# GitHub Actions example
jobs:
  build-linux:
    runs-on: ubuntu-latest
    steps:
      - run: npm run dist:linux

  build-windows:
    runs-on: windows-latest
    steps:
      - run: npm run dist:win  # Can build MSI natively

  build-macos:
    runs-on: macos-latest
    steps:
      - run: npm run dist:mac  # Can build DMG/PKG natively
```

### Single-Machine Builds (Development/Testing)

For quick iteration on Linux:

```bash
# Build all platforms from Linux
npm run dist:all

# Output:
# dist-electron/
#   MyApp-1.0.0.AppImage          # Linux portable
#   myapp_1.0.0_amd64.deb         # Debian package
#   MyApp Setup 1.0.0.exe         # Windows NSIS installer
#   MyApp-1.0.0-mac.zip           # macOS app bundle
#   MyApp-1.0.0-arm64-mac.zip     # macOS ARM app bundle
```

## Troubleshooting

### Wine Build Issues

**"spawn wine ENOENT"**
Wine is not in PATH. Run:
```bash
node scripts/setup-wine.js
export PATH="$HOME/.local/bin:$PATH"
```

**WiX/MSI errors under Wine**
Switch to NSIS target (see "Setting Up Cross-Platform Builds").

### macOS Build Issues

**"spawn productbuild ENOENT" or "spawn hdiutil ENOENT"**
These tools only exist on macOS. Use `zip` target instead:
```json
"mac": { "target": [{ "target": "zip" }] }
```

### Icon Not Appearing (Windows)

The `signAndEditExecutable: false` setting disables icon embedding. The `post-build-windows.js` script handles this manually via rcedit.

If icons still don't appear:
```bash
node scripts/fix-rcedit.js  # Patches rcedit for Wine compatibility
```

### Build Taking Too Long

Large apps can take 5-10 minutes per platform. Use parallel builds:
```bash
# Build platforms in parallel (if you have resources)
npm run dist:linux &
npm run dist:win &
npm run dist:mac &
wait
```

## Output Directory Structure

After building, `dist-electron/` contains:

```
dist-electron/
├── MyApp-1.0.0.AppImage              # Linux portable
├── myapp_1.0.0_amd64.deb             # Linux Debian package
├── MyApp Setup 1.0.0.exe             # Windows NSIS installer
├── MyApp Setup 1.0.0.exe.blockmap    # Windows update metadata
├── MyApp-1.0.0-mac.zip               # macOS x64 bundle
├── MyApp-1.0.0-mac.zip.blockmap      # macOS x64 update metadata
├── MyApp-1.0.0-arm64-mac.zip         # macOS ARM bundle
├── MyApp-1.0.0-arm64-mac.zip.blockmap
├── latest.yml                        # Windows auto-update manifest
├── latest-linux.yml                  # Linux auto-update manifest
├── latest-mac.yml                    # macOS auto-update manifest
├── linux-unpacked/                   # Unpacked Linux app
├── mac/                              # Unpacked macOS x64 app
│   └── MyApp.app/
├── mac-arm64/                        # Unpacked macOS ARM app
│   └── MyApp.app/
└── win-unpacked/                     # Unpacked Windows app
    └── MyApp.exe
```

## Related Documentation

- [Wine Installation Guide](./WINE_INSTALLATION.md) - Detailed Wine setup
- [Debugging Windows Apps](./DEBUGGING_WINDOWS.md) - Windows-specific troubleshooting
- [electron-builder Targets](https://www.electron.build/multi-platform-build) - Official docs
