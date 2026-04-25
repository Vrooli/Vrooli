# Implementation Plan: Address prompt-manager Tidiness Failures

## 1. Purpose

Fix all tidiness failures flagged by the GCT review for the prompt-manager scenario, bringing it from red/yellow to green across standards, tests, and code quality dimensions.

## 2. Required Reading

```bash
prompt-manager skill read implementation-plan-authoring documentation-health
```

## 3. Hard Constraints

**Greenfield fix.** No compatibility shims, deprecation aliases, or transitional symlinks are allowed. The renamed CLI binary, updated requirement linkages, and consolidated `resolveApiBase` must replace prior artifacts outright. Update every in-repo reference rather than leaving compatibility wrappers behind.

## 4. Problem Statement

The GCT review flagged prompt-manager with 5 test failures and 162 standards violations (5 critical, 19 high). The failures span five areas:

1. **Standards — critical**: Missing `cli/prompt-manager` binary (standards expect it; actual binary is `cli/pm`)
2. **Standards — critical**: 4 P0 operational targets missing linked requirements in `requirements/index.json`
3. **Standards — high**: Go CLI builds without workspace mode, PRD linkage broken for ~12 requirements, ESLint safety rules missing, resolveApiBase scattered, tsconfig missing protective comment
4. **Test failures**: docs validation, UI unit tests, CLI unknown-command check (returns 0 instead of non-zero), missing `bas/registry.json`
5. **Code quality**: Stale (needs re-run)

## 5. Scope

**In scope** (`acceptance_allow: scenarios/prompt-manager/**`):
- Fix all critical and high standards violations
- Fix any medium violations that sit in the same file/area as a critical/high fix and take <5 min each (workshop decision d3=B)
- Fix all 5 test failures
- Regenerate `bas/registry.json`
- Re-run code quality checks

**Out of scope:**
- Standalone low/info violations and medium violations not adjacent to a critical/high fix
- New features or refactors beyond the resolveApiBase consolidation already required by standards
- Changes outside `scenarios/prompt-manager/`

## 6. Current Technical Context

| Area | File/Path | Issue |
|------|-----------|-------|
| CLI binary naming | `cli/pm` (ELF binary) | Standards expect `cli/prompt-manager` |
| Requirements linkage | `requirements/index.json` | 4 P0 targets + 1 P1 target lack linked requirements; ~12 REQ-* entries reference missing PRD sections |
| CLI unknown command | `cli/install.sh` → delegates to `packages/cli-core/install.sh` | Returns exit 0 for unknown commands |
| BAS registry | `bas/` | Missing `registry.json` |
| Docs validation | `docs/` | Docs validation test fails |
| UI unit tests | `ui/` | Unit tests fail |
| ESLint config | `ui/` | Missing safety rules (import/no-cycle, etc.) |
| tsconfig | `ui/tsconfig.json` | Missing protective comment block |
| Go workspace | `cli/go.mod` | Not using Go workspace mode with API |
| resolveApiBase | `ui/` (multiple files) | Should be consolidated to single config/hook |

## 7. Target End State

- All 5 test phases pass (standards, docs, unit, integration, playbooks)
- 0 critical violations, 0 high violations from standards audit
- GCT re-review shows green/yellow-improving across all dimensions

## 8. Contract Decisions

Captured from workshop round 001:

- **d1 — CLI binary naming → Option A (Rename build output to `prompt-manager`).** Update the Go build `-o` flag in `cli/` and all `pm`-named references (install scripts, Makefile targets, docs) to `prompt-manager`. No symlink fallback (greenfield constraint).
- **d2 — PRD linkage → Option A (Update `prd_ref` fields to existing PRD sections).** Re-map each broken `prd_ref` in `requirements/index.json` to the correct existing section in `PRD.md`. Add new requirement entries for the 4 unlinked P0 targets and 1 unlinked P1 target, each linked to a real PRD section. Do not regrow PRD.md to fit stale refs.
- **d3 — Low/medium violation handling → Option B (critical + high + trivially adjacent medium).** While editing a file for a critical/high fix, also resolve medium violations in that same file/area when each takes under 5 minutes. Do not chase medium issues in unrelated files.

## 9. Implementation Strategy

