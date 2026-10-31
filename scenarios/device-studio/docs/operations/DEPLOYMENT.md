# Deployment — Device Studio

## Purpose Of This Document

Working deployment boundary, updated 2026-10-20. Development APIs and
rendering paths are implemented and exercised, and a private Linux bundle/AppImage has now been assembled and launched through the native validation owner. The exact private AppImage passes required native project-dialog input with the development app stopped. Standalone rendering and persistence remain unqualified. Research and tool-profile
qualification are evidence for bounded techniques, not a released scenario.

Architecture targets Windows/macOS/Linux and downloadable personal use from the first implementation. Runtime evidence is per exact package/OS/architecture/backend.

## Current delivery boundary

The operator-approved implementation effort requires runtime/package validation only on this machine. Windows/macOS architecture remains required; execution, signing and certification on other OS targets are deferred and cannot block this delivery. The per-target release checklist below remains the condition for any future native-platform claim, not a requirement to access those machines during this effort.

## Supported Tiers

Current evidence includes the local browser/API project and photographic rendering workflow, managed rendering tools, and bounded physical scene studies. A standalone desktop package is not yet qualified. Target modes: current-host browser access; private personal desktop bundle; explicitly configured remote/shared provider. Private desktop does not require website login for local work. Multi-user/shared exposure requires its declared identity mode. No deployed public service, signed installer or platform-native render is claimed.

## Runtime Requirements

Core target: Go API/worker, static UI, SQLite and managed per-user artifacts. Local video needs a qualified FFmpeg/FFprobe pair. 3D needs a pinned Blender renderer distribution with included Python/libraries/kernels/importers/color configuration. CPU/GPU capabilities are probed in the actual worker. Optional capture, AI, account and publication services are available only through their configured adapters; no full fleet or checkout is required for local core.

Use existing `VROOLI_STORAGE_ROOT`, `APP_DATA_DIR` and `BUNDLE_ROOT` conventions through the desktop runtime. Do not invent home-directory fallbacks or store mutable files inside a signed app. Host/tool remediation belongs in the control plane; standalone provisioning must be a supported verified artifact route, not a private shell installer.

## Packaging

Use scenario-to-desktop's private runtime manifest with explicit service artifacts, dependencies, health/readiness, hashes and writable directories. Ship a compact core and an optional qualified renderer pack with trusted manifest, dependency closure and rights/source notices. Packaged composition must include all required fonts/codecs/libraries; ambient Python/Pillow/OpenCV is not an implicit dependency. Source files use portable references, resolved at the worker boundary.

Linux x64, Windows x64 and macOS arm64 are the initial target cells. Additional architectures are explicit artifact/compatibility decisions. Shared cross-build support or an Electron platform enum does not certify native execution. Signed helpers/libraries and renderer packs need the same provenance as the outer app.

### Current packaging preparation — 2026-10-20

The supported desktop pipeline exposes explicit Linux-only bundled builds,
verified tool-artifact roots and development-local artifact trust. A private Device Studio Linux AppImage was built by pipeline
`1cbba55f-80e0-74e6-1c6a-ba09d4487a56`. Its first smoke failed before launch
because validation rediscovered the source-tree path instead of honoring the
selected staged artifact. The owner fix passes focused regressions and independent source review. Retry matrix `run-b72d397516e939e071f86668` launches that exact AppImage and records a responsive Projects window; its generic baseline then clicks empty space and fails the surface-change assertion. That result does not prove semantic input failure. A required semantic project-dialog journey now passes actual native execution in matrix`run-2062b3d706c4eb45ec7fb548` for AppImageSHA256`cb8980de7e9a0d9923c5abdf079074c2aaa57d3a5ec1e7fbc5a4ffc012cffd01`, with development Device Studio stopped. Ten steps verify click/input, enabled submit, Escape dismissal and focus restoration without creating a project. Parent inspected the actual dialog capture. Evidence:`../internal/goal/evidence/E7/desktop-private/native-dialog-5/parent-review.json`.

