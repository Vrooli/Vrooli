# React-Vite Template Validation and Polish Plan

**Plan file**: `path:docs/plans/react-vite-template-polish-and-drift-reduction-plan.md`  
**Revised**: 2026-05-06  
**Owner**: meta-optimization / template-validation  
**Primary target**: `path:templates/scenarios/react-vite/`  
**Supporting targets**:

- `path:internal/cli/scenariohandlers/template_runtime.go`
- `path:internal/cli/scenariocli/template_parsers.go`
- `path:scenarios/test-genie/`

**Status**: implemented 2026-05-06; keep as the validation contract and
historical implementation record

---

## 1. Purpose

`templates/scenarios/react-vite` is the main scenario template for new
React/Vite scenarios. It is already in strong shape: Connect-RPC is the shared
wire contract, API logic is domain-owned, CLI/UI surfaces are thin, and the
template includes useful docs and tests.

The remaining highest-leverage work is not adding more generated application
features. It is making template validation strong enough that a future agent can
ask one question and get a trustworthy answer:

> If a real scenario is generated from this template right now, does it work
> cleanly from first run?

This plan therefore focuses on adding first-class shallow/deep validation to the
template validation command, where deep validation generates a temporary real
scenario and runs test-genie against that generated output.

---

## 2. Required Reading

Run before implementing:

