# Deployment P2 Orchestration Summary

## Source
Planning session covering the second deployment cluster from the whiteboard:
- app updates testing
- Mac/Windows desktop validation
- interop with other downloaded apps and/or a local swarm
- Stripe testing for monetized desktop delivery

## Decisions Made
- Deployment P2 is a mix of extension work and genuinely new backlog.
- One canonical desktop update policy should exist by default:
  - check on startup
  - download in the background
  - show a low-friction prompt when an update is ready
  - allow per-app override later
- Downloaded desktop apps should win over same-scenario local swarm instances by default.
- Windows should aim for the same bundled discovery behavior as the other desktop targets unless a platform-specific constraint is discovered.
- Stripe work should happen in two phases:
  - first deepen automated assurance
  - then add anomaly/reconciliation guardrails

## Existing Backlog Adjustments
- `execute/lpbs-desktop-release-contract-hardening`
  - expanded to explicitly include canonical update policy defaults, customer-visible update state, and LPBS guarantees for the default update flow
- `execute/desktop-release-regression-suite`
  - expanded to include real update-flow regression, Mac/Windows launch sanity, emulator-driven smoke coverage, and human-gating evidence handoff
- `execute/desktop-release-regression-suite`
  - now depends on `execute/scenario-to-desktop-canonical-update-policy`

## New Backlog Clusters
- `execute/scenario-to-desktop-canonical-update-policy`
  - implement the default update behavior and override seam
- `desktop-runtime-interop` initiative
  - `research/desktop-runtime-discovery-precedence-contract`
  - `execute/cross-platform-bundled-discovery-parity`
  - `execute/desktop-ecosystem-interop-regression-coverage`
- `desktop-monetization-assurance` initiative
  - `execute/lpbs-stripe-monetization-assurance-suite`
  - `research/lpbs-payment-guardrails-and-reconciliation-plan`

## Dependency Notes
- Update-policy implementation is downstream of LPBS release-contract hardening.
- Desktop release regression is downstream of:
  - LPBS release-contract hardening
  - canonical update policy
  - deployment-manager LPBS orchestration
  - emulator adoption in deployment flows
- Interop regression is downstream of:
  - interop contract definition
  - cross-platform bundled discovery parity
  - emulator adoption in deployment flows
- Payment guardrails research is downstream of Stripe monetization assurance hardening.

## Context For Workshop Agents
- `scenario-to-desktop` already contains substantial updater behavior; the main remaining work is to standardize defaults and prove the behavior end to end.
- `api-core` discovery currently routes through `vrooli scenario port`.
- `scenario-to-desktop` already stages a bundle-local `vrooli` shim for non-Windows targets; Windows parity is part of the new interop work.
- LPBS already has Stripe/download-gating test coverage. The new monetization assurance item is about raising confidence to launch quality, not inventing the entire payment test surface from scratch.

## Unresolved Questions Deferred To Workshop
- Whether Windows needs a distinct implementation approach for bundle-local discovery parity even if the behavior contract matches the other targets.
- What the cleanest future override/config seam should be for discovery precedence after the default shipped behavior is in place.
- Which exact emulator-driven topologies provide the highest-signal interop regression coverage first.
