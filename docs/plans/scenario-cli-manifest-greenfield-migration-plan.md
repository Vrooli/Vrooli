# Scenario CLI Manifest Greenfield Migration Plan

This migration should be treated as a repo-wide breaking change, not an incremental refactor. The design constraint is simple: stop inferring scenario CLI implementation from file layout and make `service.json` the single source of truth.

## Phase 0: Freeze The Contract

1. Define the target CLI model before touching code.
2. Decide exactly what the new `service.json` must express:
   - CLI presence
   - installed command name
   - implementation adapter kind
   - install strategy per OS
   - invocation contract
   - freshness inputs
3. Explicitly reject legacy inference rules:
   - no `path:cli/go.mod` detection
   - no `path:cli/<scenario-name>` detection
   - no implicit `path:cli/install.sh` contract
4. Decide whether CLI is required or optional per scenario.
5. Decide whether `vrooli` can execute scenario CLIs directly from source via adapter, or only via installed command.
6. Write a short ADR in `docs/` because this is a platform contract change.

## Phase 1: Update The Schema

1. Extend the `service.json` schema with a first-class `cli` section.
2. Keep it minimal and explicit. A good shape is:
   - `cli.enabled`
   - `cli.command`
   - `cli.adapter.kind`
   - `cli.adapter.config`
   - `cli.install[]`
   - `cli.invoke`
   - `cli.freshness`
3. Remove schema assumptions that tie setup to `install-cli` or `path:cli/install.sh`.
4. Update repo contract docs and scenario docs so the manifest, not folder heuristics, defines the CLI.
5. Add schema examples for:
   - Go module CLI
   - Bash script CLI
   - External binary CLI
   - No CLI

## Phase 2: Replace Platform Discovery With Manifest-Driven Adapters

1. Replace `path:internal/cliinstall` discovery logic with manifest parsing.
2. Introduce a small adapter interface:
   - `Validate(manifestCLI)`
   - `Install(ctx, scenarioRoot, home)`
   - `ResolveExecutable(ctx, scenarioRoot, home)`
   - `IsCurrent(ctx, scenarioRoot, installedPath)`
3. Implement only the greenfield adapters you want to support immediately:
   - `go_module`
   - `shell_script`
4. Delete fallback discovery code entirely.
5. Delete code that guesses CLI type from `go.mod`, `install.sh`, or script names.
6. Keep install behavior driven only by manifest adapter config.

## Phase 3: Update Lifecycle And Handler Code

1. Update lifecycle setup checks to read `service.json.cli`, not repo layout.
2. Replace `"type": "cli"` check semantics so they validate against the declared CLI contract.
3. Update `vrooli scenario start/run/test` pre-install logic to use the new adapter path only.
4. Update hard-coded CLI locators like `test-genie` and `scenario-completeness-scoring` to use the same manifest-driven resolver.
5. Update any direct `exec.LookPath("<scenario-cli>")` consumers to go through a shared scenario CLI resolution service.
6. Delete any logic that special-cases Go CLI freshness by scanning `path:cli/*.go` unless the adapter declares that as its freshness model.

## Phase 4: Update Validation Tooling

1. Update `test-genie` CLI structure validation to validate manifest contract, not file presence patterns.
2. Remove `legacy` vs `cross-platform` terminology from validation code.
3. Update `scenario-auditor` rules so they stop enforcing:
   - `install-cli`
   - `cd cli && ./install.sh`
   - `path:cli/install.sh`
4. Replace those rules with:
   - valid `cli` manifest section
   - valid adapter config
   - valid install strategies for declared target OSes
5. Update docs and remediation messages to point to the new manifest contract.

## Phase 5: Update Scenario Templates

1. Update all scenario templates to emit the new `cli` manifest block.
2. Update Go-based templates to use the `go_module` adapter config.
3. Update Bash-based templates to use the `shell_script` adapter config.
4. Remove template text that implies `path:cli/install.sh` is the platform contract.
5. Keep install scripts only as adapter implementation assets, not as what the platform validates.

## Phase 6: Build The Migration Script

1. Write a script that inspects each scenario and proposes a `cli` manifest block based on actual implementation.
2. Inputs for classification:
   - `path:cli/go.mod` present => `go_module`
   - executable `path:cli/<scenario-name>` text script => `shell_script`
3. The script should not directly mutate everything blindly on first pass.
4. Better workflow:
   - generate a report of intended changes
   - generate candidate JSON patches
   - review sample output
   - apply after validation
5. The script should also flag ambiguous scenarios:
   - both Go and shell assets present
   - missing install surface
   - mismatched command names
   - compiled binaries checked into `cli/`
6. Because the repo guidance discourages mass blind edits, use the script to generate deterministic candidate edits and then apply or review them in controlled batches.

## Phase 7: Migrate All Scenario Manifests

1. Run the script and classify every scenario.
2. Review ambiguous cases manually first.
3. Apply manifest updates in batches:
   - templates first
   - Go scenarios
   - shell scenarios
4. Remove old manifest fields or setup assumptions that no longer match the new contract.
5. Normalize command names and install paths while migrating.

## Phase 8: Delete Legacy Code

1. Remove compatibility branches from:
   - `path:internal/cliinstall`
   - lifecycle CLI checks
   - validator code
   - auditor rules
   - any hard-coded scenario CLI finders
2. Remove docs describing legacy CLI detection.
3. Remove tests for legacy inference behavior.
4. Add failing tests that assert inference no longer exists.

## Phase 9: Validation

1. Unit tests:
   - schema validation
   - adapter parsing
   - install resolution
   - freshness checks
   - lifecycle setup checks
2. Integration tests:
   - `vrooli scenario start` on a Go CLI scenario
   - same on a Bash CLI scenario
   - scenario with no CLI
   - malformed CLI manifest
3. Migration validation:
   - every scenario manifest parses
   - every declared CLI adapter validates
   - every declared CLI command resolves or fails with a precise error
4. Repo-wide checks:
   - grep confirms no remaining layout inference code
   - grep confirms no remaining auditor rules tied to `path:cli/install.sh`
   - grep confirms templates all emit the new manifest block

## Recommended Execution Order

1. ADR + schema
2. adapter service + platform adoption
3. validator and auditor updates
4. template updates
5. migration script
6. scenario manifest migration
7. legacy deletion
8. full validation

## Non-Negotiable Acceptance Criteria

- Scenario CLI behavior is driven only by `service.json`
- No code infers CLI type from file layout
- No handler code contains fallback logic for old structures
- Templates emit only the new contract
- Every scenario manifest is migrated or explicitly marked `cli.enabled=false`
- Validation tooling enforces only the new contract

## Main Risk

The dangerous failure mode is half-migration: new schema added, but old discovery still silently works. That would preserve hidden coupling. If this is meant to be truly greenfield, the cutover needs to be hard, with old inference removed early and repo migration completed in the same branch.
