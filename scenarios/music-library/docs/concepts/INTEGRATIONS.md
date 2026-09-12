# Integrations — Music Library

Everything this scenario depends on, and what happens when it is missing.

## Purpose Of This Document

Use this document to answer:

- What must be running for the listener to do anything useful?
- What still works when the capability layer is down?
- What contract does this scenario rely on from `music-tools`?

> **Status:** the dependencies below are the intended shape.
> `.vrooli/service.json` does not yet declare them. See
> [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).

## Dependency Inventory

| Dependency | Kind | Required | Used for |
|---|---|---|---|
| `music-tools` | Vrooli scenario | **Yes** | Every attribute, every generated candidate |
| `qdrant` | Vrooli resource | Yes | Embedding index for similarity and retrieval |
| `postgres` | Vrooli resource | No | Optional backend; SQLite is the default |
| BlobStore | api-core seam | Yes | Generated audio and transcode cache |
| Listener filesystem | Host | **Yes** | Source audio, read-only |

## Vrooli Resources

### `qdrant` — embedding index

Holds track embeddings for similarity search, "more like this", and text-to-audio
retrieval. Well established in this repo with many existing consumers.

Degraded behaviour: similarity-driven surfaces and embedding-space exploration
become unavailable. Direct playback, library browsing, and the queue continue from
the relational index.

### `postgres` — optional

SQLite is the default and is sufficient for a single-listener library. Postgres is
available for larger libraries.

## Scenario Dependencies

### `music-tools` — the entire capability layer

This scenario runs **no models**. Everything comes across this boundary:

| Needed | Provided by |
|---|---|
| Semantic and musical embeddings | analysis |
| Structure boundaries and section labels | analysis |
| Tempo, key, loudness | analysis and deterministic ops |
| Tags — genre, mood, instrument | analysis |
| Lyrics | analysis |
| Isolated stems, on demand | transformation |
| Generated candidates | composition |
| Transcoding and format conversion | deterministic ops |

The contract relied upon: **any track, owned or generated, yields the same
structured description through one interface**, and every generated artifact
carries the model, licence lane, and applied profile rung that produced it. The
second half is what makes provenance disclosure possible.

Degraded behaviour when `music-tools` is stopped: **playback and library browsing
continue in full**, because attributes are cached locally against track identity.
What stops is new decomposition, new generation, and on-demand stem isolation. A
listener with an already-analysed library barely notices; a first run cannot start.

`audio-tools` is not a dependency. It owns speech.

## Third-Party Services

**None.** No streaming catalogue, no metadata service, no recommendation API, no
telemetry endpoint.

This is a design position, not an omission. The attribute layer that would
otherwise come from a commercial music API is computed locally — and the major
provider withdrew exactly those endpoints from public access, so local computation
is not a workaround but the only remaining path. Behavioural data never leaves the
host because there is nowhere for it to go.

## Failure Modes

| Condition | Behaviour |
|---|---|
| `music-tools` unavailable | Playback and browsing continue from cache; decomposition and generation queue |
| `qdrant` unavailable | Similarity surfaces degrade; queue and browsing continue |
| Source root unreachable | Tracks marked missing, not deleted; re-link by content when it returns |
| Source file moved | Re-linked by content identity; ratings and history preserved |
| Embedding model changed upstream | Attributes marked stale; affected profiles require an explicit refit rather than silently misreading their coordinate space |
| Preference model not yet fitted | Ranking states plainly that it is unfitted and routes to comparison, rather than presenting arbitrary output as recommendation |
| Generation backlog exceeds budget | Oldest unconfirmed candidates evicted first |
| Offer surface unavailable | Ranking is unaffected — offers are strictly downstream decoration |

That last row is the blindness boundary showing up as a failure mode: because
`offers` sits downstream of a final ranking, it cannot take recommendation down
with it.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — the preference model and its inputs
- [`DOMAINS.md`](DOMAINS.md) — domain ownership and the blindness boundary
- [`DATA.md`](DATA.md) — what is cached locally and what is regenerable
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — current wiring gaps
