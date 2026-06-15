# Integrations — Quality Health

## Purpose Of This Document

This document names Quality Health's scenario dependencies, resource posture, and degradation behavior.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| Code Facts | scenario | required for confident audits | surfaces, audit | surface/language/framework/parse-unit facts | Return degraded audit; do not claim pass. |
| Test Genie | scenario | required for phase integration | audit provider | `quality` phase shells to Quality Health | Phase reports unavailable provider or mapped findings. |
| Tidiness Manager | scenario context | no runtime dependency after cutover | migration only | old type-safety rules are superseded | No runtime call from Quality Health v1. |
| Scenario Auditor | scenario context | no runtime dependency after cutover | migration only | old external rule registration is superseded | No runtime call from Quality Health v1. |
| Vrooli lifecycle | platform | required | all surfaces | Makefile and `.vrooli/service.json` | Scenario should be managed through lifecycle commands. |
| SQLite | embedded storage | optional | future run history | `SQLITE_PATH` if used | Live audits can continue without history. |

## Vrooli Resources

No external Vrooli resources are required for the live-audit v1 path.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite | optional | Generated template provides local storage; Quality Health v1 can be stateless. | Add run history or latest finding lookup. |
| Qdrant/Ollama | not-applicable | Quality Health evaluates deterministic contracts; no embeddings needed. | Add semantic finding search only if explicitly planned. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| code-facts | required | Source of surface discovery and parse-unit evidence. | API/CLI client seam; degraded audit on failure. |
| test-genie | integration target | Owns orchestration, phase catalog, and report normalization. | Phase 4 provider cutover. |
| tidiness-manager | migration boundary | Existing hidden type-safety producer to retire. | Phase 5 cleanup; maintainability only afterward. |
| scenario-auditor | migration boundary | Existing external static-quality rule registration to retire. | Phase 5 cleanup; no duplicate findings. |

## Third-Party Services

None.

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| Code Facts unavailable | connection, timeout, or invalid response | `AuditQualityResponse.status = degraded` with `degraded_reason`; no clean pass. | Fake client tests. |
| Command runner timeout | timeout exceeded | command result status records timeout and finding evidence stays bounded. | Fake executor tests. |
| Autofix unsupported config | parser or shape unsupported | skip candidate with reason; no mutation in dry-run. | Temp-directory tests. |
| Test Genie provider unavailable | Quality Health cannot be resolved | Test Genie quality phase reports provider failure. | Phase 4 integration tests. |

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [DATA.md](DATA.md)
- [configuration.md](../reference/configuration.md)
- [DEPLOYMENT.md](../operations/DEPLOYMENT.md)
