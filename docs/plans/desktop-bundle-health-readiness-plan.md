# Desktop Bundle Health/Readiness Alignment Plan

## Context
- `vrooli scenario status <scenario>` uses standard health endpoints at `http://localhost:${UI_PORT}/health` and `http://localhost:${API_PORT}/health` and validates schema compliance. (See `cli/commands/scenario/modules/health.sh` and `cli/commands/scenario/schemas/health-*.schema.json`.)
- Desktop bundle preflight readiness is driven by the bundle manifest and the runtime readiness checker, but:
  - API readiness in the generated manifest currently uses `port_open` even though `/health` is available.
  - UI manifest health path is currently `/`, not `/health`.
  - `ui-bundle` readiness bypasses the runtime health checker and marks ready immediately after port bind.

## Decisions (from user responses)
- UI readiness behavior: **Use full health checker loop for ui-bundle** (no bypass).
- API readiness: **Switch to `health_success`** so `/health` is authoritative.
- UI health path: **`/health`** (confirmed consistent with scenario status).
- Health timing values: **Keep existing defaults**.
- Runtime scope: **Apply to all UI bundles** (no opt-in metadata).
- Tests: **Add unit tests + runtime test** for ui-bundle readiness behavior.
- Plan file location: **`docs/plans/`**.

## Goals
- Preflight readiness uses the same health endpoints and semantics as `vrooli scenario status`.
- API readiness reflects actual health, not just port availability.
- UI readiness uses `/health` and is validated through the runtime health checker.
- No behavior divergence between bundle preflight and scenario status health definitions.

## Non-Goals
- Change the health endpoint schema or payload format.
- Add new health endpoints to APIs/UIs (already present per standard).
- Introduce new service types or metadata flags for health behavior.

## Files to Change
- `scenarios/scenario-dependency-analyzer/api/internal/deployment/manifest.go`
  - API readiness: change `readiness.type` to `health_success`.
  - UI health path: change from `/` to `/health`.
  - UI readiness remains `health_success`.
- `scenarios/scenario-to-desktop/runtime/service_launcher.go`
  - `ui-bundle` startup should run the health checker loop (same readiness mechanism as other services).
  - Ready state should only be set after health_success check passes.
- `scenarios/scenario-to-desktop/runtime/health/checker.go`
  - Ensure UI log path usage and check semantics remain correct for ui-bundle; no schema changes.
- Tests:
  - `scenarios/scenario-dependency-analyzer/api/internal/app/deployment_manifest_test.go` (or related manifest tests)
  - `scenarios/scenario-to-desktop/runtime/service_launcher_test.go` (or runtime readiness tests)

## Implementation Steps
1) Manifest generation (scenario-dependency-analyzer)
   - Update API skeleton readiness to `health_success` with `/health` and `port_name=api`.
   - Update UI skeleton health path to `/health`.
   - Keep current intervals, timeouts, and retries.
2) Runtime readiness for ui-bundle
   - After `ui-bundle` server binds to the port, invoke the health checker loop (as `health_success` readiness) and only mark ready when it passes.
   - If health fails, mark `Ready=false` with the error and surface via `/readyz`.
3) Tests
   - Add/adjust manifest generation tests to assert API readiness type and UI health path are set correctly.
   - Add runtime tests to verify ui-bundle readiness follows health checker results (ready when `/health` returns 2xx, not ready otherwise).
4) Validate that preflight readiness now reflects `/health` status for both API and UI.

## Testing Plan
- Unit tests:
  - Manifest generation tests verifying API readiness type and UI health path.
  - Runtime readiness tests for ui-bundle health_success flow.
- Optional manual verification:
  - Run bundle preflight and confirm services readiness fails if `/health` is unhealthy for API/UI.

## Risks and Mitigations
- **Risk:** UI bundles that serve `/health` but do so slowly could delay readiness.
  - **Mitigation:** Keep current timeouts; adjust only if real scenarios show false negatives.
- **Risk:** Tight coupling between readiness and health could block startup for transient health issues.
  - **Mitigation:** Health retries and interval already provide backoff; can revisit thresholds if needed.

## Open Questions
- None (all decisions resolved in this plan).
