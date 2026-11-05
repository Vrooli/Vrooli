# Architecture — Music Tools

How the scenario is put together, and why.

## Purpose Of This Document

Use this document to answer:

- What shape is this scenario, and what forced that shape?
- Where are its boundaries, and what is deliberately outside them?
- What contracts do consumers rely on?
- Which structural choices are deviations from the house pattern, and why?

Domain ownership is in [`DOMAINS.md`](DOMAINS.md); workflows are in
[`FLOWS.md`](FLOWS.md); storage detail is in [`DATA.md`](DATA.md); durable decisions
and their rejected alternatives are in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Scenario Shape

Music models are large, mutually incompatible, and collectively larger than the
hardware. Three constraints drive every structural choice here:

1. **VRAM is contended, not free.** The reference host has a 16 GB card already
   shared with other Vrooli resources. No two heavyweight music models can be
   resident at once.
2. **The Python stacks conflict irreconcilably.** The generation stack, the
   music-information-retrieval stack, and the embedding stack pin incompatible
   Python versions, torch builds, and NumPy majors. One environment cannot hold
   them.
3. **Derived audio outgrows disk faster than anything else.** Separated stems for a
   large library exceed available disk by roughly an order of magnitude, and
   frame-level embeddings are about four orders of magnitude larger per track than
   pooled ones.

### Runtime boundaries: composition implemented, analysis planned

Isolation is at the process and environment boundary, not the container boundary,
so the desktop delivery tier does not require a container runtime.

| Runtime | Kind | Holds | Why separate |
|---|---|---|---|
| `stable-audio-3` | Managed-service resource | Default local instrumental composition | Offline pinned Medium model, resource-owned resident admission |
| `ace-step` | Managed-service resource | Composition and its native edit operations | Pins a newer Python and an aggressive inference backend |
| `music-mir` (planned; resource not created) | Managed-service resource | Structure and beat analysis, stem separation | Pins an exact torch build and an older NumPy major |
| in-scenario sidecar (planned) | Native uv-provisioned virtualenv | Embedding models, notation, loudness | Compatible stack; the always-on hot path, so no process hop |

The implemented ACE-Step resource acquires lockfile-pinned wheels and checksum-verified weights
natively; the other runtimes remain intended architecture. Because a resource is its own unit with its own environment, this needs
no change to the shared single-virtualenv sidecar provisioner.

### Residency policy

The intended analysis design keeps an embedding pool resident. ACE composition
claims operation-scoped VRAM through the control-plane broker and renews that claim.
The broker can be advisory: a claim does not establish exclusive GPU ownership.
Music applies a supported reduced rung or refuses when the loaded resource cannot
apply it without a managed restart. Separation/structure runtime policy is planned.

```
persistent   [ embedding pool ]────────────────────────────────
exclusive                       [ composition ]
                                                [ separation ]
                                                              [ structure ]
```

ACE GPU-bearing jobs declare ordered profile rungs — model variant,
precision, batch size — so the broker can degrade it under contention rather than
fail it. The applied rung travels with the result, so degraded output is never
mistaken for full-quality output.

## System Boundaries

| Inside | Outside |
|---|---|
| Composition and bounded operator audition; planned music transformation/analysis | Product playback, libraries, taste/ranking — consuming products |
| The model registry, its licence lanes, and model execution | Speech recognition and synthesis — `audio-tools` |
| GPU claims, degradation, and release | Host GPU detection and remediation — the control plane |
| Derived-artifact budget and eviction | Ownership of anyone's source audio |

`music-tools` is a capability primitive with no opinion about taste or libraries,
in the same relationship to `music-library` that `image-tools` has to
`backdrop-studio` and `asset-studio`. `audio-tools` owns speech and is not a
dependency in either direction.

The scenario depends on no other scenario. A capability primitive that depended on
a product scenario would invert the layering.

## Contracts And Data Flow

The consumer-facing contract has four parts, and all four are load-bearing:

1. **Every operation runs headless.** CLI and API parity, report-shaped output, no
   UI dependency and no external workflow tool.
