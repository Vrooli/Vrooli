# Test Genie Workspace Mapping Implementation Plan

Last updated: 2026-05-06

## Purpose

Implement a clean workspace mapping contract for test-genie so a scenario can be
read from one physical filesystem location while being validated as if it lived
at a logical repo-relative location.

This directly addresses template deep validation for
`templates/scenarios/react-vite`: the generated scenario lives under `/tmp`, but
documentation and standards checks must be able to evaluate repo-relative
contracts as if the scenario lived under `scenarios/<scenario-id>` in the Vrooli
repo.

## Required Reading

Run these before implementation:

```bash
prompt-manager skill read plan-skill-discovery implementation-plan-authoring
prompt-manager skill read documentation-health seam-discovery-and-enforcement boundary-of-responsibility-enforcement cli-steer interoperability-steer test
prompt-manager skill read api-steer utils-unification
```

Relevant files to read:

```bash
sed -n '1,240p' scenarios/test-genie/api/internal/orchestrator/workspace/workspace.go
sed -n '1,220p' scenarios/test-genie/api/internal/orchestrator/plan_preview.go
sed -n '1,260p' scenarios/test-genie/api/internal/orchestrator/suite_execution.go
sed -n '1,260p' scenarios/test-genie/cli/execute/command.go
sed -n '1,120p' scenarios/test-genie/cli/internal/execute/types.go
sed -n '1,260p' scenarios/test-genie/api/internal/orchestrator/phases/phase_docs.go
sed -n '1,700p' scenarios/test-genie/api/internal/docs/runner.go
sed -n '1,280p' scenarios/test-genie/api/internal/orchestrator/phases/phase_standards.go
sed -n '480,555p' scenarios/scenario-auditor/api/handlers_standards.go
sed -n '820,1040p' scenarios/scenario-auditor/api/handlers_standards.go
sed -n '300,340p' internal/cli/scenariohandlers/template_runtime.go
```

Discovery command already used while writing this plan:

```bash
prompt-manager discover "test-genie workspace path mapping" "scenario docs validation" "scenario-auditor standards explicit scenario path" "cli path flags contract" --complexity architectural
```

## Greenfield Constraint

This is greenfield work. Do not include compatibility shims, legacy wrappers,
duplicate flag aliases, dead code, unused re-exports, `// removed` comments, or
renamed `_unused` variables.

Keep `--scenario-path` because it is the current physical-path contract, not a
legacy shim. Rewrite its comments and surrounding model so it is explicitly one
part of the new contract.

## Problem Statement

`test-genie execute` now accepts `--scenario-path`, and template deep validation
uses it to test a generated scenario under a temp directory. This proves the
generated files can be read and tested, but it collapses two different concepts:

- **Physical location**: where test-genie reads the scenario files.
- **Logical placement**: where repo-relative validation should treat the
  scenario as living.

The difference matters for generated temp scenarios. A docs link like this in a
generated scenario:

```markdown
[port-allocation reference](../../../../docs/reference/port-allocation.md)
```

is conceptually valid from `scenarios/<id>/docs/reference/configuration.md`
inside the Vrooli repo, but fails when resolved physically from
`/tmp/.../scenarios/<id>/docs/reference/configuration.md` because the temp root
does not contain the repo-level `docs/` directory.

There is also a related standards issue. test-genie sends `scenario_path` to
scenario-auditor, and scenario-auditor uses it for the primary scan target, but
some external standards checks still receive only the scenario name and resolve
back to the real repo path. That caused a false `PRD.md missing` finding for
`/home/matthalloran8/Vrooli/scenarios/template-validation-react-vite-deep/PRD.md`
even though the retained generated scenario did contain `PRD.md`.

## Scope

In scope:

- Add an explicit workspace mapping model to test-genie.
- Extend `test-genie execute` with one physical-path field and two logical
  placement fields.
- Pass that mapping through CLI, API request structs, orchestrator planning, and
  phase environments.
- Update docs validation to resolve local links through the mapping.
- Update standards validation and scenario-auditor to preserve the mapping for
  external standards checks.
