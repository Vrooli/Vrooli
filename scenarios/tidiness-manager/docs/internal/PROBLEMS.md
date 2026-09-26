# Known Problems & Risks

Track issues, blockers, and deferred decisions here. Keep open issues at the top and move resolved items to the bottom.

## Open Issues

## Work ladder

- Rung: W3 / R0
- Evidence: W0 passes because active goal `tidiness-manager-declarative-canonical-seams` names AST-based declarative canonical seams and `OT-P0-011` requires the same capability; `business-health validate scenario tidiness-manager` and `vrooli scenario requirements validate tidiness-manager` both report `PASSED`. Fresh Test Genie run `20260906-084556-b6a3e3f1` reached terminal `FAIL` in 387 seconds with 17 of 27 phases passing, 9 failing, and 1 skipped. The scoped consolidation-relevant phases pass; the failed phases include `ui-health`, `dependencies`, `docs`, `unit`, `storage`, and `workflow`, with provider-unavailable phases also present. The direct owner suite additionally reports 723 executable artifacts against the existing 472 reserve. The scenario therefore remains at W3 / R0 despite the scoped seam-control repair.
- Blocker: the runnable-and-green baseline remains below the declarative-seam implementation work; the current consolidation repairs the scoped seam-control implementation without claiming the unrelated W3 findings or repository artifact-reserve drift are resolved.
- Measured: 2026-09-06

### Completeness Tool BATS Test Recognition Limitation
**Status**: Open (Ecosystem-level tool limitation)
**Severity**: Medium (affects completeness score calculation)
**Description**: The scenario completeness tool (`vrooli scenario completeness`) does not recognize CLI BATS tests (`test/cli/*.bats`) as valid test locations. It only recognizes:
- `api/**/*_test.go` (API unit tests)
- `ui/src/**/*.test.tsx` (UI unit tests)
- `test/playbooks/**/*.{json,yaml}` (e2e automation)

**Impact**: 21/62 requirements reference `test/cli/` paths which are flagged as "unsupported test directories" even though the BATS tests exist, are comprehensive, and pass. This incorrectly penalizes the completeness score (-8pts) and misreports 57 P0/P1 requirements as lacking "multi-layer validation" when they actually have API + CLI test layers.

**Example**: TM-LS-001 (Makefile lint execution) has:
- Unit tests: `api/lint_execution_test.go` ✓
- Integration tests: `api/light_scanner_integration_test.go` ✓
- CLI tests: `test/cli/light-scanning.bats` ✓ (but flagged as invalid)

The completeness tool says "has: API → needs: API + E2E" but for backend-only requirements, CLI tests ARE the appropriate second validation layer, not browser automation.

**Root Cause**: Completeness tool validation logic doesn't account for CLI/BATS tests as a valid test layer for backend requirements. Tool expects browser e2e tests for all P0/P1 requirements regardless of whether they involve UI.

**Workaround**: Backend requirements can legitimately be validated via:
- API layer: Go unit/integration tests
- CLI layer: BATS integration tests
- E2E layer: Browser automation (only for UI-facing features)

**Mitigation**: Document that tidiness-manager's actual test coverage is comprehensive despite completeness score penalty. The 21 requirements with "invalid" test/cli/ references all have passing tests that validate their functionality.

**Next Steps**:
1. Update ecosystem completeness tool to recognize `test/cli/**/*.bats` as valid CLI test layer
2. Or: Update tool logic to allow backend requirements to achieve multi-layer validation via API + CLI (not requiring browser e2e)
3. Or: Accept completeness score penalty as tool limitation and document actual coverage separately

---
