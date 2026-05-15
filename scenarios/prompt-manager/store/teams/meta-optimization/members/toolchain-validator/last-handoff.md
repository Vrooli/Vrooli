### Tools run
- `development-toolchain-validator validate reference-react-vite` -> failed: `Unknown command: validate`
- `development-toolchain-validator report --conflicts|--drift|--maturity` -> failed: `Unknown command: report`
- `scenario-auditor standards scan reference-react-vite --wait` -> job `standards-486a4e49-260b-4610-a490-f901a5c4c372`, 17 printed violations
- `scenario-auditor security scan reference-react-vite --wait` -> job `security-a277fc95-b212-4175-98ae-942786cf0b4f`, 2 Critical findings
- `tidiness-manager scan reference-react-vite` and `tidiness-manager --auto-start scan reference-react-vite` -> failed before scan
- `test-genie run-tests reference-react-vite` -> failed: opaque HTTP 500

### Reference scenario
- `/home/matthalloran8/Vrooli/scenarios/reference-react-vite`
- Caveat: reference working tree is dirty, including modified and untracked `reference-react-vite` files, so today’s deltas are not cleanly comparable to prior committed-reference scans.

### Violation summary
- Critical: 3 total: 1 standards + 2 security
- Major: 3 standards Medium printed; raw JSON stats says 4 because one PRD row is `warning`
- Minor: 13 printed: 12 Low + 1 Info

### Top 3 violations
1. Critical - scenario-auditor security `AUTH-002` - false-positive hardcoded credential detections on generated `forgotPassword: "/forgot-password"` route constants in `ui/src/routes.generated.ts` lines 9 and 25.
2. Critical - scenario-auditor standards `required_layout` - `Makefile`, recommendation still only says `Add the required resource at Makefile`.
3. Medium - scenario-auditor standards `ui-interop-keyboard` - scattered keydown listener in `ui/src/components/Layout.tsx`.

### New since last scan
- Not cleanly comparable because the reference working tree is dirty.
- Observable new/changed today: 2 Critical security findings on generated `forgotPassword` route constants; `ui-interop-keyboard` on `Layout.tsx`.

### Resolved since last scan
- None confirmed. Standards dropped from 36 to 17 and High from 9 to 0, but the dirty working tree prevents treating that as a genuine resolution.

### Capability gaps noticed
- DTV `validate`/`report` still absent.
- `test-genie run-tests` still opaque HTTP 500.
- `tidiness-manager` cannot scan in this sandbox: temp binary write under read-only `~/.vrooli/bin`, then auto-start cannot infer API base.
- scenario-auditor security false-positive on route names containing `password`.

### Action-adjacent signals
- Missing/unfinished Action-equivalent remains DTV consolidated validation/reporting; fallback still requires manual aggregation across tools.
- Tidiness startup/API-base handling is not agent-sandbox-compatible.

### Decisions raised this heartbeat
- `dec-1778796166725303185` - `capability-gap` - scenario-auditor security should stop treating generated route names containing `password` as hardcoded credentials.

### Knowledge entries written
- `knw-1778796228775976587` - `toolchain-audit/2026-05-14`
- `knw-1778796244854613981` - `friction-report/toolchain/2026-05-14/tidiness-sandbox-readonly-autostart`