- Update react-vite template deep validation to pass logical placement for
  generated temp scenarios.
- Add focused unit/integration tests for mapped and unmapped behavior.
- Update test-genie and react-vite template maintenance docs for the new
  contract.

Out of scope:

- Domain scaffolding commands.
- Generated CLI metadata work.
- Backward-compatibility aliases for old flag names.
- Compatibility modes that guess intent from path shape.
- Fixing unrelated template failures such as empty requirements/business
  validation.
- Reworking comprehensive phase order, except where a test needs a small
  isolated fixture to avoid unrelated smoke/performance behavior.
- Broad scenario-auditor rule rewrites unrelated to path mapping.

## Current Technical Context

`scenarios/test-genie/cli/execute/command.go` currently exposes:

```bash
--scenario-path <absolute path>
```

The flag is parsed into `execute.Args.ScenarioPath`, sent through
`scenarios/test-genie/cli/internal/execute/types.go::Request`, and then carried
to `SuiteExecutionRequest.ScenarioPath`.

`scenarios/test-genie/api/internal/orchestrator/workspace/workspace.go` currently
has:

- `ScenarioWorkspace.ScenarioDir`
- `ScenarioWorkspace.AppRoot`
- `Environment.ScenarioDir`
- `Environment.AppRoot`
- `NewWithOverride(scenariosRoot, scenario, scenarioPath)`

`NewWithOverride` treats `scenarioPath` as the scenario directory and derives
`AppRoot` from the physical path. That is correct for a scenario that truly
lives elsewhere, but wrong for an ephemeral temp copy that should be validated
as if it lives in the Vrooli repo.

`scenarios/test-genie/api/internal/docs/runner.go` resolves local links with
physical filesystem paths only:

```go
target = r.resolvePath(target, link.File)
os.Stat(target)
```

`scenarios/test-genie/api/internal/orchestrator/phases/phase_standards.go`
passes `env.ScenarioName` and `env.ScenarioDir` to scenario-auditor as
`scenario_path`.

`scenarios/scenario-auditor/api/handlers_standards.go` accepts `scenario_path`
and uses it in `buildStandardsScanTargets`, but `performStandardsCheck` invokes
`runExternalRuleChecks(ctx, effectiveScenario, requestedSet, forceDisabled)`
without the physical scenario path or logical placement. Some external checks
therefore resolve the scenario name against the normal repo and can report false
findings.

`internal/cli/scenariohandlers/template_runtime.go` currently invokes
test-genie deep validation with:

```go
test-genie execute <scenario-id> \
  --scenario-path <temp-root>/scenarios/<scenario-id> \
  --preset <preset> \
  --no-stream \
  --json
```

## Target End State

test-genie has a first-class workspace mapping contract:

```bash
test-genie execute template-validation-react-vite-deep \
  --scenario-path /tmp/vrooli-template-deep-123/scenarios/template-validation-react-vite-deep \
  --logical-repo-root /home/matthalloran8/Vrooli \
  --logical-scenario-relpath scenarios/template-validation-react-vite-deep \
  --preset comprehensive
```

Semantics:

- `--scenario-path` is the physical scenario directory to read and write.
- `--logical-repo-root` is the repo root used for repo-relative validation.
- `--logical-scenario-relpath` is the logical scenario directory relative to
  `--logical-repo-root`.
- `--logical-repo-root` and `--logical-scenario-relpath` must be provided
  together.
- If logical fields are omitted, the physical path is authoritative and the
  scenario is treated as truly living there.
- If logical fields are present, phases that need repo-relative reasoning use
  the logical placement while still reading/writing scenario-owned files from
  the physical path.

Template deep validation automatically supplies:

```text
scenario_path = <tempRoot>/scenarios/template-validation-react-vite-deep
logical_repo_root = <actual Vrooli repo root>
logical_scenario_relpath = scenarios/template-validation-react-vite-deep
```

## Contract Decisions

### CLI Flags

Add exactly these flags to `test-genie execute`:

