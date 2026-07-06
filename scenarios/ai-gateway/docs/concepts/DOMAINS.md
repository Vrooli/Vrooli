# Domains - AI Gateway

This is the canonical product capability map for AI Gateway. Code,
tests, requirements, and UI surfaces should use these domain names
unless a later decision splits or merges them.

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Source Paths |
|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Keep lifecycle and dependency status visible. | No product data. | reporting | `api/handlers/health/`, `ui/src/features/health/` |
| gateway | Validate provider-neutral AI request contracts. | Give caller scenarios a stable AI work API. | No durable data by itself. | service | planned `api/internal/gateway/`, `api/handlers/gateway/`, `cli/domains/gateway/` |
| providers | Adapt resource gateway commands and normalize provider outcomes. | Preserve resource ownership while making provider execution consistent. | No durable data; emits execution evidence. | adapter | planned `api/internal/providers/` |
| inventory | Read and compare provider role/policy/model/smoke state. | Let operators see what roles exist and whether they work. | Provider snapshots, smoke results. | query | planned `api/internal/inventory/`, `ui/src/features/inventory/` |
| routing | Resolve profiles, preview routes, execute/fallback, and record evidence. | Make AI routing explainable and auditable. | Profiles, route evidence. | workflow/service | planned `api/internal/routing/`, `ui/src/features/routing/` |
| conformance | Scan scenarios for AI integration maturity and violations. | Let test-genie enforce correct AI usage over time. | Scan runs and findings. | validation-provider | planned `api/internal/conformance/`, `cli/domains/conformance/` |
| operator | Compose UI/CLI/API surfaces for human and agent operation. | Make gateway state and actions accessible. | Operator preferences only. | presentation/orchestration | planned `ui/src/pages/`, `ui/src/features/operator/` |

## Domain Details

### gateway

- Owns: request validation, operation taxonomy, role/profile/constraint
  normalization, provider-neutral errors.
- Does not own: concrete model catalogs, provider credentials, route
  selection algorithms, provider command execution.
- Requirements: `AIGW-CONTRACT-*`.
- First implementation slice: request contract types and rejection of
  concrete model/provider fields.

### providers

- Owns: subprocess/resource-command adapters, timeout/cancellation
  wrappers, normalized provider result/error shape.
- Does not own: direct HTTP clients to Ollama/OpenRouter or provider
  credentials.
- Requirements: `AIGW-PROVIDER-*`.
- First implementation slice: fakeable command-runner seam and Ollama
  policy/gateway adapter.

### inventory

- Owns: role/policy/model inventory snapshots, role smoke-test status,
  drift indicators, cross-provider role comparison.
- Does not own: editing the resources' policy files directly.
- Requirements: `AIGW-INVENTORY-*`, `AIGW-ROLE-ALIGNMENT`.
- First implementation slice: read and normalize resource role policies
  with fake resource outputs.

### routing

- Owns: profile semantics, candidate ranking, route preview, fallback
  rules, route evidence persistence.
- Does not own: the final resource role-to-model mapping.
- Requirements: `AIGW-ROUTE-*`, `AIGW-POLICY-CONSTRAINTS`.
- First implementation slice: deterministic route preview for local-only,
  local-first, and remote-only.

### conformance

- Owns: scanner rules, finding taxonomy, maturity ladder, exception
  model, test-genie phase output, migration reports.
- Does not own: direct changes to target scenarios.
- Requirements: `AIGW-CONFORMANCE-*`, `AIGW-MIGRATION-REPORTS`.
- First implementation slice: fixture scanner for direct provider URLs,
  provider secrets, concrete model slugs, resource gateway usage, and
  gateway adoption evidence.

### operator

- Owns: user-facing composition of provider inventory, route preview,
  traces, conformance findings, filters, and CLI report rendering.
- Does not own: business rules duplicated from API domains.
- Requirements: `AIGW-UI-DASHBOARD`, `AIGW-CLI-OPERATIONS`.
- First implementation slice: dashboard route with provider and route
  preview panels backed by mocked API contracts.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Role | Provider capability label such as `chat.default` or `embedding.default`. | Resource policies define concrete mappings; AI Gateway validates use. |
| Profile | Cross-provider routing policy such as `local-first`. | AI Gateway routing domain. |
| Constraint | Request-specific limits or requirements. | Gateway contract; routing interprets them. |
| Route evidence | Structured explanation of preview/execution decisions. | Routing domain. |
| Finding | Conformance issue with severity, maturity level, source, and fix guidance. | Conformance domain. |
| Exception | Approved reason a finding does not fail a maturity threshold. | Conformance domain. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `sdk` | Useful caller helpers, but API contract should stabilize first. | P1 caller ergonomics work begins. |
| `budgeting` | Cost ceilings belong in routing initially. | Hosted usage volume requires separate policy/audit tables. |
| `feedback` | Adaptive routing needs operational history and quality signals. | P2 adaptive routing starts. |

## Non-Domains

- The generated `notes` example is template scaffolding.
- Resource policy files are owned by resources, not AI Gateway.
- Provider credentials are owned by resources.
- Image/audio model orchestration belongs to image-tools/audio-tools.
- End-user chatbot interface management belongs to chatbot/product
  scenarios, not AI Gateway.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`FLOWS.md`](FLOWS.md)
- [`DATA.md`](DATA.md)
- [`INTEGRATIONS.md`](INTEGRATIONS.md)
- [`../reference/roles-profiles-policies.md`](../reference/roles-profiles-policies.md)
- [`../reference/validation-provider.md`](../reference/validation-provider.md)
