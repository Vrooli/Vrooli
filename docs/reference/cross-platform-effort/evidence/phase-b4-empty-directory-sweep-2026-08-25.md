# Phase B4 — Empty-directory sweep

Date: 2026-08-25

## Implementation

- Root hygiene now owns the `repo-layout` provider. It derives the canonical
  `internal/` and `packages/` roots from `repo-contract-go.Layout`.
- An empty directory without `.gitkeep` is an automatic
  `repo_contract_empty_directory` finding.
- `vrooli hygiene --fix-safe --contract-only` removes only those empty
  directories and repeats the scan until newly empty parent directories are
  also resolved. A `.gitkeep` directory is preserved.
- The reviewed abandoned shells were removed: seven shareddrift/agentpolicy/
  plans directories, `internal/codingagents`, `internal/hostobservability`,
  `packages/cli-core/cmd/policy-runner`, and
  `packages/binaryfetch/cmd/syncschema`.
- `packages/proto/cmd/protogen` now discovers the existing proto root from
  repository markers instead of appending `packages/proto` based only on the
  current basename. This prevents a nested `packages/proto/packages/proto`
  output tree.
- The RCL generated-fixture validator now uses stable `rcl-fixture-positive`
  and `rcl-fixture-negative` names. Its focused test rejects timestamp-like
  fixture names.

## Validation

```text
go test ./internal/app/hygiene                         PASS
(cd packages/proto && go test ./cmd/protogen)           PASS
(cd scenarios/react-component-library/api && \
  go test ./handlers/componenttests)                    PASS
vrooli hygiene --contract-only --no-freshness \
  --fix-safe --summary                                  PASS
make -C packages/proto generate                         PASS (first run)
make -C packages/proto generate                         PASS (second run)
```

The safe-fix run removed the reviewed residue and the nested generated tree;
the post-run scan reports no empty directory under `internal/` or `packages/`
without `.gitkeep`. The `.gitkeep` scratch directory survived. Both proto
generation runs left `packages/proto/packages/proto/` absent and produced no
new timestamped fixture directory. An older untracked RCL fixture tree already
present in this shared worktree remains user-owned and was not deleted.

The separate `packages/repo-contract-go` test suite still has one pre-existing
literal-manifest audit failure in
`scenarios/scenario-to-plugin/api/handlers/pipeline/conformance.go`; it is
unrelated to B4 and no source in that package was changed by this phase.