```bash
--logical-repo-root <absolute-path>
--logical-scenario-relpath <repo-relative-path>
```

Do not add aliases such as `--repo-root`, `--virtual-root`,
`--link-root`, or `--logical-scenario-path`.

Validation rules:

- `--scenario-path`, when present, must be absolute.
- `--logical-repo-root`, when present, must be absolute.
- `--logical-scenario-relpath` must be relative, clean, non-empty, and must not
  escape with `..`.
- The two logical flags must be provided together.
- `filepath.Base(logicalScenarioRelpath)` must match the scenario name.
- If `--scenario-path` is omitted, logical flags are allowed only if the
  physical scenario path resolves normally from the scenario name. This keeps
  mapped mode useful for future callers without tying it to temp generation.

### API Request Fields

Add JSON fields to `execute.Request` and `SuiteExecutionRequest`:

```go
LogicalRepoRoot        string `json:"logicalRepoRoot,omitempty"`
LogicalScenarioRelPath string `json:"logicalScenarioRelPath,omitempty"`
```

Use the same names in the test-genie HTTP API payloads. Do not introduce a
nested mapping object unless the existing request plumbing is already being
converted wholesale; a two-field additive request shape is simpler and enough.

### Workspace Model

Introduce a single mapping type in
`scenarios/test-genie/api/internal/orchestrator/workspace`:

```go
type Mapping struct {
    PhysicalScenarioDir     string
    PhysicalAppRoot         string
    LogicalRepoRoot         string
    LogicalScenarioRelPath  string
}
```

Recommended methods:

- `HasLogicalPlacement() bool`
- `LogicalScenarioDir() string`
- `PhysicalPath(rel string) string`
- `LogicalPath(rel string) string`
- `PhysicalToLogical(absPhysical string) (string, bool)`
- `LogicalToPhysical(absLogical string) (string, bool)`
- `ResolveLocalLink(fromPhysicalFile, href string) ResolvedLink`

`ResolvedLink` should make the decision explicit:

```go
type ResolvedLink struct {
    Exists        bool
    PhysicalPath  string
    LogicalPath   string
    OutsideScenario bool
}
```

Keep this mapping in the workspace package rather than in docs or standards.
Docs and standards are consumers of workspace identity, not owners of it.

### Link Resolution Rules

Docs validation should read markdown from the physical scenario path.

For each local markdown link:

1. Convert the source file from physical path to logical path when mapped mode
   is enabled.
2. Resolve the link against the logical source file path.
3. If the resolved logical target is still under the logical scenario dir, check
   existence through the physical scenario path.
4. If the resolved logical target escapes the logical scenario dir but remains
   under the logical repo root, check existence under the logical repo root.
5. If the resolved logical target escapes the logical repo root, fail it.
6. If no logical mapping is present, use current physical-only behavior.

This preserves the two intended cases:

- Truly external physical scenario: only `--scenario-path`, physical path is the
  source of truth.
- Temp copy pretending to live in Vrooli: `--scenario-path` plus logical
  placement fields.

### Scenario-Auditor Mapping

Extend scenario-auditor standards scan requests with:

```json
{
  "scenario_path": "/tmp/.../scenarios/demo",
  "logical_repo_root": "/home/.../Vrooli",
  "logical_scenario_relpath": "scenarios/demo"
}
```

Scenario-auditor must preserve those fields in its scan target and use the
physical scenario path for all file walking and rule content reads.

External standards checks must receive enough context to avoid resolving by
scenario name alone. The minimum acceptable implementation is to pass physical
scenario path through the external check path. If an external rule genuinely
needs repo-relative context, pass the logical placement too. Do not let any
standards path re-resolve the scenario by name when a physical path is supplied.

## Implementation Strategy

### Phase 1: CLI And Request Contract

Files:

- `scenarios/test-genie/cli/execute/command.go`
- `scenarios/test-genie/cli/execute/command_test.go`
- `scenarios/test-genie/cli/internal/execute/types.go`
- `scenarios/test-genie/api/internal/orchestrator/suite_execution.go`
- `scenarios/test-genie/api/internal/orchestrator/plan_preview.go`

