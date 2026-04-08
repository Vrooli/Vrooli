# Plan: Align scenario-to-desktop setup steps with standards

## Required Reading

- `prompt-manager skill read implementation-plan-authoring`
- `docs/scenarios/PRODUCTION_BUNDLES.md` — canonical setup step patterns

## Problem Statement

The GCT standards checker flagged `.vrooli/service.json` line ~121 with a warning: "Run build-ui before show-urls so develop serves the ui/dist production bundle." While the current setup steps already have `build-ui` before `show-urls` in sequence, the `build-ui` step does not match the canonical pattern from `docs/scenarios/PRODUCTION_BUNDLES.md`:

**Current:**
```json
{
  "name": "build-ui",
  "run": "cd ui && pnpm run build",
  "description": "Build production UI with Vite",
  "condition": { "file_exists": "ui/package.json" }
}
```

**Standard pattern (from PRODUCTION_BUNDLES.md):**
```json
{
  "name": "build-ui",
  "run": "if [ -f ui/package.json ]; then if [ -z \"${API_PORT:-}\" ]; then echo 'API_PORT must be provided by the lifecycle system'; exit 1; fi; cd ui && VITE_API_BASE_URL=\"http://localhost:${API_PORT}/api/v1\" pnpm run build; else echo 'ui/ not present yet'; fi",
  "description": "Build production UI (requires lifecycle-assigned API_PORT)"
}
```

Key gaps:
1. Missing `VITE_API_BASE_URL` environment variable injection during build
2. Missing `API_PORT` validation guard
3. Conditional structure differs from the canonical pattern

## Scope

### In Scope
- Update `build-ui` step in `.vrooli/service.json` to match production bundle guidance
- Verify the fix clears the GCT standards warning

### Out of Scope
- Other GCT warnings (code quality, missing tests in runtime/infra, etc.)
- Changes to develop, stop, or test lifecycle phases

## Approach

### Phase 1: Update build-ui step
1. Edit `scenarios/scenario-to-desktop/.vrooli/service.json`
2. Replace the `build-ui` step's `run` command with the canonical pattern including `VITE_API_BASE_URL` and `API_PORT` guard
3. Update the `description` field to match the standard

### Phase 2: Verify
1. Run GCT standards check to confirm the warning is cleared
2. Run `vrooli scenario test scenario-to-desktop` to ensure no regressions

## Risks

- **Low**: The change is a configuration-only update to the build command
- The scenario already builds and runs correctly; this just adds the `VITE_API_BASE_URL` env var and a port guard

## Test Plan

- [ ] GCT standards check passes without the setup steps warning
- [ ] `vrooli scenario test scenario-to-desktop` passes