### Phase 1: Critical Standards Fixes
1. **CLI binary rename** (`cli/`): change Go build output from `pm` to `prompt-manager`; update `install.sh`, Makefile, docs, and any internal callers. Delete the old `pm` binary path entirely (no symlink).
2. **P0/P1 requirements linkage**: add requirement entries for the 4 unlinked P0 targets and 1 unlinked P1 target in `requirements/index.json`, each with a valid `prd_ref`.

### Phase 2: High Standards Fixes
3. **PRD linkage**: re-point the ~12 broken `prd_ref` fields (REQ-P0-003/005/012/016/017, REQ-P1-023/024/030/031, REQ-P2-004/027/028/029) to existing sections in `PRD.md`.
4. **Go workspace mode**: add the workspace replace directive in `cli/go.mod` so the CLI builds against the in-repo API package.
5. **ESLint safety rules** (`ui/`): add `import/no-cycle: "error"` and other missing rules listed by the standards check.
6. **tsconfig protective comment** (`ui/tsconfig.json`): add the required header comment block.
7. **resolveApiBase consolidation** (`ui/`): move to a single config/hook and import from every prior call site; delete the duplicated implementations.
8. **Adjacent medium fixes**: while inside any file touched above, resolve medium violations in that file when each takes <5 min (per d3=B).

### Phase 3: Test Failures
9. **CLI unknown command**: fix exit-code handling so unknown commands return non-zero (likely in `packages/cli-core/install.sh` or the prompt-manager `install.sh` wrapper).
10. **Docs validation**: triage the docs phase failure and fix the underlying issue.
11. **UI unit tests** (`ui/`): run, diagnose, and fix unit-test failures.
12. **BAS registry**: regenerate `bas/registry.json` via the playbook builder.

### Phase 4: Validation + Cleanup
13. Re-run `scenario-auditor audit prompt-manager --standards-only --timeout 60`.
14. Re-run `make test` in `scenarios/prompt-manager/`.
15. Confirm 0 critical/high violations and all 5 test phases pass.
16. **Final cleanup**: run `vrooli scenario restart prompt-manager` to reset scenario state and verify the scenario comes back up cleanly with the new binary name and regenerated registry.

## 10. Testing Plan

- `scenario-auditor audit prompt-manager --standards-only --timeout 60` after each phase touching standards.
- `make test` in `scenarios/prompt-manager/` after all phases.
- Each phase verified independently: standards, docs, unit, integration, playbooks.
- Final `vrooli scenario restart prompt-manager` to confirm runtime health.

## 11. Rollout/Validation Checklist

- [ ] Critical standards violations → 0
- [ ] High standards violations → 0
- [ ] No `cli/pm` references remain in repo (greenfield rename complete)
- [ ] docs test phase passes
- [ ] unit test phase passes
- [ ] integration test phase (CLI unknown command) passes
- [ ] playbooks test phase (`bas/registry.json`) passes
- [ ] standards test phase passes
- [ ] `vrooli scenario restart prompt-manager` succeeds cleanly

## 12. Risks + Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| CLI binary rename breaks external/in-repo consumers of `pm` | Medium | Grep entire repo for `pm` invocations; update or remove every reference before merging (greenfield: no alias). |
| PRD content truly missing (not just mislinked) for some refs | Low | Verify candidate PRD section exists before updating each `prd_ref`; if no section fits, surface as a follow-up rather than fabricating PRD content. |
| UI unit test failures are deep/unrelated to tidiness | Medium | Triage first; fix what is reasonable within scope; if systemic, file a separate backlog item and document in handoff. |
| Adjacent-medium fixes balloon scope | Low | Hard 5-minute-per-fix rule from d3=B; if a "medium" fix grows, defer it. |
| `vrooli scenario restart` surfaces a regression from the rename | Medium | Restart is the final gate; if it fails, do not mark done — diagnose, fix, restart again. |

## 13. Non-goals / Prohibited Patterns

- Do not fix standalone low/info violations or non-adjacent medium violations.
- Do not refactor or add features beyond what standards require.
- Do not modify files outside `scenarios/prompt-manager/`.
- Do not add a `pm` → `prompt-manager` symlink, alias, or compatibility shim (greenfield).
- Do not extend `PRD.md` to fit stale `prd_ref` values; relink instead.

## 14. Definition of Done

- All 5 GCT test phases pass.
- 0 critical and 0 high standards violations.
- No remaining `cli/pm` references in the repo.
- `vrooli scenario restart prompt-manager` runs to a clean healthy state.
- Tidiness check re-run shows pass.
