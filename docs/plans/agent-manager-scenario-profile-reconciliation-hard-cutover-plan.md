# Agent-Manager Scenario Profile Reconciliation Hard-Cutover Plan

## Purpose

Move scenario-owned agent-manager profile definitions out of scenario Go code and into scenario-owned data files, while keeping agent-manager as the runtime registry, validator, reconciler, and executor.

This plan is a greenfield hard cutover: after implementation, scenarios that depend on `agent-manager` must not build their default `AgentProfile` structs in Go. They declare profile source files in their own `.vrooli/service.json` dependency config and call a scenario-level reconciliation endpoint on startup.

## Required Reading

Future agents should start with:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health seam-discovery-and-enforcement api-steer interoperability-steer
prompt-manager skill read cli-steer utils-unification
```

Relevant code and contract files:

- `packages/repo-contract-go/paths.go`
- `packages/repo-contract-go/resolve.go`
- `packages/repo-contract-go/load.go`
- `.vrooli/schemas/service.schema.json`
- `internal/scenario/scenario.go`
- `packages/proto/schemas/agent-manager/v1/domain/profile.proto`
- `packages/proto/schemas/agent-manager/v1/api/service.proto`
- `scenarios/agent-manager/api/internal/orchestration/service.go`
- `scenarios/agent-manager/api/internal/database/schema.sql`
- `scenarios/swarm-manager/api/internal/agentmanager/profile.go`
- `scenarios/system-monitor/api/internal/agentmanager/service.go`
- `scenarios/scenario-to-desktop/api/agentmanager/service.go`
- `scenarios/app-issue-tracker/api/internal/agentmanager/service.go`

## Hard-Cutover Rule

Do not introduce a dual source of truth.

After this work lands:

- Scenario-owned profile defaults live in scenario-local profile files.
- Scenario startup calls agent-manager scenario reconciliation.
- Scenario Go code no longer builds default `AgentProfile` documents.
- `/api/v1/profiles/ensure` may remain as a low-level/manual primitive, but scenario integrations must not use it for scenario-owned defaults.
- No compatibility bridge should keep old scenario profile builders active.

## Problem Statement

Current dependent scenarios define agent-manager profiles in Go, then call `/api/v1/profiles/ensure`. This has one good property: UI edits in agent-manager persist because `ensure` does not necessarily overwrite an existing profile.

The weak point is permanent change management:

- A profile edited in the UI is not git-tracked.
- To make that change permanent, an operator must also update the scenario's Go defaults.
- Multiple scenarios duplicate small agent-manager client/profile-builder code.
- Drift between DB profile rows and code-defined defaults is easy to miss.
- The term `ensure` no longer matches the desired model when a scenario owns a list of profiles.

The target model is reconciliation: a scenario declares profile source files, agent-manager reads and validates those files, and the DB stores runtime rows plus source/drift metadata.

## Current Technical Context

### Repo-Contract Helpers

`repo-contract-go` already has the path helpers needed for this plan:

- `ResolveScenarioPath(repoRoot, scenario)` resolves a contract-aware scenario root.
- `ResolveScenarioFile(repoRoot, scenario, "service")` resolves a scenario service manifest.
- `Contract.ScenarioRoot(repoRoot, scenario)` validates the scenario identifier through `cleanIdentifier`.
- `cleanIdentifier` rejects empty identifiers and rejects values containing `..`, `/`, or `\`.

Relevant evidence:

- `packages/repo-contract-go/resolve.go`
- `packages/repo-contract-go/paths.go`
- `packages/repo-contract-go/load.go`
- `packages/repo-contract-go/paths_test.go`

There is one caveat: the standalone fallback helper `ScenarioRoot(repoRoot, scenario) string` falls back to `filepath.Join(repoRoot, "scenarios", filepath.Clean(scenario))` when the contract cannot load. New agent-manager reconciliation code must not use that fallback helper for security-sensitive or repo-authoritative resolution. It must use the error-returning `ResolveScenarioPath` / `ResolveScenarioFile` / `Contract` methods.

### Dependency Config Seam

`internal/scenario.Dependency` already captures unmodeled keys into `Dependency.Config`, and `MarshalJSON` round-trips them. The comment currently says this is resource-specific, but the implementation is generic for resources and scenarios.

However, `.vrooli/schemas/service.schema.json` currently defines `scenarioDependency.additionalProperties: false`, so schema validation rejects arbitrary scenario dependency extension fields today.

Hard-cutover decision: add one generic `config` object to both resource and scenario dependency schema/types, and stop relying on arbitrary top-level extension keys for new work.

### Agent-Manager Profile API

Agent-manager already has:

- `AgentProfile` proto/domain model.
- `CreateProfile`, `EnsureProfile`, `GetProfile`, `ListProfiles`, `UpdateProfile`, `DeleteProfile`.
- `agent_profiles` DB table.
- `DefaultRunConfig()` and `DefaultSandboxConfig()`.
- `resolveSandboxConfig()` which backfills partial sandbox config with contract defaults.

The new scenario reconciliation endpoint should reuse the existing profile validation and normalization path, but should not expose "ensure one profile" as the scenario-owned abstraction.

## Target End State

Each scenario that needs agent-manager profiles owns data files:

```text
scenarios/<scenario>/
  .vrooli/
    service.json
    agent-profiles/
      default.json
      investigation.json
      apply.json
