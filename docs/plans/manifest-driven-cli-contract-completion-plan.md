# Manifest-Driven CLI Contract Completion Plan

## Status

Assessment as of 2026-04-16:

- Scenario CLI discovery and install in the main `vrooli scenario ...` path are manifest-driven.
- Resource CLI discovery and install are also manifest-driven.
- The scenario and resource schemas both expose a top-level `cli` contract.
- Scenario and resource manifests both support `go_module` and `shell_script` adapters.
- Setup freshness, runtime resolution, and the main validator/test utility surfaces now consume the manifest contract.
- Invocation remains standardized around `cli.invoke.kind = installed_command`, by deliberate platform policy.
- The key acceptance case now passes: `vrooli scenario test scenario-auditor`.
- The key acceptance case was revalidated live on 2026-04-16 after CLI install state was refreshed:
  - `vrooli scenario test scenario-auditor`
  - `go test ./internal/cliinstall ./internal/cli/vroolicli ./internal/scenarioexec ./internal/resources ./internal/scenario`
  - `bash scripts/validate-go-cli-consumers.sh`
- The remaining residue is narrow and secondary:
  - a few fixture-specific tests still use `path:cli/go.mod` paths because they are intentionally testing Go CLI module repair behavior
  - some docs and archived review artifacts still mention historical layout assumptions
- Phase 1 contract decisions are resolved in [docs/strategy/scenario-cli-manifest-decision.md](/home/matthalloran8/Vrooli/docs/strategy/scenario-cli-manifest-decision.md:1).

This plan is for finishing the migration cleanly rather than preserving mixed behavior indefinitely.

## Goal

Make CLI behavior consistently contract-driven across the repo:

- scenario CLIs are resolved from `service.json`
- resource CLIs are resolved from manifest data instead of `path:resources/<name>/cli` assumptions
- setup freshness, validators, and helper tooling use the same manifest contract as runtime
- no runtime path guesses CLI behavior from `path:cli/go.mod`, `path:cli/<name>`, or other layout heuristics unless the declared adapter says those files matter
- any remaining intentional constraints, especially invocation, are explicit platform policy rather than accidental legacy behavior

## Current Baseline

### Already in place

- Scenario CLI schema exists in `.vrooli/schemas/service.schema.json`
- Scenario manifests can declare:
  - `cli.enabled`
  - `cli.command`
  - `cli.adapter.kind`
  - `cli.install`
  - `cli.invoke`
  - `cli.freshness`
- Main scenario discovery in `path:internal/cliinstall` reads the manifest and supports:
  - `go_module`
  - `shell_script`
- Main `vrooli scenario start/run/test` ensure-install flow already goes through manifest-driven scenario CLI discovery

### Remaining gaps

- A few fixture-specific tests still refer to `path:cli/go.mod` because they intentionally exercise Go-module repair logic
- Some long-form docs and archived artifacts still describe historical layout assumptions
- The migration validator script exists and is manifest-driven, but still needs normal repo tracking/cleanup discipline in the working tree used for this migration

## Architecture Decision

Treat manifest data as the only contract for CLI resolution.

That means:

- the manifest declares whether a CLI exists
- the manifest declares how it is installed
- the manifest declares which adapter owns the implementation files
- the manifest declares what inputs determine staleness
- runtime, setup, validators, and tests all consume the same declaration

Do not preserve dual behavior where manifests are "preferred" but filesystem inference still silently works. That creates hidden coupling and makes future migrations ambiguous.

Phase 1 decisions have now been resolved as follows:

- every scenario manifest must contain an explicit top-level `cli` block
- resources will adopt a parallel top-level `cli` block
- both scenarios and resources support `go_module` and `shell_script`
- `cli.invoke.kind = installed_command` remains deliberate platform policy for now
- every CLI must explicitly declare `cli.command`
- resource CLI command naming should usually follow `resource-<name>`, but that value must still be declared, not inferred
- CLI freshness is manifest-driven, with adapter-required files treated as part of the adapter contract
- legacy layout inference is explicitly unsupported with no migration-period contract

## Phase 1: Freeze The End-State Contract

Status: complete

1. Write a short design note or ADR that states the intended platform contract precisely.
   - completed in [docs/strategy/scenario-cli-manifest-decision.md](/home/matthalloran8/Vrooli/docs/strategy/scenario-cli-manifest-decision.md:1)
2. Define the contract for scenarios:
   - `service.json` is the sole CLI contract
   - explicit top-level `cli` block is required
   - supported adapters are `go_module` and `shell_script`
   - supported invocation remains `installed_command`
   - freshness ownership is manifest-driven
3. Define the contract for resources:
   - resources get a first-class top-level `cli` block parallel to scenarios
   - resources support both `go_module` and `shell_script`
   - resource CLIs must explicitly declare `cli.command`
