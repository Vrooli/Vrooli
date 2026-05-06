# Personal Home Path Portability Implementation Plan

## Purpose

Remove committed `/home/matthalloran8/...` path assumptions from active Vrooli code, tests, templates, and local/generated state, then enforce a repo-level guardrail so personal absolute paths cannot be reintroduced.

This plan is intentionally greenfield: fixes must remove the personal path dependency outright. Do not preserve `/home/matthalloran8` as a fallback, compatibility path, example default, or hidden escape hatch.

## Required Reading

Run these before implementation:

```bash
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Repo-contract package references:

```bash
sed -n '1,260p' packages/repo-contract-go/resolve.go
sed -n '1,220p' packages/repo-contract-go/paths.go
sed -n '1,220p' .vrooli/repo-contract.json
sed -n '1,120p' internal/repocontractcheck/checks.go
```

Discovery command used for baseline:

```bash
rg -n --hidden --glob '!**/.git/**' --glob '!**/node_modules/**' --glob '!**/.venv/**' --glob '!**/vendor/**' --glob '!**/dist/**' --glob '!**/build/**' --glob '!**/coverage/**' --glob '!**/.cache/**' --glob '!**/tmp/**' --glob '!**/temp/**' --glob '!**/logs/**' --glob '!**/*.log' -F '/home/matthalloran8' .
```

## Problem Statement

The repository contains committed literal paths rooted at `/home/matthalloran8`. Some are active runtime code and templates that make Vrooli non-portable. Others are tests or generated artifacts that leak the operator's machine path into fixtures, logs, or persisted state.

This conflicts with the repo contract's role as the source of truth for the clone's structure. Active code should discover repo, scenario, resource, template, and well-known paths through `github.com/vrooli/repo-contract-go` where appropriate, or through script-local relative paths for shell/JS utilities where importing Go is not the right boundary.

## Greenfield Constraint

- Do not add fallback behavior to `/home/matthalloran8`, `$HOME/Vrooli`, or any other operator-specific path.
- Do not replace `/home/matthalloran8/Vrooli` with another assumed clone location such as `/workspace/Vrooli` in runtime code.
- Do not hide the string in constants, env defaults, comments, or tests except in the guardrail/code-smell rules that intentionally detect it.
- If a path is persisted local state, regenerate or store a portable representation instead of keeping a compatibility reader for the old absolute value.

## Scope

In scope:

- Active Go runtime code with literal `/home/matthalloran8`.
- Shell and JS helpers/templates with literal repo paths.
- Test fixtures that use the real operator path when a neutral fixture is enough.
- Committed generated/local state that should not carry machine paths.
- Repo-contract validation guardrail in `internal/repocontractcheck`.
- Validation commands proving the literal path is gone from active surfaces and guarded in CI/local validation.

Out of scope:

- Intentional branding references to Matt Halloran, GitHub profile, funding links, press-kit identity, and contact identity.
- A full cleanup of all historical narrative documents unless they block the new guardrail's active-surface scan.
- Introducing new dependencies.

## Current Technical Context

Repo-contract primitives already exist in `packages/repo-contract-go`:

- `repocontract.LoadDefaultFromEnvOrCWD()` resolves env/CWD/executable and loads `.vrooli/repo-contract.json`.
- `repocontract.ResolveRepoRoot()` resolves the active repo root.
- `repocontract.ResolveScenarioPath(repoRoot, scenario)` and `contract.ScenarioRoot(repoRoot, scenario)` resolve scenario roots.
- `repocontract.ResolveScenarioFile(repoRoot, scenario, key)` and `contract.ScenarioFile(repoRoot, scenario, key)` resolve well-known scenario paths.
- `contract.TopLevelDir(repoRoot, "scenarios"|"resources"|"templates"|...)` resolves top-level contract dirs.

`.vrooli/repo-contract.json` currently defines:

- top-level dirs: `.vrooli`, `scenarios`, `resources`, `templates`, `packages`, `cmd`, `internal`, `docs`
- scenario well-known paths: `service`, `docs`, `requirements`, `api`, `ui`, `cli`, `initialization`
- resource well-known paths: `manifest`, `docs`, `initialization`

`internal/repocontractcheck/checks.go` already validates repo-contract drift and scans Go code for some adoption violations. `make validate-repo-contract` is documented as the CI/automation entrypoint.

Baseline scan found these high-signal active/local-state categories:

- Runtime Go:
  - `scenarios/scenario-to-ios/api/main.go`
  - `scenarios/scenario-to-ios/api/template.go`
  - `scenarios/algorithm-library/apply_migration.go`
- Shell:
  - `scenarios/api-library/scripts/init-endpoints.sh`
  - `resources/unstructured-io/test_debug.sh`
  - `resources/sagemath/test/phases/test-integration.sh`
- JS/template:
  - `templates/scenarios/react-vite/ui/perf/capture.template.js`
  - `scratch/perf-spike/capture.js`
- Config/local state:
  - `scenarios/browser-automation-studio/.vrooli/desktop-config.json`
  - `.vrooli/resources/archived-blueprint-resources.json`
  - `.vrooli/resources/deprecated-resources.json`
  - `scenarios/scenario-dependency-analyzer/.vrooli/service.json`
  - `scenarios/landing-manager/api/analytics/events.json`
  - `scenarios/browser-automation-studio/playwright-driver/test-coverage.txt`
- Tests/fixtures:
  - prompt-manager memberflow real-store tests
  - landing-manager UI tests
  - secrets-manager PII pattern tests
  - web-console TTS normalization tests

## Target End State

- No active runtime source, template, shell helper, committed local state, or neutral test fixture contains `/home/matthalloran8`.
- Runtime Go uses repo-contract helpers for repo/scenario/resource/template locations.
- Shell helpers derive paths from script location or explicit env inputs, with no personal fallback.
- JS/template helpers use environment variables or relative resolution, with no personal default.
- Generated/local state either stores portable paths or is ignored/regenerated rather than committed with machine paths.
- `vrooli contract validate` / `make validate-repo-contract` fails with clear file:line output if a personal absolute path appears in an active scanned surface.

## Implementation Strategy

### Phase 1: Tighten Inventory And Decide Active Surface

1. Re-run the baseline literal scan.
2. Classify each remaining hit as:
   - runtime source
   - test fixture
   - template/helper
   - generated/local state
   - intentional detector fixture
   - historical artifact/docs
3. Define the guardrail scan's active-surface include/exclude rules before editing, so implementation and validation converge on the same policy.

Recommended active-surface include roots:

- `cmd`
- `internal`
- `packages`
- `resources`
- `scenarios`
- `templates`
- `.vrooli`

Recommended guardrail exclusions:

- `.git`, `node_modules`, `.venv`, `vendor`, `dist`, `build`, `coverage`, `.cache`, `tmp`, `temp`, `logs`
- `**/*.log`
- `**/.swarm/**`
- `**/review/**`
- `**/evidence/**`
- `**/captures/**`
- `**/handoff/**`
- `**/acceptance-validation.json`
- historical docs/plans if the team decides those remain archival

Allowed active exceptions should be narrow and named:

- `scenarios/code-smell/initialization/rules/vrooli-specific.yaml`, because it intentionally detects the smell.
- New `internal/repocontractcheck` test fixtures that intentionally inject `/home/matthalloran8` into a temporary fixture repo to prove the guardrail fails.

### Phase 2: Replace Runtime Go With Repo Contract

1. Update `scenarios/scenario-to-ios/api/main.go`.
   - Add `repocontract "github.com/vrooli/repo-contract-go"`.
   - Resolve `contract, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD()`.
   - Use `contract.ScenarioRoot(repoRoot, scenarioName)` or `repocontract.ScenarioExists(repoRoot, scenarioName)`.
   - Return a normal API error if repo contract discovery fails.
   - Do not fall back to `/home/...` or `$HOME/Vrooli`.

2. Update `scenarios/scenario-to-ios/api/template.go`.
   - Change `NewTemplateExpander(scenarioName string)` to accept a resolved `scenarioPath`, or add a constructor that returns `(*TemplateExpander, error)` after repo-contract resolution.
   - Prefer passing the already-resolved scenario path from `handleBuild` to avoid duplicated root discovery.
   - Keep `getProjectRoot()` only if it is already scenario-local and not a repo-root detector. If it uses ad hoc repo-root detection, replace it with repo-contract resolution or scenario-local paths.

3. Update `scenarios/algorithm-library/apply_migration.go`.
   - Add repo-contract import.
   - Resolve `contract, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD()`.
   - Use `contract.ScenarioFile(repoRoot, "algorithm-library", "initialization")`.
   - Join `postgres/migration_003_problem_mapping.sql`.
   - Replace deprecated `ioutil.ReadFile` with `os.ReadFile` while touching the file.

4. Update prompt-manager real-store tests:
   - Use repo-contract to resolve `prompt-manager` scenario root and join `store`.
   - Keep the existing skip behavior when the real store is absent.
   - Candidate files:
     - `scenarios/prompt-manager/api/memberflow/classifier_purity_test.go`
     - `scenarios/prompt-manager/api/memberflow/writer_skill_registry_test.go`
     - `scenarios/prompt-manager/api/memberflow/validation_test.go`
     - `scenarios/prompt-manager/api/memberflow/runtime_attribution_test.go`

### Phase 3: Replace Shell And JS Paths With Local Relative Or Env-Based Resolution

1. Update `scenarios/api-library/scripts/init-endpoints.sh`.
   - Add `SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"`.
   - Set `SEED_FILE="${SCRIPT_DIR}/../initialization/postgres/seed_endpoints.sql"` or equivalent after checking the script's actual relative location.
   - Redirect from `"${SEED_FILE}"`.
   - Do not introduce `VROOLI_ROOT` unless needed; the seed file is scenario-local.

