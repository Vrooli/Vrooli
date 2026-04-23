# Research Conclusion: Extract Reusable Signal From The Legacy Bridge

## Research Question
What aspects of the legacy `scenarios/vrooli-bridge` implementation contain reusable signal — requirements, trust/auth assumptions, data-model ideas, UI/CLI patterns, or mistakes to avoid — that should inform the greenfield trusted-node bridge replacing it under the `trusted-node-bridge` initiative?

## Summary
Almost nothing in the legacy bridge's domain logic carries over: it solves external-project documentation injection, while the greenfield mission is trusted Vrooli↔Vrooli node collaboration. The two pieces of signal that do carry over are (1) `PRD.md` requirement `OT-P0-007` — "securely connect to remote servers; sign in only the first time" — which is the only legacy line that matches the new mission, and (2) a rich set of concrete anti-patterns, chiefly the total absence of any trust boundary: no auth middleware, no identity store, no tokens, no mTLS, CORS pinned to `http://localhost:3000`, CLI `AllowAnonymous: true`, and a scan/integrate pipeline where client-supplied paths become filesystem writes. Framework and lifecycle patterns (Vrooli scenario template, `api-core`, `cliapp.ScenarioApp`, `.vrooli/service.json` v2.0.0) are standard across all scenarios and the greenfield runtime inherits them by default.

## Methodology
Read-only static review of `scenarios/vrooli-bridge/` files:

- `PRD.md`, `README.md`, `requirements/index.json` — declared scope and requirements
- `.vrooli/service.json`, `Makefile` — packaging and lifecycle
- `api/main.go`, `api/main_test.go`, `api/test_helpers.go`, `api/test_patterns.go` — API surface, behavior, trust posture, and test conventions
- `cli/main.go`, `cli/app.go`, `cli/domains/domains.go`, `cli/domains/projects/register.go`, `cli/install.sh` — CLI surface and conventions
- `ui/src/index.html`, `ui/src/app.js`, `ui/package.json`, `ui/server.js` — UI surface
- `initialization/storage/postgres/schema.sql`, `initialization/storage/postgres/seed.sql`, `initialization/templates/*` — data model and content templates

Round 002 added a targeted trust/auth deep-dive per user direction: whole-tree grep for `auth|token|identity|session|tls|jwt|bcrypt|hmac|secret|bearer|x509|credential` across all `*.go` files to confirm the *absence* of any trust primitives, and close reading of `corsMiddleware`, the scan/integrate request handlers, and the `TestCORSMiddleware` test. Cross-referenced sibling backlog items in initiative `trusted-node-bridge` to align "reusable for what" with the new mission.

## Signal Catalog

Compact catalog keyed on (subsystem, item, verdict, evidence). Verdicts: **keep** = carry directly, **adapt** = carry the pattern, not the code, **drop** = specific to the legacy mission, **avoid** = explicit anti-pattern to not repeat, **inherit** = delivered by the Vrooli scenario template regardless.

