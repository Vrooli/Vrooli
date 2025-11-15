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

## Roadmap

1. Build read-only dashboard (no swaps) to replace spreadsheets.
2. Add swap editing with optimistic calculations.
3. Integrate secrets-manager to preview secret actions per tier.
4. Wire "Export bundle" button to scenario-to-desktop.
5. Expand to mobile/cloud packagers.

## Dependencies

- scenario-dependency-analyzer (enhanced)
- secrets-manager
- scenario-to-desktop/mobile/cloud
- app-issue-tracker (issue creation)
