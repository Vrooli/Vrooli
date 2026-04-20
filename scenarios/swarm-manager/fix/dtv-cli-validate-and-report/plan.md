# Implementation Plan: DTV CLI Commands — validate, report, expectation

## 1. Purpose

Add three missing command groups to the `development-toolchain-validator` CLI so the meta-optimization team can invoke DTV end-to-end from the command line, plus the small API handler that makes `validate` reachable:

- `validate <reference>` — trigger a synchronous validation run and display the report.
- `report {conflicts|drift|maturity|tool-baselines}` — surface cross-skill reports.
- `expectation {structural|cli} {list|create|get|delete}` — CRUD over expectation records.
- `POST /api/v1/references/{id}/validate` — thin HTTP handler over the existing `domain/validation` executor.

All three CLI groups follow the existing `reference` / `connection` patterns: subcommands built on `cli-core` that are thin HTTP wrappers around the DTV API.

**Greenfield: yes.** Every command, the new `api/handlers/validation.go`, and the route registration are net-new code in a scenario already under active development. There are no external callers to preserve, no compatibility shims, and no deprecation aliases to maintain. Pick the cleanest shape and ship it.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read scientific-debugging
prompt-manager skill read implementation-plan-authoring
```

Also review in-repo:

- `scenarios/development-toolchain-validator/cli/domains/references/register.go` — canonical subcommand-group pattern (list/get/create/update/delete + `--json`).
- `scenarios/development-toolchain-validator/cli/domains/connections/register.go` — filter/action pattern (`drift` action on a connection uses `POST`, plus `--reference`/`--skill` filters).
- `scenarios/development-toolchain-validator/api/handlers/report.go` — report endpoints & response shapes.
- `scenarios/development-toolchain-validator/api/handlers/expectation.go` — expectation CRUD endpoints, including `X-Dry-Run` handling at line 86.
- `scenarios/development-toolchain-validator/api/domain/validation/model.go` — `ReferenceValidationReport` / `ConnectionValidationResult` shapes the `validate` command must render.
- `scenarios/development-toolchain-validator/api/domain/validation/executor.go` — the service the new validation handler will call.
- `scenarios/development-toolchain-validator/api/main.go:80-104` — handler wiring site for the new `validationHandler`.

## 3. Problem Statement

The meta-optimization team depends on DTV for P2/P3 readiness checks, but the CLI currently exposes only `reference` and `connection` commands and the API has no validation route registered. Without `validate`, `report`, and `expectation` commands (and the missing handler behind `validate`), the team has no scriptable way to:

1. Trigger validation of a reference and read the outcome.
2. Inspect cross-skill conflicts, content drift, maturity scores, or tool-baseline regressions.
3. Manage the structural and CLI expectations that drive validation.

Everything else in the DTV initiative (`dtv-meta-optimization-contract`) assumes these commands exist.

## 4. Scope

**In scope**
- CLI code under `scenarios/development-toolchain-validator/cli/`.
- New domain packages: `cli/domains/validation`, `cli/domains/reports`, `cli/domains/expectations`.
- Registration in `cli/domains/domains.go`.
- CLI-level tests in `cli/cli_*_test.go` matching the existing struct-assertion style.
- New `scenarios/development-toolchain-validator/api/handlers/validation.go` plus handler wiring in `api/main.go` for `POST /api/v1/references/{id}/validate` (per D1 — the validation domain executor already exists; only the HTTP adapter is missing).
- API handler tests under `api/handlers/` mirroring the existing handler test style.

**Out of scope**
- UI changes.
- Database/migration work.
- Changes to `domain/validation/` business logic — the new handler is a thin adapter only.
- Validate history endpoint (`GET /api/v1/references/{id}/validate/history`) — deferred per D4.
- Async validation jobs / `--wait` flag — deferred per D4.

**Acceptance allow:** `scenarios/development-toolchain-validator/{cli,api}/**` (widened from CLI-only per D1).

## 5. Current Technical Context

- CLI uses `github.com/vrooli/cli-core` via `cliapp.NewStandardScenarioApp` (`cli/app.go`).
- Subcommand groups are returned from `domains/domains.go::SubcommandGroups`.
- Each domain package exports a `Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup`.
- HTTP helpers live inline per package (`get`, `getWithQuery`, `request`) and call `core.Get` / `core.Request`.
- Output uses `cliapp.ListReport`, `cliapp.MutationReport`, `cliapp.OperationalReport` with a `--json` flag from `cliutil.JSONFlag(fs)`.
- `cliutil.ParseInterspersed` is always used instead of `fs.Parse` (so `validate my-ref --json` works).
- API endpoints already implemented (`api/handlers/report.go`, `api/handlers/expectation.go`):
  - `GET /api/v1/reports/conflicts` `[?reference_id=…&skill_id=…]`
  - `POST /api/v1/reports/drift` `{current_hashes: {…}}`
  - `GET /api/v1/reports/maturity`
  - `GET /api/v1/reports/tool-baselines`
  - `GET|POST|DELETE /api/v1/expectations/structural[/{id}]`
  - `GET|POST|DELETE /api/v1/expectations/cli[/{id}]`
- Validation API endpoints **are not yet implemented** — the validation domain layer (services, models, checkers, executor) is complete, but no HTTP handler is registered. This plan adds `api/handlers/validation.go` + one wiring line in `api/main.go`.

## 6. Target End State

Running `development-toolchain-validator --help` lists three new subcommand groups:

```
reference       Manage references
connection      Manage skill connections
validate        Run validation against a reference
report          Inspect cross-skill reports
expectation     Manage structural and CLI expectations
```

`POST /api/v1/references/{id}/validate` is registered in `api/main.go` and returns a `ReferenceValidationReport`.

Each new CLI command:
- Parses args with `cliutil.ParseInterspersed`.
- Honors `--json` and `--api-base`.
- Renders human output by default, structured JSON with `--json`.
- Is exercised by at least one struct-shape test under `cli/`.
- Passes `go build ./...` and `go test ./...` inside `cli/` and `api/`.

The `dtv-meta-optimization-contract` follow-up item can rewrite prompts to call these commands directly.

## 7. Implementation Strategy (Phased)

**Phase 1 — `expectation` command group**
- No API-side work required; all endpoints exist.
- Create `cli/domains/expectations/register.go` with `Register(core)` returning a group.
- Subcommands per D3: `expectation structural list|create|get|delete` and `expectation cli list|create|get|delete` (nested groups).
- Reuse the request/response shapes in `api/domain/expectation/model.go`.
- Forward `X-Dry-Run: true` on `create` / `delete` when `--dry-run` is passed (matches `api/handlers/expectation.go:86`).
- Add `cli/cli_expectation_test.go` mirroring `cli_reference_test.go`.

**Phase 2 — `report` command group**
- No API-side work required.
- Create `cli/domains/reports/register.go`.
- Subcommands per D2: `report conflicts`, `report drift`, `report maturity`, `report tool-baselines`. Each maps 1:1 to `/api/v1/reports/…` and renders a `cliapp.ListReport`.
- Filter flags use the connection naming convention exactly: `--reference <id>`, `--skill <id>`. A single helper converts CLI flag names to API query keys (`reference` → `reference_id`, `skill` → `skill_id`).
- `report drift` uses `POST /api/v1/reports/drift` with a `--hash <skill_id>=<hash>` repeatable flag; build the `current_hashes` map client-side. Help text documents the format.
- Add `cli/cli_report_test.go`.

**Phase 3 — Validation API handler + `validate` command**
- Add `scenarios/development-toolchain-validator/api/handlers/validation.go` exposing `POST /api/v1/references/{id}/validate`. The handler resolves the reference (ID or slug), invokes the existing `domain/validation` executor, and returns the resulting `ReferenceValidationReport`.
- Wire `validationHandler` in `api/main.go` alongside the existing reference/skill/expectation/report handlers.
- Add `api/handlers/validation_test.go` covering: 200 with a successful report, 200 with failures + triage status, 404 for an unknown reference.
- Implement `cli/domains/validation/register.go` with `validate <reference>` (accepts ID or slug, mirroring `reference get`).
- Output: `cliapp.OperationalReport` with summary (pass/fail/total), per-connection rows, and triage status when failures exist.
- No `validate history`, no `--wait`, no async — D4 keeps v1 synchronous-only.
- Add `cli/cli_validate_test.go`.

**Phase 4 — Wire-up & smoke test**
- Register all three groups in `cli/domains/domains.go::SubcommandGroups`.
- Run `go build ./...` and `go test ./...` in `cli/` and `api/`.
- Document new commands in `cli/README.md` if it exists; otherwise in `scenarios/development-toolchain-validator/README.md`.

## 8. Contract Decisions

All four workshop decisions resolved in round 001 (committed):

- **D1 — Validation API blocker → A (expand scope):** Add `api/handlers/validation.go` + register the route in `api/main.go`. The `domain/validation` package already has the executor, models, and checkers — the handler is a thin adapter. `acceptance_allow` widens to `scenarios/development-toolchain-validator/{cli,api}/**`. The lost `fix/dtv-validation-api` item is not recreated.
- **D2 — `report` command shape → A (subcommands):** `report conflicts` / `report drift` / `report maturity` / `report tool-baselines`. Filters via `--reference <id>` / `--skill <id>`. Matches the existing `reference list` / `connection drift` pattern.
- **D3 — `expectation` command shape → A (nested groups):** `expectation structural {list|create|get|delete}` and `expectation cli {list|create|get|delete}`. Keeps the two resource types visibly distinct and lets each group validate its own flag set without conditional branching.
- **D4 — `validate` scope → A (synchronous v1 only):** Just `validate <reference>`. No history command, no async/wait flags. History can land later if a caller asks for it.
- **Output format:** human-readable by default, `--json` flips to `cliapp.PrintReportJSON`. No new formatters.
- **Dry-run support:** `expectation create` and `expectation delete` forward `X-Dry-Run: true` when `--dry-run` is passed.

## 9. Testing Plan

Automated only — the `feedback_testing_over_manual.md` memory says plans should use automated tests, not manual checklists.

**Unit-level (CLI structs)**
- `cli/cli_expectation_test.go`: assert request/response struct field parity with `api/domain/expectation/model.go` for both structural and CLI variants.
- `cli/cli_report_test.go`: assert each report response type parses and renders a non-empty `ListReport` for a fixture; assert the `--hash` flag builds the expected `current_hashes` map.
- `cli/cli_validate_test.go`: assert `ReferenceValidationReport` parses and renders a non-empty `OperationalReport`, including the failure + triage path.

**Command-wiring**
- `cli/cli_core_test.go`-style assertions that each new command is registered, callable with `--help`, and returns a non-zero exit for missing required args.

**API handler**
- `api/handlers/validation_test.go`: 200 success, 200 with failures + triage status, 404 unknown reference, 400 malformed reference identifier.
- Must integrate with the existing handler test harness (no new test infra).

**Build/integration**
- `(cd scenarios/development-toolchain-validator/cli && go build ./... && go test ./... -timeout 300s)` is the gate.
- `(cd scenarios/development-toolchain-validator/api && go build ./... && go test ./... -timeout 300s)` is the gate.

**Pre-existing failures**
- Per planning guidelines, if any pre-existing `go test` failures surface in `cli/` or `api/` during this work, fix them in the same PR — the build must be clean at merge.

## 10. Rollout / Validation Checklist

1. PR touches only files under `scenarios/development-toolchain-validator/{cli,api}/**`.
2. `go build ./...` passes in `cli/` and `api/`.
3. `go test ./...` passes in `cli/` and `api/` (including the new validation handler test).
4. User restarts `vrooli scenario restart development-toolchain-validator` and confirms commands work end-to-end.
5. `dtv-meta-optimization-contract` can reference the new command surface without scope slippage.

> Per `feedback_no_restart_active_scenario.md`, this plan does **not** restart the scenario in code — the user runs `vrooli scenario restart development-toolchain-validator` after the patches land.

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| ~~Validation HTTP handler is not registered~~ — **resolved by D1**: this plan adds the handler and route. | — | — | Phase 3 covers handler + wiring + tests. |
| CLI flags drift from API query param names (`reference_id` vs `--reference`). | Medium | Confusing UX | Mirror `connections/register.go` filter naming (`--reference`, `--skill`) exactly; keep query-key conversion in one helper. |
| Drift report body shape (`current_hashes`) is awkward from a shell. | Medium | Poor UX for the one command that uses `POST`. | Accept `--hash <skill_id>=<hash>` repeatable; build the map client-side. Document in help text. |
| Users pass slugs where IDs are required (or vice versa) on `validate`. | Medium | Errors mid-pipeline. | Reuse the ID-or-slug fallback pattern from `references/register.go:136` in both the new validation handler and the CLI command. |
| Output format divergence between `report`/`validate` and existing commands. | Low | Inconsistent UX. | Stick to `cliapp.ListReport` / `OperationalReport` — no bespoke renderers. |
| Validation handler accidentally introduces business logic instead of being a thin adapter. | Low | Domain logic in the wrong layer. | Code review check: handler may resolve the reference, call `executor.Run`, and serialize the result — nothing else. |

## 12. Non-goals / Prohibited Patterns

- Do **not** implement business logic in the CLI; reports/validation stay in the API.
- Do **not** add new domain logic to the validation handler — it is a thin adapter over the existing executor.
- Do **not** introduce a new output format or bespoke renderer — use `cliapp.*Report` types.
- Do **not** add retries, polling, or spinners; commands are synchronous one-shots.
- Do **not** widen acceptance to other scenarios or to the UI.
- Do **not** add pagination flags that aren't enforced by the API.
- Do **not** build a shared "client" package inside the CLI — inline helpers per existing domain packages.
- Do **not** add backwards-compat shims, deprecation aliases, or unused-var renames — this is greenfield code.

## 13. Definition of Done

- [ ] `development-toolchain-validator validate <reference>` returns a real report from the new `POST /api/v1/references/{id}/validate` route.
- [ ] `development-toolchain-validator report {conflicts|drift|maturity|tool-baselines}` works against a live API.
- [ ] `development-toolchain-validator expectation structural {list|create|get|delete}` works.
- [ ] `development-toolchain-validator expectation cli {list|create|get|delete}` works.
- [ ] Every new command supports `--json`.
- [ ] `expectation create` and `expectation delete` honor `--dry-run` (forward `X-Dry-Run: true`).
- [ ] `go build ./...` and `go test ./...` pass in `cli/` and `api/`.
- [ ] No files outside `scenarios/development-toolchain-validator/{cli,api}/**` are modified.
- [ ] User can run `vrooli scenario restart development-toolchain-validator` and exercise all commands successfully.
- [ ] `dtv-meta-optimization-contract` can be queued without CLI gaps.
