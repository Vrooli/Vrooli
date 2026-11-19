# Data — Music Library

Storage ownership, schemas, retention, and privacy. Domain ownership is in
[`DOMAINS.md`](DOMAINS.md); system structure is in [`ARCHITECTURE.md`](ARCHITECTURE.md).


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

### Track catalog and project usage

The library owns listener judgments and use history, not Music Tools' generated
take pool. A track is identified by the SHA-256 of its source audio and references
the caller-owned original file and generation take/receipt. Listener approval,
descriptive tags, and intended purposes are separate from use. Record a draft
association with a specific project independently from an accepted use; only an
accepted use makes a track used for an unused-track search. A used track remains
discoverable for reuse in that same project. Keep source files read-only, retain the
exact take provenance, and never infer preference from pool reservation or play.
The implemented slice is a catalog and use ledger; it does not claim
that automatic taste prediction, playback, missing-file monitoring, or generated-audio
retention exists. Paths are references: callers must retain their files and receipts.
The CLI verifies the source digest on registration; API clients provide metadata and
are responsible for verifying it. The catalog currently searches its full set in
memory, so it is intended for modest collections rather than unbounded scale.

Use `music-library catalog put --help`, `catalog list --help` and `catalog use --help`
for the generic commands. The exact-tag and project filters can be combined with
`--unused true`; a draft association does not make the source used. A project and
revision identify a use, and an accepted use cannot be downgraded to a draft by
repeating the command.

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
| `music_decompositions` | decomposition | track SHA-256, Music Tools job id, exact result envelope, receive time |
| `music_decomposition_reviews` | decomposition | track/job identity, human verdict and timestamped corrections |
| `music_decomposition_taste_features` | decomposition | compact bounded feature projection, exact track/job/review provenance, confidence |
| `music_taste_profile_versions` | preference | context, monotonic revision, full readable profile snapshot, created time |
| `music_taste_judgments` | elicitation | append-only judgment id, context, exact left/right SHA-256, distinct choice, evidence snapshots, source time |
| `music_taste_audit_events` | preference | append-only explicit-control or judgment event and JSON payload |
| `music_releases` | releases | stable local release identity and creation time |
| `music_release_versions` | releases | immutable album/EP metadata revision and pinned external artist-subject reference |
| `music_release_track_occurrences` | releases | ordered occurrence bound to an exact catalog SHA-256 |
| `music_release_recording_provenance` | releases | immutable take, receipt path and existing track judgment snapshotted for each exact occurrence |
| `music_release_credits` | releases | role, credited spelling, optional external subject reference, revision scope |
| `music_release_credit_identity_versions` | releases | pinned external subject version for each credited artist identity |
| `music_release_assets` | releases | owner-issued asset ref/hash and image-operation/model/license/rights provenance |
| `music_release_track_rights` | releases | exact-hash license lane, usage scope, operator decision and evidence ref |
| `music_release_rights_decisions` | releases | append-only operator use-right attestation bound to revision, exact hash and recording receipt |
| `music_release_derivations` | releases | exact source/derived hashes, remix/sample spans, operation receipt, inherited rights |
| `music_release_reviews` | releases | append-only operator review decision for one immutable edition revision |
| `music_release_review_pins` | releases | edition digest pinned by each durable operator review |
| `music_release_review_attestations` | releases | ordered exact hashes the operator confirmed hearing in that batch |
| `events` | signals | track id, session id, kind, position, timestamp |
| `candidates` | generation | id, profile id, caption, params, blob ref, state |
| `lineage` | provenance | blob ref, model, model version, licence lane, seed, params, source refs |

**Track identity is content-derived**, so a listener can reorganise their files
without losing ratings, comparisons, or history. A track whose path disappears is
marked missing rather than deleted, and re-links by content when it reappears.

The preference feature space consumes explicit tags stored on the exact catalog row;
a tag with the explicit `genre:<name>` prefix is exposed as a genre attribute without
classifying generic tags. Each value carries the catalog SHA-256 as evidence provenance.
Hash-matched decomposition source metadata supplies artist labels for diversification
only, never as a taste score. Duration, tempo and key enter the taste feature space
only as compact projections from the latest exact-hash analysis job explicitly
confirmed by the operator; a newer job or later non-confirming review hides them.
Unreviewed analysis is omitted. No reference analysis is performed here. Taste controls are stored
separately from learned weights. Pairwise choice, reason, attribute scope, context,
correction link, and exact feature snapshots remain append-only; a reversal adds a
new judgment instead of rewriting the old event. A profile version is a recoverable
snapshot, not the label source of truth.

Release editions use a separate immutable version chain: localized metadata, track
order, credits, assets, rights and derivations are saved together as one revision.
Exact track foreign keys prevent a release from pointing at a hash absent from the
catalog. Artwork bytes stay with the image owner; Library records only exact asset
refs/hashes and the source operation/license/rights provenance. Artist IDs remain
opaque until their canonical subject owner verifies the pinned version. An approval
is append-only and must be refused while any eligibility proof is missing. Release
storage holds no audio and adds no publishing or distribution action.

## Migrations And Compatibility

Music Tools results are cached as immutable job-addressed records rather than
flattened into assumptions about one analysis model. `POST
/api/v1/decompositions/{track_sha256}` accepts the complete Music Tools result
envelope and verifies its source digest, job id, and `music-mir` provenance.
The source digest must match the track key. `GET` returns the latest result and
its review history; `?analysis_job_id=...` retrieves a specific run. Music
Library stores only this JSON and the listener's feedback. Music Tools remains
the owner of analysis and stem blobs, so this path does not copy source audio or
stem files.
`GET /api/v1/decompositions?limit=20&cursor=...` provides stable, newest-first
pages of track/job summaries and review counts for a review queue. It does not
load the potentially large result envelopes; callers fetch one selected track
with the detail route above. The cursor is opaque and page size is capped at 50.

Reviews are append-only and tied to one analysis job. A verdict can be
`confirmed`, `corrected`, or `needs_work`; timestamped corrections carry an
exclusive start/end range in seconds, a section label, and an optional note.
This provides durable evidence for the MQ1 listening review. The responsive review
page now selects an exact analysis job, privately streams source audio from Music
Tools, and appends verdicts and timestamped corrections; the topic taste-register
reconciliation is still outstanding.

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
