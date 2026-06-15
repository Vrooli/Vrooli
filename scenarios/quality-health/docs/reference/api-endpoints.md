# API Endpoints — Quality Health

## Target API Shape

Quality Health should expose proto/Connect services unless a generated REST exception is explicitly documented.

```text
QualityHealthService
  AuditQuality(AuditQualityRequest) returns (AuditQualityResponse)
  ListContracts(ListContractsRequest) returns (ListContractsResponse)
  ExplainFinding(ExplainFindingRequest) returns (ExplainFindingResponse)
  PreviewFixConfig(FixConfigRequest) returns (FixConfigResponse)
  ApplyFixConfig(FixConfigRequest) returns (FixConfigResponse)
```

## AuditQuality

Request fields:

- `target.kind`: `scenario`, `path`, or `project`
- `target.scenario`
- `target.path`
- `target.repo_root`
- `surfaces[]`
- `rule_ids[]`
- `include_command_execution`
- `include_autofix_preview`
- `use_cache`

Response fields:

- `run_id`
- `scenario`
- `status`: `passed`, `failed`, `degraded`, or `error`
- `started_at`
- `finished_at`
- `surfaces[]`
- `contracts[]`
- `findings[]`
- `command_results[]`
- `maturity`
- `summary`
- `next_steps[]`
- `degraded_reason`

## ListContracts

Lists contract definitions filtered by language, framework, surface kind, tooling, or rule ID. This endpoint powers CLI discoverability and UI contract detail views.

## ExplainFinding

Returns remediation for a stable finding ID. v1 may require the caller to pass `scenario` so the API can re-audit or locate the latest finding context.

## PreviewFixConfig

Returns deterministic config edits without changing files. Preview is the default mutation posture.

## ApplyFixConfig

Applies only supported config edits and returns the applied diff/result. Source-code suppression edits are out of scope for v1.

## Existing Generated Endpoints

The generated health endpoint remains valid lifecycle infrastructure. The generated sample CRUD endpoints were removed during Phase 1.
