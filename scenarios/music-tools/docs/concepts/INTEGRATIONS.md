# Integrations — Music Tools

Everything this scenario depends on, and everything that depends on it.

## Purpose Of This Document

Use this document to answer:

- What must be running for an operation to succeed?
- What degrades, and what fails outright, when a dependency is unavailable?
- What contract do consuming scenarios rely on?

> **Status:** the dependencies below are declared here and in `PRD.md` as the
> intended shape. `.vrooli/service.json` does not yet declare them, and the two
> managed-service resources do not yet exist. See
> [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).

## Dependency Inventory

| Dependency | Kind | Required | Used for |
|---|---|---|---|
| `ace-step` | Vrooli resource | For composition | Generation, cover, section repaint |
| `music-mir` | Vrooli resource | For structure and separation | Structure, beats, downbeats, stem separation |
| `qdrant` | Vrooli resource | No | Embedding index, primarily for consumers |
| `ollama` | Vrooli resource | No | Local caption assistance |
| `openrouter` | Vrooli resource | No | BYOK cloud caption assistance |
| Capacity broker | Control plane | **Yes, for any GPU operation** | Claims, admission, degradation, release |
| BlobStore | api-core seam | Yes | Output storage with ownership metadata |
| In-scenario sidecar | Native virtualenv | Yes, for analysis | Embeddings, notation, loudness |

## Vrooli Resources

### `ace-step` — composition runtime

Owns the generation stack in its own native environment. Holds an **exclusive** GPU
lease while generating. Declares ordered profile rungs — model variant, planner
size, precision, batch — so the broker degrades rather than fails under contention.

Degraded behaviour: composition operations queue; if the resource is stopped, they
fail explicitly rather than silently falling back to a cloud provider.

### `music-mir` — music-information-retrieval runtime

Owns structure and beat analysis and stem separation. Exists as a separate runtime
because its dependencies pin an exact torch build and an older NumPy major that
cannot coexist with the composition stack.

Degraded behaviour: structure and separation operations become unavailable.
Embedding, loudness, and notation analysis continue, so a track still yields a
partial description rather than none.

### `qdrant` — embedding index

Optional here and primarily for consumers. This scenario computes embeddings; it
does not require a vector index to do so.

### `ollama` / `openrouter` — caption assistance

Both optional. Used to turn a loose description into a structured caption. Absent
either, callers supply captions directly and every operation still works.

## Scenario Dependencies

This scenario depends on **no other scenario**. That is deliberate: a capability
primitive that depended on a product scenario would invert the layering.

Consumers, in expected order of adoption:

| Consumer | Uses |
|---|---|
| `music-library` | Analysis for every track; composition for candidates; transformation for edits |
| `asset-studio` | Composition for marketing video beds |
| `backdrop-studio` | Composition for ambient surfaces |
| `bedtime-story-generator` | Composition for scored, mood-matched music |
| `content-desk` | Composition and deterministic delivery operations |

The consumer contract is: **every operation is available headless from the CLI and
the API, returns report-shaped output, and records the model, licence lane, and
applied profile rung against every artifact.** Consumers never reach into the model
registry or claim GPU capacity themselves.

`audio-tools` is not a dependency in either direction. It owns speech; this
scenario owns music. Lyric transcription lives here because it is a music-specific
model, not a general speech capability.

## Third-Party Services

**None at runtime.** Every model runs locally. The only outbound network activity is
model acquisition — declared sources, checksum-verified — and optional BYOK caption
assistance if the operator configures it.

This is the property that makes zero marginal cost real: no per-generation royalty,
no per-request billing, no service that can change its terms.

## Failure Modes

| Condition | Behaviour |
|---|---|
| Capacity broker denies admission | Operation queues with a stated reason; never proceeds unclaimed |
| Contention during a claim | Degrade to the next declared rung; applied rung travels with the result |
| Composition resource stopped | Composition fails explicitly; analysis unaffected |
| MIR resource stopped | Structure and separation unavailable; other analysis continues |
| Free disk below the declared floor | Model install refuses to start rather than filling the volume |
| Derived-artifact budget exhausted | LRU eviction; regenerable artifacts are never the only copy |
| Model weights fail checksum | Install fails; the model stays unavailable rather than running unverified |
| Permissive-lane build requests a restricted model | Resolution refuses — it does not silently substitute |

The consistent rule: **refuse explicitly rather than degrade silently.** An operator
should always be able to tell whether output is full-quality, degraded, or absent.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — the three-runtime split and residency policy
- [`DOMAINS.md`](DOMAINS.md) — which domain owns each seam
- [`../reference/model-registry.md`](../reference/model-registry.md) — what each runtime holds
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — current wiring gaps
