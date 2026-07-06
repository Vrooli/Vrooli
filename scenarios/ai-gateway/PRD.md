# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: AI Gateway is the policy-driven inference boundary for Vrooli scenarios. It lets scenarios request AI work by intent, role, profile, and constraints instead of binding themselves to concrete providers, direct Ollama/OpenRouter APIs, hard-coded model names, embedding dimensions, or provider secrets.
- **Primary users/verticals**: Scenario authors, coding agents, operators, test-genie phase authors, and any scenario that needs text inference, structured extraction, summarization, classification, code assistance, routing decisions, or embedding governance.
- **Deployment surfaces**: Go API, scenario CLI, React operator UI, test-genie validation-provider phase, generated policy/report artifacts, and scenario-to-scenario automation contracts.
- **Value promise**: Scenarios gain a stable AI capability contract while local and hosted providers can evolve independently. Operators can inspect, test, and govern routing decisions from one place, and test-genie can identify AI integration debt before it becomes permanent platform drift.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Gateway request contract | Expose a stable request contract for chat, generation, summarization, classification, structured extraction, and embeddings that accepts role/profile/constraints rather than concrete model slugs.
- [ ] OT-P0-002 | Resource-backed provider execution | Route every provider call through approved resource gateway commands such as `resource-ollama gateway ...` and `resource-openrouter ...`, with no direct provider HTTP clients, localhost assumptions, or provider secret handling in this scenario.
- [ ] OT-P0-003 | Role and profile inventory | Discover, normalize, and display provider roles, policies, available models, constraints, and smoke-test status for Ollama and OpenRouter without making AI Gateway the source of truth for provider model catalogs.
- [ ] OT-P0-004 | Route preview and evidence | Explain why a request would use a provider/model/role path before execution and record auditable route evidence after execution.
- [ ] OT-P0-005 | AI conformance validation provider | Provide a test-genie phase that reports AI integration violations across scenarios, including direct provider access, invalid provider env vars, concrete model coupling, embedding metadata risks, and missing gateway adoption at higher maturities.
- [ ] OT-P0-006 | Professional operator surface | Replace the generated placeholder UI with a dense, accessible operator experience for provider status, route preview, request traces, conformance findings, and profile management.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Embedding governance and retarget planning | Help scenarios declare embedding role, dimensions, content hash/version metadata, vector-store schema expectations, and migration/retarget risk when embedding models change.
- [ ] OT-P1-002 | Cross-resource role alignment | Add policy-management workflows that surface gaps between Ollama and OpenRouter roles, including `extract.structured` on Ollama and an investigated `embedding.default` path on OpenRouter if the provider contract supports it.
- [ ] OT-P1-003 | Migration assistance | Produce scenario-by-scenario adoption reports and fix guidance for moving from direct Ollama/OpenRouter/resource calls to AI Gateway profiles.
- [ ] OT-P1-004 | Budget, privacy, and locality constraints | Enforce operator-controlled routing policies such as local-only, local-first, remote-only, quality-first, cheap-first, privacy-sensitive, and max-cost ceilings.
- [ ] OT-P1-005 | SDK and caller ergonomics | Provide small typed client helpers for common scenario callers while keeping the wire/API contract provider-neutral.

### 🟢 P2 – Future / expansion
- [x] OT-P2-001 | Adaptive routing intelligence | Learn from quality, latency, cost, failure, and operator feedback to improve routing decisions without hiding policy reasons.
- [x] OT-P2-002 | Fleet enforcement | Allow higher-maturity suites to fail scenarios that bypass AI Gateway where a gateway contract exists and no approved exception is recorded.
- [ ] OT-P2-003 | Multi-provider expansion | Add additional text/embedding providers behind the same role/profile contract after Ollama and OpenRouter are stable.
- [ ] OT-P2-004 | Related-modal coordination | Coordinate with image-tools and audio-tools for cross-modal workflows while leaving image/audio provider execution under those specialized scenarios.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API and CLI, React + Vite UI with the `vrooli-default` design kit, JSON policy artifacts, generated protobuf contracts where scenario-to-scenario APIs require typed boundaries.
- Data + storage expectations: SQLite from the template is acceptable for early local route evidence and operator settings. If route history becomes shared/fleet-scale state, migrate to the repository-standard Postgres pattern with explicit retention and privacy controls.
- Integration strategy: scenario callers use AI Gateway contracts; AI Gateway uses resource CLIs and their role policies; resources own provider credentials, model installation, concrete model catalogs, and provider-specific gateway behavior.
- Non-goals / guardrails: Do not make AI Gateway a second model catalog source of truth. Do not store provider API keys. Do not call Ollama/OpenRouter HTTP APIs directly. Do not hard-code model slugs, context windows, embedding dimensions, or provider-specific endpoint URLs in caller scenarios. Do not absorb image/audio provider orchestration that already belongs to image-tools/audio-tools.

## 🤝 Dependencies & Launch Plan
- Required resources: `resource-ollama` for local roles, `resource-openrouter` for hosted roles, and the project control plane for scenario/test-genie integration. At least one text provider must be healthy for execution; both are required for cross-provider policy comparison.
- Scenario dependencies: `test-genie` for phase execution, `experience-manager` for UI contract validation, `business-health` for PRD/requirements validation, `scenario-dependency-analyzer` for dependency governance, and existing resource scenarios for actual provider calls.
- Operational risks: provider policy drift, stale resource README/JSON mismatches, accidental reintroduction of direct provider calls, embedding schema migrations after model changes, hidden provider secrets in scenario env/config, and ambiguous exceptions for scenarios that legitimately need lower-level resource operations.
- Launch sequencing: complete documentation and requirements first; implement gateway request/preview/provider inventory; add conformance phase in advisory mode; pilot migration with a small set of AI-using scenarios; then deprecate `ai-model-orchestra-controller` once the gateway covers its intended role.

## 🎨 UX & Branding
- Look & feel: Quiet, operational, and information-dense. The UI should feel like a control surface for repeated engineering decisions, not a marketing page or chat demo.
- Accessibility: Keyboard-first navigation, clear focus states, semantic tables/lists for inventories and findings, WCAG AA contrast, reduced-motion support, and responsive layouts that preserve dense comparison views without overlapping content.
- Voice & messaging: Direct and diagnostic. Say what route will be used, why, what failed, and what the operator can change.
- Branding hooks: Use Vrooli default design tokens and replace generated app icons when product branding is finalized. Provider names should be shown as capabilities and health states, not as brand-driven primary navigation.

## 📎 Appendix
- Resource role policies currently live in `resources/ollama/model-policy.json` and `resources/openrouter/model-policy.json`.
- Ollama already exposes role-oriented gateway and policy commands; its README and JSON policy should be checked for drift as part of P1 role alignment.
- OpenRouter already has text/image roles and should be investigated before adding `embedding.default`, because embedding endpoint support must be validated against the resource implementation and provider contract.
- `ai-model-orchestra-controller` appears to be the stale predecessor for routing/model orchestration and should be superseded after AI Gateway proves the durable contract.
