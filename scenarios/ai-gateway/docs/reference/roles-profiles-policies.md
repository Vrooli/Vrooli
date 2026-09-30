# Roles, Profiles, And Policies

AI Gateway separates concepts that are currently easy to blur:

- **Role**: what kind of model capability the caller needs.
- **Profile**: how AI Gateway should route across providers.
- **Policy**: the provider/resource-specific mapping and constraints
  that make a role executable.

## Ownership

| Layer | Owner | Examples | Source Of Truth |
|---|---|---|---|
| Provider model catalog | Resource | `llama3.2`, hosted provider slugs, context windows, embedding dimensions. | `resource-ollama`, `resource-openrouter` policy/model commands. |
| Role mapping | Resource | `chat.default`, `summarize.default`, `embedding.default`. | Resource `model-policy.json` and resource policy CLI. |
| Routing profile | AI Gateway | `local-first`, `privacy-sensitive`, `cheap-first`. | AI Gateway routing policy. |
| Typed inference role | AI Gateway | `classify.fast`, `extract.structured`, `write.default`, schema subset, sampling stance, provider fallback order. | `config/inference-role-catalog.json`; candidates resolve through resource policy CLIs. |
| Caller request | Scenario caller | "summarize this", "extract JSON", "embed these chunks". | Gateway API/CLI contract. |

AI Gateway should display resource role policies and route through them,
but should not edit them as its primary storage path. Resource policy
changes belong in resource scenario work.

### Multimodal input

`GatewayRequest.attachments` adds input modalities to the existing request
kind; it does not create a parallel vision operation. The gateway requires
the selected resource role to declare every required input modality (for
example, text plus image) and rejects a mismatch with `capability_mismatch`
before invoking a provider. `RunRequest` additionally accepts ordered
`turns`; turns are request-scoped and the gateway stores no conversation
identifier or history. Route evidence retains only a hash, byte count, image
count, dimensions, and redaction flags.

## Baseline Roles

The gateway should understand these logical roles even when a specific
provider does not support every role yet:

| Role | Purpose | Ollama Status | OpenRouter Status | Notes |
|---|---|---|---|---|
| `chat.small` | Low-cost/simple chat. | existing | existing | Useful default for short interactions. |
| `chat.default` | General chat/generation. | existing | existing | Primary general-purpose role. |
| `chat.quality` | Higher-quality hosted reasoning/chat. | not expected locally initially | existing | May route remote-only/quality-first. |
| `summarize.default` | Summarization. | existing | existing | Common scenario utility. |
| `classify.routing` | Classification/routing decisions. | existing | existing | Should be cheap and reliable. |
| `extract.structured` | JSON/structured extraction. | should be added | existing | User explicitly wants this added for Ollama. |
| `locate.visual` | Visual grounding. | local-first | hosted fallback | Returns canonical bounds and caller-owned confidence. |
| `embedding.default` | General embeddings. | existing | investigate/add if supported | OpenRouter support must be verified in resource-openrouter first. |
| `rerank.llm_fallback` | LLM fallback reranking. | existing | existing | Not the same as a dedicated reranker. |
| `code.default` / `code.local` | Code assistance. | local/hosted split | existing hosted | Gateway may normalize caller intent later. |
| `agent.tools` | Tool-using agent model. | not local initially | existing | Likely hosted-only until local support exists. |

Status should be read from resource policies at runtime. This table is
a planning baseline, not an executable model catalog.

Typed inference is the narrower execution contract layered on top of those
provider roles. Its gateway-owned catalog selects `classify.fast`,
`extract.structured`, `judge.default`, `locate.visual`, `author.generator`,
`write.default`, and `write.diverse`, orders local and hosted candidates,
and records the resolved provider/model in every response. The catalog does not
edit resource policy or treat schema descriptions as caller instructions. The
current local structured-extraction candidate deliberately uses Ollama's
existing `chat.default` policy role until the resource exposes a dedicated
extraction role; the gateway still enforces the JSON Schema subset locally.

## Sampling

Sampling is declared at two layers, and the split is the point: the gateway
catalog declares *intent* (what this role should do and whether a caller may
change it), and resource policy declares *capability* (whether the provider
behind the role will actually apply the control).

### Role-declared stance

Each gateway inference role may declare:

```json
"sampling": { "temperature": 0.0, "overridable": false }
```

`overridable` defaults to false. A role is deterministic unless its author
deliberately opened it, so adding a role never accidentally admits caller
sampling.

