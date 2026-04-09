# Plan: Align scenario-to-desktop setup steps with standards

## Purpose

Bring the `build-ui` and `install-ui-deps` setup steps in `scenarios/scenario-to-desktop/.vrooli/service.json` into alignment with the canonical patterns defined in `docs/scenarios/PRODUCTION_BUNDLES.md`. The GCT standards checker flagged line ~121 because the `build-ui` step is missing `VITE_API_BASE_URL` injection and the `API_PORT` validation guard.

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

- `docs/scenarios/PRODUCTION_BUNDLES.md` — canonical setup step patterns

## Problem Statement

The GCT standards checker flagged `.vrooli/service.json` line ~121 with a warning: "Run build-ui before show-urls so develop serves the ui/dist production bundle." While the current setup steps already have `build-ui` before `show-urls` in sequence, the `build-ui` step does not match the canonical pattern from `docs/scenarios/PRODUCTION_BUNDLES.md`:

**Current `build-ui`:**
```json
{
  "name": "build-ui",
  "run": "cd ui && pnpm run build",
  "description": "Build production UI with Vite",
  "condition": { "file_exists": "ui/package.json" }
}
```

**Canonical pattern (from PRODUCTION_BUNDLES.md):**
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
3. Uses `condition` field instead of inline conditionals (canonical pattern uses inline shell)

Similarly, the `install-ui-deps` step uses the `condition` field instead of inline conditionals.

## Scope

### In Scope
- Update `build-ui` step in `.vrooli/service.json` to match canonical pattern (inline conditionals + `VITE_API_BASE_URL` + `API_PORT` guard)
- Update `install-ui-deps` step to use inline conditionals for consistency
- Verify the fix clears the GCT standards warning

### Out of Scope
- Other GCT warnings (code quality, missing tests in runtime/infra, etc.)
- Changes to develop, stop, or test lifecycle phases
- Changes to any other scenario

## Current Technical Context

### Key Files
- `scenarios/scenario-to-desktop/.vrooli/service.json` — lifecycle configuration containing the setup steps (lines ~100-135)
- `docs/scenarios/PRODUCTION_BUNDLES.md` — canonical step patterns (source of truth)

### Current State
- `build-ui` is already positioned before `show-urls` (index 3 vs 4) — ordering is correct
- The step uses `condition.file_exists` instead of inline `if [ -f ... ]`
- The step lacks `VITE_API_BASE_URL` env var injection
- The step lacks `API_PORT` guard
- `install-ui-deps` also uses the `condition` field instead of inline conditionals

## Target End State

Both `install-ui-deps` and `build-ui` steps match the canonical patterns from `PRODUCTION_BUNDLES.md`:

1. **`install-ui-deps`**: Uses inline `if [ -f ui/package.json ]` conditional instead of `condition` field
2. **`build-ui`**: Uses inline conditional, includes `API_PORT` guard, injects `VITE_API_BASE_URL` env var during build
3. GCT standards check no longer flags the setup steps warning
4. Scenario still builds and runs correctly

**Greenfield declaration:** This is a direct config update — no compatibility shims or migration paths needed. The old step definitions are simply replaced with the canonical versions.

## Implementation Strategy

### Phase 1: Update setup steps (single file edit)

1. Edit `scenarios/scenario-to-desktop/.vrooli/service.json`
2. Replace the `install-ui-deps` step:
   - Remove the `condition` field
   - Wrap the `run` command in `if [ -f ui/package.json ]; then ... else echo 'ui/ not present yet'; fi`
3. Replace the `build-ui` step:
   - Remove the `condition` field
   - Use the canonical pattern with `API_PORT` guard and `VITE_API_BASE_URL` injection
   - Update `description` to `"Build production UI (requires lifecycle-assigned API_PORT)"`

### Phase 2: Verify

1. Run GCT standards check to confirm the setup steps warning is cleared
2. Run `vrooli scenario test scenario-to-desktop` to ensure no regressions

## Contract Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Match canonical pattern exactly (option A) | Yes | Full canonical pattern with inline conditionals + VITE_API_BASE_URL + API_PORT guard. Most likely to clear GCT since the checker pattern-matches against this. |
| Update both install-ui-deps and build-ui (option A) | Yes | Consistent canonical patterns across both UI-related setup steps. Reduces chance of other GCT warnings. |

## Testing Plan

- [ ] GCT standards check passes without the setup steps warning on `.vrooli/service.json` line ~121
- [ ] `vrooli scenario test scenario-to-desktop` passes (all 11 existing tests)
- [ ] Scenario can be restarted successfully with `vrooli scenario restart scenario-to-desktop`

## Rollout/Validation Checklist

- [ ] Edit `scenarios/scenario-to-desktop/.vrooli/service.json` — update `install-ui-deps` and `build-ui` steps
- [ ] Run GCT standards check and confirm setup steps warning is cleared
- [ ] Run `vrooli scenario test scenario-to-desktop` — confirm all tests pass
- [ ] Run `vrooli scenario restart scenario-to-desktop` — confirm scenario starts correctly with updated steps

## Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `API_PORT` not set when setup runs | Low | Build fails with clear error message | The guard is the canonical pattern; lifecycle system always sets `API_PORT` |
| Inline conditional syntax error | Low | Setup step fails | Copy exact pattern from PRODUCTION_BUNDLES.md; verify with test run |

## Non-goals / Prohibited Patterns

- Do not change develop, stop, or test lifecycle phases
- Do not address other GCT warnings (code quality, missing tests, etc.)
- Do not add compatibility shims or fallbacks for the old step format
- Do not modify any files outside `scenarios/scenario-to-desktop/.vrooli/service.json`

## Definition of Done

1. `install-ui-deps` and `build-ui` steps in `scenarios/scenario-to-desktop/.vrooli/service.json` match the canonical patterns from `docs/scenarios/PRODUCTION_BUNDLES.md`
2. GCT standards check no longer flags the setup steps warning
3. All existing scenario tests pass
4. Scenario restarts successfully: `vrooli scenario restart scenario-to-desktop`