2. **Uniform decomposition.** Any track — owned or generated — yields the same
   structured description through one interface.
3. **Provenance on every artifact.** Model, model version, licence lane, and the
   applied profile rung travel with the output.
4. **Explicit partiality.** When a runtime is unavailable the result says which
   layers are missing rather than returning a quietly shorter description.

Composition resolves and snapshots its provider within the configured lane. SA3 validates
resource readiness and identity, then uses resource-owned residency and its generation
slot. ACE jobs declare profile rungs, claim/renew capacity, and release after execution.
Both paths write through the BlobStore seam with provenance. Full state machines are in
[`FLOWS.md`](FLOWS.md).

## Shared Infrastructure

- **Capacity broker** (`vrooli capacity`) — the sole authority for GPU admission.
  The durable local GPU job lane serializes this scenario’s work; it does not arbitrate host GPU capacity.
- **api-core BlobStore** — generated server outputs and ownership receipts. The
  accepted caller-local arrangement CLI writes an explicit new output bundle.
- **`pyenv-go`** — provisions the single in-scenario virtualenv. Unmodified; the
  additional environments live in resources precisely so this package needs no
  change.
- **`qdrant`** — available for embedding indexing, primarily for consumers.

### Storage strategy

Derived artifacts are produced on demand and held under a declared budget with
least-recently-used eviction. There is deliberately **no** entrypoint that
materialises stems for an entire library — the absence is a design constraint, not
an unimplemented feature. Embeddings persist pooled or segment-level; frame-level
output requires explicit opt-in for a single track and is never written to the
index.

## Extension Rules

- **New model** — add a registry entry with hardware gates, disk cost, checksum,
  licence, and lane. An unknown licence defaults to the restricted lane, never the
  permissive one.
- **New operation** — name it from the existing operation vocabulary; do not
  re-declare vocabulary in a new domain.
- **New GPU path** — it must claim through the broker and declare degradation rungs.
  A code path that touches the card without a claim is a defect.
- **New Python dependency** — it goes in the runtime whose pins it matches. If it
  matches none, that is a signal for a fourth runtime, not for loosening a pin.
- **Never** add a private GPU manager, a library-wide stem materialisation path, or
  a direct filesystem output path that bypasses the BlobStore seam.

### Licence lanes

Model licences vary more than model quality, and several of the strongest analysis
models are non-commercial; at least one useful tool declares no licence at all. The
registry therefore treats commercial usability as a first-class property, recorded
per model rather than assumed per scenario.

- **Permissive lane** — usable in commercially distributed output. A build
  configured for this lane refuses to resolve anything outside it and must still
  satisfy the composition and analysis contracts.
- **Restricted lane** — non-commercial, share-alike, or unknown licence. Available
  for personal use. Unknown licences default here.
- **Copyleft tools** are invoked as separate processes and never linked in-process,
  so their obligations do not reach the scenario binary.

## Video delivery reliability contract (2026-10-16)

Caller-local arrangement stays in the bounded CLI delivery domain. It validates
sample arithmetic before work, snapshots regular source bytes, decodes into
bounded disk spools, and mixes fixed-size PCM blocks. Context cancellation covers
copy, decode, mix and publication. Raw snapshots are bounded to 256 MiB each /
512 MiB total, decoded spools to 128 MiB each / 256 MiB total, and output to
600 seconds. Process/codec peak memory requires measurement. A complete WAV/recipe/hash receipt bundle is
staged beside the destination and published under an exclusive owner publication
lock; existing outputs are preserved. The sibling lock fences cooperating owner
invocations; arbitrary external writers racing the final rename are outside
that portable guarantee. Hard termination can leave a lock/scratch directory;
inspect ownership before explicit cleanup rather than automatically deleting it. Source-byte bounds and decoded-byte bounds
are separate from measured process memory. The receipt records the decoder and
algorithm identity; listening remains unreviewed.

