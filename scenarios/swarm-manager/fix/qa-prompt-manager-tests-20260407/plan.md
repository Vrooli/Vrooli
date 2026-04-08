# Fix prompt-manager test failures

## Required Reading
- `prompt-manager skill read scientific-debugging` — hypothesis-driven root cause analysis
- `prompt-manager skill read documentation-health` — docs validation patterns and fixes
- `prompt-manager skill read test` — test infrastructure conventions
- `prompt-manager skill read cli-steer` — CLI architecture patterns and unknown-command handling

## Problem Statement

GCT review of the prompt-manager scenario shows 5 of 11 test phases failing:

| # | Phase | Failure | Root Cause (Hypothesis) |
|---|-------|---------|------------------------|
| 1 | **standards** | "standards violations exceed fail_on=high (highest=critical)" — 8 blocking, 154 warnings | Missing `cli/prompt-manager` compiled binary (critical); P0 targets in `requirements/index.json` missing linked requirements (4× critical) |
| 2 | **docs** | "Docs validation failed" | Likely broken links or missing files referenced in `docs/manifest.json` |
| 3 | **unit** | "node unit tests failed in ui: exit status 1" | UI unit test failures — need to run and inspect output |
| 4 | **integration** | "CLI returned success (exit 0) for unknown command" | `cli-core` unknown-command handling returns 0 instead of non-zero; prompt-manager delegates to cli-core via `install.sh` |
| 5 | **playbooks** | "registry not found: .../bas/registry.json" | `bas/` directory has `actions/`, `cases/`, `flows/` but no `registry.json` — needs generation |

## Scope

### Acceptance Boundaries
- **acceptance_allow**: `scenarios/prompt-manager/**`
- **acceptance_deny**: not set (no sensitive paths identified)

### In Scope
- Fix all 5 failing test phases within `scenarios/prompt-manager/`
- Standards: resolve 8 critical/high violations
- Docs: fix validation errors
- Unit: fix UI test failures
- Integration: fix CLI unknown-command exit code
- Playbooks: generate `bas/registry.json`

### Out of Scope
- Addressing the 154 warning-level standards violations (not blocking)
- Changes to `packages/cli-core/` (if the unknown-command bug is there, that's a separate item)
- New feature work

## Technical Context

### Scenario Structure
```
scenarios/prompt-manager/
├── api/           — Go API server
├── bas/           — Browser automation (actions/, cases/, flows/ — NO registry.json)
├── cli/           — Go CLI (main.go, app.go, subcommand packages)
├── docs/          — manifest.json + markdown docs
├── requirements/  — index.json with 20+ modules, P0-P3
├── ui/            — React/Vite frontend with vitest tests
└── ...
```

### Key Observations
- `cli/prompt-manager` binary is not committed (expected — it's a build artifact). The standards checker flags its absence as "Scenario Required Structure" critical.
- `cli/app.go` uses `cli-core/cliapp.ScenarioApp` which should handle unknown commands. The `install.sh` delegates entirely to `packages/cli-core/install.sh`.
- `requirements/index.json` lists 20 modules with P0-P3 priorities. The critical violation is that P0/P1 modules lack linked requirement entries in their `module.json` files. Checking module 01 shows requirements DO exist in the module.json — so the issue may be in how `index.json` references or validates them.
- `bas/` has playbook content but no `registry.json` index file.

## Approach

### Phase 1: Standards (Critical Path)
1. **CLI binary**: Determine if the standards gate expects a committed binary or a build step. If it expects a build step, add a `go build` to the test pipeline or adjust the standards config to exclude the binary path.
2. **Requirements linking**: Investigate why P0 targets show "missing requirements" when `module.json` files contain requirement entries. May be a schema mismatch or missing field.
3. Run `scenario-auditor audit prompt-manager --standards-only --timeout 60` to get full violation list.

### Phase 2: Docs Validation
1. Run docs validation to see exact failures.
2. Cross-reference `docs/manifest.json` entries against actual files on disk.
3. Fix broken references or missing files.

### Phase 3: UI Unit Tests
1. Run `cd scenarios/prompt-manager/ui && npm test` to see specific failures.
2. Fix failing tests — likely broken imports, missing mocks, or stale snapshots.

### Phase 4: CLI Unknown Command
1. Test locally: `prompt-manager __test_genie_nonexistent_command_12345__; echo $?`
2. If cli-core is the root cause, determine if a prompt-manager-level wrapper can catch and return non-zero, or if this requires a cli-core fix (out of scope → new backlog item).

### Phase 5: Playbooks Registry
1. Check if a generation command exists (e.g., `playbook-builder` or a Makefile target).
2. Generate `bas/registry.json` from existing `actions/`, `cases/`, `flows/` content.

## Verification

- Run full GCT test suite: `vrooli scenario test prompt-manager`
- All 11 phases should pass (currently 6/11)
- No regression in the 6 currently-passing phases

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| CLI unknown-command bug is in cli-core, not prompt-manager | Can't fix within acceptance_allow scope | Create separate backlog item for cli-core fix; add prompt-manager shim if possible |
| UI test failures require dependency updates | Scope creep | Only fix tests, don't upgrade deps unless required |
| Standards violations require structural changes to requirements/ | Time | Focus on critical violations only; warnings are not blocking |
| bas/registry.json format unknown | Can't generate correctly | Check other scenarios for registry.json examples |

## Dependencies
- `packages/cli-core/` — may need changes for unknown-command handling (out of scope if so)
- `scenario-auditor` — needed to validate standards fixes
- `test-genie` — for running the full test suite

## Estimated Phases
1. Standards fixes (highest priority — unblocks other phases)
2. Docs + Playbooks (likely quick fixes)
3. UI unit tests (unknown scope until we see failures)
4. CLI integration (may be out of scope)