Explicit bundled-runtime mode routes lease installation through the bound package API and removes development endpoint overrides. Legacy launch validation keeps its existing discovery path. The bound renderer URL is supplied through BAS's existing typed StartUrl so normal navigation checks still apply. Earlier optional-cell aggregate pass and navigation-guard failure remain retained, not reinterpreted as successes. Full offline import/save/render/restart, optional renderer packs, rights and release checks remain open. Source inspection found these remaining integration requirements:

- The service manifest currently declares no scenario/resource dependencies and
  no deployment profile. That empty graph does not prove a complete render bundle.
- The desktop runtime's finite packaged-profile resolver and shim command are
  implemented. Device Studio's physical/video worker adapters use the resolved VROOLI_BIN entrypoint, which the desktop runtime sets to its bundled shim. The shim already resolves the finite packaged profile declarations; full offline worker execution is still unverified. A private runtime must not depend on an operator-home tool installation.
- The dedicated Blender and FFmpeg declarations remain `host-required` for the
  general host-tool model. A separate explicit renderer-profile selector now
  packages only owner-verified managed trees into a release-v1 directory; it
  does not change the global classifier or implicitly copy host tools.
- Photographic composition itself now runs in the Go worker and does not require
  an Image Tools server. Video and physical workers still need their pinned native
  tool closures and the platform operation owner.

### Private renderer profile integration — implementation status, 2026-10-20

The release assembler now accepts a finite optional selector:
`--renderer-profiles blender,ffmpeg-video`. It reads only the exact managed
install roots recorded in the invoking user's install ledger, acquires each
tool's install lock, verifies the owner-pinned full tree and fixed binary
probes, and stages the selected tree without overwriting an existing package.
The resulting release-v1 directory contains the profile declaration index and
both selected package trees with owner-pinned profile IDs, Linux/amd64 platform
facts, tree digests, and explicit renderer roles. It does not change global
`host-required` declarations or download/install tools.

Example for a fresh development-local release directory:

```bash
cd /home/matthalloran8/Vrooli
RELEASE_DIR="$PWD/scenarios/device-studio/docs/internal/goal/evidence/E7/desktop-private/renderer-release-1"
go run ./cmd/vrooli-dist --root "$PWD" --out-dir "$RELEASE_DIR" --renderer-profiles blender,ffmpeg-video
go run ./cmd/vrooli-dist --verify-release-manifest --release-artifact-root "$RELEASE_DIR" --trust-mode development-local
```

The retained E7 release assembled successfully, and the verification command
exited 0 with all three artifacts verified (`/tmp/device-studio-renderer-release1-verify.log`).
This proves managed-tree packaging only. The directory is unsigned,
development-local, and non-promotable; it does not establish redistribution
rights, OS-library portability, offline application behavior, or a completed
Device Studio desktop bundle. The desktop pipeline still owns the next step:
select the explicit profile IDs, verify the release directory, and stage the
packs beneath the bundle's fixed tool locations. Keep license review, signing,
host prerequisites, and private-bundle render/decode proof as separate gates.

## Release Checklist

For each advertised target retain clean-user install without checkout, startup/readiness, import/save/reopen, offline plate render, video decode, supported native-render sample, CPU/GPU fallback behavior, cancellation/descendant cleanup, secure-store/account callback tests, signed update/relaunch, project retention and uninstall behavior. Qualify filesystem spaces/Unicode/permissions, disk-full and interrupted download.

Require actual target signing/notarization/Gatekeeper evidence where applicable; do not infer credentials exist from configuration code. Verify renderer/tool license/build flags and asset redistribution closure. Host FFmpeg's GPL/libx264 build is not an interchangeable LGPL-only redistribution artifact. Current product/platform cells remain unverified.

## Rollback

Pin a renderer/profile for each active job; updates cannot replace it mid-execution. Stage and verify a successor before atomic activation; retain the prior qualified package for recovery. Back up project metadata and referenced artifacts before schema migration. Test restoring/reopening real saved shots after failed update. A UI reload is not proof of migration safety.

## Cross-References

[RUNBOOK.md](RUNBOOK.md), [security](../internal/SECURITY.md), [integrations](../concepts/INTEGRATIONS.md), [research](../research/README.md).