2. Update `resources/unstructured-io/test_debug.sh`.
   - Derive `RESOURCE_DIR` from `BASH_SOURCE[0]`.
   - Derive `REPO_ROOT` by walking from `RESOURCE_DIR` to the repo root or by using `VROOLI_ROOT` as an explicit override with script-relative fallback.
   - Keep all sources based on derived variables.

3. Update `resources/sagemath/test/phases/test-integration.sh`.
   - Use existing `RESOURCE_DIR` if already available in the script; otherwise derive it from `BASH_SOURCE[0]`.
   - Invoke `"${RESOURCE_DIR}/lib/gpu.sh" check`.

4. Update `templates/scenarios/react-vite/ui/perf/capture.template.js`.
   - Prefer `process.env.BAS_NODE_MODULES`.
   - If unset, resolve from `process.env.VROOLI_ROOT` or `process.env.VROOLI_SOURCE_ROOT`.
   - Fail with an actionable error if neither env value is present.
   - Do not default to a personal path.

5. Update `scratch/perf-spike/capture.js` or delete it if obsolete.
   - If kept, mirror the template's env-based BAS resolution.
   - If deleted, verify no docs/scripts still instruct users to run it.

### Phase 4: Clean Or Regenerate Local/Generated State

1. `.vrooli/resources/archived-blueprint-resources.json` and `.vrooli/resources/deprecated-resources.json`.
   - Replace absolute archive paths with one of:
     - `~/.vrooli/archive/...` if the archive is intentionally user-home state.
     - `${VROOLI_HOME}/archive/...` if Vrooli has or should define a home env var.
     - A relative archive ID/path if consumers resolve it elsewhere.
   - Inspect readers before changing to ensure they expand the chosen representation.
   - Do not keep support for the old absolute path except through generic `~` or env expansion.

