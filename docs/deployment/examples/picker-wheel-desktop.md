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

## Next Steps for Tier 2 Readiness

1. Add deployment metadata to picker-wheel dependencies (mark Postgres as desktop-unfit, suggest SQLite).
2. Use deployment-manager (once available) to simulate swaps and generate bundle manifest.
3. Enhance scenario-to-desktop to bundle the Picker Wheel API + swapped dependencies.
4. Implement first-run wizard prompting for optional AI provider (local Ollama vs OpenRouter).

Until then, document this limitation anywhere picker-wheel desktop is shared.