| Subsystem | Item | Verdict | Evidence |
|---|---|---|---|
| Requirements | `OT-P0-007` — secure remote connection, sign in first time only | **keep** (as requirement) | `PRD.md:23` |
| Requirements | `VRB-REQ-001..015` (project discovery, doc injection, project types) | **drop** | `requirements/index.json` |
| Requirements | `OT-P0-001..006` (discovery, doc gen, CLAUDE.md injection, integration status, CLI, postgres registry) | **drop** | `PRD.md:17-22` |
| Trust/auth | Authentication middleware | **missing — build it** | no match for `auth/token/jwt/bearer/tls` in any `*.go` |
| Trust/auth | Identity store / per-node credentials | **missing — build it** | no match anywhere |
| Trust/auth | `AllowAnonymous: true` CLI posture | **avoid** (must be authenticated per call) | `cli/app.go:36` |
| Trust/auth | CORS pinned to `http://localhost:3000`, all routes public | **avoid** | `api/main.go:158-171` |
| Trust/auth | Client-supplied paths flow to `filepath.Walk` / `ioutil.WriteFile` | **avoid** (no containment across trust boundary) | `api/main.go:222-234, 432, 449-465` |
| Trust/auth | TLS / mTLS / cert pinning | **missing — build it** | plain `server.Run`, all endpoints on `http://localhost` |
| Data model | `UUID PRIMARY KEY DEFAULT gen_random_uuid()` | **adapt** | `initialization/storage/postgres/schema.sql` |
| Data model | `JSONB metadata` column | **adapt** | ibid |
| Data model | `status TEXT CHECK (status IN (...))` enums | **adapt** | ibid |
| Data model | Separate audit/history table with `action TEXT CHECK (...)` | **adapt** | ibid (`integration_history`) |
| Data model | `BEFORE UPDATE` trigger to maintain `last_updated` | **adapt** | ibid |
| Data model | plpgsql insert-seam function (e.g., `add_integration_history`) | **adapt** | ibid |
| Data model | Tables `projects`, `integration_history`, `project_files`, `project_tags` | **drop** | specific to doc-injection mission |
| CLI | `cliapp.NewStandardScenarioApp` with domain groups | **inherit** | `cli/app.go:27-39` (template default) |
| CLI | Subcommand-group shape (`Register(core)` returning `cliapp.SubcommandGroup` with verb-style `list/scan/integrate/remove`) | **adapt** (use domain verbs: likely `node list/connect/revoke`, `session attach/detach`) | `cli/domains/projects/register.go` |
| CLI | `support.NewFlagSet` + `cliutil.JSONFlag` + `--body-file` escape hatch | **keep** | same file |
| API | `preflight.Run` + `server.Run` + `health.New().Version(...).Check(health.DB(...), health.Critical)` | **inherit** | `api/main.go:82-114, 152-153` |
| API | `mux.Router` with `/api/v1` `PathPrefix` | **keep** | `api/main.go:139-155` |
| API | `corsMiddleware` as written | **avoid** | `api/main.go:158-171` |
| UI | Vanilla JS, single `index.html` + `app.js`, `cp -R` "build" | **drop** (violates react-vite template) | `ui/src/*`, `ui/package.json` |
| UI | Page composition (header actions, stat cards, filter row, list, action modal) and per-row verb set | **adapt** (compositional ideas only, no code) | `ui/src/index.html`, `ui/src/app.js` |
| UI | `getApiPort()` derived as `current_port + 5000` | **avoid** | `ui/src/app.js` |
| UI | `getDefaultScanPath()` reading `process.env.HOME` from the browser | **avoid** | `ui/src/app.js` |
| UI | 30s polling refresh | **avoid** (use push or stale-while-revalidate) | `ui/src/app.js` |
| Lifecycle | `.vrooli/service.json` lifecycle v2.0.0 (setup/develop/test/stop + HTTP health checks + `dependencies.resources`) | **inherit** | `.vrooli/service.json` |
| Lifecycle | `Makefile` as thin wrapper around `vrooli scenario <verb>` | **inherit** | `Makefile` |

Long-form rationale follows in Findings.

## Findings

### Finding 1: Mission mismatch — most domain logic does NOT carry over
The legacy bridge solves "discover external projects on disk and inject `VROOLI_INTEGRATION.md` / `CLAUDE.md` so they can call Vrooli scenarios" (`PRD.md` §🎯 Overview, requirements `VRB-REQ-001..006`). The greenfield mission is "trusted Vrooli↔Vrooli node collaboration for remote validation and emulator transport" (initiative `trusted-node-bridge`, idea `greenfield-vrooli-bridge-foundation-spec`). Project discovery, doc injection, project-type detection, integration-status tracking, and the `VROOLI_INTEGRATION.md.template` / `CLAUDE_ADDITIONS.md.template` artifacts are domain-specific to the old mission and should not be ported.

### Finding 2: Single requirement that maps to the new mission — OT-P0-007
`PRD.md:23` lists `OT-P0-007 | Remote server connection | Securely connect to remote servers. Require sign-in only the first time`. This is the only line in the legacy PRD that aligns with the greenfield direction (trust-on-first-use across machines). Notably, **there is no implementation** of OT-P0-007 anywhere in `api/`, `cli/`, or `ui/`: no auth code, no identity store, no token persistence, no remote-host config, and no entry in `requirements/index.json`. Treat this as a *requirement statement to inherit*, not a code reference.

### Finding 3: Trust/auth posture — the total absence, documented
A whole-tree grep of all `*.go` files for any token, identity, session, JWT, HMAC, TLS, certificate, or bearer primitive returns **zero matches** outside the POSTGRES_* env variables used for the database connection. There is no auth middleware, no identity store, no per-node credential, no token issuance or verification. The posture consists of four concrete artifacts, each worth calling out explicitly because the greenfield mission inverts all of them:

