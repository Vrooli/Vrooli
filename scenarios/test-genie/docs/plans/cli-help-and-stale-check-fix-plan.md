# CLI Help And Stale-Check Fix Plan

This plan covers two related but distinct CLI problems discovered during validation of the `test-genie` lint redesign:

1. stale-check / auto-rebuild fingerprint loops
2. broken `... --help` handling for many Go scenario CLIs

The two issues interact in user-visible output, but they do not have the same root cause and should be fixed separately.

This should be implemented as a **clean fix**, not as compatibility layering. Do not add temporary fallback logic that keeps conflicting fingerprint contracts or multiple help-dispatch models alive indefinitely.

## Investigation Summary

### 1. Rebuild loop is a shared `cli-core` bug

This is not specific to `test-genie`.

It reproduced with:

- `test-genie`
- `agent-inbox`
- `prompt-manager`

Observed pattern:

- the CLI detects itself as stale
- it auto-rebuilds
- it restarts
- it still considers itself stale
- it prints a rebuild-loop warning and stops rebuilding

The current warning in [packages/cli-core/cliutil/stalechecker.go](/home/matthalloran8/Vrooli/packages/cli-core/cliutil/stalechecker.go) mentions binary-name mismatch, but the deeper issue is broader: the stale checker and installer are not using one canonical fingerprint contract.

#### Root cause

The stale checker and installer compute fingerprints from different inputs.

In [packages/cli-core/cliutil/stalechecker.go](/home/matthalloran8/Vrooli/packages/cli-core/cliutil/stalechecker.go):

- when `FreshnessInputs` is present, stale checking uses:
  - `SourceContextPath`
  - declared `FreshnessInputs`
  - `computeFingerprintFromDeclaredInputs(...)`

In [packages/cli-core/cmd/cli-installer/main.go](/home/matthalloran8/Vrooli/packages/cli-core/cmd/cli-installer/main.go):

- installer always computes fingerprint with:
  - `buildinfo.ComputeFingerprint(modulePath, binaryName)`

For standard scenario CLIs wired through [packages/cli-core/cliapp/scenario_app.go](/home/matthalloran8/Vrooli/packages/cli-core/cliapp/scenario_app.go), the stale checker uses:

- `SourceContextPath: ".."`
- `FreshnessInputs: []string{"cli/**", ".vrooli/service.json"}`

But the installer fingerprints the module root only.

That means rebuild verification is structurally inconsistent by design.

### 2. `... --help` breakage is partly shared and partly scenario-specific

This is also not specific to `test-genie`.

Observed behavior:

- `test-genie --help` works
- `test-genie execute --help` fails
- `test-genie generate --help` fails
- `test-genie status --help` fails
- `agent-inbox --help` works
- `agent-inbox status --help` fails
- `agent-inbox chat --help` works
- `prompt-manager --help` works
- `prompt-manager skill --help` fails

#### Shared root cause

`cli-core` currently supports:

- top-level help: `app --help`
- explicit meta help: `app help`
- subgroup help: `app group --help`

But it does **not** provide first-class handling for:

- `app command --help` for flat commands
- `app group subcommand --help` when the subcommand tree is implemented inside a flat command handler rather than a `cli-core` subgroup

This is visible in [packages/cli-core/cliapp/app.go](/home/matthalloran8/Vrooli/packages/cli-core/cliapp/app.go):

- `Run()` handles global help before dispatch
- `runSubcommand()` handles subgroup help
- flat commands are otherwise handed directly to `cmd.Run(...)`

So `cli-core` does not currently guarantee command-level help dispatch.

#### Scenario-specific manifestation

Each affected scenario depends on how its commands are wired:

- `test-genie` registers `execute`, `generate`, and `status` as flat commands
- `test-genie execute` in [scenarios/test-genie/cli/execute/command.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/execute/command.go) treats the first positional argument as the scenario name
- so `execute --help` is parsed as `scenarioName="--help"`

Similarly:

- `agent-inbox status` is a flat command and fails under the same framework limitation
- `agent-inbox chat --help` works because `chat` is a real `cli-core` subgroup
- `prompt-manager skill --help` fails because `skill` is a flat command that implements its own nested dispatch internally instead of using `cli-core` subgroup help

## Fix Order

Implement in this order:

1. `cli-core` rebuild/fingerprint unification
2. `cli-core` command-help dispatch
3. scenario CLI adoption/tests

The rebuild issue should be fixed first because it pollutes every CLI invocation and can obscure whether help routing is working correctly.

## Track 1: Rebuild Loop Fix

### Goal

Make stale checking and installer rebuilding compute the exact same fingerprint from the exact same declared freshness contract.

### Target design

Introduce one canonical freshness-spec API in shared code, used by both:

- `cliutil.StaleChecker`
- `cmd/cli-installer`

Suggested shape:

```go
type FreshnessSpec struct {
    SourceRoot  string
    ContextRoot string
    Inputs      []string
    SkipFiles   []string
}

func ComputeFreshnessFingerprint(spec FreshnessSpec) (string, error)
```

### Implementation steps

1. Add shared freshness fingerprint helper in `cli-core`.
   Likely location:
   - new file in `packages/cli-core/cliutil/` or `packages/cli-core/buildinfo/`

2. Change [packages/cli-core/cliutil/stalechecker.go](/home/matthalloran8/Vrooli/packages/cli-core/cliutil/stalechecker.go) to build a `FreshnessSpec` and call the shared helper.

3. Change [packages/cli-core/cmd/cli-installer/main.go](/home/matthalloran8/Vrooli/packages/cli-core/cmd/cli-installer/main.go) to accept and honor the same freshness inputs.

