# Test-Genie Lint Phase Greenfield Redesign Plan

This plan covers a full redesign of the `test-genie` lint phase so it validates scenario codebases by discovered component shape and lint contract, not by hardcoded assumptions like "`api/` means Go" or "`ui/` means React/TypeScript".

The objective is not just to make the current lint phase less flaky. The objective is to make lint validation:

- component-based
- evidence-driven
- extensible
- explicit about unmatched code-bearing components
- deterministic in unit tests
- aligned with scenario freedom rather than repo habits

This is a **greenfield redesign**. It must be treated as a breaking cleanup, not as an additive compatibility exercise.

## Greenfield Constraint

This work is explicitly **greenfield**.

That means:

- do **not** preserve the current `api -> Go`, `ui -> Node`, `root -> Python` execution model
- do **not** add compatibility wrappers that let both the old folder-assumption model and the new component-discovery model coexist indefinitely
- do **not** keep legacy `languages.go`, `languages.node`, or `languages.python` config as a first-class long-term contract once the new component/handler contract exists
- do **not** add fallback shims that silently reinterpret old config into the new model
- do **not** keep dead runner code, dead tests, dead docs, or dead schema fields after cutover
- do **not** document the old layout-based behavior as an acceptable alternative

If a current behavior is wrong or overly narrow, replace it cleanly and remove it in the same implementation stream.

## Why This Exists

The current lint phase is structurally mis-modeled for a scenario platform.

It assumes:

- Go belongs in `api/` and `cli/`
- Node belongs in `ui/`
- Python belongs at scenario root
- only Go, Node, and Python matter

That is not the real scenario contract. The only hard layout requirement is that a scenario has an `api/` directory. Beyond that, scenarios should be free to choose different languages, frameworks, or additional top-level sidecars.

The current design therefore has two problems:

1. It misses legitimate lintable components because they do not live in a blessed folder/language pairing.
2. It reports success/skips based on folder conventions instead of whether the scenario actually contains configured code units.

This plan exists to replace that with one coherent model: discover top-level components, detect supported lint contracts from evidence, lint every supported component, and explicitly surface unsupported or unconfigured common components.

## Investigation Summary

These findings are the factual baseline for the redesign.

### 1. The current runner is hardcoded around folder/language assumptions

The main orchestrator in [runner.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/runner.go:1) currently constructs exactly four linters:

- Go for `api/`
- Go for `cli/`
- Node for `ui/`
- Python for scenario root

The detection logic is therefore:

- `api/go.mod` or `cli/go.mod` => Go target
- `ui/package.json` => Node target
- Python indicators at scenario root => Python target

This is the core behavior to replace, not extend.

### 2. The current phase and docs still encode the old assumptions

Current documentation and catalog text explicitly describe the lint phase as "Go, TypeScript, Python":

- [phase_lint.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/orchestrator/phases/phase_lint.go:1)
- [catalog.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/orchestrator/phases/catalog.go:60)
- [docs/phases/lint/README.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/phases/lint/README.md:1)
- [docs/phases/README.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/phases/README.md:1)

The redesign must update the documentation contract at the same time as the code.

### 3. The current config schema is language-centric, not component/contract-centric

The current `.vrooli/testing.json` and schema express lint settings like:

- `lint.languages.go`
- `lint.languages.node`
- `lint.languages.python`

Relevant files:

- [schemas/testing.schema.json](/home/matthalloran8/Vrooli/scenarios/test-genie/schemas/testing.schema.json:1)
- [.vrooli/testing.json](/home/matthalloran8/Vrooli/scenarios/test-genie/.vrooli/testing.json:17)
- [config.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/config.go:1)

That config shape bakes in the same assumption as the code. The greenfield redesign needs a new configuration model centered on handlers, policies, and component overrides.

### 4. The current tests mix unit expectations with real tool execution

The current phase tests stub command lookup but not actual command execution:

- [phase_lint_test.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/orchestrator/phases/phase_lint_test.go:1)
- [shared/command.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/shared/command.go:1)
- [golang/linter.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/golang/linter.go:145)
- [nodejs/linter.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/nodejs/linter.go:198)

This causes environment-sensitive behavior and invalid test fixtures to look like product bugs.

The redesign must make command execution injectable from day one.

### 5. The existing plan is now the wrong architecture

The original plan at [docs/plans/lint-phase-implementation.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/plans/lint-phase-implementation.md:1) is useful historical context, but it is based on the same language/folder assumptions as the current implementation and should not guide the redesign.

This new plan supersedes it.

## Target End State

At the end of this work, the lint phase should follow this model.

### 1. Top-Level Component Discovery

The lint phase scans the scenario root and builds a set of candidate top-level components.

Default scope:

- scenario root metadata
- direct child directories only

Examples:

- `api/`
- `ui/`
- `cli/`
- `worker/`
- `desktop/`
- `proxy/`
- `scripts/`

The phase should not recursively discover arbitrary nested subprojects by default.

### 2. Evidence-Driven Handler Matching

Each component is matched by supported lint handlers based on project evidence, not directory name.

Examples of evidence:

- `go.mod`
- `package.json`
- `tsconfig.json`
- `eslint.config.*`
- `pyproject.toml`
- `mypy.ini`
- `ruff.toml`
- future: `Cargo.toml`, `deno.json`, etc.

Folder names like `api`, `ui`, and `cli` are policy inputs, not detection rules.

### 3. Handler Registry

Lint execution is driven by a registry of handlers implementing one shared interface.

The registry should support multiple handlers over time, for example:

- Go module handler
- Node package handler
- Python project handler

Handlers should represent supported lint contracts, not just raw languages.

Examples:

- "Go module with golangci-lint/go vet"
- "Node package with TypeScript + ESLint"
- "Python project with Ruff/Flake8 + Mypy"

### 4. Policy Layer

After matching, the phase evaluates policy separately from execution.

Policy decisions should include:

- whether unmatched `api/`, `ui/`, or `cli/` is a warning or failure
- whether unconfigured but clearly code-bearing top-level sidecars are warnings
- whether lint warnings from a matched handler fail the phase (`strict`)
- whether unsupported component types are reported as informational, warning, or error

### 5. Component-Level Reporting

The phase result should report at the component level, not just by language.

Each component should have:

- component name
- component path
- matched handler ID, or explicit unmatched reason
- tools run
- issues found
- skip reason, if skipped
- policy findings, if any

### 6. Deterministic Testability

The orchestration and handlers must be unit-testable without real tool binaries.

That requires:

- injectable command execution
- clear separation between detection, policy, and execution
- real-tool integration tests only where explicitly intended

## Desired Architecture

The greenfield architecture should be organized around these responsibilities.

### Component Discovery

Suggested package responsibility:

- discover top-level candidate components
- classify obvious non-code directories for ignore/exclusion
- provide stable component metadata for later stages

Possible package shape:

```text
api/internal/lint/
├── discover/
│   ├── components.go
│   ├── filters.go
│   └── components_test.go
```

### Handler Registry

Suggested package responsibility:

- register supported handlers
- return match results for a component
- reject ambiguous or conflicting matches cleanly

Possible package shape:

```text
api/internal/lint/
├── handlers/
│   ├── registry.go
│   ├── types.go
│   ├── go_module/
│   ├── node_package/
│   └── python_project/
```

### Policy Engine

Suggested package responsibility:

- evaluate unmatched/common-component rules
- apply strictness rules
- convert handler findings into phase pass/fail semantics

Possible package shape:

```text
api/internal/lint/
├── policy/
│   ├── evaluator.go
│   ├── defaults.go
│   └── evaluator_test.go
```

### Orchestrator

Suggested package responsibility:

- wire discovery, matching, execution, and policy
- aggregate component results into phase summary

Possible package shape:

```text
api/internal/lint/
├── runner.go
├── summary.go
└── result.go
```

