# Prompt-Manager CLI ↔ API Parity — Decision-Accept & Systematic Audit

**Backlog reference:** `execute/prompt-manager-cli-api-parity` (swarm-manager, priority 5, P1, tags: prompt-manager, cli, thin-wrapper, platform-fix)

**Motivation:** The `prompt-manager team decision-accept` subcommand referenced by the morning-vision-walk skill does not exist, even though the underlying API (`PATCH /teams/{id}/decisions/{decisionId}`) is fully implemented. This has now blocked two consecutive vision walks from executing 4+ pending decisions. The root principle — **every prompt-manager CLI must be a thin fully-featured wrapper over the API** — is violated more broadly than just this one subcommand, so the fix is scoped to include a systematic parity audit and a CI guard that prevents regression.

---

## Required Reading

```
prompt-manager skill read skill-principles cli-steer api-steer test
```

Rationale: `cli-steer` defines the thin-wrapper principle being enforced; `api-steer` anchors the API conventions the CLI must faithfully expose; `test` covers the testing discipline for Part A unit + integration tests and Part C CI guard; `skill-principles` is mandatory by convention.

---

## Greenfield Declaration

**This is greenfield work.** Do not add compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, renamed `_unused` variables, or deprecated aliases. The API surface is stable; the CLI catches up by addition, not adaptation.

---

## Context & Current State

### API side (complete, do not modify)
- `scenarios/prompt-manager/api/heartbeat/handlers.go:2114` — `UpdateDecisionHandler` implements `PATCH /teams/{id}/decisions/{decisionId}`
- `scenarios/prompt-manager/api/heartbeat/handlers.go:2297` — `DeleteDecisionHandler` implements `DELETE /teams/{id}/decisions/{decisionId}`
- `scenarios/prompt-manager/api/heartbeat/models.go:207-220` — `UpdateDecisionRequest` has 11 optional fields: `Decision`, `Rationale`, `Context`, `Status`, `Supersedes`, `Topic`, `Description`, `Options`, `Selected`, `Freeform`, `Notes`
- `scenarios/prompt-manager/api/main.go:574-575` — routes registered
- Approval enforcement (`checkApprovalEnforcement`) already distinguishes human callers from agent callers via `X-Caller-ID` header; humans bypass approval restrictions

### CLI side (incomplete — this plan's focus)
- `scenarios/prompt-manager/cli/teams/teams.go:619-622` — only `decision-add` and `decision-list` wired up in switch dispatch
- `scenarios/prompt-manager/cli/teams/teams.go:691-693` — usage text only advertises two decision commands
- Missing: `decision-update`, `decision-accept`, `decision-reject`, `decision-delete`

### Existing patterns to mirror
- `cmdKnowledgeUpdate` (teams.go) already wraps `PATCH /teams/{id}/knowledge/{entryId}` with pointer-optional flags; follow this exact pattern for `cmdDecisionUpdate`
- `cmdTaskUpdate` shows how to flow optional fields through a flag.FlagSet and build a JSON PATCH body with only the flags the user provided
- `teams_test.go` `TestCmdDecisionAdd` is the template for unit tests

---

## Scope

### Part A — Decision-Update / Accept / Reject / Delete CLI Subcommands
**Goal:** Unblock the morning vision walk.

A1. **Add `cmdDecisionUpdate`** — generic PATCH wrapper exposing all 11 `UpdateDecisionRequest` fields as optional flags (`--decision`, `--rationale`, `--context`, `--status`, `--supersedes`, `--topic`, `--description`, `--options` as JSON, `--selected`, `--freeform`, `--notes`). Build the PATCH body from only the flags the user actually passed (use pointer-optional pattern). Attach `X-Caller-ID: ui-user` header so humans bypass approval restrictions (standard pattern for operator-driven CLI).

A2. **Add `cmdDecisionAccept`** — convenience wrapper over `cmdDecisionUpdate` with `--status accepted` preset. Accepts positional `<team-id> <decision-id>` plus `--selected <option-key>` and `--notes "..."`. If `--selected` is omitted, error out (accept-without-choice is almost always a mistake).

