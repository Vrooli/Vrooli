# Operation Contract Drift Prevention Plan

Status: draft for workshop refinement.

## Purpose

Create the first version of a shared operation-contract spine that prevents drift between scenario APIs, scenario CLIs, generated command catalogs, and prompt-manager Actions.

This plan exists because the Action entity work exposed a deeper platform problem: prompt-manager can only safely expose executable Actions if the commands behind those Actions are real, owned, typed, permission-classified, and still aligned with the APIs or lifecycle operations they wrap. Solving that only inside Actions would add another drift surface. The goal here is broader and more valuable: make Vrooli operations reliable across API, CLI, UI, agents, and future Action discovery by giving them one contract-backed source of truth.

This is intentionally a first-version foundation plan, not the final end-state for every scenario in Vrooli. It should be workshopped and strengthened before implementation. The first implementation should establish the contract model, prove it on prompt-manager and root Vrooli commands, and create tests that make future drift visible. Later plans can expand adoption across more scenarios.

## Why This Matters

Treat this as platform reliability work, not only as a prompt-manager Action prerequisite. Actions make the drift problem more visible because they need to expose executable commands as searchable capabilities, but the root issue already exists anywhere a scenario API, scenario CLI, UI flow, agent instruction, or command catalog can describe the same operation differently.

The desired outcome is one practical ground truth for operation shape, validation, effects, permissions, and structured output. When a scenario API changes, the corresponding CLI binding and command catalog should either update from the same contract or fail a targeted check that identifies the drift. When a CLI command is local-only and has no API operation, it should still publish a typed command contract instead of relying on prose or first-token ownership.

Future workshop rounds should strengthen this plan against that bar:

- the plan should prevent new Action drift without ignoring existing API/CLI drift
- the plan should reduce duplicated schemas, duplicated request structs, and hand-maintained parity tables over time
- the plan should make incomplete coverage explicit through certainty levels and drift checks
- the first implementation can be partial, but it should leave a clear expansion path for other scenarios
- the design should keep CLIs ergonomic instead of forcing every API route to become a one-to-one command

## Required Reading