2. `scenarios/browser-automation-studio/.vrooli/desktop-config.json`.
   - Determine whether this file is committed product config or per-clone generated config.
   - If product config: store relative paths from scenario root, e.g. `ui/dist/...` and `platforms/electron/bundle/bundle.json`, and update consumers to resolve relative values against the scenario root.
   - If per-clone generated: remove from committed state and add the generated path to `.gitignore`, then ensure setup/lifecycle regenerates it.

3. `scenarios/scenario-dependency-analyzer/.vrooli/service.json`.
   - Replace the verification fallback message with a portable value:
     - `echo "  ${VROOLI_HOME:-$HOME/.vrooli}/bin/scenario-dependency-analyzer"` if shell expansion is desired.
     - Or better, avoid printing a guessed path at all and print a clear error: `scenario-dependency-analyzer not found on PATH`.

4. Runtime captures and analytics:
   - `scenarios/landing-manager/api/analytics/events.json`
   - `scenarios/browser-automation-studio/playwright-driver/test-coverage.txt`
   - These should not preserve machine-local absolute paths. Prefer removing from committed state and ignoring regenerated files. If they must stay as fixtures, sanitize to `<repo-root>/...`.

### Phase 5: Neutralize Test Fixtures

1. `scenarios/secrets-manager/api/pii_patterns_test.go`.
   - Replace `/home/matthalloran8/data` with `/home/alice/data`.
   - Keep the detector behavior identical.

