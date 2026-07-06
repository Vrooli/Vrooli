# Roles, Profiles, And Policies

AI Gateway separates three concepts that are currently easy to blur:

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
| Caller request | Scenario caller | "summarize this", "extract JSON", "embed these chunks". | Gateway API/CLI contract. |

AI Gateway should display resource role policies and route through them,
but should not edit them as its primary storage path. Resource policy
changes belong in resource scenario work.

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
| `embedding.default` | General embeddings. | existing | investigate/add if supported | OpenRouter support must be verified in resource-openrouter first. |
| `rerank.llm_fallback` | LLM fallback reranking. | existing | existing | Not the same as a dedicated reranker. |
| `code.default` / `code.local` | Code assistance. | local/hosted split | existing hosted | Gateway may normalize caller intent later. |
| `agent.tools` | Tool-using agent model. | not local initially | existing | Likely hosted-only until local support exists. |

Status should be read from resource policies at runtime. This table is
a planning baseline, not an executable model catalog.

## Recommended Gateway Profiles

| Profile | Provider Preference | Fallback | Hard Blocks |
|---|---|---|---|
| `local-only` | Ollama only. | none | hosted providers. |
| `local-first` | Ollama first. | OpenRouter only if constraints permit. | hosted fallback for sensitive data unless explicitly allowed. |
| `remote-only` | OpenRouter only. | none | local providers. |
| `quality-first` | Highest quality viable role. | lower quality only if caller allows. | routes that violate privacy/cost. |
| `cheap-first` | Lowest cost viable role. | next cheapest viable route. | routes that violate role/capability needs. |
| `privacy-sensitive` | Local or explicitly approved private route. | none by default. | unapproved hosted providers. |

Profiles should be deterministic and previewable. If two routes tie, the
route preview must say what tie-breaker was used.

## Resource Alignment Recommendations

1. Add Ollama `extract.structured` as a resource role if it can be
   backed by a local model that reliably emits JSON.
2. Investigate OpenRouter embeddings in `resource-openrouter` before
   adding `embedding.default`; confirm endpoint support, response shape,
   dimensions, pricing, and policy command behavior.
3. Keep role names mostly shared across Ollama and OpenRouter. A role
   can be unsupported by one provider, but callers should not need
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

- Ollama has `embedding.default`; `extract.structured` is still a
  resource-policy follow-up.
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