4. Change [packages/cli-core/install.sh](/home/matthalloran8/Vrooli/packages/cli-core/install.sh) and stale-checker rebuild invocation to pass the canonical freshness inputs through explicitly.

5. Remove the current split where:
   - runtime stale check uses declared inputs
   - installer uses whole-module fingerprinting

### Tests to add

In `cli-core`:

- stale checker and installer produce identical fingerprint for:
  - plain module-root CLI
  - standard scenario CLI with `SourceContextPath=".."` and `FreshnessInputs=["cli/**",".vrooli/service.json"]`
- no rebuild loop after one rebuild
- manifest changes affect fingerprint when declared
- undeclared file changes do not affect fingerprint

Likely test files:

- [packages/cli-core/cliutil/stalechecker_test.go](/home/matthalloran8/Vrooli/packages/cli-core/cliutil/stalechecker_test.go)
- [packages/cli-core/cmd/cli-installer/main_test.go](/home/matthalloran8/Vrooli/packages/cli-core/cmd/cli-installer/main_test.go)

### Acceptance criteria

- `test-genie` no longer prints rebuild-loop warnings after one rebuild
- same for `agent-inbox` and `prompt-manager`
- auto-rebuild stabilizes after first refresh

## Track 2: Command Help Dispatch Fix

### Goal

Make help/version resolution happen before stale-check, preflight, API resolution, or command execution.

### Target behavior

Supported help forms should be:

- `app --help`
- `app help`
- `app command --help`
- `app group --help`
- `app group subcommand --help`

All help paths should be side-effect free:

- no stale-check
- no API preflight
- no command body invoked

### Recommended shared API changes

Extend `cli-core` command metadata so commands can render framework-driven help.

Suggested additions on `cliapp.Command`:

- `Usage string`
- `LongDescription string`
- or `Help func() string`

The exact shape can be chosen during implementation, but the framework needs enough information to print command-specific help without entering the command body.

### Implementation steps

1. Extend [packages/cli-core/cliapp/app.go](/home/matthalloran8/Vrooli/packages/cli-core/cliapp/app.go) command model with command-level help metadata.

2. Add shared helpers such as:

- `isHelpToken(string) bool`
- `wantsHelp(args []string) bool`

3. In `Run()`:

- detect `command --help`
- print command help
- return before stale-check or preflight

4. In `runSubcommand()`:

- detect `group subcommand --help`
- print subcommand help
- return before stale-check or preflight

5. Keep subgroup help behavior (`group --help`) intact.

### Tests to add

In [packages/cli-core/cliapp/app_test.go](/home/matthalloran8/Vrooli/packages/cli-core/cliapp/app_test.go):

- `demo --help`
- `demo help`
- `demo status --help`
- `demo group --help`
- `demo group sub --help`
- assert stale checker is not invoked for help
- assert preflight is not invoked for help

### Acceptance criteria

- flat command help works generically across `cli-core` consumers
- subgroup subcommand help works generically
- no help path makes API calls or triggers rebuilds

## Track 3: Scenario CLI Adoption

After `cli-core` is fixed, update scenario CLIs to use the shared help contract cleanly.

### `test-genie`

Files:

- [scenarios/test-genie/cli/domains/suites/register.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/domains/suites/register.go)
- [scenarios/test-genie/cli/execute/command.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/execute/command.go)
- [scenarios/test-genie/cli/generate/command.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/generate/command.go)
- [scenarios/test-genie/cli/status/command.go](/home/matthalloran8/Vrooli/scenarios/test-genie/cli/status/command.go)

Actions:

- provide proper command usage/help metadata
- stop relying on positional parsing to implicitly reject/consume `--help`
- add command tests for:
  - `execute --help`
  - `generate --help`
  - `status --help`

### `agent-inbox`

Primary affected path:

- flat `status --help`

Actions:

- add help metadata for flat commands
- add regression test for `status --help`

### `prompt-manager`

This one is mixed:

- top-level help works
- custom internal command trees like `skill --help` still fail because those commands implement nested dispatch inside a flat command

Actions:

- either migrate those major command trees to true `cli-core` subgroups
- or normalize their internal dispatchers so `--help` is handled explicitly and consistently at the command boundary

This should not become an excuse for ad hoc per-command hacks everywhere. Prefer shared dispatch where possible.

### Acceptance criteria

- `test-genie execute --help` works
- `test-genie generate --help` works
- `test-genie status --help` works
- `agent-inbox status --help` works
- `agent-inbox chat --help` still works
- `prompt-manager skill --help` works

## Validation Plan

### Shared

Run:

```bash
cd packages/cli-core
go test ./...
```

### Scenario CLIs

Add targeted CLI tests and run them in each scenario.

Manual validation:

```bash
test-genie --help
test-genie execute --help
test-genie generate --help
test-genie status --help

agent-inbox --help
agent-inbox status --help
agent-inbox chat --help

prompt-manager --help
prompt-manager skill --help
```

### Regression requirements

Verify that:

- help commands do not hit APIs
- help commands do not trigger stale-check rebuilds
- first real command after source change rebuilds once and then stabilizes

## Cleanup Requirements

This work is not complete until the old conflicting behavior is removed cleanly.

Required cleanup:

- no dual fingerprint paths remain in `cli-core`
- stale checker and installer share one canonical freshness contract
- no stale-check on help/version paths
- no misleading comments claiming matching fingerprint behavior while code diverges
- no scenario-specific workaround code where shared `cli-core` dispatch should own behavior

## Recommended Delivery Split

Two implementation chunks:

1. `cli-core` rebuild/fingerprint unification + help dispatch
2. scenario CLI adoption + tests

That keeps the shared contract change separate from scenario-specific cleanup while still allowing a clean end-to-end rollout.
