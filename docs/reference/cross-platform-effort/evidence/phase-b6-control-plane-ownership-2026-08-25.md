# Phase B6 — Control-plane ownership

Date: 2026-08-25

## Implemented

- Added the host ownership map to `docs/resources/architecture.md`, covering
  host facts, inventory, lifecycle, presentation, pressure, session, and the
  five `hostreq*` boundaries.
- Replaced the private model-policy-drift repository walk with the canonical
  `repo-contract-go` resolver, while preserving the test-only `PROJECT_ROOT`
  fixture override through that same canonical path resolver.
- Confirmed zero root-module importers of `internal/reporoot` and
  `internal/scheduler`, then removed both obsolete forwarding packages. The
  public `packages/scheduler` package remains the scheduler authority.
- Added package-purpose documentation for all 173 packages under `internal/`,
  including explicit ownership and non-ownership statements for the six large
  control-plane packages and the `hostreq*` cluster.
- Added the `repo_contract_internal_package_doc` hygiene finding and a negative
  scratch-package test. `internal/app/hygiene` now reports an undocumented
  package as an error before it can enter the tree.
- Audited the five `hostreq*` packages and retained their boundaries. The
  importer/API graph confirms distinct contracts: `hostreqspec` is the pure
  vocabulary, `hostreq` resolves declarations, `hostreqcheck` validates static
  consistency, `hostreqkit` owns shared actuation contracts, and `hostreqrun`
  orchestrates resolve-then-ensure. `packages/hostreq` remains the documented
  public façade over `internal/hostreq`.
- Retained `internal/resources/runtime/operations` with an explicit package
  contract after confirming it is a live internal boundary.
- Fixed the layout provider's zero-findings case: a repository with no empty
  directories now receives a passing check instead of an unconditional error.
  Added a regression test for that contract.

## Validation

```text
go test ./internal/app/contract ./internal/safeguards/model-policy-drift ./internal/api
ok

go test ./internal/app/hygiene
ok

go test ./internal/hostreq ./internal/hostreqcheck ./internal/hostreqkit ./internal/hostreqrun ./internal/hostreqspec
ok

zero root/scheduler importers and both files absent

package-doc census: 173 packages, 0 undocumented

go build ./...
ok

Scenario Dependency Analyzer freshness provider: available; 274 touched
surfaces checked, 237 clean, 37 stale, 0 errors. The stale surfaces are
pre-existing dirty-worktree baseline across unrelated scenarios and packages;
no broad dependency rewrite was applied in this phase.
```

## Phase status

The B6 implementation is complete. Focused acceptance passes, including the
control-plane package tests, host-request boundary tests, package-doc rule, and
root build. Plan-wide validation remains non-comparable because the
`react-component-library` member fails its comprehensive run on pre-existing
CLI/component conformance debt, and the dirty-worktree freshness baseline has
37 unrelated stale surfaces.
