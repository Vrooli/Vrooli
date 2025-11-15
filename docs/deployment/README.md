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
| 1 | Full Vrooli stack running locally or on a dev server, proxied through app-monitor + Cloudflare tunnel | ✅ Production ready for us today | [Local / Developer Stack](tiers/tier-1-local-dev.md) |
| 2 | Portable desktop bundles (Windows/macOS/Linux) where UI + API + dependencies ship together | ⚠️ Thin client only today | [Desktop](tiers/tier-2-desktop.md) |
| 3 | Mobile packages (iOS/Android) | 🚧 Not started | [Mobile](tiers/tier-3-mobile.md) |
| 4 | SaaS / Cloud installs (DigitalOcean, AWS, bare metal) | ⚠️ Requires dependency fitness + secret prep | [SaaS / Cloud](tiers/tier-4-saas.md) |
| 5 | Enterprise / Hardware appliance deployments | 🧭 Vision stage | [Enterprise / Appliance](tiers/tier-5-enterprise.md) |

Each tier page captures **current state → gaps → roadmap** so we can coordinate scenario updates.

## Scenario Orchestration Loop

Deployment is a scenario in its own right:

1. `deployment-manager` (future) drives the workflow.
2. It queries `scenario-dependency-analyzer` to pull the full dependency DAG (resources *and* other scenarios) plus their metadata.
3. It scores fitness for the requested tier, highlighting blockers and suggesting swaps.
4. It coordinates with `secrets-manager` to classify/create secrets per tier.
5. It triggers the appropriate `scenario-to-*` packager (desktop/mobile/cloud) to generate installers or remote bundles.
6. When manual work is required (e.g., swapping Postgres → SQLite), it files `app-issue-tracker` tasks.

That loop is spelled out in the [Scenario Docs](./scenarios) section.

## Guides

- [Dependency Swapping](guides/dependency-swapping.md) — use deployment metadata to swap in fitter alternatives.
- [Fitness Scoring](guides/fitness-scoring.md) — scoring rubric and metadata schema extension for `service.json`.
- [Secrets Management](guides/secrets-management.md) — infrastructure vs service vs user secrets lineage.
- [Deployment Checklist](guides/deployment-checklist.md) — per-tier readiness check.
- [Packaging Matrix](guides/packaging-matrix.md) — what `scenario-to-*` can actually produce today.

## Providers & Infrastructure Notes

Legacy platform-specific instructions were preserved for reference:

- [DigitalOcean](providers/digitalocean.md) — VPS/Kubernetes setup details (costing, `doctl`, etc.).
- [Cloudflare Tunnel](providers/cloudflare-tunnel.md) — Secure Tier 1 remote access via app-monitor.
- [Hardware Appliance](providers/hardware-appliance.md) — Planning notes for Tier 5 devices.

These provider notes feed into the SaaS/Enterprise tiers once the deployment-manager can emit infrastructure manifests.

## Examples

We document the true experience per tier using real scenarios:

- [Picker Wheel Desktop](examples/picker-wheel-desktop.md) — thin client reality + bundling gaps.
- [Picker Wheel Cloud](examples/picker-wheel-cloud.md) — what running the scenario on a VPS entails today.
- [System Monitor Desktop](examples/system-monitor-desktop.md) — another case study for dependency swapping.

## Historical Docs

Everything that described the old "package-scenario-deployment.sh" era now lives in [history](history). The content is still useful when we eventually support Kubernetes/SaaS installs, but the guidance is clearly marked as legacy so it doesn't mislead agents.

## Roadmap Snapshot

1. Document current truth (this hub + spokes). ✅
2. Extend `service.json` with `deployment.platforms` metadata (fitness, requirements, alternatives). 🔄
3. Upgrade `scenario-dependency-analyzer` to compute resource tallies and cascade fitness scores. 🔄
4. Build the `deployment-manager` scenario UI (dependency visualization, swap tool, secret prep). 🔜
5. Teach `scenario-to-desktop/mobile/cloud` to read deployment bundles produced by deployment-manager. 🔜
6. Close the loop with app-issue-tracker automation for required swaps/migrations. 🔜

Until the automation exists, the docs act as the contract for how deployment *should* work, preventing another scenario-to-desktop surprise.
