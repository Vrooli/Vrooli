### Tools run
- `scenario-auditor standards scan reference-react-vite --wait` (job `standards-18dd3de9-…`)
- `scenario-auditor security scan reference-react-vite --wait` (job `security-87a99c05-…`)
- `scenario-auditor scenarios health reference-react-vite`
- `tidiness-manager scan reference-react-vite`
- `test-genie run-tests reference-react-vite` (failed: HTTP 500, opaque)
- `development-toolchain-validator reference list` (validate/report still not shipped)

### Reference scenario
- `reference-react-vite` — operator-nominated 2026-04-24 (`dec-1776981723540926630`). Registered in DTV (id `37cbef99-12c7-4513-a78e-0648ec6136a8`). History row added to `docs/meta-optimization/REFERENCE_SCENARIOS.md`.

### Violation summary
- Critical: 0 (severity scale tops out at High)
- Major (High): **41**
- Minor (Medium): 10 · Low: 20 · Info: 1
- **Total: 72**

### Top 3 violations
1. **High · scenario-auditor `stack-governance` (×37) · `scenarios/reference-react-vite/Makefile`** — header incomplete, color constants wrong, required PHONY targets undefined (`check`, `lint-go`, `lint-ui`, `fmt-ui`, etc.), lifecycle targets don't call `vrooli scenario …`. Confirmed by tidiness-manager: `make lint`/`make type` exit 2.
2. **High · scenario-auditor `type-safety` (×6) · `ui/tsconfig.json` + `ui/eslint.config.js`** — `strict`/`noUncheckedIndexedAccess` off, no protective comment block, missing `tseslint.configs.strictTypeChecked`. 31 dangerous TS patterns flagged (mostly `as Type`).
3. **High · scenario-auditor `quality-gates` (×4) · `ui/package.json` + `.vrooli/testing.json`** — `vite build` runs without `tsc --noEmit`; `lint.handlers.{node_package,go_module}.strict` are off so lint findings can pass silently.

### New since last scan
- All 72 violations (prior scan was BLOCKED, no baseline).

### Resolved since last scan
- Reference-designation blocker (`dec-1776981723540926630`) — operator accepted on 2026-04-24, nominated `reference-react-vite`.

### Capability gaps noticed
- `test-genie run-tests` returns flat HTTP 500 with no structured failure output; HEARTBEAT.md's documented verb `test-genie run <reference>` doesn't exist (actual verb is `run-tests`).
- DTV `validate <reference>` and `report --conflicts | --drift | --maturity | --tool-baselines` still not shipped — now consequential because a reference exists.

### Decisions raised this heartbeat
- `dec-1777068246086430656` · `toolchain-violation` · resolve reference rot on `reference-react-vite` (72 violations / 41 High); operator chooses cleanup vs. demote-and-re-nominate.
- `dec-1777068259096417622` · `capability-gap` · ship structured `test-genie run-tests` output + DTV `validate`/`report` surface so triage isn't blocked by opaque tooling.

### Knowledge entries written
- `knw-1777068225938400246` · `toolchain-scan-2026-04-24` (supersedes `toolchain-scan-2026-04-23`)