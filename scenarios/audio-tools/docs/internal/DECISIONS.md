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
| 2026-05-16 | `handlers/health`: HTTP-only domain — `handler.go` is the canonical HTTP handler (no Connect surface); `module.go` + `endpoints.go` follow the rest of the convention. | Phase 3 handler-shape unification audit. Health uses `api-core/health` for the standardized envelope and registers a plain `http.HandlerFunc`; there is no Connect service definition, so no `connect_handler.go` exists. | Readers should not expect a `connect_handler.go` under `handlers/health/`. Module wiring continues to import `NewHandler` from `handler.go`. | Revisit if a `HealthService` proto ever lands and we mount it via Connect. |
| 2026-05-17 | Post-extraction cleanup. Rename CLI `voice` → `stt`; fold standalone `diagnose` CLI into `settings providers`; keep `internal/ai/*chain/` as canonical provider orchestrators with `internal/{stt,tts,summarize}/` as primitives; keep `internal/text/normalizer/` shared between TTS + summarize; declare audio domain scope as the ffmpeg-backed ops already wired in `api/internal/audio/` with anything else PROBLEMS-deferred. | 9-phase extraction from web-console left naming drift (`voice` CLI for STT proto), an undefined `diagnose` domain, and unclear ownership between `internal/ai/*chain` (provider routing) and `internal/{stt,tts,summarize}` (primitives). Inspection shows the chains layer is consumed by every handler and bootstrap; primitive layers host segmenter/strategy/cache/normalizer with no overlap. Summarize has two files: `summarizer.go` is the Ollama primitive, `summarization_service.go` is the orchestrator — complementary, not duplicate. Normalizer is imported by `handlers/tts/connect_handler.go` and `internal/summarize/summarization_service.go`, so it is genuinely shared. `cli/domains/diagnose` exposes a single `providers` subcommand built from `SettingsService.GetProviderConfig` + `TTSService.GetStatus`; folds cleanly under `settings`. | CLI surface changes: `audio-tools voice …` → `audio-tools stt …`; `audio-tools diagnose providers` → `audio-tools settings providers`. No proto changes. `internal/text/normalizer/doc.go` lists its two consumers. | Revisit if a third consumer of normalizer appears outside `tts`/`summarize` (consider promoting), or if `audio` domain gains a CLI/UI surface for additional ffmpeg ops (re-scope). |
| 2026-05-16 | Decouple streaming **strategy** from streaming **provider** in the STT pipeline. | The 5 PRD-committed BYOK starter adapters plus Local Whisper and Vrooli/LPBS span three distinct techniques (VAD-segment, overlap-and-agree, native passthrough). Fusing technique with provider — as the current `voice.Service` does for Local Whisper — would duplicate VAD logic across every batch-only adapter and leave no clean home for the local-only quality tier. The selector becomes one explicit decision boundary instead of conditionals scattered across providers. | Providers declare `ProviderTraits` capability bits; `StrategySelector` enforces a compatibility matrix and returns typed errors on forbidden pairs. Both transports (browser WS, Connect bidi) reach STT through the same `Segmenter` + `StrategySelector` pipeline, eliminating the "two parallel pipelines silently drift" risk. Full design in [`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md). | Revisit if a future provider class cannot fit the matrix, or if the strategy/provider axes prove to collapse to one in practice across every supported vendor. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
