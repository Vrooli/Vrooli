# Phase B7 — God-file splits (partial)

Date: 2026-08-25

## Implemented

- Added `tools/symbolset`, a small AST-based declaration inventory used to
  compare a Go file set before and after structural moves without depending on
  source filenames.
- Split the target declarations from `internal/cliinstall/cliinstall.go` and
  `internal/cliinstall/uninstall.go` into whole-symbol parts. The resulting
  parts are 700/322 lines and 672/700/100 lines respectively.
- Split `internal/scenario/scenario.go` into whole-symbol parts of 671, 685,
  and 203 lines.
- Split `internal/resources/drivers.go` into whole-symbol parts of 674, 720,
  and 520 lines. The package-only source files remain as stable entrypoints.
- Confirmed `internal/setup/setup.go` was already below the 800-line target at
  602 lines, so no artificial split was needed.

## Validation

Passed:

```text
go test ./tools/symbolset
go test ./internal/cliinstall/...
go test ./internal/scenario
go test ./internal/resources -run '^$'
vrooli capability conformance --json
git diff --check
```

The symbol inventory was captured before and after each split; all declaration
set diffs were empty. `go test ./internal/resources -run '^$'` is a compile-only
check because the broader resources suite has unrelated pre-existing failures
in this worktree.

The capability-conformance gate exited 0 with no findings. The only modified
test files under the split packages are the separate B1 platform-skip
instrumentation edits; no split introduced a test behavior change.

Fresh targeted tidiness scans also passed with zero required, advisory, or
uncheckable findings for `internal/cliinstall`, `internal/scenario`, and
`internal/resources`. The `internal/setup` package was scanned as the control
case; it also passed. This proves the split packages did not introduce a new
duplicate block even though the earlier repository-wide campaign reported
unrelated baseline duplication debt.

The server-owned tidiness-manager run
`20260825-083054-1e6d8e4f` returned `FAIL`. Its relevant findings were
scenario-wide duplication debt (38 duplicated-code findings and 45
duplicated-boilerplate findings), plus unrelated UI/provider/template findings;
it did not identify a declaration-loss or compile failure from these moves.

## Open boundary

The repository-wide tidiness campaign remains red because it reports broad
pre-existing duplication and UI/provider/template findings. The phase-specific
split acceptance is nevertheless complete: symbol sets are unchanged,
focused tests pass, targeted tidiness scans report zero findings, and
conformance is clean. The campaign result remains recorded as a plan-wide
baseline limitation rather than being presented as a green repository-wide
tidiness result.
