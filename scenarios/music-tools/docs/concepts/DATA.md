# Data — Music Tools

Storage ownership, schemas, retention, and the size arithmetic that constrains
the design. Domain ownership is in [`DOMAINS.md`](DOMAINS.md); system structure is
in [`ARCHITECTURE.md`](ARCHITECTURE.md).

## Purpose Of This Document

Use this document to answer:

- What does this scenario persist, and which domain owns it?
- Where does audio live, and what is derived versus source?
- What is the storage budget, and what happens when it is exhausted?
- What is retained, for how long, and what deletes it?

## Storage Overview

This scenario holds **no product data of its own**. It owns operational state —
the model registry, job records, style definitions, budget accounting — plus
derived audio artifacts that are treated as a cache, not a record.

| Store | Backend | Holds |
|---|---|---|
| Scenario database | SQLite by default | Registry and install state, job records, style definitions, budget ledger, measure samples |
| Blob storage | shared api-core BlobStore seam | Generated audio, separated stems, rendered artifacts |
| Model weights | scenario data directory | Downloaded weights, checksum-verified |
| Runtime environments | scenario data directory | Provisioned virtualenv for the embedding stack |

Outputs go through the BlobStore seam with ownership metadata rather than to
ad-hoc filesystem paths.

## Data Ownership

| Domain | Owns | Notes |
|---|---|---|
| models | Registry records, install state, licence lane, checksums | The registry seed is read-only; install state is mutable |
| jobs | Job records, status, progress | Server-owned; survive client disconnect |
| storage | Blob references, derived-artifact budget, LRU accounting | Owns eviction policy |
| styles | Style definitions | Built-in styles read-only; custom styles mutable |
| capacity | Claim state | Mirrors the control-plane broker; never the authority |
| measures | Latency, queue wait, degradation frequency samples | Aggregated, bounded retention |

Source audio is **never owned by this scenario**. Callers pass audio in; this
scenario does not manage anyone's library.

## Schema Map

| Table | Domain | Key fields |
|---|---|---|
| `models` | models | id, operations, architecture, disk bytes, min VRAM, licence, lane, checksum, enabled |
| `model_installs` | models | model id, state, installed bytes, verified at |
| `jobs` | jobs | id, operation, params, state, progress, applied profile rung, error |
| `blobs` | storage | id, kind (generated / stem / render), source track ref, bytes, last accessed |
| `budget` | storage | kind, budget bytes, used bytes |
| `styles` | styles | id, caption template, params, built-in flag |
| `takes` | composition | id, job id, style id, seed, caption as sent, caption as authored, blob ref, pool state, reserved by, reserved at, consumed at, times offered |
| `style_inventory` | styles | style id, target depth, replenish threshold, last replenish job |
| `measures` | measures | operation, duration, queue wait, degraded flag, timestamp |

Every generated blob records the model, licence lane, and applied profile rung that
produced it, so a degraded or restricted-lane output is never mistaken for a
full-quality permissive one.

### Why `takes` is a table and not a column on `jobs`

A composition job produces a **set** of candidates, not one artifact — see the
2026-09-18 batch decision in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).
Modelling that as one output per job would have to be broken later.

Two fields carry decisions rather than convenience:

- **`caption as sent` alongside `caption as authored`.** The generator's planner
  rewrites captions by default, and measurably ruins authored briefs. Keeping both
  makes the rewrite auditable instead of invisible; if they differ, a caller can
  see what the model was actually asked.
- **`pool state` and its companions.** Records where a take is in the reservation
  lifecycle below, and which caller holds it. This is stored as a **fact**, not
  interpreted: inferring taste from picks and skips belongs to a consuming
  scenario, and these columns are the seam it reads. The distinction is the line
  that keeps the PRD's "holds no opinion about taste" non-goal honest while still
  letting an inventory exist here.

### The reservation lifecycle

A take is not simply chosen or not chosen. It moves through a lifecycle, because
the pool has more than one consumer and a track that has shipped in one video must
not ship in the next.

```text
                   ┌──────────── release ────────────┐
                   ↓                                 │
generated → available → reserved → consumed          │
                   │         │                       │
                   │         └───────────────────────┘
                   └── discard ──→ discarded
```

| State | Meaning |
|---|---|
| `available` | In the pool, offerable to any caller |
| `reserved` | A caller holds it; no other caller will be offered it |
| `consumed` | It shipped. Terminal — the take is never offered again |
| `discarded` | Explicitly rejected. Terminal, and excluded from depth accounting |

Three properties are deliberate:

- **Reservation is what prevents collision.** Without it, two launch videos
  generated the same afternoon can ship the same bed. Selection state alone cannot
  express this, which is why `selection state` was replaced.
