# Wine Installation Guide (No Sudo Required)

## Overview

Wine is required for building Windows `.msi` installers on Linux. This guide shows how to install Wine **without sudo/root access** using Flatpak.

## Why Flatpak?

- ✅ **No sudo required**: Install with `--user` flag
- ✅ **Isolated**: Doesn't affect system packages
- ✅ **Up-to-date**: Maintained by Flathub community
- ✅ **Safe**: Wine should never be run as root anyway

## Installation Steps

### 1. Ensure Flatpak is Available

Check if Flatpak is installed:
```bash
flatpak --version
```

If not installed, Flatpak itself requires sudo for initial system setup. However, many modern Linux distributions come with Flatpak pre-installed.

### 2. Add Flathub Repository (User-Space)

```bash
flatpak --user remote-add --if-not-exists flathub https://flathub.org/repo/flathub.flatpakrepo
```

### 3. Install Wine (User-Space)

```bash
flatpak --user install flathub org.winehq.Wine
```

**Note**: This installs Wine in `~/.local/share/flatpak/` - no system modification required.

### 4. Verify Installation

```bash
flatpak run org.winehq.Wine --version
```

You should see Wine version information (e.g., `wine-9.0`).

## Using Wine for Electron Builder

### Method 1: Set Wine Path Environment Variable

Tell electron-builder where to find Wine:

```bash
export WINE_BIN="flatpak run org.winehq.Wine"
npm run dist:win
```

### Method 2: Create Wine Wrapper Script

Create a wrapper script at `~/.local/bin/wine`:

```bash
#!/bin/bash
exec flatpak run org.winehq.Wine "$@"
```

Make it executable:
```bash
chmod +x ~/.local/bin/wine
```

Ensure `~/.local/bin` is in your PATH:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then electron-builder will find `wine` automatically.

### Method 3: Alias in Shell Config

Add to `~/.bashrc` or `~/.zshrc`:

```bash
alias wine='flatpak run org.winehq.Wine'
```

Reload shell:
```bash
source ~/.bashrc
```

## Automated Installation (API Endpoint)

The scenario-to-desktop API can detect if Wine is installed and guide users through installation.

### Check Wine Installation Status

```bash
# API endpoint to check Wine status
curl http://localhost:${API_PORT}/api/v1/desktop/wine/status
```

Response:
```json
{
  "installed": false,
  "method": "flatpak",
  "install_command": "flatpak --user install flathub org.winehq.Wine",
  "instructions_url": "/docs/WINE_INSTALLATION.md"
}
```

### UI Integration

The UI can display a banner when Wine is not installed:

```
⚠️  Wine Not Installed

Windows builds require Wine. Install it without sudo:

[Install Wine via Flatpak] (opens terminal command)

Learn more: Wine Installation Guide
```

## Verifying Wine Works for Builds

Test Wine with a simple build:

```bash
cd scenarios/picker-wheel/platforms/electron
npm install
npm run dist:win
```

If successful, you'll see:
- `dist-electron/Picker Wheel Setup 1.0.0.msi` created
- No Wine-related errors in build log

## Troubleshooting

### "flatpak: command not found"

Flatpak is not installed on your system. Options:
1. Ask system administrator to install Flatpak (requires sudo once)
2. Use Wine AppImage instead (portable, no installation)
3. Build only for Linux/Mac platforms

### "Runtime org.freedesktop.Platform not found"

Flatpak needs to download the freedesktop runtime:
```bash
flatpak --user install flathub org.freedesktop.Platform//22.08
```

### Wine Build Fails with "spawn wine ENOENT"

electron-builder can't find Wine executable. Use Method 2 (wrapper script) or Method 1 (environment variable).

### Permission Denied Errors

Never run Wine with sudo. Wine creates `.wine` directory in your home folder automatically.

## Alternative: Wine AppImage (Advanced)

If Flatpak is not available, Wine AppImages provide a portable Wine installation:

1. Download Wine AppImage from https://github.com/ferion11/Wine_Appimage/releases
2. Make executable: `chmod +x wine-*.AppImage`
3. Run: `./wine-*.AppImage --version`
4. Create wrapper script pointing to AppImage

**Note**: AppImages are larger (~200MB) and may have compatibility issues.

## Security Notes

- ✅ **Safe**: Running Wine as normal user (via Flatpak) is secure
- ❌ **Never**: Run `sudo wine` - gives Windows programs full system access
- ✅ **Isolated**: Flatpak Wine runs in sandbox with limited permissions

## References

- [Flathub Wine Package](https://flathub.org/apps/org.winehq.Wine)
- [Wine HQ Documentation](https://www.winehq.org/documentation)
- [electron-builder Wine Configuration](https://www.electron.build/multi-platform-build#linux)

---

**Last Updated**: 2025-11-15
**Maintained By**: Vrooli Team
