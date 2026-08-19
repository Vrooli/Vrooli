# Domains — Music Library

Canonical map of product capabilities, bounded contexts, and ownership.
System architecture is in [`ARCHITECTURE.md`](ARCHITECTURE.md); workflows in
[`FLOWS.md`](FLOWS.md); storage in [`DATA.md`](DATA.md).

music-library is a **Lifestyle bundle** scenario — the first one — and a consumer
of the `music-tools` capability primitive. It owns listening, the preference model,
and everything that makes both legible. It owns **no** audio understanding: every
embedding, segment boundary, tempo, key, tag, and generated waveform comes from
`music-tools`.

## Domain inventory

| Domain | Purpose | Archetype | Owns data | Surfaces | Requirements |
|---|---|---|---|---|---|
| health | Runtime readiness and dependency reachability. | Reporting | No | API, UI | Starter scaffold |
| library | Source roots, incremental scan, content-derived track identity, metadata. Source files are read-only. | Registry | Track index, scan state | API, CLI, UI | MLIB-P0-001 … MLIB-P0-002 |
| playback | Sessions, queue, transport, on-demand transcoding, gapless boundaries. | Session | Session and queue state | API, UI | MLIB-P0-003, MLIB-P0-004 |
| decomposition | Orchestrates attribute extraction through music-tools; owns batch progress and resumability. | Orchestration | Attribute records, batch state | API, CLI, UI | MLIB-P0-005 … MLIB-P0-008 |
| preference | The taste model: expected preference and uncertainty over the embedding space, plus listener constraints and per-context profiles. | Model | Model state, constraints | API, CLI, UI | MLIB-P0-009 … MLIB-P1-002 |
| elicitation | Pairwise comparison, informative pair selection, rating updates, component-scoped comparison. | Interaction | Comparison history, ratings | API, UI | MLIB-P0-012 … MLIB-P1-003 |
| signals | Completion, replay, skip; position-bias correction; reliability-gated section attribution. | Ingest | Interaction events | API | MLIB-P1-007 |
| ranking | Scoring, calibration, exploration policy, and the derived explanation. **Structurally blind to monetisation.** | Policy | No | API, UI | MLIB-P0-015 … MLIB-P1-006 |
| generation | Background queue conditioned on the profile, directed requests, retention and eviction. | Lifecycle | Candidate state | API, CLI, UI | MLIB-P1-008 … MLIB-P1-011 |
| provenance | Generated-content disclosure and complete generation lineage. | Reporting | Lineage records | API, UI | MLIB-P0-018 … MLIB-P0-019 |
| offers | Post-processing decoration of an already-final ranking. Cannot reorder or filter. | Post-process | Offer state | API, UI | MLIB-P0-020 |

## Concepts that are deliberately NOT domains

- **Audio model execution.** All of it belongs to `music-tools`. This scenario has
  no model registry, no GPU claim, and no inference path.
- **Publishing or distribution.** Nothing here uploads music anywhere.
- **Speech.** `audio-tools` owns it.

## The blindness boundary

`ranking` and `generation` must not be able to observe what is sold, promoted, or
commission-bearing. `offers` sits strictly downstream of a final ranking and may
only decorate it. This follows the lifestyle-bundle recommendation-blindness rule
in monetisation strategy, and it is enforced by package boundary rather than by
convention — the authority the listener is paying for is the reason the bundle
exists at all.