- **`release` is a first-class transition, not an error path.** "Give me a
  different one" is the single most likely thing a caller says, and it must be
  cheap: the take returns to `available` and `times offered` increments. Nothing is
  regenerated.
- **`times offered` is a fact, not a score.** It records that a take was put in
  front of someone and passed over. This scenario never reads it to rank, filter,
  or weight anything. A consuming scenario may; that is the seam.

A reservation held by a caller that never returns would leak pool depth, so
reservations carry a timestamp and expire back to `available`. Expiry is not a
failure and is not reported as one.

### Inventory depth

`style_inventory` holds the declared policy for a named style: a `target depth` of
ready takes and a `replenish threshold` below which a batch is enqueued. Depth
counts `available` plus `reserved`; `consumed` and `discarded` takes are gone from
the count, which is what makes consumption trigger replenishment.

The trigger that runs replenishment is **pluggable**, and the v1 needs no platform
facility: when a reservation or consumption drops depth below the threshold,
enqueue a batch. Idle-time replenishment is an optimisation over *when* that batch
runs, not what makes the pool possible — see
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).

Note the divergence from `image-tools`, which is deliberate rather than
accidental: its `--variations N` produces one primary result plus indexed extras
inside a single job. Here the takes are **peers**. There is no primary, because the
2026-09-18 decision holds that selection is the quality mechanism and the scenario
does not rank.

Storage budgets must be sized for the discarded takes, not the kept one. If a
usable track costs ten generations, the `generated` blob budget in `budget` is
roughly an order of magnitude larger than a naive per-delivered-track estimate,
and LRU eviction should prefer takes in the `discarded` state first, then
`available` takes that have never been reserved. A `reserved` take is never
evicted — a caller is holding it — and a `consumed` take's bytes are the
consumer's concern to release.

## The size arithmetic

This is the constraint that shapes the storage design, so it is recorded rather
than rediscovered.

| Artifact | Per track | At 10,000 tracks |
|---|---|---|
| Four separated stems, lossless | ~100 MB | **~1 TB** |
| Four separated stems, compressed lossless | ~60 MB | **~1 TB scale** |
| Pooled or segment-level embeddings | ~100 KB | ~1 GB |
| Frame-level embeddings, multi-layer | **~900 MB** | far beyond any available disk |

The reference host has 274 GB free on a volume at 85%. Stems for a large library
exceed available disk by roughly an order of magnitude, and frame-level embeddings
exceed pooled ones by about four orders of magnitude per track.

Two consequences, both structural:

- **There is no library-wide stem materialisation path.** Separation is on-demand
  under an LRU budget. The absence of a batch entrypoint is deliberate.
- **Embeddings persist pooled or segment-level.** Frame-level output requires
  explicit opt-in for a single track, is treated as interactive, and is never
  written to the shared index.

## Migrations And Compatibility

The registry seed carries a schema version. Registry changes are additive:
new fields default to a safe value, and an unknown licence defaults to the
restricted lane rather than the permissive one. Removing a model from the seed does
not delete its installed weights; installs are reconciled explicitly so a removed
model cannot silently orphan disk.

Job records are forward-compatible: an unknown operation or profile rung is
retained and reported rather than dropped.

## Import / Export

- **Model weights** are acquired from declared sources with checksum verification;
  an install refuses to start when free disk is below the declared floor.
- **Custom styles** import and export as data, so a house sound is portable.
- **Derived audio** is exportable through the BlobStore seam with its provenance
  metadata attached.
- There is no bulk import of anyone's music library. That is the consumer's
  concern.

## Retention And Deletion

| Class | Retention |
|---|---|
| Generated audio | Retained until the consumer releases it or the budget evicts it |
| Stems and renders | Cache — LRU eviction under a declared budget, regenerable on demand |
| Frame-level embeddings | Not persisted |
| Job records | Bounded retention; terminal jobs pruned after a retention window |
| Measure samples | Aggregated and bounded |
| Model weights | Retained until explicitly uninstalled |

Because every derived artifact is regenerable, eviction is always safe. Nothing in
this scenario is the only copy of anything a person cares about.

## Privacy Notes

This scenario sees audio it is asked to process and the captions used to generate
it. It does not see listening behaviour, ratings, or taste — those never leave the
consumer scenario. Prompts and captions may contain personal content and are
treated as caller data: retained with the job, pruned with it, and never
transmitted off-host except through an explicitly configured BYOK provider.

Generated output records its provenance so downstream disclosure obligations can be
met by the consumer.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — storage strategy in context
- [`DOMAINS.md`](DOMAINS.md) — domain ownership
- [`../reference/model-registry.md`](../reference/model-registry.md) — registry disk budget
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — cost of producing these artifacts