2. `scenarios/landing-manager/ui/src/pages/TemplatePreviewLinks.test.tsx`.
   - Replace generated scenario paths with neutral fixture paths such as `/tmp/vrooli-test/scenarios/...` or `<repo-root>/scenarios/...`.
   - Prefer `<repo-root>` if the UI only displays/parses path text and does not require an actual filesystem path.

3. `scenarios/web-console/api/tts_normalizer_test.go`.
   - Replace the real path in the assistant-message fixture with a neutral Unix path.
   - Keep the assertion checking that full paths are reduced to basenames.

4. `scenarios/web-console/ui/src/__tests__/terminal-pane-auto-tts.test.tsx`.
   - Replace the real path with a neutral path.
   - Keep the markdown-vs-terminal matching behavior under test.

5. `scenarios/vrooli-autoheal/api/internal/checks/infra/display_test.go`, if included in the broader identity cleanup.
   - Use a fake username such as `alice` in mocked commands.
   - Ensure production code gets user identity from the system/runtime, not hard-coded fixtures.

### Phase 6: Add Repo-Contract Guardrail

1. Add a new check in `internal/repocontractcheck/checks.go`, for example:
   - name: `personal_absolute_paths`
   - function: `checkNoPersonalAbsolutePaths`

2. Implement scanner behavior:
   - Walk the active-surface roots from Phase 1.
   - Read text files only. Skip binaries either by extension or by checking for NUL bytes.
   - Match literal `/home/matthalloran8` with file and line reporting.
   - Return a sorted, deterministic error message.

3. Add path allowlist helpers:
   - `isPersonalPathScanSkipped(rel string) bool`
   - Skip generated/history directories listed in Phase 1.
   - Explicitly allow only intentional detector rules and guardrail tests.

4. Add tests in `internal/repocontractcheck/checks_test.go`.
   - Fails when a fixture repo contains `/home/matthalloran8/Vrooli` in active Go source.
   - Fails when a fixture repo contains it in an active shell/JS/config file.
   - Does not fail for the intentional code-smell rule path.
   - Does not fail for ignored generated/history paths.
   - Verifies the error message includes relative path and line number.

5. Ensure `Run` includes the new check before or near `adoption_rules_alignment`.

## Contract Decisions

- Runtime Go path resolution contract: use `repo-contract-go`, not ad hoc root detection.
- Shell path resolution contract: use script-local relative paths for scenario/resource-local files; use explicit env only where the target is outside the script's local tree.
- JS/template path resolution contract: use explicit env (`BAS_NODE_MODULES`, `VROOLI_ROOT`, or `VROOLI_SOURCE_ROOT`) and fail clearly when missing.
- Persisted local state contract: store relative paths, `~`, or named env placeholders; do not store absolute operator home paths.
- Guardrail contract: active committed surfaces must not contain `/home/matthalloran8` except narrow intentional detector fixtures.

