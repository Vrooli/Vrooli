# Decisions — AI Gateway

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-07-05 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-07-05 | Name the greenfield replacement `ai-gateway`. | Older AI scenarios blur routing, chat UI, and direct provider/model assumptions. | This scenario becomes the intended durable AI routing and conformance boundary; `ai-model-orchestra-controller` remains a migration reference, not a dependency. | Revisit only if the capability scope changes away from AI inference routing/governance. |
| 2026-07-05 | Resources remain the source of truth for concrete model catalogs. | Ollama and OpenRouter already own provider policy files and gateway commands. | AI Gateway owns profiles, routing, evidence, and conformance; it must call resource command surfaces instead of direct provider HTTP APIs. | Revisit if resource scenarios stop owning provider credentials/policies. |
| 2026-07-05 | AI Gateway should provide a test-genie AI conformance phase. | Scenarios can currently drift into invalid provider env vars, hard-coded model slugs, direct provider calls, and unsafe embedding schemas. | The scenario scope includes scanner rules, maturity levels, findings, exceptions, and migration guidance. | Revisit after first conformance provider implementation and pilot scans. |
| 2026-07-06 | Adaptive routing (cost/quality scoring) is blocked until descriptor-backed route measures are validated, and scoring may never hide the hard policy reasons deterministic routing already enforces. | P2 adaptive routing and fleet enforcement had been marked complete without a measurement substrate, making planned work look shipped. | Routing stays conservative: hard policy filters first, deterministic profile ordering second, and observed measures only as future explainable tie-breakers once AIGW-ROUTE-MEASURES is green. No opaque weighted scoring that hides policy reasons. AIGW-ADAPTIVE-ROUTING/AIGW-FLEET-ENFORCEMENT are `planned`, not `complete`. | Revisit when AIGW-ROUTE-MEASURES passes the measures-health probe and route evidence carries trustworthy analytics fields. |
| 2026-07-06 | Provider failure isolation, capacity-aware local eligibility, and route analytics are the next durable hardening layers, built on resource-owned contracts. | Provider failures were only local to a single execution, local-vs-remote decisions never consulted the capacity broker, and route evidence was not a descriptor-backed measures surface. | Adds persisted provider-health/circuit-breaker state (AIGW-PROVIDER-BREAKER), a `vrooli capacity` broker adapter seam for local route fit (AIGW-CAPACITY-ROUTING), and route measures over durable evidence (AIGW-ROUTE-MEASURES). Capacity uses the broker CLI/role metadata, never direct GPU/RAM probing; resources remain the source of model footprint and role policy. | Revisit if the capacity broker contract changes or resources stop exposing role/footprint metadata. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
