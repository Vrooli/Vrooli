# Domains — Audio Tools

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| stt | Speech-to-text (batch + streaming) with three-tier provider routing. | Streaming / chain-routed | Stream config, wakeword phrases, enrolled speaker embeddings, transcription usage rows. | API, CLI, UI, WS, Embed | OT-P0-001, OT-P0-007, OT-P0-008, OT-P1-014 | `api/internal/stt/`, `api/internal/ai/sttchain/`, `api/handlers/stt/`, `cli/domains/stt/`, `ui/src/features/diagnostics/`, `embed/`, `packages/proto/schemas/audio-tools/v1/stt/` |
| tts | Text-to-speech synthesis (batch + streaming) with on-disk cache. | Synthesis / chain-routed | Voice catalog snapshot, TTS config, content-addressable audio cache, playback events. | API, CLI, UI, Embed | OT-P0-002, OT-P0-006 | `api/internal/tts/`, `api/internal/ai/ttschain/`, `api/handlers/tts/`, `cli/domains/tts/`, `ui/src/features/diagnostics/`, `embed/`, `packages/proto/schemas/audio-tools/v1/tts/` |
| summarize | Text summarization with normalization preprocessing. | Inference / chain-routed | Per-call usage rows. | API, CLI, UI | OT-P0-003, OT-P0-006 | `api/internal/summarize/`, `api/internal/ai/summarizechain/`, `api/handlers/summarize/`, `cli/domains/summarize/`, `ui/src/features/diagnostics/`, `packages/proto/schemas/audio-tools/v1/summarize/` |
| audio | Audio file processing (transcode/trim/merge/split/fade/volume/normalize/metadata). | Pipeline / shellout | None (operates on multipart payloads). | API, CLI, UI | OT-P0-004 | `api/internal/audio/`, `api/handlers/audio/`, `cli/domains/audio/`, `ui/src/features/diagnostics/`, `packages/proto/schemas/audio-tools/v1/audio/` |
| session | Voice-session pub/sub fan-out for live STT + TTS streams. | Pub/sub / streaming | Ephemeral in-memory session state. | API, WS | OT-P0-007, OT-P0-008 | `api/internal/session/`, `api/handlers/stt/stream_ws.go`, `api/handlers/session/`, `packages/proto/schemas/audio-tools/v1/session/` |
| settings | Operator configuration: provider defaults, per-capability precedence, BYOK creds (AES-GCM at rest). | CRUD / config | Provider config, BYOK secrets, voice overrides. | API, CLI, UI | OT-P0-009 | `api/internal/store/`, `api/internal/byokstore/`, `api/handlers/settings/`, `cli/domains/settings/`, `ui/src/features/configuration/`, `packages/proto/schemas/audio-tools/v1/settings/` |
| usage | Per-operation usage + cost ledger and rollup queries for the dashboard. | Reporting / ledger | Usage rows (provider, op, ms, credits). | API, CLI, UI | OT-P0-011 | `api/internal/store/usage.go`, `api/internal/usagereport/`, `api/handlers/usage/`, `cli/domains/usage/`, `ui/src/features/usage/`, `packages/proto/schemas/audio-tools/v1/usage/` |
| corpus | Speech-eval clip store: operator-recorded audio + corrected ground-truth transcripts for the eval harness. | CRUD / blob+metadata | Clip metadata (`corpus` SQLite domain) + audio bytes (blob store, git-ignored). | API, CLI, UI | (eval harness) | `api/internal/corpus/`, `api/handlers/corpus/`, `cli/domains/corpus/`, `ui/src/features/dictation-studio/`, `packages/proto/schemas/audio-tools/v1/corpus/` |
| eval | STT strategy comparison harness: replays the corpus through batch/vad/overlap and reports WER, compute cost, safety, length curves, and backend-owned duration-scaling classifications. | Measurement / replay | None (reads the corpus domain; stateless). | API, CLI, UI report renderer | (eval harness) | `api/internal/eval/`, `api/handlers/eval/`, `packages/proto/schemas/audio-tools/v1/eval/`, shared report rendering under `cli/domains/experiment/` and `ui/src/features/dictation-studio/` |
| experiment | Persisted async STT experiment lifecycle: reproducible recipes, server-owned execution, stored reports, deterministic long-form, augmentation, speaker-dimension inputs, and comparisons. | Long-running operation / lab | Experiment metadata, lifecycle state, reproducible recipe realization, per-run metric cells, augmentation/speaker condition notes, and report blob references. | API, CLI, UI | STT experiment lab | `api/internal/experiment/`, `api/handlers/experiment/`, `cli/domains/experiment/`, `ui/src/features/dictation-studio/`, `packages/proto/schemas/audio-tools/v1/experiment/` |
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/overview/`, `packages/proto/schemas/audio-tools/v1/health/` |

## Domain Details

### stt

- Purpose: convert spoken audio to text via BYOK → Vrooli/LPBS → Local chain; support batch unary + bidi streaming with segmenter-strategy decoupling (VAD, overlap-agree, passthrough).
- Primary archetype: streaming / chain-routed.
- Secondary traits: WS browser transport, wakeword admin, speaker enrollment + verification.
- Owns: segmenter, strategy selector, streaming strategies, STT chain providers, STT admin (stream config / wakeword / speaker) handlers.
- Does not own: BYOK credential persistence (settings), usage ledger (usage), generic audio file ops (audio).
- API: `api/handlers/stt/` (Connect-RPC + WS `/api/v1/voice/stream`).
- CLI: `cli/domains/stt/` (verb `audio-tools stt …`; renamed from `voice` on 2026-05-17 — see [`../internal/DECISIONS.md`](../internal/DECISIONS.md)).
- UI: `ui/src/features/diagnostics/` (try-it row), `ui/src/features/configuration/` (admin forms).
- Storage: `stt_stream_config`, `wakeword_phrases`, `speaker_embeddings` tables; see [`DATA.md`](DATA.md).
- Requirements: OT-P0-001 (local), OT-P0-007 (session fan-out), OT-P0-008 (barge-in), OT-P1-014 (streaming).
- Tests: chain unit, selector table-driven, segmenter parity (HTTP/2 httptest), strategy tests; per-table store tests; pipeline test coverage.
- Related docs: [`../internal/SEAMS.md`](../internal/SEAMS.md) (Segmenter, StrategySelector, StreamingStrategy seams).

### tts

- Purpose: synthesize audio from text via the same three-tier chain; cache results by content hash; emit playback events for analytics.
- Primary archetype: synthesis / chain-routed.
- Owns: voice catalog snapshot + verification gate, TTS config doc, audio cache, normalizer (text → speech-ready text), kokoro local synth client, summarize-adjacent voice surface.
- Does not own: text summarization (summarize), provider credentials (settings).
- API: `api/handlers/tts/`.
- CLI: `cli/domains/tts/`.
- UI: `ui/src/features/diagnostics/` (synthesize try-it), `ui/src/features/voices/` (browse), `ui/src/features/configuration/`.
- Storage: `tts_config_doc`, `tts_cache/`, `playback_events`.
- Requirements: OT-P0-002, OT-P0-006.
- Tests: cache, chunker, config, service, summarization-service (shared upstream), summarizer, normalizer, voice catalog — all per-file tests.

### summarize

- Purpose: convert long text to short text via the chain, with a normalization preprocessing step.
- Primary archetype: inference / chain-routed.
- Owns: summarization service, summarize config, normalizer (text→text), and the `internal/ai/summarizechain/` providers.
- Does not own: TTS voice synthesis (tts), settings persistence (settings).
- API: `api/handlers/summarize/`.
- CLI: `cli/domains/summarize/`.
- UI: `ui/src/features/diagnostics/`.
- Storage: usage rows only.
- Requirements: OT-P0-003, OT-P0-006.
- Tests: summarization service, summarizer, summarize config, normalizer — see `internal/summarize/`.

### audio

- Purpose: best-effort ffmpeg/ffprobe-backed audio transformations exposed as a multipart endpoint.
- Primary archetype: pipeline / shellout.
- Owns: `Ops` shellout layer, ffprobe parser, multipart route handler.
- Does not own: transcription (stt), synthesis (tts).
- API: `api/handlers/audio/` (multipart route is the documented REST exception).
- CLI: `cli/domains/audio/`.
- UI: `ui/src/features/diagnostics/`.
- Storage: none.
- Requirements: OT-P0-004.
- Tests: per-op tests with a `Runner` seam substituting `exec.Cmd`.

### session

- Purpose: own the per-voice-session lifecycle and fan out STT/TTS events to multiple observers (UI + diagnostics + downstream subscribers).
- Primary archetype: pub/sub / streaming.
- Owns: `session.Session`, browser-WS transport bridge.
- API: `api/handlers/session/` (Connect-RPC + WS `/api/v1/voice/stream`).
- UI: consumed by diagnostics and any embed component subscribing.
- Storage: ephemeral in-memory only.
- Requirements: OT-P0-007, OT-P0-008.

### settings

- Purpose: operator-facing configuration: provider defaults, per-capability tier order (BYOK → Vrooli → Local), BYOK credentials encrypted at rest, voice overrides.
- Primary archetype: CRUD / config.
- Owns: provider config doc, BYOK store + AES-GCM encryptor + fingerprint, voice overrides, settings handlers, chain `Reconfigure` plumbing.
- API: `api/handlers/settings/`.
- CLI: `cli/domains/settings/`.
- UI: `ui/src/features/configuration/`.
- Storage: `provider_config_doc`, `byok_secrets`, `voice_overrides`.
- Requirements: OT-P0-009.
- Tests: per-table store tests; `byokstore.Encryptor` round-trip + tamper tests; handler tests.

### usage

- Purpose: record every chain-routed op (provider, ms, credits) and serve rollup queries for the dashboard.
- Primary archetype: reporting / ledger.
- Owns: `usagereport.Recorder` interface + async recorder, `internal/store/usage.go`, usage handlers, dashboard.
- API: `api/handlers/usage/`.
- CLI: `cli/domains/usage/`.
- UI: `ui/src/features/usage/`.
- Storage: `usage` table.
- Requirements: OT-P0-011.

### corpus

- Purpose: store operator-recorded audio clips with corrected ground-truth transcripts as the substrate the eval harness replays against.
- Primary archetype: CRUD over a blob+metadata split.
- Owns: `CorpusService` (CreateClip/ListClips/GetClip/GetClipAudio/DeleteClip), the `corpus_clips` SQLite table, and the audio blob store.
- API: `api/handlers/corpus/`. CLI: `cli/domains/corpus/`. UI: `ui/src/features/dictation-studio/`.
- Storage: `corpus_clips` table (metadata only) + audio bytes in the blob store under the git-ignored runtime data dir (variant-aware namespace). Audio never enters git or the DB.
- Does not own: the strategies it evaluates (stt) or the metrics/replay (eval).

### eval

- Purpose: provide the internal measurement harness and shared report contract for WER, compute cost (Whisper-calls / audio-seconds / RTF), safety gates, length curves, duration-scaling points/classifications, and finalization latency (p50/p95).
- Primary archetype: internal deterministic replay/report assembly.
- Owns: the offline harness (`internal/eval`: WER/normalizer/metered-provider/replay/report/scaling) and shared proto report messages consumed by `ExperimentService`. Scaling model fitting and strategy recommendation policy are backend-owned here; UI and CLI render the typed output and do not fit models or re-rank strategies.
- API: no standalone public API. The former blocking eval RPC and eval CLI were retired on 2026-07-01; persisted experiments are the only agent/operator evaluation surface. The Dictation Studio UI retains the eval report renderer only as a shared report component.
- Storage: none — experiment runs materialize corpus inputs and persist reports in the experiment domain.
- Reference: [`../reference/eval-harness.md`](../reference/eval-harness.md). Requires a live Whisper backend for production experiments (build-tagged integration test; deterministic fake-provider tests in the default suite).

### experiment

- Purpose: provide the agent-facing STT experiment lab surface: persisted recipes, async execution, wait/watch lifecycle, stored reports, duration sweeps, and N-way comparison.
- Primary archetype: long-running operation / lab.
- Owns: `ExperimentService`, the `experiments` and `experiment_runs` SQLite tables, report artifact refs, deterministic long-form and augmentation recipe realization, speaker extraction/verification condition binding, safety gate configuration, and the server-lifetime async runner.
- API: `api/handlers/experiment/`. CLI: `cli/domains/experiment/`. UI: `ui/src/features/dictation-studio/ExperimentLabView.tsx` drives builder, history, report, and compare flows through `ExperimentService`.
- Storage: metadata and metric cells in SQLite; large reports in the variant-aware `experiment-blobs` store. Audio/report bytes do not enter git.
- Does not own: the STT strategies, live STT/speaker config mutation, or metric computation/ranking policy; it orchestrates the eval harness with per-run config snapshots and persists its safety/curve/scaling results.

### health

- Purpose: expose API/database readiness for boot checks and the overview card.
- Primary archetype: reporting / query.
- API: `api/handlers/health/`.
- CLI: built-in `status` command via cli-core.
- UI: `ui/src/features/overview/`.
- Storage: none.
- Requirements: starter scaffold health only.

## Naming Pitfalls

- **`ui/src/features/voices/` is part of `tts`, not `stt`.** It is the
  TTS voice-catalog browser. The CLI verb for STT is `audio-tools stt …`
  (renamed from `voice` on 2026-05-17 — see
  [`../internal/DECISIONS.md`](../internal/DECISIONS.md)). The
  `voice` flag on `audio-tools tts synthesize` and the proto field
  `Voice` likewise refer to the TTS voice catalog.
- **`cli/domains/stt` ≠ `internal/stt`** but they cover the same domain.
  The CLI is generated against the proto Connect client; primitives
  (segmenter/strategy/selector) live in `api/internal/stt/`. Provider
  routing lives one level up in `api/internal/ai/sttchain/`.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, WS, or embed layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Tier | One of BYOK / Vrooli (LPBS) / Local in the per-capability provider chain. | `internal/ai/chains/`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| adoption | Out-of-process integration health for scenarios consuming `@audio-tools/embed`. | OT-P1-013 ramp. |
| twilio-transport | Twilio media-stream WS bridge. | OT-P2-001 / `execute/audio-tools-twilio-media-stream-transport`. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/modulekit/` — shared module descriptor type defs.
- `api/internal/modules/` — thin static registry consumed by `main.go` and `cmd/gen-endpoints`.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/httpx/`, `api/internal/httpc/` — HTTP request/response helpers.
- `api/internal/clock/` — clock seam.
- `api/internal/middleware/` — request-pipeline middleware.
- `api/internal/testutil/` — cross-domain test harnesses.
- `api/internal/ai/chains/` — generic per-capability chain primitives shared by stt/tts/summarize.
- `api/internal/capabilities/` — capability-flag derivation.
- `api/integrations/lpbs/` — LPBS-tier client adapters + usage reporter (cross-scenario integration, not a product domain).
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