4. Decide whether invocation remains intentionally constrained to `installed_command`.
   - yes; this is deliberate platform policy for now
5. Document what is explicitly no longer allowed:
   - implicit `path:cli/go.mod` inference
   - implicit `path:cli/<scenario-name>` inference
   - implicit `path:resources/<name>/cli` discovery
   - layout-only freshness logic
   - silent fallback from manifest-driven resolution to repo-layout guessing

## Phase 2: Add A First-Class Resource CLI Contract

Status: complete

1. Extend the resource schema with a top-level CLI declaration parallel to the scenario model.
   - completed by adding top-level `cli` to `.vrooli/schemas/resource.schema.json`
2. Add matching Go manifest types and validation logic for resources.
   - completed in `path:internal/resources/manifest`
3. Support the intended resource adapter set:
   - supported by contract as `go_module` and `shell_script`
4. Ensure invalid or missing resource CLI declarations fail during manifest parsing, not later during CLI install.
   - completed by requiring explicit `cli` during resource manifest validation
5. Update resource docs so the manifest, not `path:resources/<name>/cli`, defines the contract.
   - completed for the canonical resource template guidance

## Phase 3: Unify CLI Discovery Around Manifest Resolvers

Status: complete

1. Refactor `path:internal/cliinstall` so both scenarios and resources resolve from manifest data.
   - completed for resource discovery and ensure/install flows
2. Keep scenario discovery behavior as the reference model.
   - completed; resource discovery now mirrors the scenario resolver shape
3. Replace `DiscoverResourceCLI` hardcoding with adapter-aware manifest discovery.
   - completed for `go_module` and `shell_script`
4. Centralize adapter-specific requirements:
   - required files
   - install mechanics
   - fingerprint inputs
   - freshness inputs
   - completed inside `path:internal/cliinstall`
5. Remove layout-driven resource discovery logic once manifest-based discovery is in place.
   - completed for `DiscoverResourceCLI`

## Phase 4: Replace Layout-Based Setup Freshness Logic

Status: complete

1. Update setup freshness checks so `"type": "cli"` is evaluated from manifest data.
   - completed in both the Go-native lifecycle evaluator and the shell compatibility layer
2. For `go_module` adapters, watch:
   - declared module dir
   - `go.mod`
   - `go.sum`
   - relevant replace targets
   - completed
3. For `shell_script` adapters, watch:
   - declared `script_path`
   - declared `install_script`
   - declared freshness inputs
   - completed
4. Stop assuming every CLI lives at `app_root/cli` or is implemented in Go.
   - completed
5. Keep setup behavior aligned with runtime behavior so one system does not declare a CLI current while the other considers it stale.
   - completed for the current CLI setup-check surface

## Phase 5: Remove Hardcoded Special-Case CLI Locators

Status: complete

1. Replace fixed-path helpers for:
   - `test-genie`
   - `scenario-completeness-scoring`
   - completed
2. Route these through the same manifest-driven resolution service used by `vrooli`.
   - completed by reusing the shared scenario CLI resolver
3. If a bootstrap-only fallback is temporarily necessary, isolate it behind an explicit compatibility layer and document the exit condition.
   - no additional fallback retained beyond the explicit `VROOLI_TEST_GENIE_CLI` override
4. Delete repo-path assumptions like:
   - `path:scenarios/test-genie/cli/test-genie`
   - `path:scenarios/scenario-completeness-scoring/cli/scenario-completeness-scoring`
   - completed for `path:internal/scenarioexec`

## Phase 6: Make Validators And Test Utilities Manifest-Aware

Status: substantially complete

1. Update any runtime detector that currently infers Go from:
   - `path:cli/go.mod`
   - `path:cli/*.go`
   - completed for the active `test-genie` runtime detector path
2. Update validation scripts that only search for `**/cli/go.mod`.
   - completed in `path:scripts/validate-go-cli-consumers.sh`
3. Replace resource compatibility tests that enforce undeclared file-layout contracts.
   - completed in `path:internal/resources/resource_cli_compat_test.go`
4. Ensure structure validation checks the declared adapter contract instead of generic folder presence.
   - completed for the main `test-genie` structure validation path
5. Keep validation messages oriented around manifest remediation, not layout folklore.
   - substantially complete; remaining references are fixture-specific or documentary rather than core validator behavior

## Phase 7: Decide Whether Invocation Should Stay Narrow

The repo is adapter-aware for install and freshness, but invocation is still effectively standardized.

Status: resolved

1. `cli.invoke.kind = installed_command` is a deliberate platform rule for now.
2. Keep validation strict around that rule.
3. Do not describe the current platform as fully invocation-agnostic.
4. Any future broadening of invoke kinds should be treated as a new platform decision, not as unfinished migration work hidden inside this plan.

## Phase 8: Add Migration Coverage Before Deleting Compatibility Paths

Status: substantially complete

