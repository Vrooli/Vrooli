# Architecture - AI Gateway

AI Gateway is the policy boundary between scenario callers and AI
providers. It is not a model host, provider SDK, chat application, or
second resource policy file. It turns caller intent into a provider
route, executes through approved resources, and emits evidence that can
be inspected by operators and test-genie.

## Architectural Position

```text
Scenario caller
  - role: chat.default, summarize.default, extract.structured, embedding.default
  - profile: local-first, local-only, remote-only, quality-first, cheap-first, privacy-sensitive
  - constraints: privacy, budget, latency, context, structured-output, embedding metadata
        |
        v
AI Gateway
  - validates request contract
  - resolves profile and route policy
  - previews or executes route
  - records route evidence
  - reports conformance findings
        |
        v
Resource gateway commands
  - resource-ollama policy/gateway commands
  - resource-openrouter policy/gateway commands
        |
        v
Concrete provider implementation
  - local Ollama models
  - hosted OpenRouter models
```

The core invariant is simple: callers should bind to AI work, not to a
provider implementation. Resources own concrete model catalogs,
credentials, installation, and provider-specific API behavior. AI Gateway
owns cross-provider routing, caller-facing profiles, evidence, and
conformance checks.

## Owned Domains

| Domain | Responsibility | P0 Targets |
|---|---|---|
| `gateway` | Public request contract for text and embedding operations. | OT-P0-001 |
| `providers` | Resource command adapters and normalized provider outcomes. | OT-P0-002 |
| `inventory` | Provider role/policy/model/smoke-test inventory. | OT-P0-003 |
| `routing` | Profile resolution, route preview, fallback decisions, and route evidence. | OT-P0-004 |
| `conformance` | Test-genie validation provider and migration findings. | OT-P0-005 |
| `operator` | UI/CLI/API surfaces for inspection and operation. | OT-P0-006 |

`health` remains a real scaffold domain for lifecycle readiness. The
generated example domain is not product scope and has been removed with
`vrooli scenario detemplate ai-gateway` before implementing the first
real product slice.

## Surface Shape

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Gateway business logic and integration boundary. | Contract validation, routing, provider adapters, evidence, conformance scans. | Browser-only state, CLI formatting, concrete model catalogs. |
| CLI (`cli/`) | Scriptable operator and agent interface. | JSON commands for inventory, preview, smoke tests, execution, and scans. | Business rules duplicated from API. |
| UI (`ui/`) | Operator control surface. | Provider status, route preview, traces, conformance findings, profile inspection. | Chat-product UX, provider credentials, hidden routing. |
| Experience (`experience/`) | Declarative UI contract. | Route/page intent, priorities, and later machine-checkable claims. | Runtime implementation. |
| Requirements (`requirements/`) | Test target registry. | `[REQ:...]` anchors for future tests. | Product behavior by itself. |

## Request Contract

The gateway request contract must be provider-neutral:

```json
{
  "operation": "generate | chat | summarize | classify | extract | embed",
  "role": "chat.default",
  "profile": "local-first",
  "constraints": {
    "privacy": "normal | sensitive",
    "maxCostUsd": 0.05,
    "maxLatencyMs": 30000,
    "structuredOutput": true,
    "embeddingSchema": {
      "store": "qdrant",
      "existingDimensions": 768
    }
  },
  "input": {}
}
```

Forbidden caller fields include concrete provider model slugs, provider
base URLs, provider API keys, and provider-specific endpoint names. A
debug-only route override may exist later, but it must be explicit,
audited, and unavailable to normal scenario callers.

## Routing Profiles

AI Gateway owns profiles because profiles are cross-provider policy,
not concrete model selection:

| Profile | Meaning | Typical Use |
|---|---|---|
| `local-only` | Refuse hosted providers. | Private data, offline/local development, deterministic local policy. |
| `local-first` | Prefer Ollama, fall back only when policy permits. | Default Vrooli local development. |
| `remote-only` | Refuse local providers. | Hosted deployments without local inference. |
| `quality-first` | Prefer strongest available provider role within constraints. | Complex reasoning, high-value extraction. |
| `cheap-first` | Prefer lowest-cost viable provider path. | Bulk summarization/classification. |
| `privacy-sensitive` | Permit only routes that satisfy declared privacy constraints. | Secrets, user data, regulated content. |

Resources still own the final role-to-model mapping. For example,
`chat.default` may resolve differently in Ollama and OpenRouter, and
AI Gateway should explain which resource role was selected and why.

## Provider Boundary

Allowed:

- `resource-ollama policy roles/models/resolve`
- `resource-ollama gateway generate/chat/embed`
- `resource-openrouter policy roles/models/resolve`
- `resource-openrouter generate/chat/images` or equivalent approved
  resource gateway commands

Forbidden inside AI Gateway:

- `http://localhost:11434`
- `api.openrouter.ai`
- `OLLAMA_URL`, `OPENROUTER_API_KEY`, or provider-specific secrets
- hard-coded model names such as `llama3.2`, `codellama`, or hosted
  provider slugs
- hard-coded embedding dimensions or context windows as routing truth

If a resource lacks a command needed by AI Gateway, the fix belongs in
the resource first. AI Gateway should not bypass the resource to keep
moving.

## Route Evidence

Every preview and execution should produce structured evidence:

- caller scenario and operation
- role, profile, constraints, and request class
- candidate providers and rejection reasons
- selected provider, resource command, resource role, and resolved
  concrete model when the resource reports it
- fallback path, if any
- latency, token/size metadata when available, and error classification
- redaction state and privacy policy outcome

Evidence is for operations and testing. It must not store prompts,
responses, or sensitive content unless a future retention policy
explicitly permits that.

## Test-Genie Phase

The conformance provider is a first-class product surface. Lower
maturities inventory and warn on AI integration drift. Higher maturities
can require gateway adoption where the gateway provides an equivalent
contract. See [`../reference/validation-provider.md`](../reference/validation-provider.md).

## Extension Rules

1. Add or update the owning domain in `docs/concepts/DOMAINS.md`.
2. Define the contract in requirements before implementation.
3. Add provider behavior only through resource command adapters.
4. Add route/profile changes with preview tests before execution tests.
5. Add UI pages to `experience/` before building the page.
6. Update conformance findings when a new invalid pattern or approved
   exception class is discovered.

## Intentional Non-Goals

- Replacing `resource-ollama` or `resource-openrouter`.
- Becoming the catalog authority for concrete model slugs.
- Providing end-user chat widgets; that belongs in chatbot/product
  scenarios.
- Owning image/audio model orchestration; coordinate with image-tools
  and audio-tools instead.
- Retiring `ai-model-orchestra-controller` during documentation
  inception. Retirement should happen after AI Gateway proves the
  intended replacement contract.
