### Tools run
- `development-toolchain-validator validate /home/matthalloran8/Vrooli/scenarios/reference-react-vite` -> failed: `Unknown command: validate`
- `development-toolchain-validator report --conflicts|--drift|--maturity|--tool-baselines` -> failed: `Unknown command: report`
- `scenario-auditor --auto-start status` -> healthy
- `test-genie --auto-start status` -> healthy
- `scenario-auditor standards scan reference-react-vite --wait` -> job `standards-1d600f94-2209-433b-bed5-0d54ba7e2f3f`, 17 violations
- `scenario-auditor security scan reference-react-vite --wait` -> job `security-e994fa5e-735b-4a8e-a54b-476ea82294a4`, 2 Critical findings
- `test-genie run-tests reference-react-vite` -> failed: opaque API 500, `scenario tests failed`
- `tidiness-manager scan reference-react-vite` and `tidiness-manager --auto-start scan reference-react-vite` -> failed before scan: read-only temp binary write plus API-base/autostart failure
- Absolute-path probes for `scenario-auditor standards scan` and `test-genie run-tests` -> API 404; slug still works

### Reference scenario
- `/home/matthalloran8/Vrooli/scenarios/reference-react-vite`
- Caveat: reference working tree remains dirty with untracked reference files, including `ui/src/routes.generated.ts`, new pages, contexts, route tests, flow artifacts, and BAS flow artifacts. Deltas are not cleanly attributable to committed reference state.

### Violation summary
- Critical: 3 total: 1 standards + 2 security
- Major: 3 standards Medium
- Minor: 13 standards: 12 Low + 1 Info

### Top 3 violations
1. Critical - scenario-auditor security `AUTH-002` - false-positive hardcoded credential detections on generated `forgotPassword: "/forgot-password"` route constants in `ui/src/routes.generated.ts` lines 9 and 25.
2. Critical - scenario-auditor standards `required_layout` - `Makefile`, recommendation remains terse: `Add the required resource at Makefile`.
3. Medium - scenario-auditor standards `testing-standards-v1` - missing test file in `reference-react-vite/cli/domains`.

### New since last scan
- None material. Counts and failure modes match 2026-05-15.
- Persistent confirmation: absolute path arguments still return API 404 while scenario slug works.

### Resolved since last scan
- None confirmed.

### Capability gaps noticed
- DTV `validate`/`report` still absent.
- `test-genie run-tests` still returns an opaque API 500.
- `tidiness-manager` still cannot scan in this sandbox due read-only temp binary write and API-base/autostart detection failure.
- scenario-auditor security false-positive on generated route names containing `password` persists.
- Fallback CLI argument contracts still create manual retry/filter pressure.

### Action-adjacent signals
- Missing Action-equivalent remains DTV consolidated validation/reporting.
- Repeated deterministic workaround remains: manually aggregate scenario-auditor standards/security, test-genie, and tidiness-manager, while remembering slug-vs-path behavior.
- Existing accepted/pending decisions cover the durable gaps, so no new Action/capability proposal was added.

### Decisions raised this heartbeat
- None. No material change, and existing accepted/pending decisions cover DTV/test-genie/tidiness and the route-name security false positive.

### Knowledge entries written
- `knw-1778968916512606378` - `toolchain-audit/2026-05-16`
- `knw-1778968928960534958` - `friction-report/toolchain/2026-05-16/fallback-toolchain-still-requires-manual-aggregation`