1. **CLI is anonymous by design** — `cli/app.go:36` sets `AllowAnonymous: true` on the `StandardScenarioOptions`. The cli-core framework supports authenticated scenarios; the legacy bridge opts out.
2. **CORS is pinned to one origin, applied globally, independent of request origin** — `api/main.go:158-171` always emits `Access-Control-Allow-Origin: http://localhost:3000` and `Access-Control-Allow-Headers: Content-Type, Authorization` regardless of what the request's `Origin` header actually is. The `Authorization` header is *allowed* by the middleware but never *read* by any handler. The middleware returns 200 for every OPTIONS request on any path (even non-existent ones) before routing happens.
3. **Client-supplied filesystem paths become server-side filesystem operations without containment** — `scanProjects` (`api/main.go:212-264`) accepts `directories []string` from the request body, falls back to `os.UserHomeDir()` if empty, and passes each straight into `filepath.Walk`. The resulting `Project.Path` values are inserted into the database, and `integrateProject` (`api/main.go:354-473`) later reads those paths from the DB and `WriteFile`s into them. There is no allowlist, no denylist, no symlink resolution check. In a single-user localhost tool this is acceptable; across a trust boundary (the greenfield mission) it would let any authenticated caller — or any actor that can write to the DB — drive arbitrary filesystem writes on the server.
4. **No TLS, no mTLS, no cert pinning, no transport auth** — `server.Run` from `api-core` starts plain HTTP. All documented endpoints (`.vrooli/service.json:66`, `README.md:42-43`, `ui/server.js:115`) are `http://localhost:...`. The only transport security in the tree is `sslmode=disable` explicitly in test DSNs (`test_helpers.go:143`, `main_test.go:1705,1729`), which confirms the "trusted local" posture.

The greenfield bridge inverts all four: **every cross-node call must originate from an authenticated, capability-scoped identity over an authenticated transport**, and the server must not accept free-form paths as inputs to filesystem operations.

**Test-coverage gap that hides the problem**: `TestCORSMiddleware` in `api/main_test.go:935-998` only asserts `origin != ""` and `methods != ""`. It never asserts that the value of `Access-Control-Allow-Origin` matches the request's `Origin` header, so the hardcoded-origin bug is invisible to the test suite. Any greenfield CORS test should assert the *value* matches an allowlist keyed on registered node identity, not just the header's presence.

### Finding 4: Data-model ideas worth keeping
From `initialization/storage/postgres/schema.sql`:

- **Schema patterns that generalize:** `UUID PRIMARY KEY DEFAULT gen_random_uuid()`, `JSONB metadata` columns, `status TEXT CHECK (status IN (...))` enums, a separate history/audit table with `action TEXT CHECK (...)`, a `BEFORE UPDATE` trigger to maintain `last_updated`, and a plpgsql function (`add_integration_history`) used as a single insert seam from handlers.
- **Domain tables that do NOT carry over:** `projects`, `integration_history`, `project_files`, `project_tags`. The greenfield equivalents will likely be `nodes`, `sessions` / `tasks`, and an audit table — but those are new tables, not migrations.

### Finding 5: CLI pattern to adapt (not just inherit)
Every Vrooli scenario inherits `cliapp.NewStandardScenarioApp` from the template. What's *bridge-specific* and worth naming explicitly is the subcommand-group shape documented in `cli/domains/projects/register.go`: a single `Register(core)` function returning a `cliapp.SubcommandGroup` with `NeedsAPI: true` and verb-style `Subcommands` (`list`, `scan`, `integrate`, `remove`), each handler calling `core.Get` / `core.Request` and rendering via `cliapp.ListReport` / `cliapp.MutationReport`. The greenfield bridge should follow the same shape but with domain verbs — likely a `node` subcommand group (`list`, `connect`, `revoke`) and a `session` subcommand group (`attach`, `detach`, `logs`). The `support.NewFlagSet` + `cliutil.JSONFlag` per command + `--body-file` escape hatch is a keep-as-is convention for handler tests and machine-driven invocations.

