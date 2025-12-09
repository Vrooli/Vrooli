# Scenario: deployment-manager

## Purpose

Provide an orchestration UI + API that turns "I want scenario X on platform Y" into actionable steps: analyze dependencies, score fitness, suggest swaps, coordinate secrets, and trigger packagers.

## Responsibilities

1. **Inventory** — Read scenario/service dependency graphs from `scenario-dependency-analyzer` outputs.
2. **Visualization** — Display dependencies with fitness badges, requirements, and swap recommendations.
3. **What-If Editing** — Let users toggle dependencies or choose suggested alternatives to see new fitness scores.
4. **Secret Planning** — Pull secret manifests from `secrets-manager`, label each as infrastructure/service/user, and configure prompts or generators.
5. **Bundle Export** — Generate bundle manifests consumed by `scenario-to-desktop/mobile/cloud` packagers.
6. **Issue Creation** — File app-issue-tracker issues when swaps require engineering changes.

## UI Outline

- **Target Picker** — Choose scenario + tier + platform (desktop OS, mobile platform, cloud provider).
- **Fitness Dashboard** — Table similar to the one described in the prompt (`Dependency | Current | Fitness | Suggested Swap`).
- **Swap Drawer** — Inspect alternatives, read pros/cons, accept or reject.
- **Secrets Panel** — Shows which secrets will be bundled/generated/prompted.
- **Export Modal** — Download bundle manifest, or trigger `scenario-to-*` automatically.

## Data Contracts

- **Input:**
  - `service.json` + `.vrooli/service-dependencies.json` (output of scenario-dependency-analyzer).
  - Deployment metadata across scenarios/resources.
- **Output:**
  - `bundle.json` describing chosen dependencies, versions, requirements, secrets, configuration.
  - `swap-decisions.json` for issue tracking.

### Bundle Schema

- Desktop bundle exports must validate against `docs/deployment/bundle-schema.desktop.v0.1.json` before they are handed to packagers.
- Reference manifests (baseline SQLite + API, Playwright-enabled) live in `docs/deployment/examples/manifests/` and should be used in tests/fixtures until deployment-manager emits its own.

## Roadmap

1. Build read-only dashboard (no swaps) to replace spreadsheets.
2. Add swap editing with optimistic calculations.
3. Integrate secrets-manager to preview secret actions per tier.
4. Wire "Export bundle" button to scenario-to-desktop.
5. Expand to mobile/cloud packagers.

## Success Metrics

- **Coverage:** 100% of high-priority scenarios publish deployment metadata that surfaces cleanly in the dashboard.
- **Insight Speed:** Fitness view renders in <2s for scenarios with up to 200 dependencies thanks to cached analyzer output.
- **Actionability:** Every swap decision can auto-create an app-issue-tracker issue (pre-filled context) with a single click.
- **Bundle Accuracy:** Bundle manifests produced here drive at least one downstream packager (desktop/cloud) without manual edits.

## Milestones

| Phase | Deliverable | Notes |
|-------|-------------|-------|
| M1 | Read-only inspector + dependency heat map | Blocks manual doc hunting. Pulls analyzer cache + secrets classification. |
| M2 | Swap sandbox + recalculated scores | Accept/undo dependency swaps, persist proposals, push deltas to app-issue-tracker. |
| M3 | Bundle manifest export + desktop automation | Trigger scenario-to-desktop with the manifest; capture telemetry back into deployment-manager. |
| M4 | Provider integrations | Kick scenario-to-cloud/mobile with validated manifests; surface deploy status + logs. |

## Risks & Open Questions

- **Data Freshness:** Need background jobs (or CLI hook) to keep analyzer data up to date; stale inputs produce misleading scores.
- **Secret Propagation:** How do we guarantee user-secret prompts are localized per tier? Might require new secrets-manager contracts.
- **Swap Explosion:** Avoid presenting dozens of swap combos; heuristics (cost/impact) needed to keep UI manageable.
- **Access Control:** Enterprise tiers will need RBAC + audit logging for bundle exports.

## Dependencies

- scenario-dependency-analyzer (enhanced)
- secrets-manager
- scenario-to-desktop/mobile/cloud
- app-issue-tracker (issue creation)
