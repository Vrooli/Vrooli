# Build & Packaging Checklist

Centralize prerequisites and failure points before cutting installers.

## Prerequisites
- **UI build present** in the target scenario at `ui/dist` (Scenario Inventory now validates this).
- **Node 18+** and **npm** available for `platforms/electron`.
- **electron-builder** bundled via template dependencies (no global install required).
- **Wine (Linux → Windows builds)**: follow `docs/WINE_INSTALLATION.md` for a no-sudo Flatpak install.

## Build Commands (inside generated `platforms/electron/`)
- Dev: `npm run dev`
- Current platform: `npm run dist`
- Targets: `npm run dist:win | dist:mac | dist:linux | dist:all`
- Debug unpacked Windows build: `npm run dist:win:debug` then run from `dist-electron/win-unpacked/`.

## Common Issues
- **App won’t start (Windows)**: run the unpacked exe from PowerShell to see logs; see `docs/DEBUGGING_WINDOWS.md`.
- **Wine not found**: set `WINE_BIN="flatpak run org.winehq.Wine"` or add the wrapper script from `docs/WINE_INSTALLATION.md`.
- **No icon in .exe**: ensure the `afterPack` hook and rcedit/Wine are available; rebuild after fixing Wine.
- **Cannot reach server**: verify proxy URL, firewall, or SSH tunnel; thin clients require the remote scenario to be running.

## Outputs & Paths
- Installers: `dist/` (MSI/PKG/AppImage/DEB; legacy EXE/DMG if configured)
- Unpacked builds: `dist-electron/<platform>-unpacked/`
- Telemetry: `deployment-telemetry.jsonl` under OS user data (see `docs/telemetry.md`)

## Tests
- Run scenario tests from repo root: `make test` or `vrooli test help` for options.
- UI/unit tests live under `ui/`; API/CLI tests under `api/` and `cli/`.