Potential repo-contract extension:

- If prompt-manager store resolution appears repeatedly, add `store` to `scenario.well_known_paths` in `.vrooli/repo-contract.json` and update schema/tests/docs as needed. Otherwise, join `store` locally after resolving the `prompt-manager` scenario root.

## Testing Plan

Targeted tests:

```bash
go test ./internal/repocontractcheck
go test ./scenarios/scenario-to-ios/api
go test ./scenarios/algorithm-library
go test ./scenarios/prompt-manager/api/memberflow
go test ./scenarios/secrets-manager/api
go test ./scenarios/web-console/api
```

Scenario/UI tests where applicable:

```bash
cd scenarios/landing-manager && make test
cd scenarios/web-console && make test
```

Repo contract validation:

```bash
make validate-repo-contract
vrooli contract validate
```

Literal path verification:

```bash
rg -n --hidden --glob '!**/.git/**' --glob '!**/node_modules/**' --glob '!**/.venv/**' --glob '!**/vendor/**' --glob '!**/dist/**' --glob '!**/build/**' --glob '!**/coverage/**' --glob '!**/.cache/**' --glob '!**/tmp/**' --glob '!**/temp/**' --glob '!**/logs/**' --glob '!**/*.log' -F '/home/matthalloran8' .
```

Expected final output:

- Either no matches, or only explicitly allowed detector/guardrail fixtures.
- `make validate-repo-contract` must fail if a new active-surface match is added.

## Rollout / Validation Checklist

1. Baseline scan output saved in implementation notes or PR description.
2. Runtime Go paths replaced with repo-contract helpers.
3. Shell helpers no longer contain personal paths.
4. JS/template helpers no longer contain personal paths.
5. Local/generated state cleaned, ignored, or made portable.
6. Tests use neutral fixture paths.
7. `internal/repocontractcheck` has guardrail coverage.
8. `make validate-repo-contract` passes on the cleaned repo.
9. Literal path scan shows no unexpected `/home/matthalloran8` matches.
10. `git diff` reviewed to ensure no unrelated user changes were reverted.

## Risks And Mitigations

- Risk: over-broad guardrail fails on historical artifacts.
  - Mitigation: define active-surface scan and explicit archival skips first; keep allowlist narrow and documented.

- Risk: generated/local state has consumers that expect absolute paths.
  - Mitigation: inspect readers before changing persisted representation; update readers to resolve relative or placeholder paths generically.

- Risk: replacing shell paths with repo-root discovery recreates an ad hoc root detector.
  - Mitigation: prefer script-local relative paths when the target is scenario/resource-local.

- Risk: repo-contract use in tests makes unit tests depend on live repo structure.
  - Mitigation: keep existing skip-on-missing behavior and use test fixtures where possible.

- Risk: guardrail itself needs to contain the forbidden literal for tests.
  - Mitigation: keep those occurrences isolated to `internal/repocontractcheck` test fixtures and document the allowlist.

## Non-Goals / Prohibited Patterns

- No fallback to `/home/matthalloran8`.
- No fallback to `$HOME/Vrooli`.
- No new package dependencies.
- No broad rewrite of historical docs or branding references.
- No mass automated rewrite without reviewing each file category.
- No direct scenario execution for validation; use scenario Makefiles or `vrooli scenario ...` commands.

## Definition Of Done

- All active runtime code resolves repo/scenario/resource paths through repo-contract helpers or local relative script logic.
- No active shell, JS, Go, JSON, YAML, or TS/TSX source relies on `/home/matthalloran8`.
- Test fixtures use neutral identities/paths unless intentionally testing the guardrail.
- Generated/local files no longer commit operator-specific absolute paths, or are ignored/regenerated.
- `internal/repocontractcheck` enforces `personal_absolute_paths` with deterministic file:line reporting.
- `go test ./internal/repocontractcheck` passes.
- `make validate-repo-contract` passes on the cleaned repo and fails on an injected active-surface personal path.
- Final literal scan has only approved intentional matches, if any.
