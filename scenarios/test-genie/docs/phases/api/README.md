# API Phase

The `api` phase delegates API readiness validation to **api-health** through the shared `ScenarioValidationService`.

## What It Owns

API Health owns API-specific readiness:

- `.vrooli/service.json` API and health metadata
- API startup lifecycle and preflight wiring
- `/health` schema and optional live probe evidence
- route-aware HTTP response semantics
- API-runtime hygiene such as outbound HTTP timeouts, response body closure, request-context propagation, and cancellable long-running work
- deterministic API fix preview/apply metadata

Other standards-like checks live with their focused health providers rather than a catch-all Test Genie phase.

## How It Runs

Test Genie resolves `api-health` and calls its provider contract:

```bash
api-health validate scenario <scenario>
```

Execution mode is not requested by Test Genie by default, so the phase stays bounded and static unless API Health is run directly with execution enabled.

## Opt-Out

Skip for a single run:

```bash
test-genie execute <scenario> --skip api
```

Disable per scenario via `.vrooli/testing.json`:

```json
{
  "phases": {
    "api": { "enabled": false }
  }
}
```

## Configuration

Per-scenario timeout override via `.vrooli/testing.json`:

```json
{
  "phases": {
    "api": { "timeout": "120s" }
  }
}
```

## Provider Detail

Run the provider directly for native API Health evidence and fix previews:

```bash
api-health validate scenario <scenario>
api-health validate fix-preview <scenario>
```