```bash
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Read these files before editing:

- `path:templates/scenarios/react-vite/template.json`
- `path:templates/scenarios/react-vite/docs/START-HERE.md`
- `path:templates/scenarios/react-vite/docs/internal/TEMPLATE-GENERATION-CONTRACT.md`
- `path:templates/scenarios/react-vite/docs/internal/TEMPLATE-MAINTENANCE.md`
- `path:templates/scenarios/react-vite/docs/internal/TESTING.md`
- `path:templates/scenarios/react-vite/.vrooli/service.json`
- `path:templates/scenarios/react-vite/.vrooli/testing.json`
- `path:internal/cli/scenariohandlers/template_runtime.go`
- `path:internal/cli/scenariocli/template_parsers.go`
- `path:internal/cli/scenariocli/help.go`
- `path:internal/cli/scenariocli/scenariocli_test.go`
- `path:scenarios/test-genie/cli/execute/command.go`
- `path:scenarios/test-genie/cli/internal/execute/types.go`
- `path:scenarios/test-genie/api/internal/orchestrator/workspace/workspace.go`
- `path:scenarios/test-genie/api/internal/orchestrator/plan_preview.go`
- `path:scenarios/test-genie/api/internal/orchestrator/suite_execution.go`

Current docs note: `docs/internal/REPLACING-NOTES.md` no longer exists in the
template. Replacement and cleanup guidance now belongs in
`docs/START-HERE.md`, with template-only generation rules in
`docs/internal/TEMPLATE-GENERATION-CONTRACT.md`.

---

## 3. Hard Rules

1. Do not implement a project-level command for adding domains to scenarios.
   Different templates may use different languages, frameworks, layouts, and
   architectural choices. A generic domain scaffold command would hard-code
   assumptions that only fit this template.
2. Do not add generated CLI command metadata work to this plan. It may be a
   separate future idea, but it is outside this implementation.
3. Do not add an attachments reference-scope phase. It is not part of this
   validation plan.
4. Do not do broad standalone CI or Git provider workflow work. No Git
   operations are part of this plan.
5. Do not install new third-party dependencies without explicit permission.
6. Preserve the default developer path: normal validation must stay fast enough
   for routine local use.
7. Deep validation must be automated. A manual smoke checklist is not an
   acceptable substitute.
8. Temporary generated scenarios must be cleaned up by default, including
   generated proto relocation outputs. Keep a retention flag only for debugging.

---

## 4. Problem Statement

`vrooli scenario template validate` currently catches useful template issues:

- malformed or missing `template.json`
- unresolved placeholders after copying
- default design copy problems
- relocation source problems
- generated-scenario checks in a temporary directory
- selected generated API/CLI/UI commands when subprocess execution is available

That is valuable, but it is still a template-local validator. It does not expose
clear validation modes, and it does not run the same ground-truth suite that a
real scenario uses through test-genie.

The gap is especially important because this template can be copied hundreds or
thousands of times. Small omissions in setup, docs, generated paths, lifecycle
configuration, lint/test expectations, or template placeholders multiply.

---

## 5. Scope

### In Scope

- Add explicit validation modes to `vrooli scenario template validate`.
- Keep the existing behavior as the default shallow mode, with focused
  improvements where needed.
- Add a deep mode that:
  - generates a real scenario from `react-vite` into a temporary workspace,
  - replaces placeholders using deterministic validation seed values,
  - performs template relocations and post-generation setup needed for a real
    first-run scenario,
  - runs test-genie against the generated scenario path,
  - reports failures through the template validation report,
  - cleans temporary workspace and relocation outputs by default.
- Add any small test-genie CLI/API support needed to run against an explicit
  generated scenario path.
- Update template docs so maintainers know when to use shallow vs deep
  validation.
- Apply only small template polish items that are directly validated by the new
  validation flow or remove known first-run confusion.

### Out of Scope

- Generic scenario domain scaffolding.
- Template-defined "how to add a domain" metadata in `template.json`.
- Generated CLI command metadata.
- Attachments reference architecture decisions.
- Generated scenario migration support.
- Broad Git provider workflow validation.
- Any compatibility layer for old generated scenarios.

---

## 6. Current Technical Context

### Template Validation CLI

`internal/cli/scenariohandlers/template_runtime.go` owns
`runTemplateValidate`. Current validation loads every template, validates the
template source, copies a generated scenario to an OS temp directory, verifies
placeholders, runs relocations with `template-validation-*` seed values, cleans
relocation targets, and calls `validateGeneratedScenario`.

`internal/cli/scenariocli/template_parsers.go` currently defines
`scenario template validate` without validate-specific flags.

### Test-Genie Path Support

`scenarios/test-genie/cli/execute/command.go` currently accepts
`test-genie execute <scenario> ...` and sets `ScenarioPath` by calling
`cliutil.ResolveScenarioPath(parsed.Scenario)`.

The API side already understands scenario path overrides:

- `scenarios/test-genie/cli/internal/execute/types.go` has `ScenarioPath`.
- `scenarios/test-genie/api/internal/orchestrator/workspace/workspace.go` has
  `NewWithOverride`.
- `scenarios/test-genie/api/internal/orchestrator/plan_preview.go` passes
  `req.ScenarioPath` to `NewWithOverride`.
- `scenarios/test-genie/api/internal/orchestrator/suite_execution.go` carries
  `ScenarioPath` in execution requests.

Likely implementation implication: test-genie may only need an execute CLI flag
such as `--scenario-path <abs-path>`, plus tests and help text, rather than a
large API change.

### Current Template Docs

The current template docs under `templates/scenarios/react-vite/docs` are:

- `QUICKSTART.md`
- `START-HERE.md`
- `concepts/ARCHITECTURE.md`
- `guides/troubleshooting.md`
- `internal/ERROR-HANDLING.md`
- `internal/PROBLEMS.md`
- `internal/PROGRESS.md`
- `internal/SEAMS.md`
- `internal/TEMPLATE-GENERATION-CONTRACT.md`
- `internal/TEMPLATE-MAINTENANCE.md`
- `internal/TESTING.md`
- `manifest.json`
- `reference/api-endpoints.md`
- `reference/cli-commands.md`
- `reference/configuration.md`

Use those current files. Do not reference removed docs.

---

## 7. Target End State

The command surface should support this operational shape:

```bash
vrooli scenario template validate
vrooli scenario template validate --mode shallow
vrooli scenario template validate --mode deep --template react-vite
vrooli scenario template validate --mode deep --template react-vite --retain-temp
vrooli scenario template validate --mode deep --template react-vite --test-preset comprehensive
```

Exact flag names may be adjusted to match local CLI conventions, but these
contracts must hold:

- Default mode is shallow.
- Shallow mode is compatible with the current command's intent.
- Deep mode is a built-in command path, not a documented manual checklist.
- Deep mode generates a real temporary scenario and runs test-genie on that
  generated scenario.
- Human output gives a concise status, grouped findings, and next steps.
- JSON output includes mode, template name, generated scenario id, retained temp
  path when applicable, test-genie preset, per-phase failures when available, and
  cleanup status.

---

## 8. Implementation Strategy

### Phase 0: Re-Baseline the Current State

Before editing, rerun a short discovery pass:

```bash
find templates/scenarios/react-vite/docs -maxdepth 3 -type f | sort
rg -n "func runTemplateValidate|validateGeneratedScenario|TemplateValidateRequest|TemplateValidationReport" internal/cli
rg -n "ScenarioPath|NewWithOverride|ResolveScenarioPath|test-genie execute" scenarios/test-genie/cli scenarios/test-genie/api/internal
```

Record any surprise in this plan before implementation. In particular, verify
whether test-genie already has a usable explicit scenario path flag by the time
implementation starts.

### Phase 1: Add Validate Mode Contracts

Files:

- `path:internal/cli/scenariocli/template_parsers.go`
- `path:internal/cli/scenariocli/help.go`
- `path:internal/cli/scenariocli/scenariocli_test.go`
- request/report types in `path:internal/cli/scenariocli/`
- handler tests in `path:internal/cli/scenariohandlers/`

Implementation:

1. Extend `TemplateValidateRequest` with:
   - `Mode` using a constrained enum or string constant set: `shallow`, `deep`.
   - optional `TemplateName` filter.
   - optional `RetainTemp`.
   - optional `TestPreset`, defaulting to the repo's standard comprehensive
     test-genie preset for deep validation.
2. Extend parser/help to accept:
   - `--mode shallow|deep`
   - `--template <name>`
   - `--retain-temp`
   - `--test-preset <name>`
3. Reject invalid combinations early:
   - unknown mode,
   - unknown template filter,
   - deep mode with a missing test-genie command when subprocess execution is
     required.
4. Extend the validation report model with mode and deep-run metadata without
   weakening existing JSON consumers.
5. Update CLI parser/help tests so the new flags are stable.

Acceptance criteria:

- `vrooli scenario template validate` still parses with no flags.
- `--mode shallow` and `--mode deep` parse.
- invalid modes fail with a usage error.
- `--template react-vite` limits validation to that template.

### Phase 2: Make Current Validation the Shallow Mode

Files:

- `path:internal/cli/scenariohandlers/template_runtime.go`
- tests under `path:internal/cli/scenariohandlers/`

Implementation:

1. Extract current per-template validation into a named shallow validation
   function.
2. Keep the current generated-copy checks in shallow mode unless profiling shows
   they are too expensive. The key is that shallow stays fast and local; it does
   not invoke test-genie.
3. Make source validation use current docs:
   - `docs/START-HERE.md` as the generated scenario onboarding doc.
   - `docs/internal/TEMPLATE-GENERATION-CONTRACT.md` as the template-only
     generation contract.
4. Improve issue messages where they still imply stale docs or manual smoke
   steps.
5. Keep relocation cleanup behavior deterministic for seed ids with
   `template-validation-*`.

Acceptance criteria:

- Existing template validation tests still pass after being renamed or adjusted
  for shallow mode.
- Shallow mode does not mention removed docs.
- Shallow mode remains the default.

### Phase 3: Add Deep Validation Engine

Files:

- `path:internal/cli/scenariohandlers/template_runtime.go`
- helper files in `path:internal/cli/scenariohandlers/` only if extraction keeps
  the code clearer
- tests under `path:internal/cli/scenariohandlers/`

Implementation:

1. Create a deep validation path separate from shallow validation.
2. Generate a temporary workspace that looks enough like a Vrooli repository for
   generated scenario tooling to resolve paths:
   - temp root,
   - `scenarios/<generated-id>/`,
   - any minimal repo markers required by `repo-contract-go` or test-genie.
3. Use deterministic generated ids, for example:
   - template: `react-vite`
   - generated id: `template-validation-react-vite-deep`
4. Generate the scenario through existing generation functions, not by manually
   copying partial folders.
5. Run relocations using the generated id and clean all relocation targets after
   validation unless `--retain-temp` is set.
6. Run template post hooks needed for realistic first-run validation. If this is
   too expensive for every deep run, support a later opt-out flag, but the
   default deep contract should represent a real generated scenario.
7. Invoke test-genie with:

```bash
test-genie execute template-validation-react-vite-deep \
  --scenario-path <absolute-temp-root>/scenarios/template-validation-react-vite-deep \
  --preset comprehensive \
  --no-stream \
  --json