Tasks:

1. Add `LogicalRepoRoot` and `LogicalScenarioRelPath` to CLI args and request
   structs.
2. Parse `--logical-repo-root` and `--logical-scenario-relpath`.
3. Enforce the validation rules from Contract Decisions.
4. Include mapping fields in both plan preview and execute requests.
5. Update help text and examples with exactly one mapped-mode example.
6. Rewrite `ScenarioPath` comments so it means physical path only.

Acceptance criteria:

- CLI rejects one logical flag without the other.
- CLI rejects relative `--logical-repo-root`.
- CLI rejects absolute or escaping `--logical-scenario-relpath`.
- CLI accepts the mapped temp scenario shape.
- Existing unmapped `--scenario-path` tests remain meaningful but comments no
  longer frame the feature as sandbox-only.

### Phase 2: Central Workspace Mapping

Files:

- `scenarios/test-genie/api/internal/orchestrator/workspace/workspace.go`
- `scenarios/test-genie/api/internal/orchestrator/workspace/scenario_workspace_test.go`
- `scenarios/test-genie/api/internal/orchestrator/workspace/README.md`

Tasks:

1. Add `Mapping` to the workspace package.
2. Add mapping fields to `ScenarioWorkspace` and `Environment`.
3. Replace `NewWithOverride(scenariosRoot, scenario, scenarioPath)` with a
   cleaner constructor that takes an options struct, for example:

   ```go
   type Options struct {
       ScenarioPath            string
       LogicalRepoRoot         string
       LogicalScenarioRelPath  string
   }
   ```

   Keep `New` only if it delegates directly with zero options and does not
   create a second path model.
4. Ensure `AppRoot` has a documented meaning. Prefer:
   - `PhysicalAppRoot` for the derived physical root.
   - `LogicalRepoRoot` for mapped repo-relative validation.
5. Update plan preview and suite execution to construct workspaces through the
   new options struct.

Acceptance criteria:

- Workspace tests cover unmapped default resolution.
- Workspace tests cover physical override only.
- Workspace tests cover physical override plus logical placement.
- Mapping methods are covered for inside-scenario links, repo-root links, and
  escaping links.

### Phase 3: Docs Validation Uses Mapping

Files:

- `scenarios/test-genie/api/internal/orchestrator/phases/phase_docs.go`
- `scenarios/test-genie/api/internal/docs/runner.go`
- `scenarios/test-genie/api/internal/docs/runner_test.go`

Tasks:

1. Add workspace mapping to `docs.Config`.
2. Replace `resolvePath` and local link existence checks with mapping-aware
   resolver calls.
3. Keep collection and file reading physical.
4. Apply mapping to `[CODE: ...]`, `// DOC: ...`, marked refs, and manifest
   coverage where those checks resolve filesystem paths.
5. Preserve physical file paths in observation locations for files inside the
   temp scenario so agents can open the failing file directly.
6. Include logical target context in broken-link messages when the failure is
   about repo-relative mapping.

Acceptance criteria:

- A fixture with physical temp scenario at `/tmp/root/scenarios/demo` and
  logical placement under `<repo>/scenarios/demo` passes a link from
  `docs/reference/configuration.md` to `../../../../docs/reference/foo.md` when
  `<repo>/docs/reference/foo.md` exists.
- The same fixture fails without logical placement.
- A link that stays inside the scenario checks the physical temp scenario file.
- A link escaping above the logical repo root fails.
- Existing docs tests are updated to use the new resolver instead of relying on
  physical-only assumptions.

### Phase 4: Standards Mapping Through Test-Genie

Files:

- `scenarios/test-genie/api/internal/orchestrator/phases/phase_standards.go`
- `scenarios/test-genie/api/internal/orchestrator/phases/phase_standards_test.go`

Tasks:

1. Send `logical_repo_root` and `logical_scenario_relpath` to scenario-auditor
   when mapping is present.