| Role | Stance | Why |
|---|---|---|
| `classify.fast` | `0.0`, closed | identical sources must classify identically |
| `extract.structured` | `0.0`, closed | same |
| `judge.default` | `0.0`, closed | a judge that drifts is not a judge |
| `locate.visual` | `0.0`, closed | bounds must be reproducible |
| `author.generator` | `0.2`, open | every generator is validated before storage, so variety costs a rejection rather than a bad artifact |
| `write.default` | `0.9`, open | prose wants variety; the experimental sweep needs per-call control |
| `write.diverse` | `1.0`, open | same, and the candidate-set lane pins temperature per call |

### Declared provider support

Resource roles declare, per parameter, how their provider treats an explicit
control:

| Value | Meaning |
|---|---|
| `honored` | the provider applies the value |
| `ignored` | the provider accepts the field and silently discards it |
| `rejected` | the provider fails the request when the field is present |
| `unknown` | not established; treated as `ignored` |

**These are declarations, never probes.** There are three real states and only
one is detectable at runtime: a rejection surfaces as an error, but a provider
that accepts a control and silently discards it is indistinguishable at the call
site from one that honours it. A probe would report success and be wrong. Two
live examples make this concrete rather than theoretical: Anthropic's newest
models reject sampling parameters with a 400, while Gemini 3.x accepts
`temperature` and documents that it ignores it.

Declare from first-party provider documentation. An aggregator's
`supported_parameters` metadata is not evidence — it is known to contradict
upstream vendors' own migration guidance.

### Precedence

Resolved once, per candidate, in `ResourceRepository.Run`:

| Caller | Role | Result |
|---|---|---|
| sets temperature | `overridable: true` | caller's value, but only on a candidate declaring `honored`; any other candidate is skipped and the walk continues |
| sets temperature | closed | `INVALID_REQUEST`, construct `sampling.temperature` |
| sends nothing | declares temperature | role's value, sent to every candidate except one declaring `rejected`, which **omits** it and still serves the request |
| sends nothing | declares nothing | nothing sent; the resource role's own policy default applies |

Rows 1 and 3 draw the line in different places, and the difference is the whole
design.

A caller-supplied control is a **promise**, so only `honored` keeps it.
`ignored` fails that promise exactly as thoroughly as `rejected` does — the only
difference is whether the failure is visible — so both refuse the candidate, and
an exhausted walk refuses by name rather than returning output that quietly
ignored the caller. That is the same rule the schema gate follows when it
declines to degrade a rejected schema into unconstrained generation.

A role-declared default is a **preference**, and the question there is not "will
this be honoured" but "will sending it break the call". Only `rejected` breaks
it. So `ignored` and `unknown` are still sent: harmless where the provider
discards the value, and correct where an undeclared provider turns out to honour
it. This is what treating `unknown` as best-effort means — try it, rather than
silently drop the role's stated intent. It also keeps a deterministic role
deterministic against a resource whose declaration has not shipped yet.

Row 2 is `INVALID_REQUEST` rather than `UNSUPPORTED_SAMPLING` because the two
failures need different fixes: a role that forbids overrides is a request defect
the caller can correct, while provider incapacity is not something any request
change would repair.

### What the response reports

`RunResponse.applied` and `ExecuteRouteResponse.applied` carry
`temperature_sent`, `temperature_support`, `max_output_tokens_effective`, and
`max_output_tokens_source`. There is deliberately **no single "applied
temperature" field**: for a provider that accepts and silently ignores the
control, such a field would have to lie in exactly the case the caller most
needs the truth. A caller comparing two candidate sets must treat differing
support states as differing conditions even when `temperature_sent` matches.

`max_output_tokens_source` distinguishes `request`, `role_policy`, and
`none_imposed`. The last is a real statement, not a gap: it means the gateway
sent no cap at all, and the provider's own default applies unobserved. The
gateway never guesses at a cap it did not impose.

The routing path additionally persists `sampling_temperature` and
`sampling_temperature_support` on route evidence. `sampling_temperature` is
nullable so an omitted control stays distinguishable from a deterministic `0`.

### `locate.visual` coordinate contract

The role returns this provider-neutral value shape:

```json
{
  "found": true,
  "bounds": [0.10, 0.20, 0.40, 0.80],
  "confidence": 0.73
}
```

`bounds` is always four finite floats in canonical `[0,1]` space, ordered as
`[left, top, right, bottom]` relative to the submitted image. `confidence` is
returned unchanged in `[0,1]`; the gateway does not apply a threshold because
the caller owns the consequence of accepting a weak grounding result.

Resources declare the model response convention in `policy resolve` output
through `coordinate_convention`. The gateway currently accepts
`normalized_1000` (coordinates relative to a 0-to-1000 square) and
`absolute_pixels` (pixels in the submitted image). Conversion is selected from
that declaration, never guessed from numeric range. The gateway requires
positive attachment dimensions and rejects unknown conventions or converted
bounds outside the image, so callers never need provider-specific coordinates.

