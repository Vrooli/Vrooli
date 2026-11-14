# Known Issues & Follow-ups

## Resolved Issues

### 1. BAS Playbook Format Incompatibility (Iteration 15-16) ✅ RESOLVED
**Severity**: Critical (blocked all integration tests)
**Status**: ✅ **RESOLVED via Playwright pivot**

**Resolution (Iteration 16)**:
- **Pivoted to Playwright** for UI integration testing instead of converting BAS playbooks
- Created `test/integration.spec.ts` with 9 Playwright tests covering:
  - All 5 core chart types (bar, line, pie, scatter, area)
  - 3 style themes (professional, minimal, vibrant)
  - UI load validation
- **All 9 Playwright tests passing** (11.6s total)
- Rewrote `test/phases/test-integration.sh` to use Playwright instead of BAS workflow runner
- **Test suite**: 6/6 phases passing ✅ (structure, dependencies, unit 107 tests/50.9%, integration 9 Playwright tests, business, performance)

**Original Problem (Iteration 15)**:
- Chart-generator playbooks used step-based format (`steps` array)
- BAS API required node-based format (`nodes`/`edges` arrays)
- All 10 BAS workflows failed with `"json: unknown field \"steps\""`

**Why Playwright was chosen**:
- Faster implementation (no format conversion needed)
- Simpler test authoring (TypeScript vs BAS JSON)
- No dependency on BAS infrastructure
- Better debugging (Playwright DevTools vs BAS logs)

**Legacy playbooks**: Deprecated playbook references have been removed; use `bas/cases/` going forward

## Notes for Future Agents

- Test files are correctly located in `api/*_test.go` and `cli/chart-generator.bats`
- Module structure exists in `requirements/01-*/module.json` format
- All 15 operational targets are passing
- CLI has 15/15 BATS tests passing
- Go unit tests have 123/123 passing