2. Include mapping fields in test doubles and expected JSON payload tests.
3. Update remediation text for mapped scans so commands are accurate and do not
   imply the scenario exists in the normal repo path when it does not.

Acceptance criteria:

- Standards phase request payload contains `scenario_path`,
  `logical_repo_root`, and `logical_scenario_relpath` in mapped mode.
- Standards phase request payload omits logical fields in unmapped mode.

### Phase 5: Scenario-Auditor Standards Mapping

Files:

- `scenarios/scenario-auditor/api/handlers_standards.go`
- `scenarios/scenario-auditor/api/handlers_standards_test.go`
- External standards rule call sites under `scenarios/scenario-auditor/api/`
  that currently resolve by scenario name.

Tasks:

1. Add logical mapping fields to standards check request parsing.
2. Introduce a standards scan target struct that carries:
   - scenario name
   - physical scenario path
   - logical repo root
   - logical scenario relpath
3. Thread that target through `StartScan`, `performStandardsCheck`, and external
   rule checks.
4. Remove paths that re-resolve the target scenario by name when a physical path
   is present.
5. Update violation reporting to keep `file_path` scenario-relative where
   possible, even when the physical file was under `/tmp`.
6. Add tests that reproduce the false `PRD.md missing` case:
   - no scenario exists at `<realRepo>/scenarios/demo`
   - physical temp scenario contains `PRD.md`
   - mapped scan must not report missing PRD

Acceptance criteria:

- All standards rule paths scan the physical target when supplied.
- External checks do not resolve by scenario name in mapped scans.
- Top violations use stable scenario-relative file paths.

### Phase 6: Template Deep Validation Uses Mapped Mode

Files:

- `internal/cli/scenariohandlers/template_runtime.go`
- `internal/cli/scenariohandlers/template_runtime_test.go`

Tasks:

1. Add the new logical flags to the test-genie invocation:

   ```go
   "--logical-repo-root", deps.Root(ctx),
   "--logical-scenario-relpath", filepath.Join("scenarios", scenarioID),
   ```

2. Update tests to assert all three path fields are passed:
   - `--scenario-path`
   - `--logical-repo-root`
   - `--logical-scenario-relpath`
3. Keep temp cleanup behavior unchanged.
4. Do not copy repo-level docs into the temp workspace as a workaround.

Acceptance criteria:

- Deep validation no longer fails docs solely because repo-level docs are absent
  from the temp root.
- Template deep validation still tests the generated scenario files from the
  temp physical path.

### Phase 7: Documentation

Files:

- `scenarios/test-genie/docs/reference/cli-commands.md`
- `scenarios/test-genie/docs/concepts/infrastructure.md`
- `scenarios/test-genie/docs/guides/troubleshooting.md`
- `templates/scenarios/react-vite/docs/internal/TEMPLATE-GENERATION-CONTRACT.md`
- `templates/scenarios/react-vite/docs/internal/TEMPLATE-MAINTENANCE.md`

Tasks:

1. Document physical path vs logical placement in test-genie docs.
2. Add examples for:
   - testing a true external scenario with only `--scenario-path`
   - testing an ephemeral scenario with logical placement
3. Update react-vite template maintenance docs so deep validation explains the
   mapped temp scenario behavior.
4. Avoid docs that imply temp validation copies the full repo.

Acceptance criteria:

- Docs use the same flag names as the CLI.
- Docs explain when logical placement is needed.
- Docs do not describe legacy sandbox-only semantics for `--scenario-path`.

## Testing Plan

Run focused package tests first:

```bash
go test ./internal/cli/scenariohandlers
cd scenarios/test-genie/cli && go test ./...
cd scenarios/test-genie/api && go test ./internal/orchestrator/workspace ./internal/orchestrator/... ./internal/docs/...
cd scenarios/scenario-auditor/api && go test ./...
```

Run mapped-mode command validation against a retained generated scenario from
template deep validation, replacing `<retained-temp-root>` with the path printed
by `--retain-temp`:

```bash
test-genie execute template-validation-react-vite-deep \
  --scenario-path <retained-temp-root>/scenarios/template-validation-react-vite-deep \
  --logical-repo-root /home/matthalloran8/Vrooli \
  --logical-scenario-relpath scenarios/template-validation-react-vite-deep \
  --preset quick \
  --json
```

Use a real generated template run as the final ground-truth validation:

```bash
vrooli scenario template validate --mode shallow --template react-vite
vrooli scenario template validate --mode deep --template react-vite --test-preset comprehensive --retain-temp
```

Expected after this plan:

- The docs phase should not fail because repo-level links are resolved against
  the logical repo root.
- The standards phase should not report `PRD.md missing` against the normal repo
  path when the physical temp scenario has `PRD.md`.
- Any remaining failures should be real template/test-genie policy issues, not
  path identity confusion.

## Rollout And Validation Checklist

- [ ] CLI help shows all three path-related flags with distinct meanings.
- [ ] API request structs carry logical mapping fields.
- [ ] Workspace package owns all physical/logical path decisions.
- [ ] Docs runner has no ad hoc temp-workspace special cases.
- [ ] Standards phase passes mapping to scenario-auditor.
- [ ] Scenario-auditor does not re-resolve mapped scan targets by name.
- [ ] Template deep validation passes mapped mode fields.
- [ ] Test artifacts point to physical paths that agents can open.
- [ ] Violation file paths remain stable and scenario-relative where possible.
- [ ] React-vite shallow validation passes.
- [ ] React-vite deep validation reaches real template policy failures, if any,
      rather than docs/standards false positives from path mapping.

## Risks And Mitigations

**Risk: Logical mapping leaks into phases that should use physical paths.**  
Mitigation: the workspace mapping API must expose intent-specific methods.
Callers should choose physical reads/writes explicitly. Only repo-relative
validation should use logical paths.

**Risk: Docs observations become confusing because a logical target differs from
the physical source file.**  
Mitigation: keep source locations physical and include logical target context
only when it explains a mapped-resolution failure.

**Risk: Scenario-auditor external checks keep hidden name-based resolution.**  
Mitigation: add a test where the scenario exists only at the physical temp path.
That test fails if any high-level standards check resolves by name.

**Risk: The new mapping becomes another loose bundle of strings.**  
Mitigation: centralize it in `orchestrator/workspace` and pass the typed mapping
through phase environments.

## Non-Goals And Prohibited Patterns

- Do not add compatibility aliases for flag names.
- Do not infer logical placement by looking for `/tmp` or path prefixes.
- Do not copy repo-level `docs/` into temp roots as the primary fix.
- Do not add docs-only link root flags.
- Do not let each phase implement its own physical/logical resolver.
- Do not keep sandbox-only comments for `--scenario-path`.
- Do not re-resolve a mapped standards scan by scenario name.
- Do not leave unused fields or helper wrappers after refactoring constructors.

## Definition Of Done

- `test-genie execute` has a documented, validated physical/logical path
  contract.
- The workspace package is the single source of truth for path mapping.
- Docs validation correctly handles links from a physical temp scenario to
  logical repo-root docs.
- Scenario-auditor standards scans honor mapped physical scenario paths end to
  end, including external checks.
- React-vite template deep validation invokes test-genie in mapped mode.
- All touched Go packages pass tests.
- No compatibility shims, legacy aliases, duplicate resolvers, dead code, or
  unused re-exports are introduced.

## Final Cleanup And Scenario Health Verification

Because this plan modifies scenarios (`test-genie` and `scenario-auditor`), the
implementing agent must finish with:

```bash
cd scenarios/test-genie && make test
cd scenarios/scenario-auditor && make test
vrooli scenario restart test-genie
vrooli scenario status test-genie
vrooli scenario restart scenario-auditor
vrooli scenario status scenario-auditor
```

Fix all lint, type, and unit test issues in modified files, including issues
that appear pre-existing. Use scenario Makefiles or `vrooli scenario ...`
lifecycle commands only; do not run scenario binaries directly.