```

8. Convert test-genie failures into `TemplateValidationIssue` entries. Preserve
   enough raw metadata in JSON output for debugging.
9. In human output, show retained temp path only when retained or on failure if
   the implementation chooses to retain failed runs for debugging.

Acceptance criteria:

- Deep mode creates a generated scenario with no unresolved placeholders.
- Deep mode invokes test-genie against the generated scenario path.
- Deep mode reports test-genie failures as template validation failures.
- Deep mode cleans temp scenario and relocation outputs by default.
- `--retain-temp` leaves the temp generated scenario available and prints its
  path.

### Phase 4: Expose Explicit Scenario Path in Test-Genie CLI

Files:

- `path:scenarios/test-genie/cli/execute/command.go`
- `path:scenarios/test-genie/cli/internal/execute/types.go`
- tests under `path:scenarios/test-genie/cli/`
- API tests only if API behavior changes

Implementation:

1. Add `--scenario-path <absolute-path>` to `test-genie execute`.
2. Validate that the provided path is absolute.
3. Preserve current default behavior when the flag is omitted:
   `cliutil.ResolveScenarioPath(parsed.Scenario)`.
4. Pass the explicit path through the existing `ScenarioPath` request field.
5. Update help examples to show the temp-path use case without making template
   validation depend on prose.
6. Add parser tests and, if practical, a command-level test proving the request
   carries the explicit path.

Acceptance criteria:

- Existing `test-genie execute <scenario>` behavior is unchanged.
- `test-genie execute demo --scenario-path /tmp/root/scenarios/demo` sends that
  absolute path.
- relative `--scenario-path` values fail before API invocation.

### Phase 5: Focused Template Polish

Only do the following if still applicable after re-baselining. These are small
items because they either affect first-run correctness or prevent validation
drift.

Files depend on the finding, but likely include:

- `path:templates/scenarios/react-vite/ui/pnpm-lock.yaml`
- `path:templates/scenarios/react-vite/docs/internal/TESTING.md`
- `path:templates/scenarios/react-vite/docs/internal/TEMPLATE-MAINTENANCE.md`
- `path:templates/scenarios/react-vite/docs/START-HERE.md`
- `path:templates/scenarios/react-vite/api/internal/app/server.go`
- `path:templates/scenarios/react-vite/api/internal/app/server_test.go`
- `path:templates/scenarios/react-vite/api/internal/testutil/`

Implementation:

1. Add `ui/pnpm-lock.yaml` if the template still has `package.json` lockfile
   expectations but no lockfile. Validate with the template's existing package
   manager flow.
2. Remove or update any docs that refer to old validation, old replacement-note
   files, old lifecycle commands, or manual generated-smoke steps.
3. Add a focused guard that API mocks/test helpers cannot accidentally import
   production transport wiring if the current tests still leave that drift path
   open.
4. Add or tighten server construction validation if `server.New` can still
   accept invalid required dependencies such as a nil clock.

Acceptance criteria:

- Each polish item has a direct validation or test.
- No broad refactor is introduced under this phase.
- Docs point to shallow/deep validation, not manual smoke instructions.

### Phase 6: Documentation

Files:

- `path:templates/scenarios/react-vite/docs/internal/TEMPLATE-GENERATION-CONTRACT.md`
- `path:templates/scenarios/react-vite/docs/internal/TEMPLATE-MAINTENANCE.md`
- `path:templates/scenarios/react-vite/docs/internal/TESTING.md`
- `path:templates/scenarios/react-vite/docs/START-HERE.md`
- `path:templates/scenarios/react-vite/docs/reference/cli-commands.md`

Implementation:

1. Document the validation modes:
   - shallow for routine template source validation,
   - deep for generated-scenario first-run validation through test-genie.
2. Document expected commands:

```bash
vrooli scenario template validate --mode shallow --template react-vite
vrooli scenario template validate --mode deep --template react-vite --test-preset comprehensive
```

3. Make `TEMPLATE-GENERATION-CONTRACT.md` clear that deep validation is the
   source of truth for "does this template generate a working scenario?"
4. Keep generated scenario onboarding in `START-HERE.md`. Keep template-only
   maintenance instructions in `docs/internal/*`.
5. Remove references to removed docs and manual generated-smoke procedures.

Acceptance criteria:

- `rg "REPLACING-NOTES|generated-smoke|manual smoke" templates/scenarios/react-vite/docs`
  has no stale references unless intentionally discussing removal history.
- Docs explain how to run both validation modes.

---

## 9. Contract Decisions

1. **Default validation mode**: shallow.
2. **Deep validation scope**: one or more selected templates, with
   `react-vite` as the main target for this plan.
3. **Generated scenario id**: deterministic and prefixed with
   `template-validation-`.
4. **Temporary path behavior**: cleanup by default, retain only with
   `--retain-temp` or a deliberately documented failure-retention behavior.
5. **Test-genie preset**: comprehensive by default for deep mode unless current
   test-genie conventions point to a better all-purpose preset.
6. **Path matching**: test-genie requires `filepath.Base(scenarioPath)` to match
   `scenarioName`; deep validation must generate into a matching basename.
7. **No generic domain assumptions**: all domain-addition workflow remains human
   or template-doc guided, not a repo-level command.

---

## 10. Testing Plan

Run focused tests while implementing:

```bash
go test ./internal/cli/scenariocli ./internal/cli/scenariohandlers
cd scenarios/test-genie/cli && go test ./...
cd scenarios/test-genie/api && go test ./internal/orchestrator/... ./internal/app/httpserver/...
```

Run template validation gates:

```bash
vrooli scenario template validate --mode shallow --template react-vite
vrooli scenario template validate --mode deep --template react-vite --test-preset comprehensive
```

If the deep command is expensive, still run it before marking the plan complete.
Deep validation is the core deliverable.

For template-local polish, run the relevant generated scenario checks selected
by test-genie plus any direct package tests changed by the polish.

---

## 11. Rollout and Validation Checklist

- [x] Current docs/files re-baselined and stale references removed from this
      plan if the tree changes again.
- [x] `scenario template validate` accepts validation mode flags.
- [x] Shallow mode preserves current behavior and remains default.
- [x] Deep mode generates a real temporary scenario from `react-vite`.
- [x] Deep mode invokes test-genie with an explicit scenario path.
- [x] Test-genie execute supports `--scenario-path` or equivalent.
- [x] Deep validation failures are surfaced as template validation issues.
- [x] Deep validation cleans temp scenario and relocation outputs by default.
- [x] `--retain-temp` is available for debugging.
- [x] Template docs describe shallow/deep validation accurately.
- [x] Focused polish items have tests or are covered by deep validation.
- [x] All commands in the Testing Plan pass or failures are documented with
      concrete follow-up.

---

## 12. Risks and Mitigations

**Risk: Deep validation is slow.**  
Mitigation: keep shallow as default, make deep explicit, and allow selecting one
template via `--template react-vite`.

**Risk: Test-genie assumes scenarios live under the real repo root.**  
Mitigation: use its existing `ScenarioPath` override and generate a temp
workspace with enough repo shape for `AppRootFromScenario` and package-relative
checks. If a phase truly requires the real repo root, document and narrow that
phase rather than silently skipping all deep validation.

**Risk: Post hooks make deep validation modify shared package outputs.**  
Mitigation: use deterministic `template-validation-*` ids and reuse existing
relocation cleanup. Explicitly verify all three proto output forms when relevant:
`packages/proto/gen/go/<id>`, `packages/proto/gen/typescript/js/<id>`, and
`packages/proto/gen/python/<id_with_underscores>`.

**Risk: Human output hides useful test-genie detail.**  
Mitigation: keep human output concise, but include JSON metadata and retained
temp paths for debugging.

---

## 13. Non-Goals and Prohibited Patterns

- No generic domain scaffolding command.
- No `template.json` domain-addition metadata.
- No generated CLI command metadata work.
- No attachments reference-scope decision.
- No broad CI/Git provider workflow work.
- No manual-only smoke validation as the plan's endpoint.
- No new dependencies without explicit permission.
- No direct execution of scenario binaries outside lifecycle/test-genie-managed
  paths.
- No compatibility layers for old generated scenarios.

---

## 14. Definition of Done

This plan is complete when:

1. `vrooli scenario template validate` has explicit shallow/deep modes.
2. Shallow mode preserves and clarifies the current validation behavior.
3. Deep mode generates a temporary real scenario from `react-vite` and runs
   test-genie against that generated scenario path.
4. Test-genie can execute against an explicit absolute scenario path from its
   CLI.
5. Temp files and relocation outputs are cleaned by default, with a retained
   debug path available.
6. Current template docs describe the real validation workflow and do not point
   to removed docs.
7. Focused first-run polish items are either fixed with tests or explicitly
   ruled out after re-baselining.
8. The Testing Plan commands pass, or any remaining failure is documented with a
   concrete reason and next action.
