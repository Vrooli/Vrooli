# UI Smoke Harness Implementation Plan

## Why this matters
- Agents frequently miss UI crashes that only appear once the production bundle is served. `vrooli scenario status` only pings `/health` or the bare root URL, so serious bundling/runtime failures slip through even though we force production builds.
- We need deterministic evidence that every scenario's UI boots, renders, and connects to its API before claiming operational targets are complete. Adding a shared smoke harness gives ecosystem-manager loops the feedback they lack today.
- Browserless is already part of the Tier 1 stack and provides console/network capture plus screenshots. Leveraging it avoids per-scenario Playwright installs and keeps the validation stack consistent with other tooling.

## Scope
1. Shared smoke runner in `scripts/scenarios/testing/shell/ui-smoke.sh`
2. CLI entry point `vrooli scenario ui-smoke <scenario>` (with `--json` support for automation)
3. Automatic invocation inside the structure phase so `make test` / `vrooli scenario test` fails when the UI is broken (reuses the existing runtime auto-start helpers)
4. Diagnosis surfaced via `vrooli scenario status <name>` and required in scenario-improver prompts
5. Minimal opt-out config + per-scenario timeout override in `.vrooli/testing.json` under the `structure` key
6. Bundle staleness detection surfaced everywhere the smoke harness reports results (including human-readable reasons)

## Detailed plan

### 1. Shared smoke runner
- Create `scripts/scenarios/testing/shell/ui-smoke.sh` alongside other shared helpers.
- Responsibilities:
  - Discover scenario + UI port using `vrooli scenario port <name> UI_PORT` (reuse logic from `testing::connectivity::get_ui_url`).
  - Reuse `testing::runtime` to auto-start the scenario when the harness requires a running UI. This keeps start/wait/stop behavior consistent with integration/performance phases and avoids inventing bespoke lifecycle code.
  - Detect stale UI builds by invoking the existing `scripts/lib/setup-conditions/ui-bundle-check.sh` helper. If it returns "setup needed", fail with a targeted message instructing agents to run the `build-ui` lifecycle step before rerunning the smoke test. Record the stale flag in the JSON artifact so downstream tools (scenario status, integration phase) can display the warning even if the Browserless call subsequently succeeds or fails.
  - Run a Browserless preflight before launching the smoke session. Add a helper that shells out to `resource-browserless status` (Tier 1 already installs this CLI) to verify the service is running and capture its port, falling back to the `resource-browserless manage start` instructions when offline. If Browserless is unreachable, exit with a dedicated status code (e.g., 50) so callers can render "blocked: browserless offline" but still fail the phase.
  - Enforce iframe-bridge adoption before declaring the smoke pass:
    - Read the scenario’s `ui/package.json` and assert that `@vrooli/iframe-bridge` (or the shared workspace package) appears under dependencies/devDependencies. If it’s missing, fail with a targeted message that links to the iframe-bridge README and explains why it’s required for app-monitor/app-monitor proxies.
    - While controlling Browserless, inject a short script that waits for `window.IFRAME_BRIDGE_READY === true` or `window.IframeBridge?.ready === true`. If the flag never appears within the smoke timeout, treat it as a bridge failure and include remediation instructions (e.g., “import the bridge in ui/src/main.tsx and expose it with iframeBridge.ready()”). This ensures legacy scenarios upgrade rather than bypassing the requirement.
  - Call Browserless via the existing adapters (`resources/browserless/lib/actions.sh`/`api.sh`): navigate to the UI port, wait for DOM ready, wait for the iframe-bridge handshake flag, capture a screenshot, dump console logs, and record network errors/HAR data. Treat any uncaught exception, navigation timeout, or HTTP 4xx/5xx on initial load as failure. **Do not add Playwright/Selenium fallbacks**—Browserless is the single supported runtime so we stay aligned with production workflows.
  - Emit `coverage/<scenario>/ui-smoke/latest.json` capturing pass/fail, duration, bundle-staleness state (include reason text surfaced from the bundle check helper), iframe-bridge dependency/handshake status, Browserless status, and artifact paths. Store artifacts under `coverage/<scenario>/ui-smoke/` (PNG screenshot, console JSON, HAR, HTML snapshot). Always write the JSON even when the harness fails so downstream tooling has structured diagnostics.
  - Exit non-zero on failures so callers fail immediately: `1` for functional/UI errors, custom `50` for Browserless offline, `60` for stale bundle, etc. All dependencies live inside Tier 1; when Browserless is unreachable we intentionally fail so loops fix the environment before claiming success.