### Finding 6: API pattern to adapt
The scenario-API scaffolding — `preflight.Run`, `server.Run`, `health.New().Version(...).Check(health.DB(...), health.Critical).Handler()`, `database.Connect` from `api-core`, and `mux.Router` with a `/api/v1` `PathPrefix` — is template-default and the greenfield runtime will use it unchanged. The one bridge-specific API lesson worth naming is negative: the `corsMiddleware` as written (Finding 3) must be replaced with origin policy keyed on registered node identities, not a fixed string.

### Finding 7: UI — rebuild, but keep the composition
`ui/` is a vanilla-JS dashboard (`src/index.html`, `src/app.js`, `src/styles.css`) served by a tiny static `server.js` / `server.py`. Its `package.json` "build" script is literally `rm -rf dist && mkdir -p dist && cp -R src/* dist/`. This violates the project guideline that every scenario uses the react-vite template. The only reusable UX ideas are the page composition (header actions, stat cards, filter row, list view, action modal) and the action set per row — not code.

## Conventions Inherited By Default (not bridge-specific)
Per user direction (round 001 d4=B), the following are standard across every Vrooli scenario and will be delivered by the scenario template regardless of what this conclusion says. Named here only for completeness; the foundation spec and runtime do not need to re-derive them:

- `.vrooli/service.json` lifecycle v2.0.0 with `setup` / `develop` / `test` / `stop` phases, `health.checks` (HTTP), and `dependencies.resources`.
- `Makefile` as a thin wrapper around `vrooli scenario <verb>`.
- `api-core` building blocks (`preflight`, `server`, `health`, `database`).
- `cliapp.NewStandardScenarioApp` + `StandardScenarioOptions`.
- Port `env_var` allocation and HTTP health-check pattern.
- Standard PostgreSQL + Redis resource declarations.

## Anti-patterns — Mistakes To Avoid In The Greenfield Bridge

These are concrete mistakes harvested from the legacy implementation. The greenfield foundation spec should treat each as a negative acceptance criterion.

1. **Anonymous-by-default CLI across a trust boundary.** `cli/app.go:36` sets `AllowAnonymous: true`. Acceptable for a single-user local tool, disqualifying for node-to-node. Every greenfield CLI verb that talks to a remote node must require an authenticated identity.
2. **CORS pinned to one origin, independent of request.** `api/main.go:158-171` emits `Access-Control-Allow-Origin: http://localhost:3000` for every request. Replace with an allowlist keyed on the registered node identity of the caller; reject or omit the header when the origin is not known.
3. **Authorization header allowed by CORS but never read by any handler.** `api/main.go:162` advertises `Authorization` as an allowed header; no handler authenticates. Greenfield handlers must authenticate *before* doing anything domain-specific, not rely on CORS posture.
4. **Client-supplied filesystem paths become server-side filesystem operations with no containment.** `scanProjects` at `api/main.go:222-234` and `performIntegration` at `api/main.go:432-465` accept/act on caller-provided paths. Greenfield handlers must not accept free-form paths; use opaque identifiers resolved server-side against a bounded, server-owned root.
5. **CORS test asserts presence, not value.** `api/main_test.go:963-965, 994-996` tests `origin != ""` instead of matching the request's origin against the expected allowlist. A greenfield CORS / auth test must assert the *value* a middleware emits for a specific request, not just that any value is present.
6. **Plain HTTP for scenario-to-scenario transport.** No TLS anywhere in the tree, all endpoints on `http://localhost:...`. Greenfield cross-node transport must run TLS (and, for trust-on-first-use, key pinning).
7. **Append-by-substring merge into external files.** `api/main.go:455` appends to `CLAUDE.md` only if `!strings.Contains(content, "Vrooli Integration")`. Any prior text with that substring blocks the append; any rename silently double-appends. Never use free-text markers for idempotency — use content-addressed sentinels (delimited fenced blocks with hashes) or a separate state record.
8. **Hard-coded versions on every mutation.** `api/main.go:388` writes literal `"1.0.0"` for `vrooli_version` and `bridge_version` every integrate call. No source of truth, no update path. Versioning must come from build metadata (`BuildFingerprint`, `BuildTimestamp` already available in `cli/app.go:16-19`).
9. **`ioutil` instead of `os`.** `api/main.go` uses `ioutil.ReadFile` / `ioutil.WriteFile` throughout — deprecated since Go 1.16.
10. **Test fixtures don't match the catalog vocabulary.** `api/main_test.go` uses `"nodejs"` while `initialization/templates/project-types.json` defines `"npm"`. The bridge tests must match the type vocabulary exactly.
11. **Test pollution of process env.** `TestInitDB` / `TestDatabaseConnection` in `api/main_test.go:1011-1030,1134-1156,1182-1199` use `os.Unsetenv` + conditional `os.Setenv` restore, irreversible if the original was unset-but-present. New tests must use `t.Setenv` and/or testcontainers.
12. **UI derives API port as `current_port + 5000` and reads `process.env.HOME` from the browser.** `ui/src/app.js`. Both are broken in the general case. Use server-injected config or a `/api/v1/config` endpoint.
13. **30s polling refresh.** `ui/src/app.js`. Use backend push or stale-while-revalidate with ETags.