### Execution Abstraction

Suggested package responsibility:

- abstract command execution
- support deterministic unit tests
- capture stdout/stderr/exit metadata in one place

Possible package shape:

```text
api/internal/lint/
├── exec/
│   ├── runner.go
│   ├── types.go
│   └── runner_test.go
```

## Proposed Contracts

The exact names can change, but the responsibilities should remain.

### Component

```go
type Component struct {
    Name string
    Path string
    Kind string
    Entries []string
}
```

`Kind` should be discovery-oriented, not language-oriented. Examples:

- `top_level_dir`
- `scenario_root`

### Handler

```go
type Handler interface {
    ID() string
    Detect(Component) (MatchResult, error)
    Run(context.Context, RunRequest) (ComponentResult, error)
}
```

`Detect` should provide reasoning, not just a boolean.

### MatchResult

```go
type MatchResult struct {
    Matched    bool
    Confidence int
    Reason     string
}
```

The orchestration should reject ambiguous multi-handler matches unless there is an explicit precedence rule.

### Policy

```go
type Evaluation struct {
    Success      bool
    FailureClass FailureClass
    Findings     []PolicyFinding
}
```

Policy findings should be distinct from linter findings so the output can say:

- "handler found 3 issues"
- "component is present without supported lint contract"

## Default Greenfield Policy

The redesign should ship with a clear default policy.

### Components That Must Be Evaluated

- Every top-level code-bearing component matched by a supported handler

### Common Folder Rules

- `api/` present with no supported lint contract: fail
- `ui/` present with no supported lint contract: warn
- `cli/` present with no supported lint contract: warn

These should be policy defaults, not hardcoded discovery behavior.

### Other Top-Level Sidecars

- if clearly code-bearing and unmatched: warn
- if clearly non-code: ignore

### Strictness

Strictness should be configurable per handler and per component override, not only per language.

## Configuration Redesign

The current `lint.languages.*` configuration should be replaced with a new greenfield schema.

Suggested direction:

```json
{
  "lint": {
    "handlers": {
      "go_module": { "enabled": true, "strict": true },
      "node_package": { "enabled": true, "strict": true },
      "python_project": { "enabled": true, "strict": false }
    },
    "policy": {
      "unconfigured_common_components": {
        "api": "error",
        "ui": "warning",
        "cli": "warning"
      },
      "unmatched_code_components": "warning"
    },
    "components": {
      "worker": {
        "handler": "go_module",
        "strict": true
      }
    },
    "ignore": ["docs", "assets", "coverage", "test"]
  }
}
```

Key points:

- handler-centric, not language-centric
- policy is explicit
- component overrides are explicit
- ignore rules are explicit

Greenfield constraint:

- do not preserve the current `languages.go/node/python` schema as a parallel long-term contract
- cut over to the new schema and update scenarios/templates in the same implementation stream

## Implementation Phases

### Phase 0: Design Lock

Objective:

- finalize the component model, handler interface, policy contract, and configuration schema before coding

Tasks:

- define exact `Component`, `Handler`, `MatchResult`, `ComponentResult`, and `PolicyFinding` types
- define handler precedence and ambiguity rules
- define the default ignore list for obvious non-code directories
- define the new `lint` schema contract in `testing.schema.json`
- define how summaries/observations are rendered in CLI/API output

Exit criteria:

- one agreed architecture
- no unresolved ambiguity about discovery scope or config shape

### Phase 1: Discovery Layer

Objective:

- build top-level component discovery with deterministic tests

Tasks:

- scan scenario root and enumerate candidate components
- classify obvious non-code directories
- support scenario-root metadata where needed
- produce stable component descriptors for downstream stages

Validation:

- unit tests for discovery across mixed layouts
- fixtures covering `api`, `ui`, `cli`, sidecars, ignored directories, and empty directories

### Phase 2: Execution Abstraction

Objective:

- remove direct `exec.CommandContext(...)` coupling from handler logic

