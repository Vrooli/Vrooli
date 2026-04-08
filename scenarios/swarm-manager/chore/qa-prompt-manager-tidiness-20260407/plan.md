# Implementation Plan: Address prompt-manager Tidiness Failures

## 1. Purpose

Fix all tidiness failures flagged by the GCT review for the prompt-manager scenario, bringing it from red/yellow to green across standards, tests, and code quality dimensions.

## 2. Required Reading

```bash
prompt-manager skill read implementation-plan-authoring documentation-health
```

## 3. Problem Statement

The GCT review flagged prompt-manager with 5 test failures and 162 standards violations (5 critical, 19 high). The failures span five areas:

1. **Standards — critical**: Missing `cli/prompt-manager` binary (standards expect it; actual binary is `cli/pm`)
2. **Standards — critical**: 4 P0 operational targets missing linked requirements in `requirements/index.json`
3. **Standards — high**: Go CLI builds without workspace mode, PRD linkage broken for ~12 requirements, ESLint safety rules missing, resolveApiBase scattered, tsconfig missing protective comment
4. **Test failures**: docs validation, UI unit tests, CLI unknown-command check (returns 0 instead of non-zero), missing `bas/registry.json`
5. **Code quality**: Stale (needs re-run)

## 4. Scope

**In scope** (`acceptance_allow: scenarios/prompt-manager/**`):
- Fix critical and high standards violations
- Fix all 5 test failures
- Regenerate `bas/registry.json`
- Re-run code quality checks

**Out of scope:**
- Low/medium/info standards violations (154 of 162) — address only if trivially fixable alongside high+ fixes
- New features or refactors
- Changes outside `scenarios/prompt-manager/`

## 5. Current Technical Context

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

## 6. Target End State

- All 5 test phases pass (standards, docs, unit, integration, playbooks)
- 0 critical violations, 0 high violations from standards audit
- GCT re-review shows green/yellow-improving across all dimensions

## 7. Implementation Strategy

### Phase 1: Critical Standards Fixes
1. **CLI binary naming**: Either symlink `cli/prompt-manager` → `cli/pm` or rename the build output to `prompt-manager`
2. **P0/P1 requirements linkage**: Add requirement entries for the 4 unlinked P0 targets and 1 P1 target in `requirements/index.json`

### Phase 2: High Standards Fixes
3. **PRD linkage**: Fix ~12 REQ-* entries that reference missing PRD sections (update `prd_ref` fields to existing sections)
4. **Go workspace mode**: Add workspace replace directive in `cli/go.mod`
5. **ESLint safety rules**: Add `import/no-cycle: "error"` and other missing rules to ESLint config
6. **tsconfig protective comment**: Add required comment block
7. **resolveApiBase consolidation**: Move to single config/hook, import elsewhere

### Phase 3: Test Failures
8. **CLI unknown command**: Fix exit code handling in CLI core or prompt-manager's install.sh wrapper
9. **Docs validation**: Investigate and fix docs issues
10. **UI unit tests**: Run tests, diagnose failures, fix
11. **BAS registry**: Regenerate `bas/registry.json` via playbook builder

### Phase 4: Validation
12. Re-run `scenario-auditor audit prompt-manager --standards-only`
13. Re-run full test suite via `make test`
14. Verify 0 critical/high violations and all test phases pass

## 8. Contract Decisions

<!-- TBD — pending workshop decisions on CLI naming approach -->

## 9. Testing Plan

- Run `scenario-auditor audit prompt-manager --standards-only --timeout 60` after standards fixes
- Run `make test` in `scenarios/prompt-manager/` after all fixes
- Verify each test phase passes: standards, docs, unit, integration, playbooks

## 10. Rollout/Validation Checklist

- [ ] Critical standards violations → 0
- [ ] High standards violations → 0
- [ ] docs test phase passes
- [ ] unit test phase passes
- [ ] integration test phase (CLI unknown command) passes
- [ ] playbooks test phase (bas/registry.json) passes
- [ ] standards test phase passes

## 11. Risks + Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| CLI binary rename breaks other consumers | Medium | Check for references to `pm` binary name in other scenarios |
| PRD content truly missing (not just mislinked) | Low | Verify PRD sections exist before updating prd_ref |
| UI unit test failures are deep/unrelated | Medium | Triage — fix what's reasonable, flag rest if systemic |

## 12. Non-goals / Prohibited Patterns

- Do not fix low/medium/info violations unless trivially adjacent
- Do not refactor or add features
- Do not modify files outside `scenarios/prompt-manager/`

## 13. Definition of Done

All 5 GCT test phases pass, 0 critical/high standards violations, and tidiness check re-run shows pass.