Production composition has one durable manager and one generation path. Domain
mutations must persist before becoming observable. Persistence failures are
errors, never terminal success. A failed metadata finalization preserves audio
and remains explicitly recoverable; clients must receive the error through
Get/Wait rather than an invented durable result. The pool schema is installed by
production EnsureSchemas, not opportunistic constructor DDL. Existing dev data
is retained; incompatible old schema requires an owner-directed maintenance step.

Generated media uses bounded streaming ingestion and retains actual model/rung
identity supplied by ACE-Step. Invalid requests fail before capacity admission.
Operation-scoped capacity claims renew via the control-plane heartbeat contract;
Music never implements host policy or stops another tenant. Audition is a bounded
candidate session: unloading playback bytes does not delete takes or erase
hash-bound findings; listening remains an explicit operator attestation.

### Reviewed concurrency and recovery obligations

Cleanup uses conditional deletion under the same storage lock that protects a
reservation's durable commit. A stale preview cannot delete a newly reserved,
changed, or recently accessed artifact. Explicit administrative deletion is a
separate operation. ACE-Step serializes load/degrade acknowledgement and checks
Music's explicit admitted-rung set before inference: full budget permits loaded
full or offload, reduced budget permits offload only. Existing strict
`expected_rung` callers remain strict, including when both restrictions are set.
Job `capacity_admitted_rung` records admission; actual output headers, take
provenance, and uniform successful job summaries record execution. Mixed batches
mark `output_identity: mixed` and omit each varying model/rung summary field;
per-take identities remain actual. The authoritative request contract is
`resources/ace-step/docs/API.md`. Historical job reads distinguish a
missing row from unavailable/canceled/corrupt storage. Terminal notifications
retain one final event even for slow subscribers. Persisted terminal cache entries
are bounded to 256 by default; active/subscribed work and pending failed
finalizations remain retained, and historical observers use durable rows. Pool
targets are 1–100, matching supported batch size. Authorized cleanup of available
audio makes a later selection durably mark that take unavailable, exclude it
from ready depth, and request refill under the existing threshold policy. Reservation
checks a regular physical blob and bounded readable handle under the storage lock;
only confirmed absence retires availability. Permission/transient I/O errors remain
explicit failures. Partial owned deletion removes catalog availability once audio
is gone even if receipt cleanup fails, preserving that cleanup error and receipt. Audition draft restore reconciles
loaded and attested hashes before a listening finding can be exported.

## Architecture Maturity

| Aspect | State |
|---|---|
| Documented | Yes — this document, `DOMAINS.md`, `FLOWS.md`, `DATA.md`, model registry |
| Implemented | **Partial.** Composition, styles, durable jobs, takes/pool, model registry, storage and cleanup exist; the full analysis/transform inventory is not implemented |
| Resources created | ACE-Step exists; music-mir remains planned |
| Dependencies declared | SA3 is enabled, optional, try-start; ACE is optional/on-demand |
| Measured | Prior composition and exact arrangement trials are retained; fresh reliability changes need focused checks and supported-runtime qualification |

Do not infer whole-scenario readiness from the implemented composition slice.
Historical broad validation remains non-green. See [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).

## Intentional Deviations

| Deviation | House pattern | Why |
|---|---|---|
| Two managed resources plus a sidecar | `image-tools` keeps all model code in one in-scenario sidecar | The dependency stacks cannot share one environment, and the shared provisioner supports exactly one |
| No container runtime | Existing GPU resources use compose overlays | The desktop delivery tier cannot assume a container runtime is present |
| Licence lane as a build configuration | Most scenarios assume all models are usable | The strongest analysis models are non-commercial, and this feeds a bundle intended to be sold |
| A deliberately absent batch path | Batch operations are normally a feature | Library-wide stem materialisation cannot fit any plausible disk budget |

## Documentation Architecture