Tasks:

- add a lint-specific execution interface with injectable fake runners
- make stdout/stderr/exit-code capture consistent
- provide production runner implementation

Validation:

- unit tests for command success, non-zero exits, missing binaries, parse failures, and context cancellation

### Phase 3: Handler Registry And First-Class Handlers

Objective:

- implement the first greenfield handler set

Initial handlers:

- `go_module`
- `node_package`
- `python_project`

Tasks:

- move existing Go/Node/Python parsing logic behind the new handler contract where it is still correct
- delete assumptions that a handler belongs to a specific folder name
- make detection evidence-based
- ensure each handler can lint any top-level component path that matches its contract

Validation:

- unit tests for detection and execution per handler
- explicit ambiguous-match tests
- explicit unsupported-component tests

### Phase 4: Policy Engine

Objective:

- evaluate component outcomes and unmatched components consistently

Tasks:

- implement default policy for common folders and unmatched code-bearing components
- implement strictness semantics at handler and component-override levels
- map policy findings to phase pass/fail/failure class behavior

Validation:

- unit tests for:
  - `api/` unmatched => failure
  - `ui/` unmatched => warning
  - `cli/` unmatched => warning
  - strict handler warnings => failure
  - ignored directories => no finding

### Phase 5: Runner And Phase Integration

Objective:

- replace the current lint runner with the new orchestrator

Tasks:

- rewrite [runner.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/runner.go:1) around discovery + registry + policy
- update [phase_lint.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/orchestrator/phases/phase_lint.go:1) summary text to be component-oriented
- update lint observations and JSON output schema if needed
- preserve the lint phase position in the test flow unless a separate plan changes phase ordering

Validation:

- unit tests for orchestration
- API/CLI contract tests for phase summaries and result serialization

### Phase 6: Schema, Template, And Scenario Adoption

Objective:

- move the repo to the new lint config contract

Tasks:

- update [schemas/testing.schema.json](/home/matthalloran8/Vrooli/scenarios/test-genie/schemas/testing.schema.json:1)
- update test-genie docs/examples
- update scenario templates to emit the new lint config shape
- update modern scenarios that currently rely on `lint.languages.*`

Validation:

- schema validation tests
- generated scenario spot checks
- targeted validation on representative scenarios with different component layouts

### Phase 7: Documentation Cutover

Objective:

- make the new model the only documented model

Tasks:

- rewrite [docs/phases/lint/README.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/phases/lint/README.md:1)
- update [docs/phases/README.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/phases/README.md:1)
- update API/CLI docs that describe lint behavior
- replace references that describe lint as only "Go/TS/Python" where no longer true
- mark [docs/plans/lint-phase-implementation.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/plans/lint-phase-implementation.md:1) as superseded or remove it during cleanup

Validation:

- docs search for stale language/folder-assumption phrasing
- docs examples match the shipped config schema

### Phase 8: Cleanup Phase

Objective:

- remove all obsolete code, tests, docs, and config shapes from the repo

This phase is mandatory. The redesign is not complete until the old model is deleted.

Cleanup tasks:

- delete old folder-specific runner branches and helpers
- delete old `go/api`, `go/cli`, `node/ui`, `python/root` assumptions wherever still encoded
- delete obsolete tests that assert the old behavior
- delete or fully replace the old language-centric config parsing
- remove dead helper functions and unused parser/runner code
- remove stale docs/examples that describe the old model
- remove the superseded lint-phase plan or clearly archive it as historical context only

Cleanup validation:

- repo search returns no active docs or runtime code describing the old folder-to-language contract
- there is no dual-path execution model
- there is no unused config parser for the removed schema

## Test Strategy

The redesign should ship with a layered validation strategy.

### Unit Tests

Focus:

- discovery
- handler detection
- handler parsing
- policy evaluation
- orchestration with fake command runner

Requirements:

- no ambient tool dependence
- no accidental reliance on host `golangci-lint`, `go`, `tsc`, `eslint`, `ruff`, or `mypy`

