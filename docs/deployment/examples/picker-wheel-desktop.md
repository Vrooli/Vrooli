# Example: Picker Wheel — Desktop

## What We Built

- Scenario: `picker-wheel`
- Packager: `scenario-to-desktop` (Electron)
- Output: Windows `.exe` bundling the UI and Electron runtime.

## Reality Check

| Component | Status |
|-----------|--------|
| UI Bundle | ✅ Bundled (production build)
| API Server | ❌ Not bundled — still runs in Tier 1 environment
| Resources (Postgres, Redis) | ❌ Not bundled
| Secrets | ⚠️ Uses dev secrets by connecting back to Tier 1 server
| Offline Support | ❌ None

Result: The desktop app only works when a developer Tier 1 environment is online and reachable.

## Lessons Learned

1. **Thin client ≠ deployment** — Users expect the desktop app to work standalone.
2. **Dependency graph matters** — Without bundling Postgres/Redis (or swapping to SQLite/embedded caches) the app cannot store data.
3. **Secrets strategy needed** — Shipping dev secrets is unsafe; we need onboarding prompts for user-provided API keys or generated credentials.

## How to Run the Thin Client Today

1. Install/verify the Vrooli CLI on the host that will run `picker-wheel` (`vrooli --version`). Download the release zip or clone the repo if it is missing, then run `./scripts/manage.sh setup --yes yes` (skip privileged prompts when the OS cannot elevate).
2. Start the scenario in that Tier 1 stack: `vrooli scenario start picker-wheel` (or `cd scenarios/picker-wheel && make start`) and wait for `vrooli scenario status picker-wheel` to show healthy.
3. Decide how the desktop wrapper reaches the UI/API: either keep the LAN URL (`http://<host>:<UI_PORT>`) or copy the Cloudflare proxy URL from app-monitor.
4. Before building the Electron app, set `APP_CONFIG.SERVER_PATH` and `APP_CONFIG.API_ENDPOINT` in `platforms/electron/src/main.ts` to the URL from step 3 and keep `APP_CONFIG.DEPLOYMENT_MODE` on `external-server`.
5. Build + distribute via `npm run dist:win|mac|linux`. When testers are done, stop the Tier 1 scenario with `vrooli scenario stop picker-wheel` to free resources.

Each desktop build writes telemetry to `<userData>/deployment-telemetry.jsonl` (Platform-specific: `%APPDATA%/Picker Wheel/…`, `~/Library/Application Support/Picker Wheel/…`, or `~/.config/Picker Wheel/…`). deployment-manager will parse those `server_ready` / `dependency_unreachable` events to prioritize bundling work.

> **Automation assist:** The generated Electron wrapper ships with `AUTO_MANAGE_TIER1=true`, so on launch it will look for the `vrooli` CLI, run `vrooli setup --yes yes --skip-sudo yes` if needed, start `vrooli scenario start picker-wheel`, and call `vrooli scenario stop picker-wheel` when the desktop app exits. The manual checklist remains valuable when the CLI is missing or you deliberately target a remote Tier 1 instance.

## Next Steps for Tier 2 Readiness

1. Add deployment metadata to picker-wheel dependencies (mark Postgres as desktop-unfit, suggest SQLite).
2. Use deployment-manager (once available) to simulate swaps and generate bundle manifest.
3. Enhance scenario-to-desktop to bundle the Picker Wheel API + swapped dependencies.
4. Implement first-run wizard prompting for optional AI provider (local Ollama vs OpenRouter).

Until then, document this limitation anywhere picker-wheel desktop is shared.
