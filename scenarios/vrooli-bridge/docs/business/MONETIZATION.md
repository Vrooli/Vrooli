# Monetization — Vrooli Bridge

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: Vrooli Bridge is
infrastructure, not a directly-sold product in v1, and this document
says so plainly while sketching the realistic future paid angle.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

Vrooli Bridge is **internal infrastructure / a meta-capability**, not a
standalone consumer product. Its primary customer is the Vrooli system
itself and the operator running it.

- Direct product: not in v1. Bridge is an enabler, not a SKU.
- Internal capability: **primary role.** Bridge supplies the fleet
  control plane (node registry, dial-out presence, allowlisted remote
  execution, privileged provisioning) that the rest of the system lacks
  today. It unlocks the **cross-platform deployment-validation tier**:
  the ability to build a scenario's desktop app and run its test suite
  natively on Ubuntu, macOS, and Windows nodes and aggregate a single
  "production-ready on every OS" verdict.
- SKU/bundle candidate: indirect. Bridge does not ship as its own line;
  it makes the desktop/multi-OS *deployment* capability (deployment-manager
  + scenario-to-desktop) trustworthy enough to charge for.
- Revenue line: none directly attributed to bridge in v1. Future paid
  angle is "deployment assurance / managed cross-OS fleet validation"
  layered on the deployment tier — bridge is the load-bearing primitive
  underneath, not the billed unit.

## Customer / Buyer

- Primary user: the Vrooli operator validating that a scenario is
  production-ready on every target OS before it ships, plus the internal
  release/deployment pipeline (deployment-manager, scenario-to-desktop)
  that needs a cross-OS gate, and agents that need to run a known, safe
  operation on a specific machine in the owner's fleet.
- Buyer (internal): the Vrooli platform itself — bridge is funded as
  capability investment because cross-OS validation is otherwise manual
  and unrepeatable.
- Buyer (future external): a team shipping a cross-platform desktop or
  CLI application that wants an audited "validated on Ubuntu + macOS +
  Windows" gate without hand-carrying builds to three machines. They
  would pay for the *assurance and automation*, not for bridge as such.
- Pain: today, answering "does this scenario actually work on macOS and
  Windows?" means hand-carrying builds to three machines and running
  tests by hand — no fleet, no remote provisioning, no safe remote
  execution. That is slow, error-prone, and impossible to gate a release
  on. Bridge collapses it to one audited command.
- Existing alternatives: self-managed CI matrices (GitHub Actions
  runners, Buildkite, BrowserStack-style farms) solve adjacent problems
  but none run *Vrooli's own* provisioning + scenario CLI natively on the
  owner's trusted machines; they are CI tooling, not a Vrooli fleet
  control plane.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone app | not-applicable (v1) | Bridge is infrastructure; it is not packaged or sold on its own. |
| Bundle component | candidate | Most likely path: bridge is the engine inside a "cross-OS deployment assurance" capability built on deployment-manager + scenario-to-desktop. |
| Add-on | future | A "managed fleet validation" add-on to the deployment tier for teams shipping cross-platform apps. |
| Service/consulting assist | future | Done-for-you cross-OS release validation could lean on bridge as the delivery mechanism. |

Guardrail: **self-hosting / bring-your-own-machines is always free and
core.** An owner registering and validating on their own devices is the
baseline, never gated or metered. Any future monetization attaches to
managed/hosted convenience (e.g. provisioned cloud-runner nodes,
managed fleet operation), never to the fundamental act of validating on
hardware you already own.

## Pricing Hypothesis

- Model: **not directly metered in v1.** Bridge has no standalone price.
  The honest near-term hypothesis is zero direct revenue; bridge's
  return is enabling the deployment tier to make a trustworthy claim.
- Future paid angle (modest, unproven): a "deployment assurance" tier
  priced per validated release or per managed node, and/or metered
  usage of *hosted* ephemeral cloud-runner nodes (the OT-P2 expansion)
  where Vrooli supplies the macOS/Windows capacity the owner lacks. Any
  such pricing must keep self-hosted/owner-owned validation free.
- Comparable products: cross-platform CI minutes and managed runner
  pricing (GitHub Actions runner minutes, Buildkite, cloud macOS runner
  fleets) are the closest reference points for the *hosted-capacity*
  angle only — not for bridge itself.
- Willingness-to-pay evidence: none captured yet. Bridge is unbuilt.
- Cost drivers: local control plane is cheap (SQLite via
  `api-core/storage`, no third-party runtime). Real cost lives in the
  nodes — native build/test compute per OS and provisioning (git fetch
  + `vrooli setup`) — and, in the future hosted angle, in supplying
  macOS/Windows runner capacity.

## Validation Plan

The question this scenario must answer is **operational, not
commercial**: does bridge actually make cross-OS deployment gates
reliable and reduce manual per-machine effort?

- Demand signal needed: deployment-manager and scenario-to-desktop
  successfully consume a bridge cross-OS gate on a real scenario, and
  the per-OS verdict is trustworthy enough to gate a promotion on.
- Effort-reduction signal: a cross-OS validation that previously took
  manual hand-carry to three machines becomes a single audited command
  with durable, re-attachable results.
- Channel: see [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — internal capability
  rollout first, dogfooded on Vrooli's own scenarios.
- Success threshold: bridge is the standing mechanism behind the
  deployment tier's cross-OS gate; only after that is proven does any
  external "deployment assurance" pricing experiment become worth
  scoping.
- Revisit trigger: deployment-manager wires the cross-OS gate (OT-P1-002)
  and it runs green on a real scenario across at least two OSes.

## Current Status

`documentation-first foundation, unbuilt` — the scenario is scaffolded
and fully specified (PRD, requirements, concept docs) but no domains are
implemented yet. Monetization posture is settled: bridge is an internal
enabler/meta-capability in v1 with a plausible future "deployment
assurance" angle layered on the deployment tier, and self-hosted
owner-owned validation stays free forever. Revisit this document when
the cross-OS gate is wired into deployment-manager and a real external
buyer for managed validation is identified.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements (fleet control plane, cross-OS gate)
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — positioning, channels, and launch motion
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — control plane, node-agent, trust tiers
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — deployment-manager / scenario-to-desktop consumers
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — remote-execution and provisioning posture
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
