# Audio Tools Cross-Platform Readiness Audit

## Last Updated

2026-08-18

## Target Tiers

| Tier | Status | Evidence / limitation |
|---|---|---|
| Local Vrooli stack | Supported | Native Linux Whisper and the shared browser capture path are lifecycle-managed. |
| Desktop bundle | Partial | The API and UI receive bundle-local roots and `VROOLI_STORAGE_ROOT`; native speech resources still require per-platform artifacts and smoke evidence. |
| Mobile | Unsupported | The current product path requires a local API and native speech resource; no mobile adapter exists. |
| Cloud / SaaS | Conditional | BYOK and remote routing can operate without local native resources; local-resource claims do not transfer to cloud deployment. |
| Enterprise / air-gapped | Conditional | The native resources can be supplied as managed artifacts, but publication, licensing, and target smoke evidence remain required. |

## Environment and Filesystem Status

- API ports remain lifecycle-injected through `API_PORT` and `UI_PORT`.
- Mutable SQLite and session state use `api-core/storage`-resolved scenario data paths.
- Server-owned soak evidence uses the repository `scenarios/audio-tools/coverage/` directory when `VROOLI_ROOT` is present. Outside a checkout it now uses `api-core/storage` data storage, so a bundled API does not require a monorepo root.
- The fixed virtual corpus endpoint resolves fixtures from `VROOLI_ROOT`, `SCENARIO_ROOT`, `BUNDLE_ROOT`, or bundle-relative working directories. It never accepts an arbitrary page-supplied filesystem path.
- Configuration and model resource paths remain owned by their respective resource manifests; audio-tools does not assume a Unix home layout.
- `storage-manager validate scenario audio-tools` passed on Linux with zero findings, including filesystem routing and isolation checks.

## Resource Dependencies

| Resource | Strategy | Current posture |
|---|---|---|
| `whisper` | Native managed service | Linux amd64 is build/live verified; Linux arm64 and Windows acquisition are declared but target smoke is pending; macOS remains unsupported until a native server artifact exists. |
| `kyutai-stt` | Optional native streaming accelerator | Linux/CUDA qualification remains a separate resource gate; the product retains other STT cells when it is unavailable. |
| `sherpa-onnx` | Optional native managed service | Linux adapter bundle is reproducible, but acquisition remains explicitly unsupported until signed Vrooli artifacts are published; macOS and Windows target artifacts and smoke are open. |
| `postgres` | Optional runtime backend | SQLite is the default local path; PostgreSQL is not required for core dictation. |
| remote/BYOK providers | Runtime fallback | Remote providers remain available where credentials and network access exist; they do not provide offline-local guarantees. |

## Build Status

- The audio-tools Go API has no `mattn/go-sqlite3` dependency in its runtime path and uses the pure-Go SQLite driver.
- The native speech resources are not represented as cross-compiled binaries. Each target must be built on a matching native host or supplied as a signed release artifact; upstream archives alone are not Vrooli evidence.
- The sherpa build targets fail closed on host/target mismatch, preventing a cross-compiled or stub binary from being presented as runnable support.

## Open Portability Gates

1. Publish and acquire signed Vrooli-owned sherpa adapter bundles for Linux arm64, macOS arm64, and Windows amd64, then smoke-test each target.
2. Complete Whisper Linux arm64 and Windows smoke, and provide a native macOS server artifact before changing its support status.
3. Compare the native cells on the same long-form corpus across supported targets; retain the exact engine/model/strategy/policy identity in evidence.
4. Complete the required 15-minute and 60-minute browser/device qualification lanes. These are runtime reliability gates, not substitutes for manifest validation.

The current workspace reports `vrooli release-authority status` as
`configured=false`; no release signature or trust-anchor mutation was created
as part of this audit. Publication must therefore remain an explicit release
operation rather than being implied by a local build.

## Recent Changes

- `internal/soak.PersistEvidence` no longer requires `VROOLI_ROOT`; it uses atomic `api-core/storage` writes outside a checkout.
- The virtual corpus lookup honors bundle and scenario roots.
- `resources/whisper` and `resources/sherpa-onnx` keep unsupported targets explicit rather than claiming portability from upstream libraries or un-smoked archives.
