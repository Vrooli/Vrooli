# Meta-Orchestrator Summary

## Source

Follow-up from `execute/deployment-manager-lpbs-desktop-release-orchestration` completion. During review of end-state operator experience, we identified that DM's release pipeline gives diagnostic text but not actionable remediation. This item is the foundation enabling preflight and skill-guided remediation.

## Codebase Inspection Already Done

- `scenarios/deployment-manager/api/deployments/orchestrator.go:72-78` — OrchestrationStep has `Name`, `Status`, `Duration`, `Message`, `Error` (all strings). No structured error code.
- `scenarios/deployment-manager/api/deployments/orchestrator.go:812-827` — startStep/successStep/failStep helpers operate on free-text Message/Error.
- `scenarios/deployment-manager/api/deployments/orchestrator_release.go` — deployCheckCloudHealth/deployCheckLPBSReadiness/deployVerifyUpdateEndpoints emit string messages via failStep. No codes.
- `scenarios/deployment-manager/api/deployments/lpbs_release_client.go:38-49` — LPBSReadinessResult has `Ready`, `Gates []ReadinessGate`, `Error string`. ReadinessGate has `Name`, `Ready`, `Message`. No remediation hint or command template.
- `scenarios/landing-page-business-suite/api/deploy_readiness_handler.go` — DeployReadinessGate matches the DM side. Remediation hints would be added symmetrically on both sides.
- `scenarios/deployment-manager/api/releases/handlers.go` — Start returns free-text error messages via shared.JSONError for most failures.

## Decisions Made

- Scope: typed error shape across every failure mode in the orchestrator.
- Both DM-originated and LPBS-originated gates need the treatment.
- CLI human rendering surfaces remediation hints inline.

## Unresolved Questions Deferred To Workshop

- **OrchestrationStep shape**: add parallel `Code string` and `RemediationCommand string` fields, vs. replace `Error string` with a structured object, vs. introduce a new `ErrorDetail` sub-struct. Backward compat: existing consumers of `Error` string should keep working.
- **Error code naming**: enumerated list in the description is a starting point. Workshop should finalize. Consider whether to split `build_failed` into subcodes (fpm_missing, compile_failed, etc.) or keep it coarse and rely on message text.
- **Remediation command templates**: should they contain placeholders (`<profile-id>`, `<sha>`) that the caller substitutes, or be fully-formed with substitutions done server-side? Server-side is more helpful but requires DM to know the operator's context.
- **LPBS side remediation**: each readiness gate needs a matching remediation. download_storage → landing-page-business-suite admin-download-storage-set ..., app_registered → landing-page-business-suite admin-download-apps-create ..., remote_profile_registered → landing-page-business-suite remote-profiles-create .... Confirm exact command shapes in workshop.

## Dependency Notes

Depends on `execute/deployment-manager-lpbs-desktop-release-orchestration` for the orchestrator step structure it extends. Blocks the preflight endpoint and skill-rewrite items.

## Greenfield Constraint

Typed errors replace free-text messages at emit sites. Do not maintain a parallel "legacy string path". Existing consumers (UI, tests) should migrate to the typed field. Backward compat only at the JSON-wire level (existing `error` and `message` fields keep working alongside new `code` and `remediation_*` fields).
