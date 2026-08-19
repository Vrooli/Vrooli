# Data — Music Library

Storage ownership, schemas, retention, and privacy. Domain ownership is in
[`DOMAINS.md`](DOMAINS.md); system structure is in [`ARCHITECTURE.md`](ARCHITECTURE.md).

## Purpose Of This Document

Use this document to answer:

- What does this scenario persist, and which domain owns it?
- What happens to a listener's own files?
- What is retained about listening behaviour, and how is it deleted?
- Where does the taste model actually live?

## Storage Overview

This scenario owns the listener's data — the library index, the preference model,
and every interaction that shaped it. It owns **no audio understanding**: every
embedding, boundary, tempo, key, and tag is fetched from `music-tools` and cached
here against a track identity.

| Store | Backend | Holds |
|---|---|---|
| Scenario database | SQLite by default, Postgres optional | Track index, attributes, preference model state, comparisons, interaction events, sessions, lineage |
| Vector index | `qdrant` | Track embeddings for similarity and retrieval |
| Blob storage | shared api-core BlobStore seam | Generated candidate audio, transcoded renditions |
| Source audio | **the listener's own filesystem** | Never written, never moved, never copied |

## Data Ownership

| Domain | Owns |
|---|---|
| library | Source roots, track index, content-derived identity, embedded metadata, scan state |
| playback | Sessions, queue state, transport position, transcode cache references |
| decomposition | Cached attribute records keyed by track identity, batch progress |
| preference | Model state per context, listener-authored constraints |
| elicitation | Comparison history and derived ratings |
| signals | Interaction events — completion, replay, skip with position |
| generation | Candidate state, retention and eviction accounting |
| provenance | Generation lineage records |
| offers | Offer state, strictly downstream of a final ranking |

`ranking` owns **no** persisted data. It is a pure policy over data owned elsewhere,
which is part of what makes the blindness boundary enforceable.

## Schema Map

| Table | Domain | Key fields |
|---|---|---|
| `source_roots` | library | path, scan mode, last scanned |
| `tracks` | library | content id, current path, duration, embedded metadata, missing-since |
| `attributes` | decomposition | track id, kind, value, model, model version, computed at |
| `segments` | decomposition | track id, start, end, label |
| `profiles` | preference | id, context label, model state, updated at |
| `constraints` | preference | profile id, attribute, direction, strength, listener-authored |
| `comparisons` | elicitation | profile id, track a, track b, winner, scope, shown at |
| `events` | signals | track id, session id, kind, position, timestamp |
| `candidates` | generation | id, profile id, caption, params, blob ref, state |
| `lineage` | provenance | blob ref, model, model version, licence lane, seed, params, source refs |

**Track identity is content-derived**, so a listener can reorganise their files
without losing ratings, comparisons, or history. A track whose path disappears is
marked missing rather than deleted, and re-links by content when it reappears.

## Migrations And Compatibility

Attribute records carry the model and model version that produced them. When
`music-tools` changes a model, existing attributes are not silently reinterpreted —
they are marked stale and recomputed, because a preference model fitted over one
embedding space cannot be read against another.

That is the sharpest compatibility constraint in this scenario: **an embedding model
change invalidates the preference model's coordinate system.** Profiles record the
embedding model they were fitted against, and a change forces an explicit refit
rather than a silent degradation.

## Import / Export

- **Source audio** is referenced in place. There is no import step.
- **The taste profile is exportable and human-readable.** A listener can read it,
  edit it, and take it with them. A profile that cannot be inspected is exactly the
  thing this scenario exists to replace.
- **Comparison and event history** export in full.
- **Generated audio** exports with its complete lineage attached.
- Nothing here uploads audio anywhere.

## Retention And Deletion

| Class | Retention |
|---|---|
| Track index and attributes | Until the source root is removed; attributes are a regenerable cache |
| Interaction events | Retained — they are the training signal; bounded by an explicit listener-set window |
| Comparison history | Retained; deleting a comparison refits the profile |
| Preference model state | Retained until reset; versioned so a reset is recoverable |
| Generated candidates | Retained until confirmed-kept or evicted by budget |
| Transcode cache | LRU under a declared budget, regenerable |
| Lineage | Retained as long as its blob exists |

**Deletion is real and complete.** Removing a source root removes its tracks,
attributes, events, and comparisons, and refits affected profiles. Resetting a
profile discards model state and derived ratings while leaving raw history intact,
so a reset is a refit rather than an amnesia.

## Privacy Notes

Listening history is among the most revealing behavioural data a person generates.
It exposes mood, routine, sleep pattern, and association far beyond musical taste.
This scenario is designed around that:

- **Everything stays on the host.** No listening event, comparison, rating, or
  profile is transmitted anywhere. There is no telemetry path for behavioural data.
- **The profile is the listener's**, readable and exportable in full.
- **Generated audio is disclosed as generated**, always, in the interface and in
  exported metadata.
- **Offers never see behaviour.** The offer surface receives a final ranking, not
  the signals behind it — a privacy consequence of the blindness boundary as much
  as an integrity one.
- Source files are read-only; a bug here cannot damage a listener's music.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — the preference model in context
- [`DOMAINS.md`](DOMAINS.md) — domain ownership and the blindness boundary
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — where attributes come from
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — sensitivity and threat model
