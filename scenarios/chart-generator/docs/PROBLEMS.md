# Known Issues & Follow-ups

## Resolved Issues

### 1. BAS Playbook Format Incompatibility (Iteration 15-16) ✅ RESOLVED
**Severity**: Critical (historical)
**Status**: Current UI integration uses the BAS-owned session contract.

**Current evidence (2026-11-19)**:
- Replaced the private Playwright integration file with `test/integration.test.mjs`; it uses typed BAS screenshot/Observe and AX-ref actions with a fixed localhost URL and AM run authority.
- Eleven fake-owner tests cover app title/preview, all five chart types, three themes, grant denial, redirect denial and session cleanup.
- The generated SVG now exposes an accurate accessible image name, and the preview container is a named region so the existing visibility assertion has an accessible surface.
- No live browser, target, grant or real UI behavior was tested. This is source/fake-owner evidence only; the full scenario suite is not claimed green.

**Historical Iteration 16**: The previous direct Playwright suite replaced incompatible BAS workflow files. That suite and its browser launcher have now been superseded by the managed-session integration above.

**Original Problem (Iteration 15)**:
- Chart-generator playbooks used step-based format (`steps` array)
- BAS API required node-based format (`nodes`/`edges` arrays)
- All 10 BAS workflows failed with `"json: unknown field \"steps\""`

**Legacy playbooks**: Deprecated playbook references have been removed; use `bas/cases/` going forward

## Notes for Future Agents

- Test files are located in `api/*_test.go`, `cli/chart-generator.bats`, and the BAS fake-owner UI suite at `test/integration.test.mjs`.
- Module structure exists in `requirements/01-*/module.json` format
- All 15 operational targets are passing
- CLI has 15/15 BATS tests passing
- Go unit tests have 123/123 passing