A3. **Add `cmdDecisionReject`** — convenience wrapper with `--status rejected` preset. Accepts positional `<team-id> <decision-id>` plus `--notes "..."`. `--notes` is required (reject-without-reason is a nudge pattern, but the API doesn't enforce it, so the CLI does).

A4. **Add `cmdDecisionDelete`** — DELETE wrapper, positional `<team-id> <decision-id>`, no body. Confirmation prompt unless `--yes` is passed (matches `cmdTaskDelete` safety pattern).

A5. **Wire dispatch** — add five cases to the switch at `teams.go:619-622` (one for each subcommand plus `decision-show` if we also add it — see Open Questions).

A6. **Update `usageText()`** at `teams.go:691-693` — extend Decision Log Commands section to list every subcommand with one-line descriptions.

A7. **Unit tests in `teams_test.go`** for each new subcommand, following the `TestCmdDecisionAdd` pattern: spin up `httptest.NewServer`, assert path + method + body shape match the PATCH/DELETE contract, assert response decoding.

A8. **Integration test (new file `teams_integration_test.go`, build-tagged `integration`)** — spin up real API server against a tmp datadir, seed a team with a decision in `pending` status, run `decision-accept` via the CLI binary, assert the decision transitions to `accepted` with the correct `selected` and `notes` fields persisted.

**Exit criterion for Part A:**
```
prompt-manager team decision-accept meta-optimization dec-1776981723540926630 --selected <option-key> --notes "..."
```
succeeds against the running API and persists the state change, and the same for decision-update / decision-reject / decision-delete on fixture decisions.

---

### Part B — Systematic Parity Audit
**Goal:** Every API endpoint has a matching CLI surface; document the current state as a baseline.

B1. **Enumerate every route** registered in `scenarios/prompt-manager/api/main.go` under the `v1.HandleFunc(...)` calls. For each: record method + path + handler name.

B2. **For each route, identify the matching CLI subcommand** — search dispatch switches in `scenarios/prompt-manager/cli/teams/teams.go` and any other CLI subcommand files (check `scenarios/prompt-manager/cli/` for additional top-level command packages).

B3. **Produce `scenarios/prompt-manager/cli/PARITY_AUDIT.md`** — a table with columns: `Method | Path | Handler | CLI Subcommand | Status`. Status values: `covered`, `gap`, `intentionally-absent` (with justification).

B4. **Close every gap found** — for each `gap` row, add the missing CLI subcommand following Part A's pointer-optional pattern for PATCH/PUT endpoints, and the conventional `list`/`get`/`add`/`delete` naming for the other verbs. Re-run the audit until every row is `covered` or `intentionally-absent`.

B5. **Commit `PARITY_AUDIT.md` as the source of truth** — this document is what Part C tests against.

**Exit criterion for Part B:** `PARITY_AUDIT.md` exists, every v1 route appears, and every row is `covered` or has a justified `intentionally-absent` entry.

---

### Part C — CI Parity Guard
**Goal:** Prevent future drift — a new API handler without a matching CLI subcommand fails CI.

C1. **Add `scenarios/prompt-manager/cli/parity/parity_test.go`** (new package). It:
   - Parses `scenarios/prompt-manager/api/main.go` as Go source (via `go/parser` + `go/ast`) and extracts every `v1.HandleFunc(pattern, handler).Methods(...)` registration. Produces a canonical `[]APIRoute`.
   - Parses `scenarios/prompt-manager/cli/teams/teams.go` (and any other CLI command files discovered in Part B) and extracts every `case "<subcommand>":` from the top-level dispatch switches. Produces a canonical `[]CLISubcommand`.
   - Loads the `intentionally-absent` allowlist from `scenarios/prompt-manager/cli/parity/allowlist.json` (list of `{method, path, reason}` tuples, mirrored from `PARITY_AUDIT.md`).
   - Asserts: for every `APIRoute` not in the allowlist, there exists a CLI subcommand that wraps it (matching rules documented in the test file).
   - Fails with an actionable error message pointing to the missing subcommand and the file where it should be added.

C2. **Wire into CI** — ensure `go test ./scenarios/prompt-manager/...` (or the scenario-level test target) picks up the new package. Verify by intentionally removing a CLI case and confirming the test fails loudly.

C3. **Document the guard** in `scenarios/prompt-manager/cli/PARITY_AUDIT.md` — add a "How to extend" section explaining the allowlist and the test's matching rules.

**Exit criterion for Part C:** Intentionally removing `case "decision-list"` from the dispatch causes `go test ./scenarios/prompt-manager/cli/parity/...` to fail with a clear message naming the uncovered endpoint.

---

## Testing Discipline (automated, not manual)

- **Unit tests** — `httptest.NewServer` stubs for every new CLI subcommand; assert request shape and response handling. No manual invocation checklists.
- **Integration test** — one end-to-end path for `decision-accept` against a real API binary + tmp datadir + fixture team. Build-tagged `integration` so it runs in CI but is optional locally.
- **Parity test (Part C)** — the drift guard itself is a test; it runs in the standard test target.

No manual "run this command and check the output" verification steps. If something needs manual inspection, encode it as a test.

---

## Final: Cleanup & Verification

1. **Build**: `cd scenarios/prompt-manager && go build ./...` — no errors. Fix **all** build errors in modified files, including any pre-existing ones you inherit (do not treat "pre-existing" as a skip signal).
2. **Format**: `gofumpt -w scenarios/prompt-manager/cli/ scenarios/prompt-manager/api/` (safe to run against both — it's idempotent).
3. **Lint**: `golangci-lint run ./scenarios/prompt-manager/cli/... ./scenarios/prompt-manager/api/...` — fix **all** warnings in modified files, including pre-existing.
4. **Test**: `go test ./scenarios/prompt-manager/cli/... ./scenarios/prompt-manager/api/... -timeout 300s` — all pass. Integration tests: `go test -tags=integration ./scenarios/prompt-manager/cli/teams/...`.

### Scenario restart — operator-led, NOT agent-led

**Do not run `vrooli scenario restart prompt-manager` from within this implementation task.** Claude Code itself runs inside prompt-manager (via the `claude-code` resource), and restarting the scenario from inside would kill the implementation agent mid-task. Per operator policy (memory: `feedback_no_restart_active_scenario.md`), write the new CLI binary to disk and leave the restart to the operator.

After operator-led restart, the operator verifies:

```
prompt-manager team decision-accept meta-optimization dec-1776981723540926630 --selected <option-key> --notes "test"
prompt-manager team decision-list meta-optimization
```

The first must succeed with a 2xx response; the second must show the decision in `accepted` status.

---

## Deliverables

1. `scenarios/prompt-manager/cli/teams/teams.go` — dispatch additions + 4 new `cmdDecision*` functions + updated `usageText()`
2. `scenarios/prompt-manager/cli/teams/teams_test.go` — unit tests for each new subcommand
3. `scenarios/prompt-manager/cli/teams/teams_integration_test.go` (new, build-tagged `integration`) — end-to-end `decision-accept` test
4. `scenarios/prompt-manager/cli/PARITY_AUDIT.md` — full endpoint ↔ subcommand table
5. `scenarios/prompt-manager/cli/parity/` — new package with `parity_test.go` + `allowlist.json`
6. Any additional `cmd*` functions required to close gaps discovered in Part B (count unknown until the audit is run)
7. Commit message cites backlog id: `refs: execute/prompt-manager-cli-api-parity`

---

## Open Questions (surface-and-resolve during Part A)

- Should we also add `decision-show <team-id> <decision-id>` as a single-item read wrapper? The API has it via the list + filter pattern, but a direct show is operator-friendly. **Recommendation:** yes, add it — costs ~15 lines.
- Approval-mode enforcement uses `X-Caller-ID` header. Should the CLI always send `ui-user`, or read from an env var / config? **Recommendation:** always send `ui-user` for now. Operator identity is future work (scope: `agent-identity-management` if it exists; file a fresh backlog item otherwise).
- For `decision-delete`, should we also soft-delete (set status to `rejected` with a note) as an alternative? **Recommendation:** no — DELETE means DELETE. If operators want a reversible path, they use `decision-reject`.

---

## Risks & Rollback

- **Risk:** Parity test false positives if the API route parser or CLI dispatch parser misreads the AST. **Mitigation:** Part C has its own unit tests using fixture Go files.
- **Risk:** Part B surfaces a larger-than-expected gap count, making this a 2-day job instead of a half-day job. **Mitigation:** Part A is independently valuable — if Part B is heavy, ship Part A as a stand-alone PR and file a follow-up backlog item for B+C. Document the split in the PR description.
- **Rollback:** Revert the PR. The changes are CLI-only additions; reverting cannot corrupt data.

---

## Non-Goals

- UI changes for decision management (separate scope)
- Schema migrations (API models are stable)
- New agent-facing affordances (this is operator/human CLI only)
- Refactoring existing `decision-add` / `decision-list` — they work; don't touch them unless Part B audit flags a parity issue
- Changes to `swarm-manager` CLI or API — this is prompt-manager-scoped

---

## Handoff Notes to Implementation Agent

- Start with Part A — it unblocks the morning vision walk, which blocks operator decision throughput across the whole system.
- Part A should be its own commit; Parts B and C can be a follow-up commit if iteration is useful.
- When the build is green and tests pass, post a handoff summary citing `execute/prompt-manager-cli-api-parity` and listing the new subcommands. Do not restart prompt-manager — operator does that manually.
- If you discover during Part B that the CLI has additional top-level command packages beyond `cli/teams/`, extend the audit to cover them all. The `thin-wrapper` principle applies to every command package.
