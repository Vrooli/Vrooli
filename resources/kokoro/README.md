# Kokoro Resource

Managed Kokoro text-to-speech runtime for local speech-synthesis workflows.

## Intent

- Resource ID: `kokoro`
- Category: `ai`
- Driver: `managed-service`
- Portability tier: `native Linux amd64` (CPU and CUDA targets)

## Use Cases

- Generate spoken audio locally for voice and multimodal workflows.
- Provide a reusable text-to-speech service for scenarios and automation.
- Pair with speech-to-text resources for end-to-end voice pipelines.

## Architecture

This resource uses the native `managed-service` structure.

- `resource.json` is the declarative authority for lifecycle, native artifact acquisition, ports, exports, health, and freshness metadata.
- The artifact combines a checksum-pinned CPython runtime, hash-locked CPU or
  CUDA wheels, and the reviewed Kokoro-FastAPI source tree.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Kokoro-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Kokoro-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/topology`: service dependency and readiness semantics
- `cli/internal/runtime`: runtime shaping helpers
- `cli/internal/health`: Kokoro-specific readiness helpers
- `cli/internal/env`: environment export helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install kokoro

# Check status through the shared control plane
resource-kokoro status
```

Default endpoint:

- API: `http://localhost:8880`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for synthesis or voice workflows.
- Keep runtime state rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/kokoro/docs/OPERATIONS.md) as the architecture boundary for future migrations.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
