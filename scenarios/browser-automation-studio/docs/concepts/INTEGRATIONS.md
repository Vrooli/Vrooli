# Integrations


List external and Vrooli-managed dependencies at the ownership boundary.

## Dependency Inventory

Dependency changes require Scenario Dependency Analyzer approval. Proto generation is repository infrastructure, not a runtime service.

## Vrooli Resources

The service manifest declares these resources. SQLite remains the default
relational state store; MinIO owns binary artifacts.

| Resource | Required? | Startup policy | Purpose | Unavailable behavior |
|---|---|---|---|---|
| `minio` | yes | `must_start` | Screenshot and artifact storage through the S3-compatible API. | Startup is blocked; artifact-backed capture cannot run without storage. |
| `ollama` | no | `try_start` | Optional local AI workflow generation and debugging assistance. | AI-assisted authoring is unavailable; browser capture and execution continue. |
| `openrouter` | yes | `must_start` | Declared hosted-provider resource, auto-detected through the resource CLI. | Startup is blocked if the required resource cannot start; this does not establish model or credential readiness. |

## Scenario Dependencies

The API supervises the in-repo Playwright driver. UI and CLI consume API
contracts rather than connecting directly to browser processes. The rows below
mirror `.vrooli/service.json`; required dependencies and optional fallback
peers have different startup and degradation behavior.

| Scenario | Required? | Startup policy | Purpose | Unavailable behavior |
|---|---|---|---|---|
| `experience-manager` | yes | `must_start` | Import-level readiness-profile contract. | Readiness-profile resolution is unavailable; capture and execution use bounded defaults. |
| `landing-page-business-suite` | yes | `must_start` | Authoritative monetization and subscription provider. | LPBS-backed Class-A entitlement and shared-session operations report unavailable; local Class-B work remains available under a valid cached lease. |
| `ai-gateway` | no | `try_start` | Managed vision calls. | Vision calls fail closed while the gateway is unavailable. |
| `agent-manager` | no | `try_start` | Optional delegated-agent workflow provider. | Browser navigation, authoring, and replay remain available; delegated supervision and telemetry are unavailable. |
| `vrooli-memory` | no | `try_start` (`bundle_policy: either`) | Scoped learning recall and capture. | Programs run without advice; captures queue for later delivery. |
| `workflow-health` | no | `try_start` (`bundle_policy: either`) | Workflow search. | `find-flows` falls back to BAS workflows/list. |
| `search-hub` | no | `try_start` (`bundle_policy: either`) | Federated workflow candidates. | `find-flows` returns local workflows only. |

## Third-Party Services

`ollama` is an optional AI resource; `openrouter` is declared required for
startup. Model and credential readiness remain separate capability checks.
Provider availability must not make core workflow execution unsafe or
nondeterministic.

## Failure Modes

Unavailable driver, storage, or AI integrations must surface actionable health/diagnostic evidence. A driver failure does not justify accepting untyped instructions.

## Cross-References

- [Configuration](../reference/configuration.md)
- [Architecture](ARCHITECTURE.md)
- [Operations](../operations/RUNBOOK.md)
