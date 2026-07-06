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

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
