# Go-To-Market — Plan Manager

This document records how plan-manager reaches its users. Because the
scenario is an internal capability, the go-to-market motion is
internal-first; external GTM is deferred until there is evidence of
external demand.

## Purpose Of This Document

Use this document to answer:

- Who is plan-manager positioned for, and how is it described to them?
- Through which channels do users adopt it?
- What is the launch motion, and what message accompanies it?
- What experiments would validate (or kill) any external GTM ambition?

## Audience And Positioning

- Audience: AI coding agents (especially small/local models) that author
  and execute implementation plans, and operators who view, manage, and
  triage those plans and handoffs.
- Positioning: "the plan-logic SSOT for Vrooli" — a guided wizard runtime
  that makes planning cheap enough for local models. It is positioned as
  shared platform plumbing, not as a competitor to external planning
  tools.

## Channels

- In-ecosystem discovery: the `plan-manager` CLI, the scenario UI, and
  Connect-RPC consumers within other scenarios.
- Skill discovery: server-side context discovery over prompt-manager/search-hub probes
  plan-manager to agents at the moment they need to plan.
- External channels (marketplace, website, docs site): deferred — not
  applicable until an external SKU exists.

## Launch Motion

- Internal launch: ship behind the standard scenario lifecycle
  (`make start` / `vrooli scenario start plan-manager`), then drive
  adoption by routing agent planning work to it via prompt-manager and by
  making `plan-manager` the canonical plan lifecycle path.
- There is no external launch event planned. A broader launch is deferred
  until plan-manager has proven local-model cost savings on real plans.

## Messaging

- Core message: "Plan once, in the SSOT; execute it on a cheap local
  model." Authoring and executing plans should not require a large cloud
  model.
- Secondary message: plans are durable in SQLite under `~/.vrooli` and rendered
  markdown mirrors are repairable from the structured record.
- Honesty guardrail: candidate findings are surfaced as unvalidated, never
  as confirmed facts — messaging must not overclaim correctness.

## Validation Experiments

- Adoption experiment: measure how often agents choose plan-manager when
  prompt-manager offers the plan skill, versus hand-rolling a plan.
- Cost experiment: compare per-plan velocity (time/tokens/iterations) for
  local-model plan work via plan-manager against a large-cloud-model
  baseline, using the meta-optimization velocity sink.
- Operator experiment: confirm operators can triage and read plans during
  a server outage through the CLI thin-client path.
- External-demand experiment: deferred until internal adoption and the
  cost experiment both clear.

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — revenue stance and pricing
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals used by the validation experiments
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system architecture
