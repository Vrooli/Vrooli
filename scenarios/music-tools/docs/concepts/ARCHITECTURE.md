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

### Three runtimes, no containers

Isolation is at the process and environment boundary, not the container boundary,
so the desktop delivery tier does not require a container runtime.

| Runtime | Kind | Holds | Why separate |
|---|---|---|---|
| `ace-step` | Managed-service resource | Composition and its native edit operations | Pins a newer Python and an aggressive inference backend |
| `music-mir` | Managed-service resource | Structure and beat analysis, stem separation | Pins an exact torch build and an older NumPy major |
| in-scenario sidecar | Native uv-provisioned virtualenv | Embedding models, notation, loudness | Compatible stack; the always-on hot path, so no process hop |

Both resources acquire lockfile-pinned wheels and checksum-verified weights
natively. Because a resource is its own unit with its own environment, this needs
no change to the shared single-virtualenv sidecar provisioner.

### Residency policy

The embedding pool is the only persistently resident model set. Composition, stem
separation, and structure analysis each take an **exclusive** GPU lease through the
control-plane capacity broker.

```
persistent   [ embedding pool ]────────────────────────────────
exclusive                       [ composition ]
                                                [ separation ]
                                                              [ structure ]
```

Every GPU-bearing operation declares ordered profile rungs — model variant,
precision, batch size — so the broker can degrade it under contention rather than
fail it. The applied rung travels with the result, so degraded output is never
mistaken for full-quality output.

## System Boundaries

| Inside | Outside |
|---|---|
| Composition, transformation, analysis over music and sound | Playback, libraries, taste, ranking — `music-library` |
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

Flow of a GPU-bearing operation: resolve model within the configured lane → declare
profile rungs → claim capacity → execute under an exclusive lease → write through
the BlobStore seam with provenance → release. Full state machines are in
[`FLOWS.md`](FLOWS.md).

## Shared Infrastructure

- **Capacity broker** (`vrooli capacity`) — the sole authority for GPU admission.
  This scenario holds no private GPU queue.
- **api-core BlobStore** — all outputs, with ownership metadata, rather than ad-hoc
  filesystem paths.
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

## Architecture Maturity

| Aspect | State |
|---|---|
| Documented | Yes — this document, `DOMAINS.md`, `FLOWS.md`, `DATA.md`, model registry |
| Implemented | **No.** No domain exists in code; the template example domain is still present |
| Resources created | **No.** `ace-step` and `music-mir` are declared but do not exist |
| Dependencies declared | **No.** `.vrooli/service.json` declares none |
| Measured | **No.** Every performance figure is vendor-stated or estimated |

This scenario is at `generated`. Everything above is intended architecture, not
running software. See [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).

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
