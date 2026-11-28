# Integrations — Music Tools

Everything this scenario depends on, and everything that depends on it.


Use this document to answer:

- What must be running for an operation to succeed?
- What degrades, and what fails outright, when a dependency is unavailable?
- What contract do consuming scenarios rely on?

> **Current manifest:** `.vrooli/service.json` declares `stable-audio-3` and
> `songgen` as enabled optional resources, and `ace-step` as disabled and
> optional. It declares no `music-mir`, Qdrant, Ollama, or OpenRouter resource.
> The analysis API still has a conditional local endpoint configured by
> `MUSIC_MIR_URL` (default `http://127.0.0.1:8898`); this is not a manifest
> dependency. Music Library outbox delivery is also implemented through
> discovery or `MUSIC_LIBRARY_URL`, but no scenario dependency is declared.
> See [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) for qualification.

## Dependency Inventory

| Dependency | Kind | Required | Used for |
|---|---|---|---|
| `stable-audio-3` | Vrooli resource, enabled optional (`ignore`) | Required when its composition provider is selected | Default instrumental composition |
| `songgen` | Vrooli resource, enabled optional (`ignore`) | Required for explicitly selected SongGen profiles | MiniMax and HeartMuLa composition profiles |
| `ace-step` | Vrooli resource, disabled optional (`ignore`) | Required for explicitly selected ACE-Step profiles | ACE-Step composition provider; not the default route |
| `music-mir` | Local HTTP endpoint; not declared in the service manifest | Required when analyze or separate is invoked | `MUSIC_MIR_URL` target, defaulting to `http://127.0.0.1:8898`; unavailable endpoint fails that job |
| `music-library` | Optional discovered scenario; not declared in the service manifest | Used for durable analysis-result forwarding | `MUSIC_LIBRARY_URL` override or scenario discovery; the outbox retries while the library is unavailable |
| Capacity broker | Control plane | **Yes, for any GPU operation** | Claims, admission, degradation, release |
| BlobStore | api-core seam | Yes | Output storage with ownership metadata |

## Vrooli Resources

### `stable-audio-3` and `songgen` — declared composition runtimes

Both are enabled and optional in `.vrooli/service.json` with startup policy
`ignore`. The default instrumental provider uses `stable-audio-3`; selected
SongGen profiles use `songgen`. Composition through a provider fails clearly
when that provider's resource is unavailable; other providers remain separate
choices.

### `ace-step` — disabled composition runtime

The manifest keeps ACE-Step disabled and optional. Explicitly selected ACE-Step
profiles use the resource's own generation stack and exclusive GPU lease;
ordinary composition does not require it.

Degraded behaviour: composition operations queue; if the resource is stopped, they
fail explicitly rather than silently falling back to a cloud provider.

### `music-mir` — unregistered analysis endpoint

The analysis module POSTs analyze/separate jobs to `MUSIC_MIR_URL`, which
defaults to loopback port 8898. The API uses its durable job manager and GPU
admission, and retains admission until the endpoint acknowledges cancellation
drain. This endpoint is not declared as a Vrooli resource or scenario
dependency; availability and installation must be qualified separately.

Degraded behaviour: analyze/separate jobs fail when the configured endpoint is
unavailable. The current analysis module does not provide a local fallback.

Qdrant, Ollama, and OpenRouter are not declared resources for this scenario, and
no corresponding client path was found in the current API. Callers supply a
caption directly to composition; this guide does not claim a caption-assistance
integration.

## Scenario Dependencies

The service manifest currently declares no scenario dependencies. The API has
an optional Music Library outbox integration resolved by `MUSIC_LIBRARY_URL` or
scenario discovery; it retries forwarding while Library is unavailable without
changing the already-terminal producer job.

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

Composition inference uses local Vrooli resources. Cross-scenario forwarding
uses the Music Library API; there is no paid or third-party service in that
path. No OpenRouter or other caption-assistance client path was found in the
current scenario API. Model acquisition is handled by the declared resource
owners.

This is the property that makes zero marginal cost real: no per-generation royalty,
no per-request billing, no service that can change its terms.

## Failure Modes

| Condition | Behaviour |
|---|---|
| Capacity broker denies admission | Operation queues with a stated reason; never proceeds unclaimed |
| Contention during a claim | Degrade to the next declared rung; applied rung travels with the result |
| Composition resource stopped | Composition fails explicitly; analysis unaffected |
| `MUSIC_MIR_URL` endpoint unavailable | Analyze and separate jobs fail; the current analysis module has no local fallback |
| Music Library stopped | Completed analysis remains successful; durable outbox retries until forwarding succeeds |
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
