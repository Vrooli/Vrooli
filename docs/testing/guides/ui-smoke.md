# UI Smoke Harness

The UI smoke harness is the final gate in the structure phase. It loads the production UI bundle through Browserless, waits for the iframe bridge to signal readiness, and captures screenshots, console logs, network traces, and HTML snapshots so regressions surface before business logic or Lighthouse tests run.

## What the harness checks

- **Bundle freshness** – runs the same `ui-bundle` setup condition used by lifecycle commands and fails if `ui/dist` is missing or stale relative to `ui/src`.
- **Browserless connectivity** – ensures the shared `resource-browserless` service is healthy before launching a session. Offline Browserless instances fail the phase immediately with an actionable error.
- **Iframe bridge readiness** – verifies `@vrooli/iframe-bridge` is declared in `ui/package.json` and loads the app inside a synthetic iframe so `initIframeBridgeChild()` executes. The harness then waits for `window.__vrooliBridgeChildInstalled` (or equivalent) before marking the run as successful.
- **Runtime capture** – stores screenshots, console messages, failed requests, and HTML snapshots under `coverage/<scenario>/ui-smoke/` for debugging and ingestion by `vrooli scenario status`.

## Running it manually

```bash
# Always start from repo root
vrooli scenario ui-smoke picker-wheel           # Human-readable summary
vrooli scenario ui-smoke picker-wheel --json    # Machine-readable summary
```

The CLI command simply wraps the shared helper at `scripts/scenarios/testing/shell/ui-smoke.sh`, so the output matches what the structure phase consumes. The helper renders a lightweight host page that iframes the target UI, which guarantees bridge initialization even though Browserless connects directly to `http://localhost:<UI_PORT>`. When the harness fails it still writes `coverage/<scenario>/ui-smoke/latest.json` so future agents can pick up from the last known state.

## Artifacts

| File | Contents |
|------|----------|
| `coverage/<scenario>/ui-smoke/latest.json` | Canonical summary (status, bundle info, Browserless metadata, handshake result) |
| `coverage/<scenario>/ui-smoke/screenshot.png` | Full-page screenshot captured via Browserless |
| `coverage/<scenario>/ui-smoke/console.json` | All console messages (level, timestamp) |
| `coverage/<scenario>/ui-smoke/network.json` | Failed requests and HTTP 4xx/5xx responses |
| `coverage/<scenario>/ui-smoke/dom.html` | HTML snapshot for offline debugging |
| `coverage/<scenario>/ui-smoke/raw.json` | Browserless response minus the screenshot payload |

Scenario status ingests the latest summary and renders a concise “UI Smoke” block so agents can see the status, handshake timing, bundle freshness, and artifact paths without inspecting the filesystem.

## Configuration knobs (optional)

The harness enables itself automatically whenever a scenario exposes a `ui/` bundle, so most scenarios do **not** need additional configuration. Add overrides to `.vrooli/testing.json` only when you need to tweak behaviour:

```json
{
  "structure": {
    "ui_smoke": {
      "enabled": false,
      "timeout_ms": 120000,
      "handshake_timeout_ms": 30000
    }
  }
}
```

- `enabled`: set to `false` for UI-less scenarios or temporary migrations; structure will record a skipped check instead of failing.
- `timeout_ms`: increase when the UI needs extra time to boot or load data.
- `handshake_timeout_ms`: extend when iframe bridge initialization is intentionally delayed (e.g., heavy auth handshakes).

If the block is omitted altogether, the harness falls back to `enabled=true`, `timeout_ms=90000`, and `handshake_timeout_ms=15000`.

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| `Browserless resource is offline` | `resource-browserless` stopped or unhealthy | `resource-browserless manage start` then rerun the harness |
| `Bundle stale` warning | `ui/dist` missing/outdated relative to `ui/src` | `pnpm --dir ui build` or `vrooli scenario run <name> --setup` to rebuild |
| `Iframe bridge dependency missing` | `@vrooli/iframe-bridge` not listed in `ui/package.json` | Add it via `pnpm --dir ui add @vrooli/iframe-bridge` and initialize in `ui/src/main.tsx` |
| `Bridge handshake: ❌` | UI never called `iframeBridge.ready()` or JS crashed before initialization | Check `console.json` and `dom.html` to identify runtime errors |

## Where it shows up

- **Structure phase (`test/phases/test-structure.sh`)** – fails the phase if the harness reports `blocked` or `failed`.
- **`vrooli scenario status <name>`** – displays the latest smoke result, handshake status, bundle freshness, and screenshot path.
- **CLI command (`vrooli scenario ui-smoke`)** – human-readable or JSON summary for manual runs.

Keep this harness green before claiming any operational target is complete. It is the fastest way to catch missing bundles, Browserless outages, or iframe bridge regressions.
