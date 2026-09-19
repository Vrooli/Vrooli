# AI Conformance Validation Provider

AI Gateway should become a test-genie validation provider for scenarios
that use AI inference. The phase should start advisory at lower
maturities and become stricter as the gateway contract matures.

## Purpose

The phase answers the core AI conformance questions:

1. Is the scenario using AI inference?
2. Is it respecting resource boundaries?
3. Is it avoiding concrete provider/model/storage coupling?
4. Has it adopted AI Gateway where a gateway contract exists?

## Maturity Ladder

| Level | Name | Expected Checks | Failure Behavior |
|---|---|---|---|
| L0 | Inventory | Detect AI-adjacent env vars, direct provider URLs, resource commands, model slugs, embedding dimensions, vector-store references, and gateway calls. | Advisory report only. |
| L1 | Resource-boundary hygiene | Flag direct Ollama/OpenRouter HTTP usage, provider secrets in scenario env/config, `OLLAMA_URL`, `OPENROUTER_API_KEY`, and unmanaged provider endpoints. | Fail only when the selected suite requires boundary hygiene. |
| L2 | Role/policy conformance | Require role-based resource gateway usage when a scenario talks to a resource directly; flag concrete model slugs outside resource policy files or approved fixtures. | Configurable fail threshold. |
| L3 | Embedding data correctness | Check vector stores and migrations for embedding role/model/dimension/content metadata and retarget/migration strategy. | Fail for unsafe embedding stores in higher maturity suites. |
| L4 | Gateway adoption | Require generic text AI calls to use AI Gateway profiles/contracts when the operation is covered; allow direct resource calls only for approved low-level exceptions. | Fail unless exception is approved. |
| L5 | Operational assurance | Require route evidence, smoke-test health, privacy/budget policy declarations, and clean migration/adoption reports. | Fail for production-ready suites. |

## Finding Taxonomy

| Finding Code | Meaning | Typical Severity | Example Fix |
|---|---|---|---|
| `ai.direct_ollama_http` | Scenario calls Ollama HTTP directly. | high | Route through AI Gateway or `resource-ollama gateway` if approved. |
| `ai.direct_openrouter_http` | Scenario calls OpenRouter HTTP directly. | high | Route through AI Gateway or `resource-openrouter`. |
| `ai.invalid_provider_secret_env` | Scenario declares provider secrets such as `OPENROUTER_API_KEY`. | high | Move secret handling to the resource. |
| `ai.invalid_provider_url_env` | Scenario declares provider URLs such as `OLLAMA_URL`. | medium/high | Use resource discovery/gateway commands. |
| `ai.concrete_model_slug` | Scenario hard-codes model names outside resource policy. | medium/high | Replace with role/profile request. |
| `ai.hardcoded_context_window` | Scenario hard-codes context limits as provider truth. | medium | Query policy/capacity metadata or declare caller constraint. |
| `ai.hardcoded_embedding_dimensions` | Scenario hard-codes embedding length without role/model metadata. | high for vector stores | Store role/model/dimensions and migration strategy. |
| `ai.embedding_metadata_missing` | Vector store lacks embedding role/model/version metadata. | high | Add metadata columns/collection config before changing models. |
| `ai.resource_gateway_missing_role` | Scenario uses a role the resource cannot resolve. | medium | Add/align resource role policy. |
| `ai.gateway_not_adopted` | Scenario uses lower-level AI calls where gateway contract exists. | advisory to high | Migrate to AI Gateway profile request or record exception. |
| `ai.unreviewed_exception` | Direct resource/provider behavior lacks an approved exception. | medium | Add documented exception or migrate. |

## Scanner Inputs

The phase should inspect:

- scenario source files
- `.vrooli/service.json`
- `.vrooli/testing.json` and test-genie config
- environment/config docs
- requirements and PRD references
- database schemas and migrations
- vector-store setup files
- resource command usage
- imports and HTTP client construction

It should avoid storing full source contents in findings. Findings need
path, line/range when available, matched pattern class, maturity level,
severity, and fix guidance.

## Approved Exception Model

Some direct resource usage may be legitimate:

- a resource scenario implementing its own provider boundary
- a diagnostic/smoke-test fixture with no production path
- a migration bridge with an expiry date
- a scenario whose AI behavior is not yet covered by AI Gateway

Exceptions should include owner, reason, source path/pattern, maturity
level waived, expiry/review date, and replacement plan.

## Embedding Checks

Embedding-backed scenarios should pass these checks before higher
maturity:

- vector store schema records embedding role
- vector store schema records provider/model identity or resource
  resolution evidence
- vector dimensions are stored with the index/collection
- corpus/content version is recorded
- migrations can distinguish old and new embedding spaces
- retargeting requires explicit rebuild or dual-index strategy

The phase should not assume one correct embedding dimension globally.
It should check whether the scenario can prove which dimensions its
stored vectors use.

## Adoption Guidance

The phase should output a migration recommendation for each scenario:

- `no-ai-detected`
- `resource-boundary-clean`
- `resource-role-clean-but-not-gateway`
- `gateway-ready`
- `gateway-adopted`
- `blocked-needs-investigation`

This lets lower-maturity suites gather evidence without forcing every
scenario to adopt AI Gateway immediately.

## Implementation Status

Phase 5 now exposes AI Gateway as the Test Genie `ai-conformance`
validation provider. The native `ConformanceService.ScanScenario` RPC
and the shared
`scenario-validation/v1.ScenarioValidationService.ValidateScenario`
contract run the same deterministic scanner, with shared maturity
mapping loaded from `scenarios/ai-gateway/.vrooli/test-genie.json`.

The scanner currently detects direct Ollama/OpenRouter HTTP coupling,
provider secret/url env vars, concrete model slugs, hard-coded context
windows, hard-coded embedding dimensions near vector code, missing
embedding metadata, direct resource command usage without visible role
policy, unreviewed exception markers, and missing AI Gateway adoption
signals. Findings include rule ID, severity, path/line, message, and
remediation, but not matched source content.

`PreviewFix` and `ApplyFix` are intentionally guidance/no-op surfaces
until safe deterministic source migrations exist.

## Implementation Notes

Start with fixture-based scanner tests before wiring test-genie. The
scanner should be deterministic and conservative: it should explain
uncertainty rather than silently passing ambiguous AI integration code.
