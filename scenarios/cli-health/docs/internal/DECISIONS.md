# Decisions — CLI Health

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-05-19 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-06-03 | Keep cli-health's local `SearchHit` / `StatusReport` / `SearchMode` instead of adopting `packages/ai-go/search`'s generic read-path types (`pkg.Service`, `pkg.SearchQuery`, `pkg.SearchHit`, `pkg.SearchMode`). | The aisearch-adoption-hardening plan (WS5) flagged read-path type duplication between the engine and its first consumer. | cli-health's `SearchHit` carries command-domain fields the generic type lacks (`Origin`, `Group`, `Binding`, `ScorePercent`) — a 1:1 merge would be wrong. `SearchMode` stays local because it is the **proto-facing** enum (`MODE_AI`/`MODE_TEXT`/`MODE_AUTO`, values `"ai"`/`"text"`/`"auto"`) that the Connect handler maps to; the generic `pkg.SearchMode` (`hybrid`/`dense`) has no `ai` member and different semantics. The local types are the **command-domain projection** of the generic read-path; the generic `pkg.Service`/`SearchQuery` read-path is intentionally exercised first by the KO docs adopter, so the contract is deferred-not-dead. | Revisit when a second 1:1 (non-doc) consumer adopts the generic read-path, or when the federation `SearchHit` shape (search-hub Appendix A.5) changes — keep that descriptor in lockstep. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
