# Research Conclusion: Extract Reusable Signal From The Legacy Bridge

## Research Question
What aspects of the legacy `scenarios/vrooli-bridge` implementation contain reusable signal — requirements, trust/auth assumptions, data-model ideas, UI/CLI patterns, or mistakes to avoid — that should inform the greenfield trusted-node bridge replacing it under the `trusted-node-bridge` initiative?

## Summary
The legacy bridge's domain (external-project documentation injection) and the greenfield mission (trusted Vrooli↔Vrooli collaboration) overlap very little. The strongest reusable signal lives in three places: (1) the lifecycle / CLI / API scaffolding pattern that every Vrooli scenario already follows, (2) `PRD.md` requirement `OT-P0-007` ("connect to remote servers; sign in only the first time") which is the single legacy line that maps onto trusted-node identity, and (3) explicit anti-patterns from things the legacy got wrong — no auth, fragile UI port math, append-by-substring CLAUDE.md handling, hard-coded versions. Detailed catalog is being assembled across rounds; this round establishes the inventory and methodology and leaves direction-of-depth open for the user.

## Methodology
Read-only static review of `scenarios/vrooli-bridge/` files:

- `PRD.md`, `README.md`, `requirements/index.json` — declared scope and requirements
- `.vrooli/service.json`, `Makefile` — packaging and lifecycle
- `api/main.go`, `api/main_test.go` — API surface, behavior, and test conventions
- `cli/main.go`, `cli/app.go`, `cli/domains/domains.go`, `cli/domains/projects/register.go`, `cli/install.sh` — CLI surface and conventions
- `ui/src/index.html`, `ui/src/app.js`, `ui/package.json` — UI surface
- `initialization/storage/postgres/schema.sql`, `initialization/storage/postgres/seed.sql`, `initialization/templates/*` — data model and content templates

Cross-referenced sibling backlog items in initiative `trusted-node-bridge` (`chore/archive-and-remove-legacy-vrooli-bridge`, `idea/greenfield-vrooli-bridge-foundation-spec`, `execute/greenfield-vrooli-bridge-core-runtime`, `execute/vrooli-emulator-remote-node-backend`, `execute/macos-real-device-validation-over-bridge`) to align "reusable for what" with the new mission. Categorized findings by the dimensions named in the item description: requirements, trust/auth, data model, UI/CLI patterns, anti-patterns.

## Findings

### Finding 1: Mission mismatch — most domain logic does NOT carry over
The legacy bridge solves "discover external projects on disk and inject `VROOLI_INTEGRATION.md` / `CLAUDE.md` so they can call Vrooli scenarios" (`PRD.md` §🎯 Overview, requirements `VRB-REQ-001..006`). The greenfield mission is "trusted Vrooli↔Vrooli node collaboration for remote validation and emulator transport" (initiative `trusted-node-bridge`, idea `greenfield-vrooli-bridge-foundation-spec`). Project discovery, doc injection, project-type detection, integration-status tracking, and the `VROOLI_INTEGRATION.md.template` / `CLAUDE_ADDITIONS.md.template` artifacts are domain-specific to the old mission and should not be ported.

### Finding 2: Single requirement that maps to the new mission — OT-P0-007
`PRD.md` lists `OT-P0-007 | Remote server connection | Securely connect to remote servers. Require sign-in only the first time`. This is the only line in the legacy PRD that aligns with the greenfield direction (trust-on-first-use across machines). Notably, **there is no implementation** of OT-P0-007 anywhere in `api/`, `cli/`, or `ui/`: no auth code, no identity store, no token persistence, no remote-host config, no entry in `requirements/index.json`. Treat this as a *requirement statement to inherit*, not a code reference.

