# Integrations - AI Gateway

This document is the dependency contract for AI Gateway. It names what
the scenario may call, what remains owned elsewhere, and how failure
should degrade.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | health, routing evidence, operator settings | `SQLITE_PATH` lifecycle env var | Health reports unhealthy if unreachable; gateway can still preview from live resource state only if evidence persistence is disabled explicitly. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be managed through lifecycle commands. |
| `resource-ollama` | Vrooli resource | conditional | providers, inventory, routing, smoke tests | role/policy/gateway CLI commands | Local routes become unavailable; profiles that require local inference fail with actionable diagnostics. |
| `resource-openrouter` | Vrooli resource | conditional | providers, inventory, routing, smoke tests | role/policy/gateway CLI commands | Remote routes become unavailable; remote-only/hosted fallback profiles fail with actionable diagnostics. |
| `vrooli capacity` broker | local platform | optional | routing (local-route eligibility) | `vrooli capacity claim/release --json` CLI contract; footprint from explicit request metadata, never model-name inference | Consulted only when a request declares `metadata.required_vram_bytes`. If unavailable/errored, capacity degrades to `unknown_capacity` (advisory) and never blocks non-local routing; a held claim falls back to its bounded TTL on crash. |
| `test-genie` | scenario | yes for conformance phase | conformance | validation-provider phase contract | Conformance scans cannot run, but normal gateway operation may continue. |
| `experience-manager` | scenario | yes for docs/UI validation | operator experience | `experience-manager spec validate ai-gateway --json` | Experience validation is unavailable; runtime is unaffected. |
| `business-health` | scenario | yes for PRD/requirements validation | docs and maturity | business phase / requirements validator | Docs validation is unavailable; runtime is unaffected. |

At least one text provider must be healthy for execution. Both Ollama
and OpenRouter should be present for cross-provider role comparison and
profile testing.

## Resource Contracts

### Ollama

AI Gateway should use Ollama through resource commands such as:

- `resource-ollama policy roles`
- `resource-ollama policy models`
- `resource-ollama policy resolve --role <role>`
- `resource-ollama gateway generate --role <role>`
- `resource-ollama gateway chat --role <role>`
- `resource-ollama gateway embed --role <role>`

Ollama remains responsible for local model installation, capacity
planning, concrete role-to-model policy, context-window knowledge, and
embedding dimensions.

Phase 3 inventory uses `resource-ollama policy roles --json` as the
bounded read/smoke command. The gateway normalizes role metadata from
that output and never reads `resources/ollama/model-policy.json`
directly. Ollama already exposes `embedding.default`; adding or
admitting a future local `extract.structured` role remains resource
policy work.

Phase 4 routing execution uses `resource-ollama gateway generate --role
<role> --json --prompt-stdin` for text generation/structured extraction
and `resource-ollama gateway embed --role <role> --json --input-stdin`
for embeddings. Transient input is passed on stdin so prompt text does
not appear in command arguments, route evidence, or command strings.

### OpenRouter

AI Gateway should use OpenRouter through resource commands such as:

- `resource-openrouter policy roles`
- `resource-openrouter policy models`
- `resource-openrouter policy resolve --role <role>`
- `resource-openrouter generate/chat` or the current approved resource
  gateway equivalent

OpenRouter remains responsible for hosted API credentials, concrete
hosted model policy, endpoint capability checks, and provider-specific
request shape. Adding `embedding.default` requires investigation in the
OpenRouter resource first; AI Gateway should display support once the
resource exposes it.

Phase 3 inventory uses `resource-openrouter policy roles --json` as the
bounded read/smoke command. OpenRouter already exposes
`extract.structured`; `embedding.default` is still absent from the
resource policy and remains a resource-owned investigation before AI
Gateway can route it.

Phase 4 routing execution uses `resource-openrouter generate --role
<role> --json` with transient prompt input on stdin for text
generation/structured extraction. AI Gateway does not pass concrete
model slugs or provider credentials; OpenRouter keeps that authority in
its resource policy/configuration.

## Forbidden Integration Paths

The following are invalid unless a future approved exception explicitly
documents the reason:

- direct Ollama HTTP calls
- direct OpenRouter HTTP calls
- provider credentials in AI Gateway config
- `OPENROUTER_API_KEY`, `OLLAMA_URL`, or equivalent env vars in caller
  scenarios
- model slug allowlists in AI Gateway
- embedding dimension constants treated as provider truth

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| test-genie | planned required | Runs the AI conformance phase. | Provider phase returns structured findings, severity, maturity, and fix guidance. |
| ai-model-orchestra-controller | superseded candidate | Historical routing/orchestration scenario. | Do not depend on it; use as migration reference only. |
| ai-chatbot-manager | adjacent, not replacement | Product/chat interface management. | It should call AI Gateway for AI work once the gateway contract exists. |
| image-tools | adjacent | Owns local/remote image model orchestration. | Coordinate cross-modal requests; do not absorb image provider logic. |
| audio-tools | adjacent | Owns local/remote audio model orchestration. | Coordinate cross-modal requests; do not absorb audio provider logic. |
| browser-automation-studio | adopted | Multimodal browser navigation. | Call `InferenceService.Run` with `extract.structured`, ordered turns, screenshot attachments, and `local_first`/`remote_only` profiles. BAS retains browser action execution; provider credentials, model selection, and usage remain gateway/resource-owned. |

Browser Automation Studio is the reference caller for multimodal inference.
It must not import resource provider clients, read provider secret environment
variables, select concrete model slugs, or maintain price tables. Its isolated
Claude computer-use exception has a dated replacement trigger in the BAS
decision log.

## Failure Modes

| Failure | Expected Behavior | Severity |
|---|---|---|
| No provider can satisfy a request | Return a typed route failure with rejected candidates and remediation. | request error |
| Local provider unavailable under `local-only` | Fail closed; do not fall back remotely. | policy error |
| Hosted provider unavailable under `local-first` | Continue local route if viable; record hosted fallback unavailable. | degraded |
| Resource policy command malformed | Mark provider inventory stale/unreadable and block execution through that provider. | provider error |
| Provider role missing | Show role gap and recommended resource policy action. | configuration error |
| Evidence persistence unavailable | Execution should fail closed until retention policy defines a no-persist mode. | operational error |
| Resource CLI binary missing | Inventory and smoke status return `missing_binary`; routes through that provider are unavailable. | provider error |
| Resource CLI times out | Inventory and smoke status return `timeout`; caller can retry or inspect the resource lifecycle. | degraded/provider error |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`DATA.md`](DATA.md)
- [`../reference/configuration.md`](../reference/configuration.md)
- [`../reference/roles-profiles-policies.md`](../reference/roles-profiles-policies.md)
- [`../reference/validation-provider.md`](../reference/validation-provider.md)
