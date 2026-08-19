# Architecture — Music Tools

How the scenario is put together, and why. Domain ownership is in
[`DOMAINS.md`](DOMAINS.md); durable decisions and their alternatives are in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## The shape of the problem

Music models are large, mutually incompatible, and collectively larger than the
hardware. Three constraints drive every structural choice here:

1. **VRAM is contended, not free.** The reference host has a 16 GB card that is
   already shared with other Vrooli resources. No two heavyweight music models can
   be resident at once.
2. **The Python stacks conflict irreconcilably.** The generation stack, the
   music-information-retrieval stack, and the embedding stack pin incompatible
   Python versions, torch builds, and NumPy majors. One environment cannot hold
   them.
3. **Derived audio outgrows disk faster than anything else.** Separated stems for a
   large library exceed available disk by roughly an order of magnitude, and
   frame-level embeddings are about four orders of magnitude larger per track than
   pooled ones.

## Three runtimes, no containers

Isolation is at the process and environment boundary, not the container boundary,
so the desktop delivery tier does not require a container runtime.

| Runtime | Kind | Holds | Why separate |
|---|---|---|---|
| `ace-step` | Managed-service resource | Composition and its native edit operations | Pins a newer Python and an aggressive inference backend |
| `music-mir` | Managed-service resource | Structure and beat analysis, stem separation | Pins an exact torch build and an older NumPy major |
| in-scenario sidecar | Native uv-provisioned virtualenv | Embedding models, notation, loudness | Compatible stack; the always-on hot path, so no process hop |

Both resources acquire lockfile-pinned wheels and checksum-verified weights
natively. Neither uses Docker. Because a resource is its own unit with its own
environment, this needs no change to the shared single-venv sidecar provisioner.

## Residency policy

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

## Licence lanes

Model licences vary more than model quality, and several of the strongest analysis
models are non-commercial; at least one useful tool declares no licence at all.
The registry therefore treats commercial usability as a first-class property, and
the lane is recorded per model rather than assumed per scenario.

- **Permissive lane** — usable in commercially distributed output. A build
  configured for this lane refuses to resolve anything outside it and must still
  satisfy the composition and analysis contracts.
- **Restricted lane** — non-commercial, share-alike, or unknown licence. Available
  for personal use. Unknown licences default here rather than to permissive.
- **Copyleft tools** are invoked as separate processes and never linked in-process,
  so their obligations do not reach the scenario binary.

## Storage strategy

Derived artifacts are produced on demand and held under a declared budget with
least-recently-used eviction. There is deliberately **no** entrypoint that
materialises stems for an entire library — the absence is a design constraint, not
an unimplemented feature. Embeddings persist pooled or segment-level; frame-level
output requires explicit opt-in for a single track and is never written to the
index.

Outputs are written through the shared api-core BlobStore seam with ownership
metadata rather than to ad-hoc filesystem paths.

## Relationship to sibling scenarios

`music-tools` is a capability primitive with no opinion about taste or libraries,
in the same relationship to `music-library` that `image-tools` has to
`backdrop-studio` and `asset-studio`. `audio-tools` owns speech and is not a
dependency in either direction.
