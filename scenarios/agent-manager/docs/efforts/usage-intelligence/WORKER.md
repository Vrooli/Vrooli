# Worker card — Usage Intelligence effort

Read this, then your assignment, then the SPEC/EVIDENCE/DEBT sections it names.
Paths are relative to the Vrooli repo root.

## Hard rules

- **Never uninstall or remove Claude Code** (operator F-5): no `vrooli resource uninstall claude-code`,
  `resource-claude-code uninstall`, live-HOME `statusline uninstall`, or deleting the claude binary/config.
  The orchestrator runs on it. Test uninstall paths only against a temp HOME.

- **No git commands at all, not even read-only ones** (`git status`, `git diff`, `git mv` included). The operator owns version control. Use `mv`, `diff`, `wc` and `find` instead.
- **No bulk formatters.** Run `gofumpt -w` only on files you changed; UI formatting only on files you changed.
- **No raw package managers** for dependency changes (`pnpm add`, `go get`, `npm install`). Dependency
  work goes through `scenario-dependency-analyzer deps install …`. Running existing test scripts is fine.
- **Never run scenario binaries directly.** Lifecycle is `make -C scenarios/agent-manager restart|logs|status`
  or `vrooli scenario …`, and **only the orchestrator restarts** the agent-manager API/UI. If you need a
  live restart to validate, finish everything else and say so in your return.
- **Protobufs** change only through `cd packages/proto && make generate SCENARIO=agent-manager`, and only
  when your assignment grants proto ownership. Never hand-edit generated output.
- **Shared tree.** Other workers edit other files in the same checkout at the same time. Edit only the
  files your assignment owns. If a build or test fails in a file you do not own, wait a few minutes and
  retry; if it persists, report it — do not "fix" a peer's file.
- **Database.** Read the live DB only read-only
  (`sqlite3 "file:scenarios/agent-manager/data/agent-manager.db?mode=ro"`). Schema changes ship as code
  the API applies on start (follow the existing migration/EnsureSchema pattern in the package you touch).

## Quality bar

- Replace, don't wrap: delete the superseded code named in your DEBT rows. No shims, dual paths,
  compatibility flags or dead code. Net non-test LOC in your area should fall.
- Tests assert the desired behavior in SPEC, not today's behavior; prefer shared fixtures/generators
  over hard-coded copies. Exercise the real production path (e.g. the real executor/interactive/recovery
  sinks), not a helper the production path skips.
- Update the docs your change affects (scenario `docs/`, requirements module refs) in the same change.
- Adjacent defects: fix when understood and blocking; otherwise list them in your return under Findings.
- Blocked means a decision, credential or approval you lack; name it. Friction you can diagnose is not
  blocked.

## Validation

- Go: `cd scenarios/agent-manager/api && go test ./internal/<pkg>/...` for the packages you touched,
  plus `go build ./...` and `go vet` on them.
- CLI: `cd scenarios/agent-manager/cli && go test ./...` when you touch it.
- UI: the scenario's existing test/type-check scripts (`make -C scenarios/agent-manager type-check`,
  vitest on touched specs).
- Scoped Test Genie phases only when your assignment names them: `vrooli scenario test agent-manager
  --phases <phases>`, then block once with `test-genie runs wait --json agent-manager <run-id>`. Never poll.

## Return format

```
Changed: files (created/modified/deleted) with one line each
Verified: each command and its result (paste key output lines)
DEBT: rows closed, before/after non-test LOC for your area
Remaining: what is not done
Unverified: what you could not prove and why (e.g. needs restart)
Findings: adjacent defects / UX issues found
```
