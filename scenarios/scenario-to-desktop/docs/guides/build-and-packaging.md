# Build & Packaging Checklist

Centralize prerequisites and failure points before cutting installers.

## Prerequisites
- **UI build present** in the target scenario at `ui/dist` (Scenario Inventory now validates this).
- **Node 18+** and **npm** available for `platforms/electron`.
- **electron-builder** bundled via template dependencies (no global install required).
- **Wine (Linux → Windows builds)**: follow `docs/guides/wine-installation.md` for a no-sudo Flatpak install.

## Build Commands (inside generated `platforms/electron/`)
- Dev: `npm run dev`
- Current platform: `npm run dist`
- Targets: `npm run dist:win | dist:mac | dist:linux | dist:all`
- Debug unpacked Windows build: `npm run dist:win:debug` then run from `dist-electron/win-unpacked/`.

## Common Issues
- **App won’t start (Windows)**: run the unpacked exe from PowerShell to see logs; see `docs/guides/debugging-windows.md`.
- **Wine not found**: set `WINE_BIN="flatpak run org.winehq.Wine"` or add the wrapper script from `docs/guides/wine-installation.md`.
- **No icon in .exe**: ensure the `afterPack` hook and rcedit/Wine are available; rebuild after fixing Wine.
- **Cannot reach server**: verify proxy URL, firewall, or SSH tunnel; thin clients require the remote scenario to be running.

## Outputs & Paths
- Installers: `dist/` (MSI/PKG/AppImage/DEB; legacy EXE/DMG if configured)
- Unpacked builds: `dist-electron/<platform>-unpacked/`
- Telemetry: `deployment-telemetry.jsonl` under OS user data (see `docs/reference/telemetry.md`)

## Tests
- Run scenario tests from repo root: `make test` or `vrooli test help` for options.
- UI/unit tests live under `ui/`; API/CLI tests under `api/` and `cli/`.


## Generator freshness and authoritative source profiles

Before consuming the compiled template generator, run the owner read operation:

```bash
cd scenarios/scenario-to-desktop/templates/build-tools
node write-build-stamp.js --check
```

This verifies the recorded input hash/schema and a regular compiled generator
entry without writing, rebuilding, installing or generating output. The existing
build command still writes the stamp after TypeScript compilation. Missing,
invalid or stale inputs fail with the governed setup instruction. The input
contract includes non-test TypeScript, package/tsconfig and the stamp producer
itself; dist/node_modules/test sources are excluded. JS producer and Go verifier
use the same byte-ordered path/content framing. Run `test:stamp` for isolated
read-only/stale-input regressions; the normal build-tools test entry includes them.
A valid input stamp is a freshness check, not a native packaging/release verdict.

A committed shell needs one reviewed generation input shared with its comparator.
BAS's source shell uses `.vrooli/electron-shell-config.json` (proxy/basic), while
its historical `.vrooli/desktop-config.json` remains a distinct saved bundled
profile. Neither profile is silently substituted for the other. The reviewed
source shell preserves current presentation and the existing bundle resource and
Linux signing hook. Source regeneration can use the supported template-generator
CLI with the explicit source-profile JSON from the BAS scenario directory;
dependency installation remains governed separately. A source-only refresh sets
`SKIP_DESKTOP_DEPENDENCY_INSTALL=1`. Do not hand-edit generated output or weaken
byte/security comparisons to reconcile a mismatched profile.


New generation build-status artifacts retain`derived_config_sha256` for the exact
temporary JSON bytes and`derived_config_status=ephemeral-not-retained`. The worker
removes that temporary file; it no longer advertises`config_path` as reusable.
The digest remains diagnostic evidence on successful and failed attempts. It does
not store arbitrary feature/native-extension/signing inputs, recover their values
or promise full reproduction. No neighboring profile is automatically asserted to
be the source. BAS's committed source profile is independently bound by its
comparator/docs; a generic typed input-reference contract remains future work.
Existing historical records are not rewritten or represented as newly qualified.
