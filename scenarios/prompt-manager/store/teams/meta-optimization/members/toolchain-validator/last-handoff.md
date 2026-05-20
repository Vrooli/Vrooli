### Tools run
- `development-toolchain-validator validate /home/matthalloran8/Vrooli/scenarios/reference-react-vite` -> failed: `Unknown command: validate`
- `development-toolchain-validator report --conflicts|--drift|--maturity|--tool-baselines` -> failed: unknown subcommand; live report subcommands are `golden-summary`, `tuple-history`, `coverage`
- `scenario-auditor --auto-start status` -> healthy
- `test-genie --auto-start status` -> healthy, but stale-binary/read-only rebuild warning persists
- `tidiness-manager --auto-start status` -> healthy
- `scenario-auditor standards scan reference-react-vite --wait` -> job `standards-6cb833e4-fa8a-4ee7-bad5-3ae9e04e315d`, 37 violations
- `scenario-auditor security scan reference-react-vite --wait` -> job `security-9f07bce3-d8ce-418a-8d1b-9d211003bd8c`, 2 Critical findings
- `test-genie run-tests reference-react-vite` -> failed with `api error (400): scenario does not define runnable tests`
- `test-genie run reference-react-vite` -> failed: `Unknown command: run`
- `tidiness-manager scan reference-react-vite` -> completed; `make lint` failed exit 2, `make type` failed exit 2
- `tidiness-manager issues reference-react-vite --limit 20` -> 20 issues: 13 High, 7 Medium
- `tidiness-manager recommend-refactors reference-react-vite --limit 5`

### Reference scenario
- `/home/matthalloran8/Vrooli/scenarios/reference-react-vite`

### Violation summary
- Critical: 3
- Major: 43
- Minor: 13

### Top 3 violations
1. Critical - scenario-auditor security `AUTH-002` - persistent generated-route false positives in `ui/src/routes.generated.ts` lines 9 and 25.
2. Critical - scenario-auditor standards `required_layout` - missing required `Makefile` structure.
3. High - scenario-auditor standards - standards count regressed from 18 to 37, including TypeScript strict/build quality gates, RoutedDB seam findings, missing golangci configs, and persistent Go CLI workspace-independence.

### New since last scan
- Standards violations increased from 18 to 37.
- Standards High count increased from 1 to 10; Medium count increased from 3 to 13.
- New/material families include RoutedDB seam findings, expanded TypeScript strict-mode findings, build-script typecheck findings, and missing golangci config findings.
- Tidiness-manager now returns 20 open issues instead of failing before scan.

### Resolved since last scan
- `test-genie run-tests` no longer timed out; it now returns a concrete 400 about no runnable tests.
- `tidiness-manager scan` no longer fails before scan in this sandbox; it produces scan/issues/recommendation output.

### Capability gaps noticed
- DTV `validate`/documented `report` flags remain absent.
- `test-genie run` remains documented but unsupported.
- `test-genie run-tests` still cannot provide test results because the reference reports no runnable tests.
- Stale CLI auto-rebuild into `/home/matthalloran8/.vrooli/bin` still fails due read-only filesystem.
- Manual aggregation across scenario-auditor, test-genie, and tidiness-manager remains required.

### Action-adjacent signals
- Missing Action-equivalent remains consolidated DTV validation/reporting.
- Repeated workaround remains manual aggregation of standards/security/test/tidiness outputs.
- No new Action proposal raised; queue is already over governance ceiling and existing decisions cover the main gaps.

### Decisions raised this heartbeat
- None. Team pending queue is above the governance ceiling; material gaps are already covered by accepted/pending decisions.

### Knowledge entries written
- `knw-1779314522205319273` - `toolchain-audit/2026-05-20`
- `knw-1779314536503574803` - `friction-report/toolchain/2026-05-20/fallback-toolchain-dtv-report-and-test-surface-drift`