### Integration Tests

Focus:

- selected end-to-end handler runs with real tools where available
- representative scenarios with mixed top-level components

Requirements:

- clearly marked as integration tests
- stable minimal fixtures with valid project inputs

### Scenario Validation

Run the redesigned lint phase against:

- `test-genie`
- `prompt-manager`
- `swarm-manager`
- `agent-inbox`
- at least one scenario with unusual sidecar layout

Validate:

- supported components are linted
- unmatched common components surface correctly
- summaries are understandable
- no false folder-based assumptions remain

## Acceptance Criteria

The implementation is complete only when all of the following are true:

- lint detection is based on discovered components plus handler evidence, not folder-language assumptions
- handlers are reusable across arbitrary top-level component names
- unmatched `api/`, `ui/`, and `cli/` behavior is enforced by policy, not detection shortcuts
- command execution is injectable and unit tests are deterministic
- the old `lint.languages.*` contract is gone from active runtime code and current docs
- templates and representative scenarios use the new config shape
- old runner code and stale tests are deleted
- docs describe only the new model

## Recommended Execution Order

1. Lock the new contracts and schema.
2. Build discovery and execution abstraction.
3. Build the handler registry and port the first handler set.
4. Add the policy engine.
5. Replace the runner and phase wiring.
6. Migrate schema, templates, and representative scenarios.
7. Rewrite docs.
8. Run the cleanup phase and delete the old model.
9. Validate against representative scenarios and phase/unit/integration suites.

## File Inventory Likely To Change

Core runtime:

- [api/internal/lint/runner.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/runner.go:1)
- [api/internal/lint/types.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/types.go:1)
- [api/internal/lint/config.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/config.go:1)
- [api/internal/orchestrator/phases/phase_lint.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/orchestrator/phases/phase_lint.go:1)

Current handler logic to refactor or replace:

- [api/internal/lint/golang/linter.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/golang/linter.go:1)
- [api/internal/lint/nodejs/linter.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/nodejs/linter.go:1)
- [api/internal/lint/python/linter.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/python/linter.go:1)

Tests:

- [api/internal/lint/runner_test.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/lint/runner_test.go:1)
- [api/internal/orchestrator/phases/phase_lint_test.go](/home/matthalloran8/Vrooli/scenarios/test-genie/api/internal/orchestrator/phases/phase_lint_test.go:1)
- handler-specific tests under `api/internal/lint/...`

Schema/config/docs:

- [schemas/testing.schema.json](/home/matthalloran8/Vrooli/scenarios/test-genie/schemas/testing.schema.json:1)
- [.vrooli/testing.json](/home/matthalloran8/Vrooli/scenarios/test-genie/.vrooli/testing.json:1)
- [docs/phases/lint/README.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/phases/lint/README.md:1)
- [docs/phases/README.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/phases/README.md:1)
- [docs/reference/api-endpoints.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/reference/api-endpoints.md:1)
- [docs/reference/cli-commands.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/reference/cli-commands.md:1)
- [docs/reference/presets.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/reference/presets.md:1)
- [docs/plans/lint-phase-implementation.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/plans/lint-phase-implementation.md:1)

Repo-level adopters likely affected:

- scenario templates under `templates/scenarios/...`
- representative scenarios with `.vrooli/testing.json`

## Risks To Manage

- over-fitting detection rules to current repo habits instead of evidence
- allowing handler ambiguity to silently pick the wrong toolchain
- preserving old schema paths "just in case", which would violate the greenfield constraint
- incomplete cleanup that leaves stale docs/tests shaping future agent behavior
- letting integration tests masquerade as unit tests again

## Bottom Line

The correct redesign is:

- discover components
- match them through a handler registry
- lint supported components
- evaluate unmatched/common components through explicit policy
- cut over the config and docs
- delete the old folder-assumption model completely

Anything less will preserve the same architectural problem under a cleaner surface.
