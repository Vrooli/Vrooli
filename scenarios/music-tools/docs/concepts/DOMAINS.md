# Domains — Music Tools

This document is the canonical map of product capabilities, bounded contexts, and
ownership for this scenario. Keep it current whenever a domain is added, renamed,
split, merged, or removed.

## Purpose Of This Document

Use this document to answer:

- What capabilities does this scenario expose?
- Which domain owns each concept, table, endpoint, CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md). Workflow
details belong in [`FLOWS.md`](FLOWS.md). Storage details belong in
[`DATA.md`](DATA.md).

music-tools is a local-first capability-primitive scenario in Vrooli's `*-tools`
family and the direct sibling of `image-tools`. The user-facing operation families —
`ops` (deterministic), `composition` (generate), `transformation` (edit existing
audio), and `analysis` (audio to data) — draw their names from one vocabulary SSOT
so no domain re-declares the vocabulary. The remaining domains make those families
reliable, hardware-aware, and composable. **Every operation runs fully headless from
the CLI; the UI is an enhancer, never a gate.**

## Domain Inventory

| Domain | Purpose | Archetype | Owns data | Surfaces | Requirements |
|---|---|---|---|---|---|
| health | Runtime readiness and dependency reachability. | Reporting | No | API, UI | Starter scaffold |
| ops | Deterministic, zero-download, zero-GPU audio editing: trim, fade, concat, convert, silence trim, loudness measurement. | Transform | No | API, CLI, UI | MUS-P0-002 |
| composition | Caption and lyrics to song; instrumental; sound effects. | AI operation | No | API, CLI, UI | MUS-P0-012, MUS-P0-013 |
| transformation | Editing existing audio: stem separation, cover, section repaint, vocal-to-accompaniment, reference mastering. | AI operation | No | API, CLI, UI | MUS-P1-003, MUS-P1-004 |
| analysis | Audio to data: dual embeddings, structure and beats, tempo and key, loudness, tags, audio-to-MIDI, lyric transcription. | Extract | No | API, CLI, UI | MUS-P0-009 … MUS-P1-002 |
| models | Declarative registry, hardware-aware selection, checksum-verified install, licence lane. | Registry | Registry and install state | API, CLI, UI | MUS-P0-003 … MUS-P0-005, MUS-P1-005 |
| capacity | Capacity-broker claims, degradation rungs, exclusive GPU leases. | Policy | Claim state | API (seam), CLI | MUS-P0-006 … MUS-P0-008 |
| backends | Per-operation provider abstraction and fallback ladder across the three runtimes. | Abstraction | No | API (seam) | MUS-P0-001 |
| jobs | Durable server-owned async jobs with progress. | Lifecycle | Job records | API, CLI, UI | MUS-P0-014 |
| storage | BlobStore integration, derived-artifact budget, LRU eviction. | Infrastructure | Blob refs, budget state | API (seam), CLI | MUS-P0-015, MUS-P0-016 |
| styles | Data-defined styles compiling to caption plus parameters. | CRUD + compile | Style definitions | API, CLI, UI | OT-P1-004 |
| measures | Operation latency, queue wait, degradation frequency, fallback usage. | Telemetry | Measure samples | API, CLI | OT-P0-005 |

## Domain Details

### The four operation families

`ops`, `composition`, `transformation`, and `analysis` are peers, distinguished by
cost and determinism rather than by subject:

- **`ops`** is exact, CPU-only, and instant — trimming, conversion, loudness
  measurement. It never downloads a model and never claims the GPU, which makes it
  the only family guaranteed available.
- **`composition`** creates audio that did not exist. Exclusive GPU lease.
- **`transformation`** changes audio that already exists — the iterative family, and
  the reason this is a toolbox rather than a generator. A near-miss is edited, not
  regenerated.
- **`analysis`** turns audio into data and is the family every consumer depends on
  most. It is layered so a partial result is still useful.

### The infrastructure domains

- **`models`** is the registry: hardware gates, disk cost, checksums, licence, and
  lane. It is the enforcement point for the lane split — resolution refuses a
  restricted model in a permissive build, so no call site can opt around it.
- **`capacity`** mirrors the control-plane broker. It owns claim, degradation rung,
  and release. It is never the authority on host state, only the claimant.
- **`backends`** abstracts which of the three runtimes serves an operation, and the
  fallback ladder between them.
- **`jobs`** makes long operations server-owned so they survive client disconnect.
- **`storage`** owns the derived-artifact budget and LRU eviction, and the invariant
  that only regenerable artifacts are evictable.
- **`styles`** compiles a named style to caption plus parameters — the reuse surface
  for a house sound.
- **`measures`** records latency, queue wait, and how often operations degraded.
  Degradation frequency is the honest health signal for a contended card.

## Shared Concepts

These cross domains and are owned by none of them alone:

- **Operation vocabulary** — one SSOT; domains consume names, never re-declare them.
- **Profile rung** — the applied model variant, precision, and batch size. Produced
  by `capacity`, recorded by `jobs`, and attached to every artifact.
- **Licence lane** — declared in `models`, enforced at resolution, recorded in
  provenance.
- **Track identity** — supplied by the caller. This scenario derives no identity of
  its own because it does not own anyone's library.

## Deferred Domains

| Deferred | Why | Revisit |
|---|---|---|
| `training` | Adapter training needs more VRAM than the reference card has. | On hardware with the headroom — `OT-P2-001`, `OT-P2-002`. |
| `notation` as its own domain | Audio-to-MIDI is currently one operation inside `analysis`. | If notation grows editing or export beyond a single conversion. |
| `mastering` as its own domain | Reference mastering is one deterministic operation. | If delivery targets multiply into a real policy surface. |

## Non-Domains

- **Taste, ranking, and recommendation.** Owned entirely by `music-library`. This
  scenario computes attributes; it never decides what is good.
- **Playback and library management.** Also `music-library`.
- **Speech.** `audio-tools` owns speech-to-text, text-to-speech, and dictation.
  Lyric transcription lives here because it is a music-specific model operating on
  sung vocals, for which general speech recognition performs poorly — but no
  general speech capability belongs in this scenario.
- **Host GPU remediation.** The control plane owns detection and remediation of
  host state. This scenario claims, degrades, and releases through the capacity
  broker; it never manages the GPU privately.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system structure and boundaries
- [`FLOWS.md`](FLOWS.md) — stateful workflows by domain
- [`DATA.md`](DATA.md) — what each domain persists
- [`../reference/model-registry.md`](../reference/model-registry.md) — the registry's contents
