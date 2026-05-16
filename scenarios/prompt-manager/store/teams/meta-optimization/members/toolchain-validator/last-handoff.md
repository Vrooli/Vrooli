### Tools run
- `development-toolchain-validator validate /home/matthalloran8/Vrooli/scenarios/reference-react-vite` -> failed: `Unknown command: validate`
- `development-toolchain-validator report --conflicts|--drift|--maturity|--tool-baselines` -> failed: `Unknown command: report`
- `scenario-auditor standards scan /home/.../reference-react-vite --wait` -> failed: API 404
- `scenario-auditor security scan /home/.../reference-react-vite --wait` -> failed: API 404
- `test-genie run-tests /home/.../reference-react-vite` -> failed: API 404
- `scenario-auditor --auto-start status` -> healthy
- `test-genie --auto-start status` -> healthy
- `scenario-auditor standards scan reference-react-vite --wait` -> job `standards-c65efa81-cc1d-403f-9b8e-6ade26f3ee05`, 17 violations
- `scenario-auditor security scan reference-react-vite --wait` -> job `security-f40fd909-4a5e-4c60-a33b-ab639ef0529f`, 2 Critical findings
- `test-genie run-tests reference-react-vite` -> failed: opaque API 500, `scenario tests failed`
- `tidiness-manager scan reference-react-vite` and `tidiness-manager --auto-start scan reference-react-vite` -> failed before scan: read-only temp binary write plus API-base/autostart failure

### Reference scenario
- `/home/matthalloran8/Vrooli/scenarios/reference-react-vite`
- Caveat: reference working tree remains dirty with untracked `reference-react-vite` files, including `ui/src/routes.generated.ts`, new pages, contexts, route tests, and BAS flow artifacts. Deltas are not cleanly attributable to committed reference state.

### Violation summary
- Critical: 3 total: 1 standards + 2 security
- Major: 3 standards Medium
- Minor: 13 standards: 12 Low + 1 Info

### Top 3 violations
1. Critical - scenario-auditor security `AUTH-002` - false-positive hardcoded credential detections on generated `forgotPassword: "/forgot-password"` route constants in `ui/src/routes.generated.ts` lines 9 and 25.
2. Critical - scenario-auditor standards `required_layout` - `Makefile`, recommendation remains terse: `Add the required resource at Makefile`.
3. Medium - scenario-auditor standards `testing-standards-v1` - missing test file in `reference-react-vite/cli/domains`.

### New since last scan
- No material scan-count change versus 2026-05-14.
- New friction observed: absolute path arguments to scenario-auditor and test-genie return API 404, while the scenario slug works.
- New friction observed: `scenario-auditor standards violations` and `security vulnerabilities` list global cached all-scenario results, not the just-run reference job.
- Storage-command drift: provided contract still says `prompt-manager team knowledge-add ... --by=...`, but CLI now rejects `--by` and uses runtime attribution.

### Resolved since last scan
- None confirmed.

### Capability gaps noticed
- DTV `validate`/`report` still absent.
- `test-genie run-tests` still returns an opaque API 500.
- `tidiness-manager` still cannot scan in this sandbox due read-only temp binary write and API-base/autostart detection failure.
- scenario-auditor security false-positive on generated route names containing `password` persists.
- Fallback CLI argument/cache contracts create manual filtering and retry pressure.

### Action-adjacent signals
- Missing/unfinished Action-equivalent remains DTV consolidated validation/reporting.
- Repeated deterministic workaround remains: manually aggregate scenario-auditor standards/security, test-genie, and tidiness-manager, then manually filter job summaries.
- `scenario-auditor standards violations` / `security vulnerabilities` would benefit from job/scenario filters or a scoped detail command.

### Decisions raised this heartbeat
- None. Existing accepted/pending capability gaps cover DTV/test-genie/tidiness, and pending `dec-1778796166725303185` covers the route-name security false positive.

### Knowledge entries written
- `knw-1778882583310798691` - `toolchain-audit/2026-05-15`
- `knw-1778882583514228331` - `friction-report/toolchain/2026-05-15/fallback-cli-argument-and-cache-contracts`