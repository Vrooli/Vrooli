# Deployment — Device Studio

## Purpose Of This Document

Foundation contract, 2026-10-19. Product behavior below is specified for implementation; the generated reference scaffold does not implement it. Research demonstrates individual techniques, not a released scenario.

Architecture targets Windows/macOS/Linux and downloadable personal use from the first implementation. Runtime evidence is per exact package/OS/architecture/backend.

## Current delivery boundary

The operator-approved implementation effort requires runtime/package validation only on this machine. Windows/macOS architecture remains required; execution, signing and certification on other OS targets are deferred and cannot block this delivery. The per-target release checklist below remains the condition for any future native-platform claim, not a requirement to access those machines during this effort.

## Supported Tiers

Current evidence is the generated local Vrooli scaffold only. Target modes: current-host browser access; private personal desktop bundle; explicitly configured remote/shared provider. Private desktop does not require website login for local work. Multi-user/shared exposure requires its declared identity mode. No deployed public service, signed installer or platform-native render is claimed.

## Runtime Requirements

Core target: Go API/worker, static UI, SQLite and managed per-user artifacts. Local video needs a qualified FFmpeg/FFprobe pair. 3D needs a pinned Blender renderer distribution with included Python/libraries/kernels/importers/color configuration. CPU/GPU capabilities are probed in the actual worker. Optional capture, AI, account and publication services are available only through their configured adapters; no full fleet or checkout is required for local core.

Use existing `VROOLI_STORAGE_ROOT`, `APP_DATA_DIR` and `BUNDLE_ROOT` conventions through the desktop runtime. Do not invent home-directory fallbacks or store mutable files inside a signed app. Host/tool remediation belongs in the control plane; standalone provisioning must be a supported verified artifact route, not a private shell installer.

## Packaging

Use scenario-to-desktop's private runtime manifest with explicit service artifacts, dependencies, health/readiness, hashes and writable directories. Ship a compact core and an optional qualified renderer pack with trusted manifest, dependency closure and rights/source notices. Packaged composition must include all required fonts/codecs/libraries; ambient Python/Pillow/OpenCV is not an implicit dependency. Source files use portable references, resolved at the worker boundary.

Linux x64, Windows x64 and macOS arm64 are the initial target cells. Additional architectures are explicit artifact/compatibility decisions. Shared cross-build support or an Electron platform enum does not certify native execution. Signed helpers/libraries and renderer packs need the same provenance as the outer app.

## Release Checklist

For each advertised target retain clean-user install without checkout, startup/readiness, import/save/reopen, offline plate render, video decode, supported native-render sample, CPU/GPU fallback behavior, cancellation/descendant cleanup, secure-store/account callback tests, signed update/relaunch, project retention and uninstall behavior. Qualify filesystem spaces/Unicode/permissions, disk-full and interrupted download.

Require actual target signing/notarization/Gatekeeper evidence where applicable; do not infer credentials exist from configuration code. Verify renderer/tool license/build flags and asset redistribution closure. Host FFmpeg's GPL/libx264 build is not an interchangeable LGPL-only redistribution artifact. Current product/platform cells remain unverified.

## Rollback

Pin a renderer/profile for each active job; updates cannot replace it mid-execution. Stage and verify a successor before atomic activation; retain the prior qualified package for recovery. Back up project metadata and referenced artifacts before schema migration. Test restoring/reopening real saved shots after failed update. A UI reload is not proof of migration safety.

## Cross-References

[RUNBOOK.md](RUNBOOK.md), [security](../internal/SECURITY.md), [integrations](../concepts/INTEGRATIONS.md), [research](../research/README.md).