### Finding 3: Trust/auth assumptions in the legacy code — none, plus permissive defaults
- `api/main.go` `corsMiddleware` (≈L158) hardcodes `Access-Control-Allow-Origin: http://localhost:3000` and exposes all routes without any authentication or authorization check.
- The CLI core (`cli/app.go`) explicitly sets `AllowAnonymous: true`.
- The handler layer trusts client-supplied `directories` for `scanDirectory` and writes to client-supplied filesystem paths during `performIntegration` with no path containment, which is acceptable for a single-user local tool but explicitly disqualifying for a node-to-node trust boundary.

**Anti-pattern to avoid:** in the new bridge, every cross-node call must originate from an authenticated, capability-scoped identity; the legacy posture (anonymous + open CORS + arbitrary FS access) cannot be reused even by accident.

### Finding 4: Data-model ideas worth keeping
From `initialization/storage/postgres/schema.sql`:

- **Schema patterns that generalize:** `UUID PRIMARY KEY DEFAULT gen_random_uuid()`, `JSONB metadata` columns, `status TEXT CHECK (status IN (...))` enums, a separate history/audit table with `action TEXT CHECK (...)`, a `BEFORE UPDATE` trigger to maintain `last_updated`, and a plpgsql function (`add_integration_history`) used as a single insert seam from handlers.
- **Domain tables that do NOT carry over:** `projects`, `integration_history`, `project_files`, `project_tags`. The greenfield equivalents will likely be `nodes`, `sessions` / `tasks`, and an audit table — but those are new tables, not migrations.

### Finding 5: CLI patterns that should be reused as-is
- `cli/main.go` + `cli/app.go` use `cliapp.NewStandardScenarioApp` with `StandardScenarioOptions{...}` and inject `domains.CommandGroups` + `domains.SubcommandGroups`. This is the current Vrooli CLI standard and should remain.
- `cli/domains/projects/register.go` shows the canonical subcommand-group shape: a single `Register(core)` function returning a `cliapp.SubcommandGroup` with `NeedsAPI: true` and verb-style `Subcommands` (`list`, `scan`, `integrate`, `remove`), each handler calling `core.Get` / `core.Request` and rendering via `cliapp.ListReport` / `cliapp.MutationReport`. The greenfield bridge should follow the same shape (likely `node` and `session` subcommand groups with verbs like `list`, `connect`, `revoke`, `attach`, `detach`).
- The `support.NewFlagSet` + `cliutil.JSONFlag` per command + `--body-file` escape hatch is a reusable convention for handler tests and machine-driven invocations.

### Finding 6: API patterns that should be reused
- `api/main.go` uses `preflight.Run`, `server.Run`, `health.New().Version(...).Check(health.DB(...), health.Critical).Handler()`, and `database.Connect` from `github.com/vrooli/api-core`. These are the standard scenario-API building blocks.
- The server registers routes via `mux.Router` with a `/api/v1` `PathPrefix`. This is the convention.

**Anti-pattern to avoid:** the legacy CORS middleware hardcodes a single origin and exposes all routes. A trusted-node bridge needs origin policy keyed on registered node identities, not a fixed string.

### Finding 7: UI patterns — minimal signal, must rebuild
- `ui/` is a vanilla-JS dashboard (`src/index.html`, `src/app.js`, `src/styles.css`) served by a tiny static `server.js` / `server.py`. Its `package.json` "build" script is literally `rm -rf dist && mkdir -p dist && cp -R src/* dist/`.
- This violates the project guideline that every scenario uses the react-vite template.
- The only reusable UX ideas are the page composition (header actions, stat cards, filter row, list view, action modal) and the action set per row (integrate / update / remove). The new UI's domain — listing trusted nodes and live sessions — will reuse those compositional ideas, not any code.

**Anti-patterns to avoid:** `getApiPort()` deriving the API port as `current_port + 5000`; `getDefaultScanPath()` reading `process.env.HOME` from the browser (always undefined, falls through to a literal `/home/user`); polling refresh every 30s instead of using a backend push or stale-while-revalidate.