1. Add scenario tests for:
   - `go_module` adapter resolution
   - `shell_script` adapter resolution
   - stale reinstall behavior
   - completed for the main shared resolution paths
2. Add resource tests for every supported resource adapter.
   - substantially complete for the active manifest-native drivers, including external-cli install/readiness regression coverage
3. Add setup freshness tests by adapter kind.
   - completed for the current setup-check surface
4. Add handler-level tests for commands that call other scenario CLIs through shared resolution paths.
   - completed for the main scenario CLI resolution flow and shared runtime helpers
5. Add failure-path tests for malformed or incomplete CLI manifests.
   - completed on the main manifest load/validation paths

## Phase 9: Migrate Manifests And Remove Dead Compatibility Code

Status: mostly complete

1. Update resource manifests to declare CLI explicitly.
   - completed
2. Remove tests and scripts that enforce undeclared legacy layout contracts.
   - mostly complete; remaining uses are targeted fixtures or archival artifacts
3. Remove special-case fallback logic once equivalent manifest-driven coverage exists.
   - completed for the active runtime path
4. Delete any now-obsolete docs that describe filesystem inference as part of the platform contract.
   - partially complete; some longer-form docs and archived artifacts remain
5. Keep cleanup strict: do not leave silent legacy inference behind "just in case".
   - completed for the active runtime path

## Phase 10: Validation

Status: key acceptance validated

1. Unit tests:
   - manifest parsing
   - adapter validation
   - install resolution
   - freshness evaluation
   - setup checks
   - passing for the migrated core packages
2. Integration tests:
   - `vrooli scenario start` with a Go CLI scenario
   - `vrooli scenario start` with a shell-script CLI scenario
   - `vrooli scenario test` on a shell-script scenario such as `scenario-auditor`
   - resource CLI ensure/install flow for every supported adapter
   - key shell-script acceptance case validated with `vrooli scenario test scenario-auditor`
   - external-cli ensure/install/readiness flow validated through `opencode`
3. Repo-wide checks:
   - no core runtime path infers CLI type from filesystem layout
   - no setup path assumes `app_root/cli` generically
   - no resource CLI discovery path requires `path:resources/<name>/cli/go.mod` unless the manifest adapter declares that layout
   - satisfied for the active runtime path

## Remaining Work

The migration work itself is functionally complete on active runtime paths. What remains is cleanup and polish rather than core implementation:

- remove or refresh stale explanatory docs that still talk about historical `path:cli/go.mod`, `path:cli/<scenario-name>`, or `path:resources/<name>/cli` assumptions as if they were current platform contract
- keep fixture-specific tests that intentionally exercise legacy Go-module repair behavior, but avoid broadening those patterns back into general guidance
- maintain normal repo hygiene around newly added validation/strategy/plan files and generated artifacts in the working tree
4. Operator validation:
   - clean install
   - CLI ensure-install on demand
   - expected commands available in `~/.vrooli/bin`
   - on-demand ensure/install validated for the scenario acceptance case and the `opencode` resource flow

## Current State

As of 2026-04-16, the migration is functionally complete for the active runtime path.

- phases 1-5 are complete
- phase 6 is substantially complete
- phase 7 is complete by explicit policy
- phase 8 is substantially complete
- phase 9 is mostly complete
- phase 10 has validated the key acceptance path

The main remaining work is cleanup, not architecture:

- remove or soften the remaining fixture-specific wording that can be misread as platform contract
- clean up residual historical documentation and archived review artifacts
- make sure the validator script introduced during migration is tracked and maintained like the rest of the repo tooling

## Recommended Execution Order

1. Freeze the contract and write the short design note
2. Add the resource CLI schema and manifest types
3. Refactor `path:internal/cliinstall` to use manifest-driven resource discovery
4. Replace setup freshness layout assumptions
5. Remove hardcoded special-case scenario CLI locators
6. Update validators, test utilities, and compatibility tests
7. Decide and implement final `cli.invoke.kind` policy
8. Migrate resource manifests and remove dead compatibility code
9. Run full validation

## Acceptance Criteria

- Scenario CLI behavior is driven only by `service.json`
- Resource CLI behavior is driven only by manifest data
- No runtime code infers CLI type from undeclared filesystem layout
- Setup freshness logic matches runtime adapter behavior
- Validation and tests consume the same CLI contract as runtime
- The remaining invocation model is explicit platform policy, not an accidental leftover
- `vrooli scenario test scenario-auditor` succeeds using the declared shell-script CLI contract

## Main Risk

The main failure mode is partial migration:

- scenario runtime uses manifests
- resources still use layout inference
- setup still watches `path:cli/go.mod`
- validators still enforce old file patterns

That state is worse than either a clean old model or a clean new model because it produces contradictory behavior. The cutover should therefore be completed in dependency order and cleaned up aggressively once the new contract is in place.
