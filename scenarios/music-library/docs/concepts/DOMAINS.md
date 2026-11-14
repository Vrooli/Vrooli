# Domains — Music Library

Canonical map of product capabilities, bounded contexts, and ownership.

## Purpose Of This Document

Use this document to answer:

- What capabilities does this scenario expose to the listener?
- Which domain owns each concept, table, endpoint, and surface?
- Where is the boundary that keeps recommendation honest?

System architecture is in [`ARCHITECTURE.md`](ARCHITECTURE.md); workflows in
[`FLOWS.md`](FLOWS.md); storage in [`DATA.md`](DATA.md).

music-library is a **Lifestyle bundle** scenario — the first one — and a consumer
of the `music-tools` capability primitive. It owns listening, the preference model,
and everything that makes both legible. It owns **no** audio understanding: every
embedding, segment boundary, tempo, key, tag, and generated waveform comes from
`music-tools`.

## Domain Inventory

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

## Domain Details

### The listening path

- **`library`** holds source roots and a content-derived track identity, so a
  listener can reorganise their files without losing history. Source files are
  opened read-only; no path in this domain writes to a source root.
- **`playback`** owns sessions, queue, transport, and the gapless boundary. Its
  correctness is judged by ear, which makes it the most demanding domain to test.
- **`decomposition`** orchestrates attribute extraction through `music-tools` and
  owns batch progress and resumability. It caches attributes locally so playback and
  browsing survive `music-tools` being down.

### The taste path

- **`preference`** is the model itself: expected preference and calibrated
  uncertainty over the embedding space, scoped per listening context, plus any
  constraints the listener has authored by hand.
- **`elicitation`** owns pairwise comparison — selecting the most informative pair,
  recording judgments, and supporting undo. Comparison is used because it is far
  more label-efficient than rating for a single listener.
- **`signals`** converts listening behaviour into evidence, and owns the corrections
  that make it admissible: position-bias correction and reliability gating before
  any section-level attribution reaches the model.
- **`ranking`** applies the model: scoring, calibration, exploration policy, and the
  explanation. It persists nothing, which is part of what makes its blindness
  enforceable.

### Generation and disclosure

- **`generation`** conditions candidate creation on the profile and owns retention
  and eviction. It exists as much for candidate-pool growth as for novelty.
- **`provenance`** records complete generation lineage and drives disclosure.
  Generated audio is always disclosed as generated.
- **`offers`** decorates an already-final ranking. It is the only domain defined
  primarily by what it may **not** do.

## Shared Concepts

- **Track identity** — content-derived, shared by every domain. It is what lets
  history survive a file move.
- **Context** — the listening situation a profile is scoped to. Owned by
  `preference`, referenced by `ranking` and `generation`.
- **Uncertainty** — produced by `preference`, consumed by `elicitation` for pair
  selection, by `ranking` for exploration, and by the surfaces for explanation.
- **Attributes** — sourced from `music-tools`, cached by `decomposition`, read by
  nearly everything. Never computed locally.

## Deferred Domains

| Deferred | Why | Revisit |
|---|---|---|
| `sharing` | Nothing here publishes or distributes music. | Only with a deliberate product decision. |
| `multi-listener` | The first deployment is single-listener. | Before any shared or bundled deployment — it is a disclosure risk, not a feature gap. |
| `sync` | Offline synchronisation is real work but has no ordered-state model yet. | When a delivery surface is chosen. |

## Non-Domains

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

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — the preference model and the loop
- [`FLOWS.md`](FLOWS.md) — stateful workflows by domain
- [`DATA.md`](DATA.md) — what each domain persists
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — blindness as an integrity control