## Limitations

- **Confidence levels.** High: Findings 1, 2, 3, 7, and every row in the Signal Catalog marked **avoid** or **drop** (file paths verified in round 002). Medium: Finding 4 (data model) — schema verified, but the *generalizability* claim rests on pattern-matching against the new mission's likely shape, which lives in the sibling idea that hasn't been refined yet. Medium: Finding 5 (CLI) — `cli/internal/support/` was skimmed only, not walked file-by-file, so additional micro-conventions may exist.
- **Scope boundaries.** This research did not audit `api/test_helpers.go`, `api/test_patterns.go`, or `cli/internal/support/` file-by-file; it did whole-tree greps sufficient for the deep-dive on trust/auth but not for exhaustive pattern extraction. If the foundation spec needs a complete inventory of reusable test helpers, a follow-up round would be needed.
- **Out-of-scope decisions.** The greenfield decisions on identity model (TOFU vs CA-backed), transport (HTTP/2 vs gRPC vs raw TCP), and session shape live in the sibling idea `idea/greenfield-vrooli-bridge-foundation-spec`. This conclusion intentionally does NOT propose those — it only names what to inherit, adapt, drop, or avoid from the legacy.
- **No external sources.** Signal is limited to the current state of files on disk; no commits, PRs, prior incidents, or design docs outside the scenario were consulted.

## Actions

### Action 1: Update document — Seed foundation spec archive with this signal catalog
- **File**: `scenarios/swarm-manager/idea/greenfield-vrooli-bridge-foundation-spec/archive/legacy-bridge-signal-catalog.md`
- **Change**: Copy this conclusion's **Signal Catalog** table and **Anti-patterns** section verbatim into the foundation spec's `archive/` folder, so its first workshop round starts with the full catalog of what to inherit / adapt / drop / avoid without re-reading the legacy tree. Prefix with a one-paragraph note citing `research/legacy-vrooli-bridge-signal-extraction/conclusion.md` as the source and stating that the archive copy is frozen at 2026-04-22 — the authoritative version lives in this research item.

### Action 2: Update backlog item — Pin trust requirements and OT-P0-007 into the foundation spec description
- **Kind**: idea
- **Name**: greenfield-vrooli-bridge-foundation-spec
- **Changes**:
  - description: append a trailing paragraph stating that the foundation spec must (a) inherit the legacy requirement `OT-P0-007` ("securely connect to remote servers; sign in only the first time") as a P0 acceptance line, and (b) answer four explicit questions surfaced by this research: identity model (TOFU vs CA-backed), transport (plain HTTP is out — TLS is mandatory), origin policy (the replacement for the legacy hardcoded CORS), and input-path hygiene (no free-form filesystem paths across the trust boundary).
- **Reason**: The foundation spec currently inherits from this research only by being downstream of it in the initiative's dependency graph; the four questions and the OT-P0-007 anchor need to be in the spec's description so any agent picking it up gets them without reading the conclusion first.

### Action 3: No further action required for execute/greenfield-vrooli-bridge-core-runtime or downstream execute items
The core runtime and the two downstream execute items (`vrooli-emulator-remote-node-backend`, `macos-real-device-validation-over-bridge`) already depend on `idea/greenfield-vrooli-bridge-foundation-spec`. Once Actions 1 and 2 land, the signal propagates through the existing dependency chain. No new items, no changes to execute-item descriptions, and no changes to `depends_on` are warranted by this research — all initiative members already sit in the right order.