- [`DOMAINS.md`](DOMAINS.md) — bounded contexts and ownership
- [`FLOWS.md`](FLOWS.md) — ordered states and illegal transitions
- [`DATA.md`](DATA.md) — storage ownership, retention, and the size arithmetic
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependencies and failure modes
- [`../reference/model-registry.md`](../reference/model-registry.md) — per-model evidence
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — rejected alternatives

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — domain ownership
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — why each choice was made
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — current gaps
- [`../reference/model-registry.md`](../reference/model-registry.md) — model evidence



## Approved default local instrumental route — 2026-10-17

SA3 owns its resident capacity claim and single generation slot. Music validates the
qualified runtime recipe, full raw receipt, and WAV before publishing a take; it never
applies ACE degradation to SA3. Exact raw receipts are protected, non-regenerable
owned blobs, linked by hash from durable job metadata. Registry declarations report
`runtime_unverified` until an execution readiness probe; they do not imply installation.
See [the routing decision](../internal/DECISIONS.md#approved-default-local-instrumental-route--2026-10-17).

### Capability-based lyric selection

The existing model registry declares composition engine, preference order and sourced
sung-lyrics capability. New composition chooses the preferred capable model; an
explicit unsupported variant is refused with its reason/source. Current SA3 has no
intelligible vocal output, so authored lyric songs use an ACE variant. This is a model
capability result, not an engine-name rule. Recorded provider validation checks its
capability without selecting a new default, preserving queued/recovered ownership.
Common style compilation preserves lyrics without imposing an engine's structured
BPM requirement; actual provider controls are validated at composition admission.

New admissions also freeze the selected exact variant, including an implicit lyric
model preference. A differently loaded runtime is refused through the existing
expected-variant/returned-variant contract; Music never switches a resident model.
Previously queued snapshots with blank variant retain their historical semantics.

## Accepted song-model integration target — 2026-10-17

The operator selected ACE take 2 for the intended KiiiKiii-like energy and
MiniMax quantized seed 170104 for realism, while finding that MiniMax take's
genre and voice mismatched. These are separate quality dimensions; neither
result establishes an artist-identity guarantee or replaces the approved
Stable Audio instrumental default.

MiniMax Music3 (the tested NF4 semantic/output-head recipe) and HeartMuLa use
one managed `songgen` resource. Each model has its own pinned source, weights,
wheel closure, capabilities and sampling profile. The resource shares the
HTTP/job/receipt/cache machinery; isolated child processes load one profile
per generation and release GPU and CPU allocations on exit. Model assumptions
belong in profile data and the corresponding backend adapter. Music's durable
job/pool/blob pipeline publishes their verified takes. No hosted API, silent
model fallback or audible artist-identity guarantee is part of this target.

Resource lifecycle declares a common exclusive group for music generators.
The control plane serializes lifecycle mutations for the group and refuses
starting a second member while another owned member runs. It must not kill an
active generation to switch models. Explicit normal stop/start is the
activation route; unrelated coding agents and GPU tenants are outside the group.

Disk retention is distinct from GPU eviction. Keep downloaded weights across
requests and restarts; default eligibility is 30 days since recorded use,
with pinning and explicit preview/apply. Unknown usage, active work, installation,
missing identity or changed previews prevent deletion. Delete whole declared
model bundles, never individual shards or operator exports. Acquisition and
generation/cleanup share the control plane's resource-data lock. Consumers express reuse intent through use and pins. The resource catalog
supplies the shared cache defaults and owns model-cache mutation.
Ordinary filesystem-age retention must not prune these bundles directly.
Weights for optional profiles are acquired by explicit declared group through
the control plane. Routine resource installation/start must not fetch a missing
on-demand group. This prevents an unused profile removed by retention from
being downloaded again when another profile is selected. Group acquisition
uses the same pinned data declarations, authorization and resource-data fence.


Acceptance requires focused provider/receipt, worker cancellation, exclusive
lifecycle and retention race regressions; normal managed activation; a fresh
bounded generation per new backend through Music; verified receipts/audio;
no overlapping music model residency; and no unused experimental models or
workers remaining. Source tests alone do not establish live adoption or sound
quality. After checks pass, run two fresh adversarial reviews and resolve
material correctness, maintainability and lifecycle findings.
