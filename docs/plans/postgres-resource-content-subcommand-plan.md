# Postgres Resource `content` Subcommand Restoration Plan

## 1. Purpose

Restore the missing `content` subcommand family on the Go-native `resource-postgres` CLI so that scenarios can reliably initialize, seed, and query their PostgreSQL databases through a clean, first-class resource abstraction — eliminating the inline `psql` / `docker inspect` workarounds that have crept into scenario `service.json` files.

The immediate trigger: `vrooli scenario restart scenario-to-desktop` fails because a transitive dependency, `secrets-manager`, cannot complete its `initialize-database` step. Root cause is that the `postgres` resource was migrated from bash (which exposed `content add / execute / create-database`) to Go (`cliapp.ResourceApp.StandardLifecycleCommands()` — lifecycle only), dropping the `content` subcommand group on the floor. Several scenarios have since silently broken.

## 2. Required Reading

An executing agent **must** run these before touching code:

```bash
prompt-manager skill read implementation-plan-authoring storage-steer cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Also read (in-repo):

- `resources/postgres/resource.json` — CLI adapter config, port/instance contract
- `resources/postgres/cli/main.go` — current (lifecycle-only) entrypoint
- `packages/cli-core/cliapp/resource_app.go` — `ResourceApp`, `StandardLifecycleCommands`, `DelegatingCommand` patterns
- `packages/cli-core/cliapp/scenario_app.go` — `SubcommandGroup` shape (lines ~40–100)
- Git reference for the old behavior: `git show 8f125336ce:resources/postgres/cli.sh`, `git show 8f125336ce:resources/postgres/lib/database.sh` (pre-migration bash, last commit with `content` handlers)

## 3. Greenfield Constraint

**This is greenfield work.** Do not add compatibility shims, dual-mode code paths, `cli.sh` wrappers, deprecated-flag aliases, or renamed `_unused` variables. The old bash CLI is already deleted and no longer referenced except through broken scenario `service.json` entries — those are fixed in Phase 3, not preserved.

## 4. Problem Statement

Observed failure (`vrooli scenario restart scenario-to-desktop`, 2026‑04‑17):

```
[INFO] [1/5] build-api
[INFO] [2/5] initialize-database
bash: line 1: ../../resources/postgres/cli.sh: No such file or directory
scenario "secrets-manager" phase "setup" step "initialize-database" failed with exit code 127
```

Underlying gaps:

1. **Go CLI is incomplete.** `resources/postgres/cli/main.go` calls `app.SetCommands(app.StandardLifecycleCommands())` — no `content` subcommand group is registered. `resource-postgres content add|execute|create-database ...` all fail with `Unknown command: content`.
2. **`secrets-manager` is locally patched with a hack.** Working tree (`git diff`) replaces the missing call with an inline 600-char bash one-liner that `docker inspect`s the postgres container for credentials then pipes a schema file into `psql`. It still fails (exit 127) because `psql` isn't guaranteed on the host, and the shape is hostile to review, portability, and the repo contract.
3. **Other scenarios silently depend on the missing subcommand.** Grep across `scenarios/**/service.json` shows:
   - `scenarios/bedtime-story-generator/.vrooli/service.json:161,166` → `resource-postgres content execute --file ...`
   - `scenarios/calendar/.vrooli/service.json:146,151` → `resource-postgres content add ...` + `resource-postgres content get main`
   - `scenarios/home-automation/.vrooli/service.json:182` → `resource-postgres content create-database ...`
   - `scenarios/scenario-authenticator/.vrooli/service.json:140` → `resource-postgres content create-database ...`
   - `scenarios/ai-chatbot-manager/scripts/init-database.sh:29,41,55` → `resource-postgres content execute ...`
   - `scenarios/secrets-manager/.vrooli/service.json` (committed) → `../../resources/postgres/cli.sh content add ...`

Net effect: every scenario in that list is broken on first `setup` unless it already has a seeded database. Swappable postgres storage is not currently a working abstraction.

## 5. Scope

**In scope**

- Implement `content` subcommand group on `resource-postgres` (Go) covering: `execute`, `create-database`, `add`, `get`, `list`, `remove` — matching the surface implied by existing callers (Section 4 grep).
- Route SQL execution through `docker exec` into the instance container (container name resolved from `resource.json` runtime or via `--instance`), using POSTGRES_USER / POSTGRES_PASSWORD sourced from the resource env contract — **no host-side `psql` dependency**.
- Fix all known broken callers to use the new CLI surface cleanly (no `docker inspect`, no inline `psql`, no Python one-liners).
- Add unit + integration tests for the new subcommand.
- Restart `scenario-to-desktop` and every affected scenario; verify health.

**Out of scope**

- Redesigning how Vrooli resources inject env vars into scenarios.
- Adding new features beyond parity with the pre-migration bash `content` surface (e.g., no new `content migrate`, `content backup`, etc.).
- Porting the remaining `manage::`, `test::`, `backup::` bash groups referenced in the old `cli.sh` — only `content::` is in scope for this plan.
- Changing the `postgres` container image, ports, or volume layout.

## 6. Current Technical Context

- **CLI entrypoint**: `resources/postgres/cli/main.go` (51 lines, `cliapp.NewResourceApp` + `StandardLifecycleCommands`).
- **Package**: Single `main` package. Internal helpers live in `resources/postgres/cli/internal/{env,health,install,runtime,status}/` — pattern for adding a new subcommand is `internal/content/` with handler + tests.
- **cli-core surface used**:
  - `cliapp.SubcommandGroup{ Name, Description, Subcommands }` — for `content <subcmd>`.
  - `cliapp.Command{ Name, Description, Run func([]string) error }` — each subcommand.
  - `cliapp.ResourceApp.SetCommandsWithSubgroups(commands, subgroups)` — replaces `SetCommands`.
- **Container contract** (from `resources/postgres/resource.json`):
  - Image: `postgres:16-alpine`
  - Container name: `vrooli-postgres-main` (single default instance; `--instance` flag selects non-default)
  - Env: `POSTGRES_DB=vrooli`, `POSTGRES_USER=vrooli`, `POSTGRES_PASSWORD=vrooli` by default; scenarios may override via the resource env injection contract.
  - Host port: `5433` → container `5432`.
- **Broken-caller inventory** (exact file:line to fix in Phase 3): see Section 4.
- **Uncommitted local drift**: `scenarios/secrets-manager/.vrooli/service.json` has a working-tree-only inline `psql`/`docker inspect` hack (`git diff HEAD scenarios/secrets-manager/.vrooli/service.json`). Phase 3 replaces it — do **not** preserve any piece of that diff.

## 7. Target End State

- `resource-postgres content <subcmd>` is a registered, help-documented subcommand group.
- No scenario `service.json` or scenario script invokes `psql`, `docker exec`, or `docker inspect` directly for postgres initialization; they all call `resource-postgres content ...`.
- `vrooli scenario restart scenario-to-desktop` reaches `healthy`; the same holds for the five other scenarios that depend on `content` (see Section 9 Rollout).
- `go test ./...` passes in `resources/postgres/cli/` with new coverage on the `content` handlers (unit + at least one integration test against a disposable container).
- `golangci-lint run ./...` is clean under `resources/postgres/cli/`.

## 8. Implementation Strategy (phased)

### Phase 1 — Design the `content` subcommand surface (no code)

Produce a short `resources/postgres/cli/internal/content/CONTRACT.md` (or inline Go doc) fixing the CLI shape. Use the pre-migration bash as the reference but drop any flags not actually used by current callers. Proposed shape:

```
resource-postgres content execute [--instance <name>] [--database <db>] [--file <path> | --sql <sql>]
resource-postgres content create-database [--instance <name>] [--owner <user>] <db-name>
resource-postgres content add [--instance <name>] [--database <db>] [--init] <file-or-dir>
resource-postgres content get [--instance <name>] <db-name>          # prints connection string + metadata
resource-postgres content list [--instance <name>]                   # list databases
resource-postgres content remove [--instance <name>] <db-name>
```

Defaults:
- `--instance` defaults to `main`.
- `--database` defaults to the resource's `POSTGRES_DB`.
- `add` without `--init` is equivalent to `execute --file` (back-compat with `calendar`/`secrets-manager` usage); with `--init` it first `create-database` then `execute --file`.

Validate the shape by mapping every existing caller in Section 4 to exactly one new command line — no caller should need a flag this shape doesn't provide.

### Phase 2 — Implement `content` handlers in Go

Files to add:

- `resources/postgres/cli/internal/content/content.go`
  - Exports `Commands() cliapp.SubcommandGroup`
  - Handlers: `execute`, `createDatabase`, `add`, `get`, `list`, `remove`
  - Shared helper `runInContainer(ctx, instance, args []string, stdin io.Reader) error` that wraps `docker exec -i vrooli-postgres-<instance> psql -U $POSTGRES_USER -d $database`
  - Container/env resolution reads from resource env (same mechanism the `status` / `health` internals already use) — no `docker inspect` fallbacks.
- `resources/postgres/cli/internal/content/content_test.go`
  - Unit tests for flag parsing, command construction, default-instance behavior, and file-not-found error paths (mock the exec runner via an injected interface, mirroring the pattern already used by `internal/runtime/`).

Files to modify:

- `resources/postgres/cli/main.go`: swap `app.SetCommands(app.StandardLifecycleCommands())` for `app.SetCommandsWithSubgroups(app.StandardLifecycleCommands(), []cliapp.SubcommandGroup{content.Commands()})`.

Rebuild + install:

```bash
cd resources/postgres/cli && go build ./... && go test ./...
bash resources/postgres/cli/install.sh
resource-postgres --help   # confirm "content" group is listed
```

### Phase 3 — Fix every broken caller

Replace each call site with the canonical Phase‑1 shape. No inline `psql`, no `docker inspect`, no Python JSON massaging.

- `scenarios/secrets-manager/.vrooli/service.json`
  - `initialize-database` → `resource-postgres content execute --database vrooli_secrets_manager --file initialization/storage/postgres/schema.sql`
  - `seed-database` → `resource-postgres content execute --database vrooli_secrets_manager --file initialization/storage/postgres/seed.sql`
  - Add a preceding `create-database` step if `vrooli_secrets_manager` doesn't already exist from a prior resource.json-level bootstrap.
  - Discard the working-tree `psql` hack — it must not appear in the final diff.
- `scenarios/bedtime-story-generator/.vrooli/service.json:161,166` — already use correct shape; verify they succeed once Phase 2 lands. No `service.json` change expected unless Phase 1 renames flags.
- `scenarios/calendar/.vrooli/service.json:146,151` — replace the Python-scripted `content get | python3 -c ...` block with a single `resource-postgres content get --instance main --database calendar_system --as-env` invocation (extend Phase‑1 shape with `--as-env` if the cleanup-friendly alternative isn't already covered; decide during Phase 1).
- `scenarios/home-automation/.vrooli/service.json:182` — verify works; adjust flag style for consistency only if Phase 1 normalizes.
- `scenarios/scenario-authenticator/.vrooli/service.json:140` — same.
- `scenarios/ai-chatbot-manager/scripts/init-database.sh:29,41,55` — same.

For each file touched, re-run its scenario's `service.json` validator (`vrooli scenario validate <name>` if available, otherwise `jq . <file>` + `vrooli scenario setup <name> --dry-run`).

### Phase 4 — Verification (runs full chain)

1. Lint + tests (even pre-existing issues — fix them):
   ```bash
   cd resources/postgres/cli && golangci-lint run ./... && go test ./... -count=1
   cd packages/cli-core && go test ./... -count=1      # resource_app_test.go touches SubcommandGroups
   ```
2. Restart the originally-failing scenario and every scenario listed in Section 9:
   ```bash
   vrooli scenario restart scenario-to-desktop
   vrooli scenario restart secrets-manager
   vrooli scenario restart bedtime-story-generator
   vrooli scenario restart calendar
   vrooli scenario restart home-automation
   vrooli scenario restart scenario-authenticator
   vrooli scenario restart ai-chatbot-manager
   ```
3. For each: confirm `vrooli scenario status <name>` reports `running` and the API `/health` endpoint returns 200.

## 9. Contract Decisions

- **Container-mediated execution.** All SQL flows through `docker exec -i <container> psql ...`. No host `psql` dependency. Rationale: matches pre-migration behavior, avoids host toolchain drift, works in CI and on user machines identically.
- **Instance model.** `--instance` defaults to `main`. Container name is `vrooli-postgres-<instance>`. Multi-instance remains supported at the CLI level but the scenarios in scope all target `main`.
- **Env/credential source.** Read `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` from the process environment as injected by the vrooli lifecycle (same mechanism `internal/health/` uses today). Never fall through to `docker inspect`.
- **Exit codes.** Non‑zero on any failure (missing instance, missing file, SQL error). Do **not** emulate the old `|| true` resilience — scenarios handle idempotency with `create-database` + separate `execute` (create-database returns 0 if the DB already exists, non-zero only on real failure).
- **Output format.** Plain text by default; `--json` on `get`/`list` for machine consumption (drives Phase 3 calendar cleanup).

## 10. Testing Plan

- **Unit (Go):** `internal/content/content_test.go` covers
  - Default instance resolution, `--instance` override.
  - `execute --file` reads file, writes to stdin of exec runner.
  - `execute --sql` passes `-c <sql>`.
  - `create-database` idempotency (returns 0 on "already exists").
  - `add --init` order: create-database → execute file.
  - Missing file → exit 1 with actionable message.
  - Mock runner interface: `type Runner interface { Run(ctx, name string, args []string, stdin io.Reader) ([]byte, error) }` injected via an unexported constructor.
- **Integration:** One test gated behind `RESOURCE_POSTGRES_INTEGRATION=1` that spins up `vrooli-postgres-integration` via `resource-postgres install && start`, runs `content execute --sql 'select 1'`, verifies output, then tears down. Runs in CI when the secret toggle is set; local devs run it on demand.
- **Scenario-level:** After Phase 3, running `vrooli scenario setup secrets-manager` on a clean machine must succeed end-to-end. Add a make target or README line if useful, but no new test harness is required — the scenario's own setup phase is the gate.

## 11. Rollout / Validation Checklist

- [ ] `resource-postgres --help` lists `content` as a subcommand group.
- [ ] `resource-postgres content --help` lists `execute`, `create-database`, `add`, `get`, `list`, `remove`.
- [ ] `go test ./...` passes in `resources/postgres/cli/` with new coverage.
- [ ] `golangci-lint run ./...` clean in `resources/postgres/cli/`.
- [ ] `gofumpt -l resources/postgres/cli/` outputs nothing.
- [ ] `git diff scenarios/secrets-manager/.vrooli/service.json` contains no `docker inspect`, no inline `psql`, no backslash-escaped SQL strings.
- [ ] `vrooli scenario restart scenario-to-desktop` succeeds; `/health` returns 200 on its API port.
- [ ] Same restart + health check for the five other scenarios listed in Section 9.
- [ ] No new references to `../../resources/postgres/cli.sh` anywhere in the repo (`rg "resources/postgres/cli\.sh"` is empty).

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Scenarios expect subtly different flag names (e.g., `--db` vs `--database`) | Medium | Breaks caller without obvious error | Phase 1 audits every caller; Phase 3 updates callers to the canonical shape rather than overloading the CLI. |
| Integration test requires running Docker in CI | Medium | CI flake | Gate the integration test behind `RESOURCE_POSTGRES_INTEGRATION=1`; unit tests cover logic via the mock Runner. |
| Credentials leak into logs via `psql -c "<sql>"` error messages | Low | Security | Wrap exec output; scrub `postgres://user:pass@...` shapes before re-emitting. Same helper the `internal/status/` package already uses. |
| `--instance` name not a valid container (first-run before `install`) | Medium | Setup fails with a confusing "No such container" | Pre-flight check in each handler: confirm container exists and is running; otherwise return a `recovery` hint pointing at `resource-postgres start`. |
| Concurrent scenario setups race on `create-database` | Low | Transient failure | Handler treats "database already exists" (SQLSTATE 42P04) as exit 0. |

## 13. Non-goals / Prohibited Patterns

- Do **not** restore `resources/postgres/cli.sh`. Anyone invoking it should get a clean "command not found" today; fix the caller, not the absence.
- Do **not** parse `docker inspect` output for credentials.
- Do **not** depend on host-installed `psql`; all SQL must go through the container.
- Do **not** add `resource-postgres content *` aliases for typos / historical spellings — scenarios update to the canonical names.
- Do **not** introduce an ORM, migration framework, or long-lived connection pool inside the resource CLI — `content` is a thin SQL passthrough.
- Do **not** change the `postgres` image version or port defaults as part of this plan.

## 14. Definition of Done

Every item in Section 11 is checked, plus:

- A follow-up memory entry captures the canonical `content` flag shape so future scenarios default to it without rediscovering (`feedback_postgres_content_shape.md`).
- The working-tree hack on `scenarios/secrets-manager/.vrooli/service.json` is gone (`git diff` on that file is clean against the new committed version).
- Running `vrooli scenario restart scenario-to-desktop` on a machine without the patched working tree succeeds.