```

The scenario manifest declares those files under the dependency-specific config for `agent-manager`:

```json
{
  "dependencies": {
    "scenarios": {
      "agent-manager": {
        "required": true,
        "startup_policy": "must_start",
        "description": "Spawning agents for automated work",
        "config": {
          "profiles": {
            "reconcile": true,
            "mode": "update_if_unmodified",
            "sources": [
              ".vrooli/agent-profiles/default.json"
            ]
          }
        }
      }
    }
  }
}
```

At startup, scenario code does one thing:

```text
scenario.Name()
POST /api/v1/profiles/reconcile-scenario { "scenario": "<name>" }
```

Agent-manager:

1. Resolves repo root using `repo-contract-go`.
2. Resolves the caller scenario service file via `ResolveScenarioFile(root, scenario, "service")`.
3. Reads `dependencies.scenarios.agent-manager.config.profiles`.
4. Resolves profile source paths relative to the scenario root.
5. Rejects path traversal, absolute paths, symlink escapes, missing files, and profile keys not owned by the scenario.
6. Parses each source file into `domainpb.AgentProfile` using proto JSON.
7. Converts to domain, validates, normalizes, and reconciles DB rows.
8. Returns a structured report of created, updated, unchanged, locally modified, skipped, and errored profiles.

## Contract Decisions

### Dependency Config Shape

Use an explicit generic `config` object:

```json
"config": {
  "profiles": {
    "reconcile": true,
    "mode": "update_if_unmodified",
    "sources": ["..."]
  }
}
```

Do not add agent-manager-specific top-level fields to `service.schema.json`.

`mode` values:

- `create_only`: create missing runtime rows, never update existing rows.
- `update_if_unmodified`: update existing rows only when the DB row has not been locally modified since the last source apply. This is the default.
- `force`: source file wins even if the DB row was locally modified.

`reconcile: false` disables automatic startup reconciliation even if sources are listed.

### Profile File Format

Use protobuf JSON for `agent_manager.v1.AgentProfile`, lowerCamelCase to match current API JSON behavior.

Example:

```json
{
  "profileKey": "swarm-manager/default",
  "name": "Swarm Manager Default",
  "description": "Agent profile for swarm-manager research and execution",
  "runnerType": "RUNNER_TYPE_CLAUDE_CODE",
  "modelPreset": "MODEL_PRESET_SMART",
  "maxTurns": 75,
  "timeout": "3600s",
  "allowedTools": ["Read", "Write", "Edit", "Glob", "Grep", "Bash"],
  "sandboxConfig": {
    "mode": "SANDBOX_MODE_PROTECTED"
  }
}
```

Profile source files must not set:

- `id`
- `createdAt`
- `updatedAt`

Agent-manager fills runtime metadata.

### Ownership Rule

Every scenario-owned `profileKey` must be namespaced:

```text
<scenario>/<profile-name>
```

Examples:

- `swarm-manager/default`
- `system-monitor/investigator`
- `scenario-to-desktop/pipeline-investigator`
- `app-issue-tracker/investigations`

This prevents accidental cross-scenario profile overwrite and makes ownership queryable.

### API Contract

Add proto messages and endpoint:

```proto
rpc ReconcileScenarioProfiles(ReconcileScenarioProfilesRequest)
  returns (ReconcileScenarioProfilesResponse) {
  option (google.api.http) = {
    post: "/api/v1/profiles/reconcile-scenario"
    body: "*"
  };
}

