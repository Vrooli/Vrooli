# AI Provider Resource Contract

AI API resources (OpenRouter, OrcaRouter, abliteration, Ollama, Ollaya) share
one contract so AI Gateway, scenario clients, and coding harnesses can treat
them uniformly. The implementation kit is
[`packages/ai-provider-go`](../../packages/ai-provider-go/README.md).

## Ownership

| Layer | Owner | Notes |
|---|---|---|
| Endpoint, credentials, health, exports | `resource.json` | The resource manifest is the only endpoint/credential authority. |
| Provider family and capabilities | `model-policy.json` `provider` block | Declares dialect and behavior, not endpoint or secret values. |
| Role/model catalog | `model-policy.json` | Concrete slugs live only here, with provenance. |
| Routing across providers | AI Gateway | Never reads provider credentials or edits provider policy. |
| Coding-role judgment | Harness resource (`opencode`, `codex`, `claude-code`) | Coding roles are not provider roles. |
| Credential values | Credential authority | Injected into the resource process only. |

## Required CLI surface

Every inference provider resource must expose:

```text
policy roles --json
policy resolve --role <role> --json
policy models --json
policy constraints --json
ensure --config-base64 <json>
generate --role <role> [--json]
list-models --json
show-config [--json]
```

`ensure` is role-only. Concrete `model` / `models` fields in a scenario
dependency are rejected with a message that names the field.

## Provider families

| Family | Wire shape | Resources |
|---|---|---|
| `openai-compatible` | OpenAI Chat Completions, `/models`, images | OpenRouter, OrcaRouter, abliteration |
| `ollama` | Ollama generate/embed gateway | Ollama |
| `decision` | System One typed decisions | Ollaya |
| `pii-detector` | Local typed PII detection | PII Detector |
| `metered` | Vrooli subscription inference | LPBS |

AI Gateway switches on the family, not the provider id. Adding another
OpenAI-compatible provider is a registry entry plus its resource.

## Policy shape

`model-policy.json` carries:

- `provider`: `id`, `family`, and provider-level capabilities.
- `roles`: intent, concrete model + fallbacks, endpoint, required capabilities,
  bounded request defaults, declared sampling support, and provenance.
- `models`: provider, family, capabilities, modalities, endpoints or local
  resource estimates, pricing, and provenance.
- `constraints`: capability/modality/endpoint vocabulary, role order, and
  provenance source kinds.

Rules:

1. A role may resolve only a model that serves its endpoint and declares every
   required capability.
2. A fallback is admitted under the same rule as the primary.
3. `sampling_support` is declared, never probed. `honored`, `ignored`,
   `rejected`, and `unknown` are the only values.
4. Every role and model carries provenance with a source kind, confidence,
   source, and observation date.
5. Provider-specific behavior (OpenRouter spend caps, abliteration research
   posture) stays in the resource or its policy, not in the shared client.

## Provider terms record and route eligibility

The shared `ai-provider-go/policy` package now defines and validates the terms record, model overrides, verified approval boundary, server-derived feature use class, and common eligibility function. The provider terms and route catalog remain inputs to that package; the durable rule-record store and Notification Hub adapter are still integration work. Callers must pass a verifier backed by Notification Hub decisions. The verifier hook is not itself an approval authority.

A supplier terms record is keyed by supplier and contains:

- `provider.id`, `provider.family`, `provider.endpoints`, and `provider.terms`.
- `terms.plan_rules[]`: the supplier plan, access tier, per-use-class permission, required conditions, and the cited clause/quote. Each permission is supported by its basis, clause and quote, evidence URL, effective/expiry dates, and reviewer.
- `terms.evidence`: source URL, document date, read date, and content digest for the captured terms evidence.
- `terms.review`: `status` and `decision_ref`; newly proposed records use `pending_review` and have no decision reference. Only an operator's recorded decision can approve a record.
- `terms.effective_from`, `next_review_at`, `expires_at`, and `on_expiry` (which must deny when expired).
- `terms.downstream`: applicable data-collection and pinned-host conditions.

A model may add a `terms_override` with narrower use-class permissions, a reason, and evidence. A server-only supplier agreement may separately record scoped permissions, approval references and expiry, and the trust decision for customer code. Model overrides can narrow supplier terms; only an applicable, active agreement can widen a permission for the models and use classes it names.

The provider-rules design assigns the schema and `eligible()` to the shared AI provider policy package. Skills and governed onboarding programs may discover evidence and propose records; the operator alone approves them. Review state is `proposed`/`pending_review` until an operator decision; an approved record can be reopened when its evidence changes, expire by its dates, or be denied. No code or agent may approve a record, set trust, or set price. Unapproved, not-yet-effective, review-due, reopened, denied, or expired records produce no eligible route; an in-flight reservation may finish after expiry. `NewVerifiedRuleRecord` and `NewVerifiedAgreement` require a Notification Hub-backed verifier before approval or trust can affect eligibility.

Scenario feature contracts are declared in `.vrooli/monetization.json` under `ai_features[]`. `packages/monetization-go` validates and generates `ai-features.generated.json`; run `go run ./packages/monetization-go/cmd/ai-feature-registry` to regenerate it and add `--check` to detect drift. The registry contains limits and schemas, never prompt text. Runtime callers must validate the invocation against that contract before constructing the policy request.

`eligible(request, account, records, now)` is a server-only shared decision. It derives the use class from the server-verified feature contract and caller, never from request input. The caller resolves the feature role to its endpoint and required capabilities before contract checking. It returns eligible routes with quotes and refusal reasons. It denies unknown features, contract violations, missing rules, permissions other than `allowed` without a matching active agreement, endpoint/capability/model/licence/harness denials, unapproved or inactive records, missing customer-code trust for `harness_access`, and data/privacy/budget mismatches. The model picker displays this result; execution recomputes it and may call only a returned route. With zero approved records, it returns zero routes.

The canonical use classes are `own_account`, `bounded_feature`, `in_app`, and `harness_access`. For a coding-agent session, the harness is a binding target, not a provider. The user selects a source at launch; the launcher resolves that choice into per-session arguments, environment, and files without editing the user's global harness configuration. A credit-funded session receives a short-lived, scoped key and routes through LPBS to AI Gateway. Do not expose an account key to the harness on the user's behalf. See [Coding agents](../configuration/coding-agents.md) for the harness lifecycle and [Paid Features](../concepts/PAID_FEATURES.md) for billing authority.

## Failure contract

Resource CLIs emit a single `VROOLI_PROVIDER_ERROR {json}` line on stderr for
typed provider failures. The JSON carries `code`, `http_status`,
`retry_after`, and `message`. The shared vocabulary lives in
`packages/ai-provider-go/providererror`.

## Adding a provider

1. Add `resources/<name>/` as a `cloud-api` resource with `resource.json`.
2. Add `model-policy.json` with a `provider` block and role/model catalog.
3. Make the CLI a thin call to `resources/internal/aiproviderapp.New`.
4. Add a conformance test using `packages/ai-provider-go/conformance`.
5. Register the provider in AI Gateway's `providers.DefaultProviderSpecs` and
   add candidates to the roles it should serve.
6. Add credentials and optional dependencies to the consumers.
7. If OpenCode should use it, declare the provider block and auth sync in
   `resources/opencode`; never hard-code a new tier in its selection order.

## Conformance

Run:

```bash
cd packages/ai-provider-go && GOWORK=off go test ./...
cd resources && go test ./<provider>/... ./openrouter/...
vrooli resource validate <provider>
```
