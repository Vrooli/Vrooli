# Decisions — Audio Tools

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
| 2026-05-16 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-05-16 | Decouple streaming **strategy** from streaming **provider** in the STT pipeline. | The 5 PRD-committed BYOK starter adapters plus Local Whisper and Vrooli/LPBS span three distinct techniques (VAD-segment, overlap-and-agree, native passthrough). Fusing technique with provider — as the current `voice.Service` does for Local Whisper — would duplicate VAD logic across every batch-only adapter and leave no clean home for the local-only quality tier. The selector becomes one explicit decision boundary instead of conditionals scattered across providers. | Providers declare `ProviderTraits` capability bits; `StrategySelector` enforces a compatibility matrix and returns typed errors on forbidden pairs. Both transports (browser WS, Connect bidi) reach STT through the same `Segmenter` + `StrategySelector` pipeline, eliminating the "two parallel pipelines silently drift" risk. Full design in [`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md). | Revisit if a future provider class cannot fit the matrix, or if the strategy/provider axes prove to collapse to one in practice across every supported vendor. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
