### Tools run
- `development-toolchain-validator validate /home/matthalloran8/Vrooli/scenarios/reference-react-vite` -> failed: `Unknown command: validate`
- `development-toolchain-validator report --conflicts|--drift|--maturity|--tool-baselines` -> failed: `Unknown command: report`
- `scenario-auditor --auto-start status` -> healthy
- `test-genie --auto-start status` -> healthy
- `tidiness-manager --auto-start status` -> failed before scan: read-only temp binary write plus API-base/autostart detection failure
- `scenario-auditor standards scan reference-react-vite --wait` -> job `standards-cda53b4b-e13e-4cbd-95b8-0a395a1c1b02`, 18 violations
- `scenario-auditor standards summary standards-cda53b4b-e13e-4cbd-95b8-0a395a1c1b02`
- `scenario-auditor security scan reference-react-vite --wait` -> job `security-453a80a6-8d1e-45da-a905-854a639813e8`, 2 Critical findings
- `scenario-auditor security summary security-453a80a6-8d1e-45da-a905-854a639813e8`
- `test-genie run-tests reference-react-vite` -> failed: context deadline exceeded awaiting headers
- `test-genie run reference-react-vite` -> failed: `Unknown command: run`
- `tidiness-manager scan reference-react-vite` -> failed before scan: unable to resolve API base plus stale binary cannot rebuild
- Absolute-path probes: `scenario-auditor standards scan /home/.../reference-react-vite --wait` and `test-genie run-tests /home/.../reference-react-vite` both API 404

### Reference scenario
- `/home/matthalloran8/Vrooli/scenarios/reference-react-vite`
- Caveat: working tree remains dirty with broad unrelated modified/untracked files across Vrooli, including `packages/cli-core`, `development-toolchain-validator`, templates, and generated `reference-react-vite` UI route/page/flow artifacts.

### Violation summary
- Critical: 3
- Major: 4
- Minor: 13

### Top 3 violations
1. Critical - scenario-auditor security `AUTH-002` - persistent false-positive hardcoded credential detections on generated `forgotPassword` route constants in `ui/src/routes.generated.ts` lines 9 and 25.
2. Critical - scenario-auditor standards `required_layout` - `Makefile`, recommendation remains terse: `Add the required resource at Makefile`.
3. High - scenario-auditor standards `stack-governance` / `GO_CLI_WORKSPACE_INDEPENDENCE` - `cli` reports missing `go.sum` entry for `github.com/santhosh-tekuri/jsonschema/v5` imported by `packages/cli-core/cliapp/manifest_proto.go`.

### New since last scan
- Standards count increased from 17 to 18.
- New High `stack-governance` finding on Go CLI module workspace independence.
- Prompt-manager also emitted stale-binary/read-only auto-rebuild warnings; commands still returned usable results. Tidiness-manager treats the same environment class as fatal before scan.

### Resolved since last scan
- None confirmed.

### Capability gaps noticed
- DTV `validate`/`report` still absent.
- `test-genie` still cannot produce test findings; `run-tests` timed out again.
- `test-genie run` remains documented but unsupported.
- `tidiness-manager` still cannot scan in this sandbox.
- Absolute path arguments still return API 404 while slug works.
- scenario-auditor security false-positive on generated route names containing `password` persists.
- Stale CLI auto-rebuild into `/home/matthalloran8/.vrooli/bin` fails because the path is read-only.

### Action-adjacent signals
- Missing Action-equivalent remains DTV consolidated validation/reporting.
- Repeated workaround remains manual aggregation of standards/security/test/tidiness outputs plus slug-vs-path handling.
- No new Action proposal raised; this run raised a toolchain-violation decision for the new High standards finding.

### Decisions raised this heartbeat
- `dec-1779141739837517904` - `toolchain-violation` - reference-react-vite standards scan now reports a new High `stack-governance` violation for Go CLI module workspace independence.

### Knowledge entries written
- `knw-1779141767132652979` - `toolchain-audit/2026-05-18`
- `knw-1779141782530159394` - `friction-report/toolchain/2026-05-18/fallback-toolchain-stack-governance-and-stale-binaries`