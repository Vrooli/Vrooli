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

## Optional managed renderer packs (development-local)

For a private Linux/amd64 bundle, the release assembler can package the exact
managed Blender and FFmpeg/FFprobe profiles already installed for the current
Vrooli user. It does not download tools or accept arbitrary paths. The source
tool must be present in the install ledger and pass the profile owner's complete
tree and fixed-probe checks. Use a fresh output directory because artifact
publication refuses to overwrite existing packages:

```bash
ROOT=/home/matthalloran8/Vrooli
RELEASE_DIR="$ROOT/scenarios/device-studio/docs/internal/goal/evidence/E7/desktop-private/renderer-release-1"
cd "$ROOT"
go run ./cmd/vrooli-dist --root "$ROOT" --out-dir "$RELEASE_DIR" --renderer-profiles blender,ffmpeg-video
go run ./cmd/vrooli-dist --verify-release-manifest --release-artifact-root "$RELEASE_DIR" --trust-mode development-local
```

The selector accepts only `blender`, `ffmpeg-video`, or both. The resulting
release-v1 manifest binds each package tree digest and role. `development-local`
verification is unsigned and non-promotable; it is useful for the private
desktop staging path but does not grant redistribution rights or qualify another
host/platform. Select the same finite IDs in the desktop pipeline with:

```bash
scenario-to-desktop pipeline run device-studio \
  --stages bundle --platforms linux --location-mode staging \
  --tool-artifact-root "$RELEASE_DIR" \
  --renderer-tool-profiles blender,ffmpeg-video \
  --artifact-trust-mode development-local
```

The pipeline still needs the scenario's valid Electron bundle manifest and a
supported Linux target. Do not call package assembly proof a completed
private-app or offline-render qualification.

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

## Bounded external experiments

Use `prompt-manager skill read local-experimentation` and `performance-health`.
For confirmation, reuse hash-pinned built artifacts first; rebuilding is not
required for every measurement. Record the source, compiled entrypoint, bundle,
tool/runtime identities and profile settings. Prepared source may differ from
old compiled output: rebuild both comparison profiles from identical inputs in
that case, and keep the old artifact as a separate calibration.

When a source experiment needs a build, copy the minimum prepared source,
bundle and already-installed approved toolchain into a task-owned directory
outside the checkout. Verify independent files and dependency closure through
Scenario Dependency Analyzer; acquire anything missing through that owner.
Do not run package-manager installs in the experiment. Copy `node_modules` as
independent project files: Electron Builder can silently omit transitive modules
when the whole directory is a symlink, even inside the experiment. Check archive
contents and dependency-resolution warnings before accepting a package. A copied
Go entrypoint needs module-graph metadata as well as compiled-package sources;
retain both inventories and build with downloads disabled. An already-prepared
TypeScript/Electron Builder fixture can invoke its copied tools directly:

```bash
node node_modules/typescript/bin/tsc --project tsconfig.json
node node_modules/electron-builder/cli.js --linux AppImage --publish never --config experiment-builder.json
```

Derive the builder configuration from the fixture's existing build settings.
Point `electronDist` at an independently copied, version-matched existing Electron
archive and retain its SHA-256; this avoids an implicit runtime download.
For an unsigned JavaScript-only fixture, `npmRebuild: false` and
`afterAllArtifactBuild: null` can suppress native rebuilding and release signing
in both profiles. These are declared experiment differences, not release settings;
native dependencies need their owning build route. Retain the exact configuration
and commands. Do not add a service or wrapper merely to run these tools.

Preview each measurement with `performance-health packaged capture --plan`.
Name the reference journey and compiled strategy/toggles explicitly. Count every
batch's discarded warmup and protocol/demo starts, including failures. Run builds
and tests outside measurement windows. Let the existing Desktop owner execute
and cancel captures; verify exact processes, temporary installs and capacity
release, and inventory retained media/scratch. A successful scratch build is not
production integration or strategy certification.

## Fast screening of an existing artifact

Use the existing `SmokeTestService.StartSmokeTest` Connect API when a pinned
artifact needs a startup and journey check without the measured capture's two
60-second resource windows. It uses the same smoke runner, recorder, journey,
status store and cancellation owner. It does not rebuild or install dependencies.
There is no standalone CLI leaf for this API.

Send this request through the normal caller-authenticated client to
`/vrooli.scenario_to_desktop.v1.domain.SmokeTestService/StartSmokeTest`:

```json
{
  "scenarioName": "hello-desktop",
  "artifactPath": "/absolute/path/to/frozen.AppImage",
  "recordDesktop": true,
  "deploymentMode": "bundled",
  "journeyId": "hello-desktop"
}
```

Use `proxy` with an explicit `proxyUrl` only for an external-server target.
Omitting mode preserves the legacy target behavior. Use explicit bundled mode
for a bundled experiment; do not substitute a live Tier 1 service. Recorded runs
use a job-owned profile; cold measured stages use separate children of that root.
The owner removes the profile root on exit, including failed or cancelled runs.
On Linux, a short job root also supplies child `TMPDIR`, so AppImage extraction
is removed with the profile. The executor checks both launch markers and owned
executable/working-directory paths when a child sanitizes its environment.

Retain `smokeTestId` before waiting. Read `GetSmokeTest` with that ID and keep
the terminal status, journey review, artifact hashes, recording and startup
traces. The status includes `deploymentMode`, `screenContentSource` and
`processLifetimeChecks` from the existing owner state. Require a nonempty
session ID, passing check and no remaining PIDs for each attempted stage.
Application failure and cleanup success are separate; an unstarted demo needs
no cleanup receipt. Missing receipts remain unknown.
These are harness cleanup checks, not proof that the application's own shutdown
stopped every service. Historical marker-only receipts do not gain stronger
coverage when the current owner is upgraded.
The job outlives the initiating HTTP request. At the declared deadline,
call `CancelSmokeTest` with the same ID and verify terminal status and exact
process/display/profile cleanup. A successful cancellation response only means
cancellation was requested.

One recorded smoke normally starts protocol and demo once each, with no owner
warmup. This route is diagnostic screening; it does not establish cold/warm
comparability, idle CPU, steady PSS or reliability. Confirm selected candidates
through Performance Health using unchanged artifacts and metric definitions.