### 2. CLI entry point
- Add a new module `cli/commands/scenario/modules/ui-smoke.sh` with `scenario::smoke::run <scenario> [--json]` that invokes the shared helper and prints a concise summary by default or machine-readable output with `--json`, matching other `vrooli` commands. Include iframe-bridge dependency/handshake highlights in the summary so agents see exactly why a run failed (e.g., “Bridge dependency missing” or “Bridge never signaled ready”).
- Wire it into `cli/commands/scenario/scenario-commands.sh` (help text + dispatch case) so agents can run `vrooli scenario ui-smoke picker-wheel` manually for debugging.
- Include exit-code mapping in the help output (success, bundle stale, Browserless offline) so downstream scripts know how to react.

### 3. Structure phase integration
- Wire the helper directly into `testing::structure::validate_all` (the shared implementation in `scripts/scenarios/testing/shell/structure.sh`). This keeps the behavior centralized—once the helper exists, every scenario automatically gains the smoke check without per-template edits.
- Flip `TESTING_SUITE_DEFAULT_RUNTIME[structure]` to `true` when the harness is enabled so the standard runtime manager auto-starts/stops the scenario before/after the smoke step. Also raise `TESTING_SUITE_DEFAULT_TIMEOUTS[structure]` from 30s to around 120s (or higher if Browserless start times demand it) because spinning up the scenario plus Browserless takes materially longer than today. Document the new expectation everywhere the structure phase runs (including `vrooli scenario status`) so teams understand why it now takes ~1 minute instead of a few seconds.
- Add a subroutine like `testing::structure::run_ui_smoke` that:
  - Skips automatically when no UI exists (no `ui/package.json`).
  - Honors `.vrooli/testing.json` flag (`structure.ui_smoke.enabled: false` disables it).
  - Calls the shared smoke helper (which includes the Browserless preflight and bundle-staleness detection) and uses `testing::phase::add_test passed|failed|skipped` so the structure phase reports the smoke result along with the other checks. If the helper exits with the Browserless-offline code, treat it as `failed (blocked)` so the phase stops and agents fix the resource.
  - Registers the smoke artifacts with `scripts/scenarios/testing/shell/artifacts.sh` (mirroring our existing artifact collectors) so the latest screenshot/log bundle always lives under `coverage/<scenario>/ui-smoke/` and `scenario status` can link directly to it. Include screenshot, HAR, console log, HTML snapshot, and JSON summary.
- Because the harness runs at the end of `validate_all`, `test/phases/test-structure.sh` in each scenario stays untouched and the only artifacts are the shared ones under `coverage/`.

### 4. Scenario status + prompt updates
- Extend `scenario::health::collect_all_diagnostic_data` (cli/commands/scenario/modules/health.sh) to read the latest smoke JSON and include it in diagnostics. The display formatter should echo a short summary such as `UI smoke: ✅ (1.2s) → coverage/.../ui-smoke/` or the failure reason so the rest of the command output stays concise, while still surfacing bundle-fresh, iframe-bridge dependency/handshake, and Browserless status fields.
- Surface the bundle-staleness bit in the same diagnostic block (e.g., `bundle_fresh: true/false` with the stale reason from the helper) so agents can see immediately whether they need to rebuild before retesting.
- Update scenario-improver prompt (`scenarios/ecosystem-manager/prompts/templates/scenario-improver.md`) Quick Validation Loop to add step 4: `vrooli scenario ui-smoke {{TARGET}}` (attach summary/artifacts). Mention this requirement in the generator template’s validation notes and in the scenario README template so every scenario explains how to rerun the smoke harness locally.

### 5. Configuration knobs
- Support an opt-out flag plus timeout override in `.vrooli/testing.json`:
  ```json
  {
    "structure": {
      "ui_smoke": {
        "enabled": false,
        "timeout_ms": 60000
      }
    }
  }
  ```
  If absent, default `enabled=true` and use the shared timeout (90–120s). The helper also accepts scenario-level overrides for selectors or wait-until options later if needed.

- Wire the smoke helper into `scripts/scenarios/testing/shell/artifacts.sh` so screenshots, console logs, HTML dump, and HAR files are collected under `coverage/<scenario>/ui-smoke`. This guarantees `scenario status` and the CLI can always display the paths that matter.
- Provide unit/smoke coverage by crafting a fake scenario with a deliberately broken UI to ensure failures propagate through `test-structure`, `vrooli scenario ui-smoke`, and `vrooli scenario status`.
- Document troubleshooting steps in a new `docs/testing/guides/ui-smoke.md` (symptoms, how to interpret artifacts, how to opt out temporarily, how to fix Browserless/resource issues, and how to restart Browserless via `resource-browserless manage start`). Include a TODO for future authentication-aware flows (pluggable login scripts/fixtures) so we call out the limitation without expanding this scope.
- Once merged, ecosystem-manager loops will naturally fail earlier (structure phase/test command). Prompts + CLI ensure agents know how to re-run the harness locally, and bundle-staleness warnings appear everywhere agents look (CLI, smoke artifacts, scenario status).