## Recommended Gateway Profiles

| Profile | Provider Preference | Fallback | Hard Blocks |
|---|---|---|---|
| `local-only` | Ollama only. | none | hosted providers. |
| `local-first` | Ollama first. | OpenRouter only if constraints permit. | hosted fallback for sensitive data unless explicitly allowed. |
| `remote-only` | OpenRouter only. | none | local providers. |
| `quality-first` | Highest quality viable role. | lower quality only if caller allows. | routes that violate privacy/cost. |
| `cheap-first` | Lowest cost viable role. | next cheapest viable route. | routes that violate role/capability needs. |
| `privacy-sensitive` | Local or explicitly approved private route. | none by default. | unapproved hosted providers. |

Profiles should be deterministic and previewable. If routes tie, the
route preview must say what tie-breaker was used.

### Capacity-Aware Local Eligibility

Before a local candidate is admitted, AI Gateway consults the platform capacity
broker (via the `vrooli capacity` CLI contract — never direct GPU/RAM probing)
when the request declares a footprint. The footprint is an explicit request
metadata constraint (`metadata.required_vram_bytes`, also `local_vram_bytes` /
`required_bytes`); it is never inferred from model names. Requests without a
declared footprint are not capacity-gated.

The broker returns a stable verdict recorded on the route candidate and route
evidence:

| Verdict | Meaning | Effect on local candidate |
|---|---|---|
| `fit` | Broker admits the footprint. | Eligible. |
| `unknown_capacity` | Broker unavailable/errored or no footprint declared. | Eligible (advisory — never blocks). |
| `reclaim_required` | Admitted, but only by reclaiming, with enforcement on. | Eligible. |
| `insufficient_capacity` | Broker cannot admit the footprint. | Rejected. |
| `advisory_reclaim_unavailable` | Admission needs reclaim but enforcement is advisory. | Rejected (treated as unavailable to avoid OOM). |

Profile behavior when a local candidate is rejected for capacity:

- `local-only` fails closed (no remote fallback exists).
- `local-first` / `cheap-first` fall back to a permitted remote provider.
- `privacy-sensitive` / secret-class requests fail closed — they never route
  remote merely because local capacity is constrained.

An op-scoped capacity claim is held around the local execution attempt and
released best-effort afterward; a crash falls back to the claim's bounded TTL.
Capacity verdicts feed analytics (the `route_events.capacity_rejections`
measure) but are policy filters only — never hidden route scoring.

## Resource Alignment Recommendations

1. Add Ollama `extract.structured` as a resource role if it can be backed by
   a local model that reliably emits JSON; until then the gateway catalog uses
   the existing `chat.default` role as its local extraction candidate.
2. Investigate OpenRouter embeddings in `resource-openrouter` before
   adding `embedding.default`; confirm endpoint support, response shape,
   dimensions, pricing, and policy command behavior.
3. Keep role names mostly shared across Ollama and OpenRouter. A role
   can be unsupported by a provider, but callers should not need
   provider-specific role names for common AI tasks.
4. Treat provider-specific roles as exceptions with clear names and
   documentation.
5. Add policy drift checks that compare resource README role tables to
   JSON/CLI policy output so stale docs are visible.

## Phase 3 Runtime Inventory

AI Gateway now reads resource policy through the resource CLIs rather
than from policy files:

- `resource-ollama policy roles --json`
- `resource-openrouter policy roles --json`

The inventory API returns normalized provider, role, capabilities,
locality, status, and policy schema version. It intentionally omits
concrete model slugs, provider URLs, and credentials. The smoke API runs
the same bounded commands and classifies missing binaries, non-zero
exits, malformed JSON, timeouts, and empty role inventories.

Current resource alignment observed for this phase:

- Ollama has `embedding.default`; it does not yet expose a dedicated
  `extract.structured` resource role. Gateway typed inference therefore uses
  `chat.default` for its local extraction candidate and keeps the role name
  provider-neutral.
- OpenRouter has `extract.structured`; `embedding.default` is still a
  resource-policy investigation before gateway routing can use it.

## Caller Contract Examples

Good:

```json
{
  "operation": "extract",
  "role": "extract.structured",
  "profile": "local-first",
  "constraints": {
    "structuredOutput": true,
    "privacy": "normal"
  }
}
```

Bad:

```json
{
  "provider": "ollama",
  "baseUrl": "http://localhost:11434",
  "model": "llama3.2:3b"
}
```

The first request lets AI Gateway and resources evolve. The second
couples the scenario to a host, provider, and concrete model.