message ReconcileScenarioProfilesRequest {
  string scenario = 1;
  bool dry_run = 2;
}

message ReconcileScenarioProfilesResponse {
  string scenario = 1;
  repeated ProfileReconcileResult results = 2;
  int32 created = 3;
  int32 updated = 4;
  int32 unchanged = 5;
  int32 skipped = 6;
  int32 conflicted = 7;
  int32 failed = 8;
}

message ProfileReconcileResult {
  string profile_key = 1;
  string source_path = 2;
  string source_hash = 3;
  string profile_id = 4;
  ProfileReconcileStatus status = 5;
  string message = 6;
}
```

Status enum should include `CREATED`, `UPDATED`, `UNCHANGED`, `SKIPPED`, `CONFLICTED_LOCAL_OVERRIDE`, and `FAILED_VALIDATION`.

### DB Metadata

Extend `agent_profiles` with source tracking:

- `owner_scenario TEXT`
- `source_path TEXT`
- `source_hash TEXT`
- `last_applied_hash TEXT`
- `source_updated_at TEXT`
- `local_override INTEGER DEFAULT 0`

Hard-cutover behavior:

- API/UI edits to a sourced profile set `local_override=true` unless the edit is an explicit "write source" operation.
- Reconciliation with `update_if_unmodified` refuses to overwrite `local_override=true`.
- Reconciliation with `force` overwrites and clears `local_override`.
- Profiles missing from current source lists should be marked orphaned/archived only in a later explicit cleanup flow, not deleted during startup reconciliation.

If the existing DB layer does not have migrations, add idempotent `ALTER TABLE ... ADD COLUMN` startup migration logic near the schema management seam.

## Implementation Strategy

### Phase 1: Manifest Extension Seam

1. Update `.vrooli/schemas/service.schema.json`:
   - Add `config` as an object to `scenarioDependency`.
   - Add the same to resource dependency if the shared `Dependency` type is intended to stay symmetric.
   - Keep `additionalProperties: false`.

2. Update `internal/scenario.Dependency`:
   - Add `Config json.RawMessage 'json:"config,omitempty"'`.
   - Stop treating unknown top-level keys as the preferred extension path.
   - For hard cutover, reject unknown top-level keys after existing manifests are corrected, or preserve only temporarily in tests if the repo still has legacy unknown keys.
   - Update comments from "resource-specific" to "dependency-specific".

3. Add tests:
   - `scenarioDependency.config` validates through schema.
   - config round-trips through `ReadService` / `json.Marshal`.
   - unknown top-level scenario dependency keys fail under hard-cutover rules.

### Phase 2: Agent-Manager Reconciliation Domain

Add a package under agent-manager, for example:

```text
scenarios/agent-manager/api/internal/profilereconcile/
```

Responsibilities:

- Resolve repo root via `repo-contract-go.ResolveRepoRoot()` or injected root for tests.
- Resolve service path via `repo-contract-go.ResolveScenarioFile(root, scenario, "service")`.
- Resolve scenario root via `repo-contract-go.ResolveScenarioPath(root, scenario)`.
- Parse the scenario service manifest enough to extract `dependencies.scenarios.agent-manager.config.profiles`.
- Resolve profile source paths relative to scenario root.
- Enforce path containment after `filepath.EvalSymlinks`.
- Parse profile source JSON with `protojson.UnmarshalOptions{DiscardUnknown:false}` into `domainpb.AgentProfile`.
- Convert with existing `protoconv.AgentProfileFromProto`.
- Validate with existing domain validation.
- Enforce profile ownership namespace.
- Compute stable source hash from canonical file bytes.
- Call orchestrator/repository reconciliation methods.

Keep this as a seam with injected filesystem/repo resolver for tests.

### Phase 3: Agent-Manager API

1. Add proto API messages and enum in `packages/proto/schemas/agent-manager/v1/api/service.proto`.
2. Run proto generation:

```bash
cd packages/proto
make generate
make lint
make breaking
make check
```

3. Add handler route in `scenarios/agent-manager/api/internal/handlers/handlers.go`.
4. Add orchestrator service method:

```go
ReconcileScenarioProfiles(ctx context.Context, req ReconcileScenarioProfilesRequest) (*ReconcileScenarioProfilesResult, error)
```

5. Return structured validation errors; do not collapse per-profile failures into an opaque 500.

### Phase 4: DB/Repository Support

1. Extend `agent_profiles` schema.
2. Extend domain `AgentProfile` with source metadata fields.
3. Extend repository scan/insert/update code.
4. Add repository methods needed by reconciliation:
   - get by `profile_key`
   - update sourced profile if unmodified
   - mark local override on normal UI/API update

Do not make source files replace the DB. Runs should continue to reference stable `agent_profile_id` values.

### Phase 5: CLI/UI Surfaces

CLI:

- Add `agent-manager profile reconcile-scenario <scenario> [--dry-run]`.
- Add `agent-manager profile list` columns or JSON fields for `owner_scenario`, `source_path`, `local_override`.

UI:

- Display source metadata on profile detail.
- Display local override/conflict state.
- Add actions:
  - "Revert to source"
  - "Force reconcile from source"
  - "Clear local override" only if profile equals source

Do not add "write back to source file" in this cutover unless explicitly scoped. It is useful, but it requires file-edit permissions and a separate safety design.

### Phase 6: Scenario Hard Cutover

Convert all current scenario-owned profile defaults:

- `swarm-manager`
- `system-monitor`
- `scenario-to-desktop`
- `app-issue-tracker`

For each:

1. Add `.vrooli/agent-profiles/*.json`.
2. Add `dependencies.scenarios.agent-manager.config.profiles`.
3. Replace startup `EnsureProfile` logic with `ReconcileScenarioProfiles(scenario.Name())`.
4. Replace `ProfileRef.Defaults` usage in run creation with `ProfileRef{ProfileKey: "<scenario>/<profile>"}` only, or use `agent_profile_id` from the reconcile response if the scenario needs exact IDs.
5. Remove local `ProfileConfig`, `DefaultProfileConfig`, and `buildProfile` code unless still needed for user-requested inline run overrides.
6. Keep per-run inline overrides for dynamic values like runner/model/manual review, but do not use inline config to restate default profile data.

### Phase 7: Documentation

Update:

- `docs/configuration/scenarios.md`: dependency `config` convention and agent-manager profile source pattern.
- `scenarios/agent-manager/docs/reference/configuration.md`: profile reconciliation config.
- `scenarios/agent-manager/docs/internal/SEAMS.md`: profile source/reconcile/DB/runtime boundary.
- `scenarios/agent-manager/docs/internal/PROBLEMS.md`: any deferred write-back/orphan-cleanup work.
- `scenarios/agent-manager/docs/manifest.json` if new docs are added.

Add `// DOC:` comments to the reconciliation service and handler.

## Testing Plan

### Repo-Contract

Run:

```bash
cd packages/repo-contract-go
go test ./...
```

Add tests only if new helper behavior is needed. Current evidence already shows scenario identifiers reject `..`, `/`, and `\`.

### Scenario Manifest

Run:

```bash
go test ./internal/scenario
```

Test:

- dependency `config` object is accepted.
- config round-trips.
- malformed config remains isolated from generic dependency parsing.
- unknown top-level dependency keys are rejected if hard-cutover strictness is implemented in Go.

### Proto

Run:

```bash
cd packages/proto
make generate
make lint
make breaking
make check
```

### Agent-Manager Unit Tests

Run:

```bash
cd scenarios/agent-manager/api
GOWORK=off go test ./...
```

Focused tests:

- valid scenario manifest reconciles multiple profiles.
- missing `agent-manager` dependency returns typed validation error.
- `reconcile=false` produces skipped results.
- source path traversal is rejected.
- absolute source path is rejected.
- symlink escape is rejected.
- profile key outside `<scenario>/...` is rejected.
- proto JSON unknown field is rejected.
- source file with `id`, `createdAt`, or `updatedAt` is rejected.
- `create_only`, `update_if_unmodified`, and `force` behave correctly.
- UI/API profile update marks local override.

### Scenario Integration Tests

For converted scenarios, run targeted Go tests after each conversion. Then run scenario tests where available:

```bash
vrooli scenario test agent-manager
vrooli scenario test swarm-manager
vrooli scenario test system-monitor
vrooli scenario test scenario-to-desktop
vrooli scenario test app-issue-tracker
```

Use longer timeouts for full suites.

## Rollout / Validation Checklist

- [ ] `service.schema.json` supports generic dependency `config`.
- [ ] `internal/scenario` preserves and validates dependency config intentionally.
- [ ] Agent-manager has `ReconcileScenarioProfiles` proto/API/CLI.
- [ ] Reconciliation uses repo-contract-go error-returning helpers, not string path assumptions.
- [ ] Profile source paths cannot escape scenario root.
- [ ] Profile source files validate through generated protobuf JSON and domain validation.
- [ ] DB stores source metadata and local override state.
- [ ] All dependent scenario Go profile builders are removed.
- [ ] Converted scenario manifests declare profile source files.
- [ ] Existing scenario profile keys are renamed/namespaced under owner scenario.
- [ ] Docs explain source-vs-runtime-vs-local-override.
- [ ] Scenario tests pass.

## Risks And Mitigations

### Risk: caller-provided scenario slug is spoofable

Mitigation: treat this as local repo-aware reconciliation, not authentication. Any trusted local scenario can request reconciliation for any scenario. Agent-manager must still enforce repo-contract scenario resolution, source path containment, and profile key ownership.

### Risk: local UI edits are overwritten

Mitigation: default mode is `update_if_unmodified`; source changes do not overwrite `local_override=true` rows unless manifest mode is `force` or operator explicitly forces reconciliation.

### Risk: schema drift between profile files and proto

Mitigation: profile source files are parsed with generated proto JSON, not a hand-maintained JSON schema. Optional editor schemas may be generated later, but protobuf/domain validation is authoritative.

### Risk: source paths become arbitrary file reads

Mitigation: profile sources are scenario-root-relative, cleaned, symlink-evaluated, and containment-checked before read.

### Risk: hard cutover breaks scenario startup

Mitigation: convert all known dependents in one PR; add tests for each adapter; fail startup with actionable errors when reconciliation fails for required profiles.

## Non-Goals / Prohibited Patterns

- Do not add agent-manager-specific top-level fields to the generic service schema.
- Do not add profile-specific logic to `api-core`.
- Do not keep scenario-local Go default profile builders active after cutover.
- Do not let agent-manager scan all scenarios for profiles at startup.
- Do not use `repo-contract-go.ScenarioRoot(repoRoot, scenario) string` fallback in reconciliation.
- Do not allow absolute profile source paths or `..` path traversal.
- Do not silently overwrite local UI changes.
- Do not delete orphaned DB profiles during startup reconciliation.

## Definition Of Done

The work is complete when a fresh checkout can start a scenario that depends on agent-manager, the scenario calls `ReconcileScenarioProfiles` with its `api-core/scenario.Name()`, agent-manager reads the scenario's own `.vrooli/service.json`, validates and reconciles every declared profile source file, and subsequent runs use those reconciled profiles without any scenario-owned default `AgentProfile` Go builders.

All tests listed in the testing plan must pass, and the updated docs must clearly explain the source profile file, runtime DB row, and local override distinction.
