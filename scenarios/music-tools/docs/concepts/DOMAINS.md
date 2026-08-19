# Domains — Music Tools

This document is the canonical map of product capabilities, bounded contexts, and
ownership for this scenario. Keep it current whenever a domain is added, renamed,
split, merged, or removed.

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md). Workflow
details belong in [`FLOWS.md`](FLOWS.md). Storage details belong in
[`DATA.md`](DATA.md).

music-tools is a local-first capability-primitive scenario in Vrooli's `*-tools`
family and the direct sibling of `image-tools`. It groups its surface into the
bounded contexts below. The user-facing operation families — `ops` (deterministic),
`composition` (generate), `transformation` (edit existing audio), and `analysis`
(audio to data) — draw their names from one vocabulary SSOT so no domain
re-declares the vocabulary. The remaining domains make those families reliable,
hardware-aware, and composable. **Every operation runs fully headless from the CLI;
the UI is an enhancer, never a gate.**

## Domain inventory

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

## Concepts that are deliberately NOT domains

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