### Finding 8: Lifecycle and packaging conventions to keep
- `.vrooli/service.json` defines `lifecycle.version: "2.0.0"` with `setup` / `develop` / `test` / `stop` phases, `health.checks` (HTTP), and `dependencies.resources` (postgres + redis). These are the standard scenario manifest patterns.
- The scenario `Makefile` is a thin wrapper around `vrooli scenario <verb>` and matches the project standard.
- **Reusable as-is:** the manifest skeleton, port `env_var` allocation, the HTTP health-check pattern.

### Finding 9: Concrete anti-patterns harvested from the implementation
1. **Append-by-substring CLAUDE.md merge** (`api/main.go` ≈L455): integration appends to `CLAUDE.md` only if `!strings.Contains(content, "Vrooli Integration")`. Any prior text containing that substring blocks the append; any rename of the marker silently double-appends. The new bridge must not rely on free-text markers for idempotency.
2. **Hard-coded versions on every integrate** (`api/main.go` ≈L382-388): `vrooli_version` and `bridge_version` are stored as the literal string `"1.0.0"`. No source of truth, no update path. Versioning must come from build metadata.
3. **`ioutil` everywhere** (`api/main.go`): deprecated since Go 1.16. New code should use `os.ReadFile` / `os.WriteFile`.
4. **Test fixtures don't match the catalog vocabulary** (`api/main_test.go` `setupTestProject(t, env, "nodejs")` and `TestProjectTypeCoverage` line `{"package.json", "nodejs"}`): the project-types catalog (`initialization/templates/project-types.json`) defines `npm`, not `nodejs`. The bridge tests must match the type vocabulary exactly.
5. **Test pollution of process env** (`api/main_test.go` `TestInitDB`, `TestDatabaseConnection`): tests `os.Unsetenv` postgres vars and "restore" with `os.Setenv` only when the original was non-empty — irreversible if the original was unset-but-present in another test. New tests must use isolated env via `t.Setenv` and/or testcontainers.
6. **CORS pinned to one origin** (`api/main.go` ≈L160): see Finding 3.
7. **UI uses Node-only globals in the browser** (`ui/src/app.js` ≈L55): see Finding 7.

## Limitations
- This round did not exhaustively walk every helper file; `api/test_helpers.go`, `api/test_patterns.go`, and `cli/internal/support/` were skimmed only. Confidence on Findings 1, 2, 3, 9 is high; confidence on the completeness of Finding 5 (CLI patterns) is medium.
- The greenfield decisions on identity model, transport, and session model live in the sibling idea `idea/greenfield-vrooli-bridge-foundation-spec`, which has not yet been refined. This conclusion intentionally does NOT propose those — it only catalogs what to carry vs. drop from the legacy.
- No external sources (commits, PRs, prior incidents) were consulted; signal is limited to the current state of files on disk.
- Open questions in workshop round 001 (depth, output format, anti-pattern emphasis, scope of "reusable") gate the depth and format of subsequent rounds.

## Actions

<!-- TBD — actions will be finalized in a later round once the user has resolved the open decisions. The likely shape: -->

### Action 1 (likely): Update document — Hand harvested signal to the foundation spec
- **File**: `scenarios/swarm-manager/.../idea/greenfield-vrooli-bridge-foundation-spec/` (and/or its `archive/`)
- **Change**: Embed Findings 2, 3, 4, 6, 8, and the anti-pattern list (Finding 9) as input context for the architecture round so it starts with eyes open.

### Action 2 (likely): Update document — Inherit OT-P0-007 verbatim
- **File**: `idea/greenfield-vrooli-bridge-foundation-spec` plan/spec
- **Change**: Carry over the single requirement (`Securely connect to remote servers. Require sign-in only the first time`) as a P0 acceptance line for the greenfield runtime, attributed to the legacy PRD.

### Action 3 (conditional): Create backlog item — Extract a genuinely reusable component
- **Trigger**: Only if a future round identifies a reusable component (none so far) that should live in `packages/` before `chore/archive-and-remove-legacy-vrooli-bridge` runs.
- **Kind**: fix
- **Initiative**: trusted-node-bridge
