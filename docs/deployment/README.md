# Deployment Hub

Vrooli deployment is no longer a single script that pushes containers. Every scenario has to be evaluated for *where* it will live, *which* resources must travel with it, and *how* secrets are provisioned. This hub replaces the legacy "package-and-ship" docs with a tiered model that matches reality today and the system we are building next.

## Why a Tiered Model?

We validated during `scenario-to-desktop` that "build an Electron app" is useless unless the UI, API, resources, and CLI dependencies all come along for the ride. That failure exposed three immovable facts:

1. **The current local stack *is* a deployment tier.** Cloudflare tunnels + app-monitor already let us access every running scenario from anywhere.
2. **Portability demands intelligence.** We have to understand dependency graphs, fitness for each platform, and offer swap suggestions before packaging.
3. **Secrets behave differently per tier.** Infrastructure credentials cannot leave the mothership, whereas per-install service secrets must be generated anew.

The hub orchestrates these ideas so future automation (deployment-manager) has a clear target.

## Deployment Tiers

| Tier | Description | Current Viability | Doc |
|------|-------------|-------------------|-----|
| 1 | Full Vrooli stack running locally or on a dev server, proxied through app-monitor + Cloudflare tunnel | ✅ Production ready for us today | [Production Operations Guide](../operations/production-guide.md) |
| 2 | Portable desktop bundles (Windows/macOS/Linux) where UI + API + dependencies ship together | ⚠️ Thin client only today | [Desktop Bundle Readiness Plan](../plans/desktop-bundle-health-readiness-plan.md) |
| 3 | Mobile packages (iOS/Android) | 🚧 Not started | [Roadmap](../strategy/roadmap.md) |
| 4 | SaaS / Cloud installs (DigitalOcean, AWS, bare metal) | ⚠️ Requires dependency fitness + secret prep | [Kubernetes Infrastructure Reference](../devops/kubernetes.md) |
| 5 | Enterprise / Hardware appliance deployments | 🧭 Vision stage | [Business Solutions](../strategy/business-solutions.md) |

Each tier page captures **current state → gaps → roadmap** so we can coordinate scenario updates.

## Scenario Orchestration Loop

Deployment is a scenario in its own right:

1. `deployment-manager` (future) drives the workflow.
2. It queries `scenario-dependency-analyzer` to pull the full dependency DAG (resources *and* other scenarios) plus their metadata.
3. It scores fitness for the requested tier, highlighting blockers and suggesting swaps.
4. It coordinates with `secrets-manager` to classify/create secrets per tier.
5. It triggers the appropriate `scenario-to-*` packager (desktop/mobile/cloud) to generate installers or remote bundles.
6. When manual work is required (e.g., swapping Postgres → SQLite), it files `app-issue-tracker` tasks.

That loop builds on the current scenario and operations docs:

- [../scenarios/README.md](../scenarios/README.md)
- [../operations/production-guide.md](../operations/production-guide.md)

## Current Deployment References

- [../operations/production-guide.md](../operations/production-guide.md) — current operational baseline for Tier 1 environments
- [reference/server-deployment.md](reference/server-deployment.md) — current server-oriented deployment guidance
- [../devops/kubernetes.md](../devops/kubernetes.md) — historical and future-facing Kubernetes context
- [storage.md](storage.md) — specialized bundle storage guidance for deployment-oriented scenarios
- [../plans/desktop-bundle-health-readiness-plan.md](../plans/desktop-bundle-health-readiness-plan.md) — current desktop portability planning
- [../plans/resource-cross-platform-migration-plan.md](../plans/resource-cross-platform-migration-plan.md) — cross-platform resource migration planning

## Provider And Infrastructure Notes

Provider-specific deployment documentation has not yet been rebuilt into a canonical leaf set.

For now, use:

- [reference/server-deployment.md](reference/server-deployment.md) for current server assumptions
- [../strategy/personal-ai-server.md](../strategy/personal-ai-server.md) for exploratory appliance-style thinking
- [../strategy/business-solutions.md](../strategy/business-solutions.md) for the enterprise and appliance framing

## Examples

The deployment examples layer has not been rebuilt yet. Until it exists again, use live scenario docs plus the planning docs above rather than relying on missing case-study pages.

## Bundled Runtime Expectations (Desktop/Mobile/Cloud)

- Bundles must be manifest-driven: deployment-manager + scenario-dependency-analyzer emit a `bundle.json` encoding the full DAG, dependency swaps, per-OS binaries/assets, env templates, port ranges, health/readiness, data dirs, and secrets strategy (generate/prompt/remote).
- A cross-platform runtime executable owns lifecycle, ports, health, logs, telemetry, and shutdown. UI shells (Electron, mobile bridges, cloud runners) only start the runtime and talk to it over a local control channel.
- Heavy/shared resources must be swapped to bundleable equivalents (e.g., Postgres→SQLite/duckdb, Redis→in-process cache, browserless→bundled Playwright driver/Chromium, Ollama→packaged models) before inclusion.
- Bundles carry migrations/seed data for swapped stores, keep data/logs under OS app data roots, and include a minimal `vrooli`-compatible shim for essentials like `scenario status/port`.
- No infrastructure secrets ship in bundles; first-run UX collects or generates only what the manifest flags as local.

## Historical Docs

Older package-and-ship material should be treated as historical reference only. It should not override the current tiered model or the current operational docs listed above.

## Roadmap Snapshot

1. Document current truth (this hub + spokes). ✅
2. Extend `service.json` with `deployment.platforms` metadata (fitness, requirements, alternatives). 🔄
3. Upgrade `scenario-dependency-analyzer` to compute resource tallies and cascade fitness scores. 🔄
4. Build the `deployment-manager` scenario UI (dependency visualization, swap tool, secret prep). 🔜
5. Teach `scenario-to-desktop/mobile/cloud` to read deployment bundles produced by deployment-manager. 🔜
6. Close the loop with app-issue-tracker automation for required swaps/migrations. 🔜

Until the automation exists, the docs act as the contract for how deployment *should* work, preventing another scenario-to-desktop surprise.