Run these before implementing any phase:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health api-steer cli-steer interoperability-steer boundary-of-responsibility-enforcement seam-discovery-and-enforcement utils-unification test assumption-mapping-and-hardening
```

Also read the relevant source and docs:

```bash
sed -n '1,220p' packages/proto/README.md
sed -n '1,160p' packages/proto/STYLE_GUIDE.md
sed -n '1,140p' internal/cli/commandtree/commandtree.go
sed -n '1,140p' internal/cli/commandtree/args.go
sed -n '1,220p' internal/cli/topcli/topcli.go
sed -n '1,180p' internal/cli/scenariocli/commands.go
sed -n '1,180p' internal/cli/resourcecli/commands.go
sed -n '1,120p' packages/cli-core/cliapp/app.go
sed -n '1,180p' packages/cli-core/cliapp/resource_app.go
sed -n '1,120p' scenarios/prompt-manager/cli/PARITY_AUDIT.md
sed -n '1,220p' scenarios/prompt-manager/cli/parity/coverage.go
sed -n '1,180p' scenarios/prompt-manager/docs/plans/action-entity-implementation-plan.md
```

## Greenfield Constraint

This is greenfield contract infrastructure. Do not add compatibility shims, duplicate hand-maintained command catalogs, legacy aliases solely for old internal code, unused re-exports, placeholder metadata fields, or TODO-only implementations.

Existing public command names should not be renamed casually, but new contract plumbing should be clean. If a command is not yet contract-backed, mark its certainty accurately instead of pretending it is fully validated.

## Problem Statement

Prompt-manager Actions need to validate argv-shaped commands before exposing them as typed executable capabilities. A shallow allowlist such as "first token is `vrooli` or `prompt-manager`" is not enough. It proves neither that the command path exists nor that arguments, output mode, permissions, and mutability still match reality.

The same drift problem already exists before Actions:

- scenario CLIs are often thin wrappers over scenario APIs, but their request/response structs and flag parsing are commonly hand-written separately
- prompt-manager has CLI/API parity tracking, but the current guard mostly tracks route coverage, not typed request/response or permission parity
- root `vrooli` commands have structured command specs, but those specs are not exposed as a normalized command catalog
- `cli-core` command registrations contain useful metadata, but scenario subcommands can still hide behind manual string routers
- `packages/proto` is the canonical schema package, but prompt-manager does not yet use proto as the source of truth for its API and CLI contracts
- installed CLI manifests prove binary ownership, not subcommand correctness

If this is not addressed, Actions would add a new layer on top of an already drifting API/CLI boundary. If this is addressed well, the benefit is larger than Actions: API handlers, CLIs, UIs, agents, command catalogs, and future Actions can all validate against the same operation spine.

## Current Technical Context

Existing useful seams:

- [CODE: packages/proto/README.md] defines proto contracts, generated Go/TypeScript/Python outputs, protovalidate usage, and package governance.
- [CODE: internal/cli/commandtree/commandtree.go] defines structured root command specs with command names, aliases, argument schemas, groups, help, and root policies.
- [CODE: internal/cli/scenariocli/commands.go] and [CODE: internal/cli/resourcecli/commands.go] already define structured command specs for root `vrooli scenario ...` and `vrooli resource ...`.
- [CODE: packages/cli-core/cliapp/app.go] defines reusable scenario/resource CLI command registration, but its `Command` type does not yet include operation references, effects, permissions, output modes, or a machine-readable catalog export.
- [CODE: packages/cli-core/cliapp/resource_app.go] defines standard resource lifecycle commands, which can be cataloged quickly.
- [CODE: scenarios/prompt-manager/cli/parity/coverage.go] and [DOC: scenarios/prompt-manager/cli/PARITY_AUDIT.md] provide a route-to-CLI parity map, but not typed parity.
- [CODE: scenarios/prompt-manager/cli/skills/skills.go] shows the current manual subcommand-router pattern: one top-level `skill` command hides subcommands from `cli-core`.
- [CODE: scenarios/prompt-manager/api/skills/models.go] and similar API model files define API shapes locally rather than using generated proto types.
- [CODE: scenarios/prompt-manager/.vrooli/service.json] and [CODE: resources/*/resource.json] declare CLI binary ownership through manifest `cli.command` / `cli.invoke.command`.
- Installed sibling manifests under `~/.vrooli/bin/*.manifest.json` copy source manifests, proving installed binary ownership but not command-path contracts.

Current gap summary:

- binary ownership exists
- some root command specs exist
- proto contract machinery exists
- API/CLI parity coverage exists
- normalized operation-to-CLI-to-Action contract does not exist
- command catalog export does not exist
- typed drift detection across API, CLI, and command catalog does not exist

## Target Architecture

Use a layered operation contract spine:

```text
Operation contract
  -> API handler implementation
  -> CLI command binding
  -> generated/exported command catalog
  -> prompt-manager Action validation
```

For API-backed operations, the operation contract should be the source of truth. Prefer proto definitions in [CODE: packages/proto] for request/response shapes, validation rules, and HTTP binding when practical.

For non-API operations, such as root lifecycle commands, resource lifecycle commands, local `configure`, and operational diagnostics, the CLI command contract is the source of truth. These still need structured args, effects, permissions, output modes, and a command catalog entry.

The target is not "generate every CLI command from every API route." That would overfit and make CLIs awkward. The target is:

```text
API-backed CLI commands bind to typed operations.
Non-API CLI commands declare typed command contracts.
Both forms produce one normalized command catalog.
Actions validate only against that normalized catalog.
```

## Contract Decisions

### Operation IDs

Every contract-backed operation needs a stable operation ID.

Recommended shapes:

```text
prompt_manager.v1.SkillService.ReadSkills
prompt_manager.v1.ActionService.RunAction
vrooli.project.Status
vrooli.scenario.Status
vrooli.resource.Logs
```

API-backed operations should use proto service/method IDs when proto exists. Local lifecycle operations should use a Vrooli-owned namespace.

### Effects

Every operation or command must declare one effect:

```text
read
write
destructive
admin
```

Use `read` for pure inspection. Use `write` for state changes, filesystem writes, or queueing. Use `destructive` for delete, stop, uninstall, kill, archive, or irreversible changes. Use `admin` for configuration, host-level setup, permissions, or privileged maintenance.

### Permissions

Use explicit permissions as machine-readable strings. Initial set:

```text
api:read
api:write
filesystem:read
filesystem:write
process:start
process:stop
network:localhost
network:external
host:configure
secret:read
secret:write
```

This list can evolve, but do not leave permissions as prose only.

### Output Modes

Commands should declare output modes:

```json
[
  {
    "mode": "json",
    "trigger": "--json",
    "schemaRef": "prompt_manager.v1.ReadSkillsResponse"
  },
  {
    "mode": "human"
  }
]
```

Actions that declare structured output should require a command output mode with a schema reference or a documented JSON shape.

### Validation Certainty

Validation should return a certainty level:

```text
none          command is not recognized
owner-only    binary is Vrooli-owned, but command path is not cataloged
command       command path and args are cataloged
operation     command path maps to a typed operation contract
```

Active runnable Actions should require `operation` where the command is API-backed, and at least `command` for lifecycle/non-API commands. Draft Actions may be allowed to persist with `owner-only` certainty if clearly marked invalid for execution.

## Proposed Catalog Shape

The normalized command catalog should be generated from executable command registrations, not maintained separately by hand.

Example:

```json
{
  "schemaVersion": 1,
  "generatedAt": "2026-04-30T00:00:00Z",
  "owner": {
    "type": "scenario",
    "id": "prompt-manager"
  },
  "binary": "prompt-manager",
  "version": "2.0.0",
  "commands": [
    {
      "path": ["skill", "read"],
      "aliases": [["skills", "read"], ["s", "read"]],
      "summary": "Read skills",
      "args": {
        "positionals": [
          {
            "name": "identifier",
            "required": true,
            "repeatable": true
          }
        ],
        "options": [
          {
            "name": "--output",
            "valueName": "format"
          },
          {
            "name": "--json"
          }
        ]
      },
      "operationRef": "prompt_manager.v1.SkillService.ReadSkills",
      "needsApi": true,
      "effect": "read",
      "permissions": ["api:read"],
      "runSurfaces": ["cli", "api", "action"],
      "outputModes": [
        {
          "mode": "json",
          "trigger": "--json",
          "schemaRef": "prompt_manager.v1.ReadSkillsResponse"
        },
        {
          "mode": "human"
        }
      ]
    }
  ]
}
```

## Implementation Strategy

### Phase 1 - Shared Operation and Command Contract Models

Deliverables:

- Add a focused command contract package, likely under [CODE: packages/cli-core/cliapp] or a new [CODE: packages/cli-core/clicontract].
- Define Go structs for:
  - command catalog
  - command entry
  - argument schema
  - operation reference
  - effect
  - permissions
  - output modes
  - validation certainty
- Add JSON schema or documented JSON contract for catalog output.
- Add helper functions to normalize command paths and aliases.
- Add tests for JSON round-trip, stable sorting, alias normalization, and hash generation.

Implementation notes:

- Keep the model independent of prompt-manager Actions.
- Do not import scenario-specific packages into `cli-core`.
- Reuse existing `commandtree.ArgSchema` shape where practical, but do not force root `vrooli` and scenario `cliapp` to share implementation internals prematurely.

Acceptance:

- `cd packages/cli-core && go test ./...` passes.
- Catalog JSON is deterministic across runs for the same command registry.
- The package can represent root commands, scenario CLI commands, resource lifecycle commands, and prompt-manager subcommands.

### Phase 2 - Catalog Export for Root Vrooli Commands

Deliverables:

- Export root command specs from [CODE: internal/cli/topcli/topcli.go], [CODE: internal/cli/scenariocli/commands.go], and [CODE: internal/cli/resourcecli/commands.go] into the shared catalog shape.
- Add `vrooli commands --json` or an equivalent non-conflicting command.
- Include nested command paths such as:
  - `["scenario", "status"]`
  - `["scenario", "test"]`
  - `["resource", "status"]`
  - `["contract", "validate"]`
- Add effect and permission classifications for the root commands.

Implementation notes:

- Start conservative: classify unknown/high-risk commands as not Action-runnable.
- Root commands already have structured arg schemas, so this phase should not parse help text.
- Hidden/internal commands should either be omitted or marked `runSurfaces: []`.

Acceptance:

- `vrooli commands --json` returns valid catalog JSON.
- Tests prove every non-hidden top-level command has a catalog entry or an explicit omission reason.
- Tests prove command arg schemas in the catalog match `commandtree.Spec` definitions.

### Phase 3 - Catalog Export in `cli-core` for Scenario and Resource CLIs

Deliverables:

- Extend [CODE: packages/cli-core/cliapp.Command] and [CODE: packages/cli-core/cliapp.SubcommandGroup] with optional contract metadata:
  - `OperationRef`
  - `Args`
  - `Effect`
  - `Permissions`
  - `OutputModes`
  - `RunSurfaces`
- Add a standard `commands --json` or `__commands --json` command generated by `cli-core`.
- Add optional installation of a sibling `*.commands.json` artifact beside installed binaries, generated from the same runtime registry.
- Update [CODE: packages/cli-core/cliapp/resource_app.go] standard lifecycle commands to emit catalog entries.

Implementation notes:

- The catalog export must be generated from the same command registrations used by runtime dispatch.
- Do not create a separate manually written catalog file per scenario.
- Preserve existing human help behavior.

Acceptance:

- A simple scenario CLI using `cli-core` can emit a machine-readable catalog.
- Resource CLIs can emit lifecycle command catalog entries.
- `commands --json` does not require the scenario API to be running.

### Phase 4 - Prompt-Manager CLI Contract Refactor

Deliverables:

- Refactor prompt-manager CLI domains so meaningful subcommands become structured command registrations rather than opaque manual routers.
- Start with domains needed by Actions:
  - `discover`
  - future `action`
  - `skill read`
  - `team decisions-pending` / decision-list surfaces
  - search/status commands
- Add contract metadata to these commands.
- Keep route handlers thin and API-backed where appropriate.

Implementation notes:

- This does not require rewriting every prompt-manager CLI command in the first pass.
- Mark unmigrated commands with lower validation certainty rather than overclaiming.
- Prefer generated/shared API request/response types where already practical; add proto adoption in Phase 5 where it is not.

Acceptance:

- `prompt-manager commands --json` returns command entries for migrated commands.
- Migrated commands expose structured arg schemas and operation refs where API-backed.
- Existing command behavior and tests remain green.

### Phase 5 - API Operation Contracts for Prompt-Manager

Deliverables:

- Add proto definitions for the prompt-manager API operations needed by Actions and command validation, or establish a local interim operation descriptor if proto adoption is too large for the first pass.
- Prefer proto for:
  - Action API once implemented
  - Discover/search request/response shapes
  - Skill read/list/show shapes
  - Team decision-list/pending shapes if used by seed Actions
- Add generation and import wiring through [CODE: packages/proto].
- Add API ingress validation where protovalidate rules exist.
- Add CLI request marshalling from generated types for migrated commands.

Implementation notes:

- Do not attempt to proto-convert the entire prompt-manager API in one phase.
- Pick the operations that need Action-grade reliability first.
- Preserve the existing REST API surface while improving its typed source of truth.

Acceptance:

- `cd packages/proto && make check` passes after new proto files are generated.
- Migrated prompt-manager API handlers and CLI commands share generated request/response types or documented adapters.
- Drift tests fail if a CLI operation ref points to a missing proto operation.

### Phase 6 - Typed API/CLI Drift Detection

Deliverables:

- Strengthen [CODE: scenarios/prompt-manager/cli/parity] from route coverage to operation parity for migrated operations.
- Add checks that:
  - every CLI `operationRef` resolves to a known operation contract
  - every required operator-facing operation has a CLI command or explicit absence reason
  - CLI arg schemas can construct the declared API request shape
  - command output modes reference valid response schemas
  - command effect/permission metadata is present and conservative
- Keep the existing coverage map for unmigrated routes, but add a migration status so the remaining work is visible.

Implementation notes:

- Do not block the first pass on all 162 existing prompt-manager routes.
- Use a ratchet: newly migrated or newly added endpoints must be contract-backed.
- Document every intentional absence with a specific consumer and reason.

Acceptance:

- `cd scenarios/prompt-manager/cli && go test ./parity` catches missing operation refs, stale command paths, and missing response schema refs for migrated commands.
- Existing route coverage checks still run.
- The audit-pending queue is not hidden; it remains visible as follow-up work.

### Phase 7 - Controlled Command Resolver for Actions

Deliverables:

- Implement a reusable resolver that prompt-manager Actions can call.
- Resolver inputs:
  - argv
  - working directory / repo root
  - expected permissions from Action
  - desired run surface
- Resolver outputs:
  - owner type/id
  - matched binary
  - matched command path
  - operation ref when available
  - validation certainty
  - effect
  - permissions
  - output modes
  - runnable/not runnable decision
  - precise validation errors
- Validate against:
  - root `vrooli` catalog
  - installed scenario/resource CLI catalogs
  - source manifests for ownership fallback

Implementation notes:

- This resolver should live outside Action handler code so graph/search/testing can reuse it.
- It should reject shell forms before catalog lookup.
- It should never validate by trusted first token alone.

Acceptance:

- Tests prove raw `bash`, `python`, `node`, `git`, `docker`, `curl`, `psql`, `rg`, shell separators, multiline args, and path-bearing argv[0] are rejected.
- Tests prove `vrooli scenario status prompt-manager --json` resolves to a cataloged command.
- Tests prove an owner-known but uncataloged scenario CLI returns `owner-only` certainty and is not runnable as an active Action.

### Phase 8 - Action Plan Integration

Deliverables:

- Update [DOC: scenarios/prompt-manager/docs/plans/action-entity-implementation-plan.md] so runnable Action validation depends on this operation-contract foundation.
- Define Action status behavior:
  - `draft`: may persist with unresolved or `owner-only` command certainty
  - `active`: must have command/operation certainty sufficient for its command class
  - `runnable`: must pass permission, effect, run-surface, and output-mode validation
- Add Action validation checks that compare Action input/output declarations against command catalog and operation contract metadata.

Acceptance:

- Active runnable Actions cannot be created over owner-only commands.
- Action validation reports drift when the command catalog hash or operation schema changes incompatibly.
- The Action implementation does not duplicate command validation logic.

## Testing Plan

Core packages:

```bash
cd packages/cli-core && go test ./...
cd packages/proto && make check
```

Root Vrooli:

```bash
go test ./internal/cli/...
vrooli commands --json
vrooli scenario --help
vrooli resource --help
```

Prompt-manager:

```bash
cd scenarios/prompt-manager/cli
go test ./...
prompt-manager commands --json
go test ./parity
```

Scenario lifecycle:

```bash
cd scenarios/prompt-manager
make test
make restart
make status
```

## Rollout and Validation Checklist

- [ ] Shared command catalog model exists.
- [ ] Root `vrooli` commands export a catalog without help-text parsing.
- [ ] `cli-core` scenario/resource CLIs can export a catalog from runtime command registrations.
- [ ] Standard resource lifecycle commands include effect and permission metadata.
- [ ] Prompt-manager migrated commands expose structured subcommand entries.
- [ ] API-backed commands can reference stable operation IDs.
- [ ] Prompt-manager parity tests check operation refs for migrated commands.
- [ ] Controlled command resolver validates commands for Actions.
- [ ] Owner-only certainty is allowed for drafts but rejected for active runnable Actions.
- [ ] Drift is detected when command path, args, output mode, permissions, or operation schema changes.
- [ ] Documentation explains the difference between API operations, CLI commands, command catalogs, and Actions.

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Plan becomes too large and stalls | Treat this as a first-version foundation. Prove root Vrooli + prompt-manager Action-relevant commands first. |
| New command catalog becomes another hand-maintained drift source | Generate catalog from runtime command registrations; prohibit per-scenario manual catalog files. |
| API-to-CLI generation overfits and makes CLIs awkward | Bind CLI commands to operations, but do not require one-to-one generated CLI surfaces. |
| Proto adoption becomes too broad | Migrate only Action-relevant prompt-manager operations first; keep unmigrated routes in parity coverage. |
| Effects/permissions are underclassified | Default unknown commands to non-runnable and require explicit metadata for Action-run surfaces. |
| Installed catalog drifts from source | Generate sibling catalog artifacts during CLI install and expose runtime `commands --json`; compare hashes in tests. |
| Existing commands hide subcommands inside manual routers | Refactor only target domains first, then ratchet for new commands. |
| Actions bypass the resolver | Make Action validation and execution depend on the shared resolver interface. |

## Non-Goals

Do not:

- implement prompt-manager Actions in this plan
- rewrite every Vrooli scenario CLI in the first pass
- require every CLI command to be generated directly from an API route
- maintain separate hand-written command catalog JSON files per scenario
- parse human help text as the source of truth
- allow Action runtime validation to fall back to trusted first-token checks
- force proto adoption for all existing prompt-manager routes before any value ships

## Definition of Done

This first-version foundation is done when:

- root Vrooli commands can export a structured command catalog
- `cli-core` can export scenario/resource command catalogs from runtime command registrations
- command entries include operation refs where API-backed, plus effects, permissions, args, output modes, and run surfaces
- prompt-manager has contract-backed catalog entries for the commands needed by Action validation and seed Actions
- parity tests detect operation-ref and output-schema drift for migrated prompt-manager commands
- a controlled command resolver can classify argv into owner, command path, operation ref, effect, permissions, output mode, certainty, and runnable decision
- active runnable Actions can depend on the resolver instead of implementing command validation themselves
- remaining unmigrated API/CLI surfaces are visible as explicit follow-up work, not hidden behind false completeness
