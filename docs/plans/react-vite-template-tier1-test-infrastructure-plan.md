# React-Vite Template — Tier 1 Test Infrastructure Hardening

## 1. Purpose

The `templates/scenarios/react-vite` template now ships with seam-disciplined
test infrastructure (Clock, Pinger, httpx live harness, renderWithProviders,
factories, mocks, three-way `no_prod_import_test.go` quarantines, ESLint
test-utils ban, codegen'd typed strings registry, axe-core a11y regression,
locale parity tests, vitest coverage thresholds). A fresh audit
(2026-05-02) found the foundation strong but flagged five **Tier 1** gaps
that would bite the next two or three new scenarios at adoption time. This
plan implements and validates fixes for all five in a single pass so the
template can be confidently forked again without follow-up template polish.

The plan is scoped to the template **and** its smoke-test scenario
(`scenarios/smoke-tier1/`), since the template itself is not runnable —
it's only validated by generating a fresh scenario and running the full
test gate against the generated copy.

## 2. Required Reading

Future agents executing this plan must first load:

```bash
prompt-manager skill read implementation-plan-authoring seam-discovery-and-enforcement test unit-testing-architecture-steer screaming-architecture-audit cross-platform-readiness storage-steer cli-steer api-steer
```

These cover: plan-authoring conventions, the seam discipline the template
already encodes (so changes preserve it), the testing-architecture mental
model, the screaming-architecture audit pattern (used to keep the worked
example domain-named, not "example"), cross-platform readiness (CGO / pure-Go
SQLite contract), the canonical storage seam shape, and the CLI/API contracts
the generated scenarios inherit.

## 3. Greenfield Constraint

**This is greenfield template work. No backwards-compatibility shims, no
legacy wrappers, no "deprecated" aliases, no `// removed` comments.** The
template is the canonical starting point for *new* scenarios; existing
scenarios that diverged from earlier template revisions do not get
compat support from this work. If a Tier 1 fix changes a public contract
(e.g., `server.Deps`), the change is unconditional. Existing forked
scenarios update on their own cadence.

## 4. Problem Statement

Five concrete gaps in the template, in priority order:

1. **No worked repository/integration test.** `api/internal/testutil/db/sqlite.go`
   ships `NewSQLite(t)` with prose explaining how to wrap it with a
   domain-specific schema, but no Go file actually exercises that path.
   The first scenario to add a database table will reinvent the
   table-test-store-repository wiring from scratch.
2. **No Go-side coverage gate.** `.github/workflows/test.yml` runs
   `go test -race ./...` on api/ and cli/ with no coverage floor. UI has
   85% thresholds across lines/branches/functions/statements enforced
   via vitest. The asymmetry is undocumented and means API/CLI coverage
   can rot silently.
3. **`server.Deps` nil-handling is asymmetric.** `internal/server/server.go::New`
   defaults `Deps.Logger` to `log.Default()` if nil, but `Pinger` and `Clock`
   nil-deref silently. Either both or neither — the inconsistency is a
   trap for the first scenario that forgets a dep.
4. **`interp()` lives in `ui/src/test-setup.ts` but is imported by tests
   as a public helper.** Setup files run side-effects (jest-dom register,
   cimode default, canvas mock); they should not also be a public API
   surface. Today `App.test.tsx` does `import { interp } from "./test-setup"`,
   coupling tests to a module whose primary contract is "side-effects on
   import."
5. **`pnpm-lock.yaml` is not committed in the template, but
   `.github/workflows/test.yml` runs `pnpm install --frozen-lockfile`.** A
   freshly-generated scenario will fail CI on the first push until
   someone manually generates and commits the lockfile. The README
   instructs adopters to run `corepack pnpm install --ignore-workspace`,
   but generators routinely forget; the postHook in `template.json`
   already runs the install but the resulting lockfile is not surfaced
   for commit.

## 5. Scope

**In scope:**
- Edits inside `templates/scenarios/react-vite/`
- Edits inside `scenarios/smoke-tier1/` (the verification scenario) only
  to keep it in sync with the regenerated template
- Updates to `docs/internal/SEAMS.md` and `docs/internal/TESTING.md`
  inside the template to reflect the new worked example and the updated
  Deps contract
- A new `.gitkeep`-style mechanism (or equivalent) so the lockfile
  travels with generated scenarios

**Out of scope:**
- Tier 2 cleanups (discardWriter/io.Discard alignment, capturing
  cli `--version` stdout, fixture wire-shape round-trip, mocks/
  package-comment polish, App.tsx first-paint coverage, CLI domain
  worked example, MSW guidance)
- Tier 3 forward-looking items (BAS workflow shipping, bundle-size
  gate, formal test taxonomy)
- Existing scenarios that forked earlier template revisions
- Coverage gate tuning beyond establishing a floor

## 6. Current Technical Context

### Template files that change
- `templates/scenarios/react-vite/api/internal/store/store.go`
- `templates/scenarios/react-vite/api/internal/store/schema.sql`
- `templates/scenarios/react-vite/api/internal/store/schema.go`
- `templates/scenarios/react-vite/api/internal/store/store_test.go` (new)
- `templates/scenarios/react-vite/api/internal/server/server.go`
- `templates/scenarios/react-vite/api/internal/server/server_test.go` (new)
- `templates/scenarios/react-vite/.github/workflows/test.yml`
- `templates/scenarios/react-vite/api/.golangci.yml` (touch only if a new lint surfaces)
- `templates/scenarios/react-vite/ui/src/test-setup.ts`
- `templates/scenarios/react-vite/ui/src/test-utils/index.ts`
- `templates/scenarios/react-vite/ui/src/test-utils/interp.ts` (new)
- `templates/scenarios/react-vite/ui/src/test-utils/interp.test.ts` (new)
- `templates/scenarios/react-vite/ui/src/App.test.tsx` (import path update)
- `templates/scenarios/react-vite/template.json`
- `templates/scenarios/react-vite/README.md`
- `templates/scenarios/react-vite/docs/internal/SEAMS.md`
- `templates/scenarios/react-vite/docs/internal/TESTING.md`

### Verification scenario files that re-sync
- `scenarios/smoke-tier1/` is regenerated (or selectively patched) from
  the updated template, then full gate is run end-to-end.

### Existing seams the work must respect
- `internal/clock.Clock` interface + `mocks.FakeClock` (do not change)
- `internal/store.Pinger` interface (extend, not replace)
- `internal/testutil/db.NewSQLite` (consume from new repo test)
- `internal/testutil/no_prod_import_test.go` AST guardrail (the new
  store_test.go must remain in the production tree, not testutil)

### Generator pipeline
- `vrooli scenario generate react-vite ...` runs `template.json::postHooks`
  after substitution. The `corepack pnpm install --ignore-workspace`
  postHook produces a `pnpm-lock.yaml` in the generated scenario but
  does not stage it for commit.
- The test scenario `scenarios/smoke-tier1/` reflects the latest generation;
  use it as the regression target.

## 7. Target End State

After this plan lands, a freshly generated scenario from the template
must:

1. Contain a working `tasks` table demonstrating the full storage seam:
   - `internal/store/store.go` declares a `TaskStore` interface
   - `internal/store/sqlite.go` implements it against `*sql.DB`
   - `internal/store/store_test.go` uses `testutil/db.NewSQLite` to
     exercise create/get/list against a real (per-test temp) SQLite file
   - `EnsureSchema` actually applies a non-empty schema and the test
     proves the table exists
2. Run with **enforced coverage floors on Go side** in CI:
   - `go test -race -coverprofile=coverage.out ./...` plus a coverage
     gate (initial floor: **70% lines** for both api/ and cli/) that
     fails the workflow when coverage drops
3. Have **uniform Deps construction**:
   - `server.New` either accepts no nil values for any required dep
     (preferred greenfield posture) or defaults all of them; the
     decision is implemented and documented in SEAMS.md
4. Expose `interp` via `test-utils/`:
   - `import { interp } from "./test-utils"` (or `@/test-utils`) works
     in tests
   - `test-setup.ts` only contains side-effects + the cimode + canvas mocks,
     no exports
   - The ESLint test-utils-quarantine continues to apply (interp is
     test-only)
5. Have a `pnpm-lock.yaml` that travels with every generated scenario:
   - `template.json` either commits the lockfile inside the template
     (preferred, since the template's deps are stable) or its postHook
     stages the generated lockfile and the README instructs the
     generator to commit it
   - `.github/workflows/test.yml::pnpm install --frozen-lockfile` succeeds
     on the first push of a freshly-generated scenario without manual
     intervention

## 8. Implementation Strategy (Phased)

### Phase 1 — Worked repository/integration test (Tier 1 #1)

**Goal:** First scenario adding a table copies one file, not invents
a pattern.

1. Add a minimal `tasks` domain to the template's storage seam:
   - **`api/internal/store/schema.sql`** — replace the placeholder with:
     ```sql
     CREATE TABLE IF NOT EXISTS tasks (
         id         TEXT PRIMARY KEY,
         title      TEXT NOT NULL,
         created_at TEXT NOT NULL
     );
     ```
   - Keep `EnsureSchema` and `stripComments` unchanged — the existing
     placeholder-noop test (`TestEnsureSchema_EmptyPlaceholderIsNoop`)
     becomes obsolete; replace it with `TestEnsureSchema_AppliesTasksTable`
     that uses `db.NewSQLite(t)` and queries `sqlite_master`.

2. Extend `api/internal/store/store.go`:
   ```go
   type Task struct {
       ID        string
       Title     string
       CreatedAt time.Time
   }

   type TaskStore interface {
       Create(ctx context.Context, t Task) error
       Get(ctx context.Context, id string) (Task, error)
       List(ctx context.Context) ([]Task, error)
   }
   ```
   Keep the `Pinger` interface untouched; both interfaces co-exist in
   `store.go`.

3. Add `api/internal/store/sqlite.go`:
   - Unexported `sqliteTaskStore` struct holding `*sql.DB`
   - Constructor `NewSQLiteTaskStore(db *sql.DB) TaskStore`
   - `Create` / `Get` / `List` methods using prepared statements
   - `var _ TaskStore = (*sqliteTaskStore)(nil)` compile-time guard

4. Add `api/internal/store/sqlite_test.go` (production-tree, not testutil):
   - Build a fresh handle with `testutil/db.NewSQLite(t)`
   - Apply schema via `store.EnsureSchema(ctx, db)`
   - Construct `NewSQLiteTaskStore(db)` and exercise:
     - happy path round-trip (Create → Get → equal)
     - `Get` on missing id returns wrapped `sql.ErrNoRows`
     - `List` returns inserted tasks in deterministic order
     - `Create` of duplicate id surfaces a constraint error
   - Use `clock.Clock` injected through the store **only if** Create
     auto-stamps `CreatedAt`. Greenfield call: caller supplies the
     timestamp; the store does no time logic. (Keeps the demo focused
     on the storage seam, not the clock seam.)

5. Update `api/internal/store/store.go` package comment to point at
   `sqlite.go` as the canonical reference for "how to add a table."

6. Update `docs/internal/SEAMS.md`:
   - Add a third row to the seams table for `TaskStore`
   - Update the "Adding a new seam" worked example to point at the now-real
     files instead of inline pseudo-code

7. Update `docs/internal/TESTING.md`:
   - Add a third canonical example file under "TL;DR — the canonical
     examples": `api/internal/store/sqlite_test.go` for repository tests
   - Add a short "Database tests" subsection under "API testing"
     pointing at `db.NewSQLite` + `EnsureSchema` + the new sqlite_test.go

**Deliverable:** A fresh `vrooli scenario generate react-vite ...` produces
a scenario whose `tasks` table is queryable, exercised by tests, and
reads as an obvious template for the next domain table.

### Phase 2 — Go coverage gate (Tier 1 #2)

**Goal:** Symmetry with the UI's 85% gate; prevent silent rot.

1. **`api/.golangci.yml`** — no change (lint, not coverage).
2. **`.github/workflows/test.yml`** — for both `api` and `cli` jobs,
   replace the single `go test -race ./...` step with:
   ```yaml
   - name: Tests (race detector + coverage)
     run: |
       go test -race -coverprofile=coverage.out -covermode=atomic ./...

   - name: Coverage gate
     run: |
       total=$(go tool cover -func=coverage.out | awk '/^total:/ {print $3}' | tr -d %)
       floor=70.0
       awk -v t="$total" -v f="$floor" 'BEGIN { if (t+0 < f+0) { exit 1 } }' \
         || { echo "Coverage $total% is below floor $floor%"; exit 1; }
       echo "Coverage $total% (floor $floor%)"
   ```
3. Floor justification: the template's current api/ and cli/ packages
   already clear 70% with the tests shipped; tightening past the
   honest baseline would make every new scenario start red. Document
   in `docs/internal/TESTING.md` (next to the existing "Coverage
   thresholds" UI section): the Go-side floor exists, lives in
   `.github/workflows/test.yml`, and graduates upward once a real
   release clears the next quartile.
4. Verify the floor before committing — if either the api or cli job
   actually falls under 70% with the Phase 1 additions counted, **fix
   the gap with real tests**, do not lower the floor.
5. Optional: add a `make test` (Go) target in `api/Makefile` /
   `cli/Makefile` if they exist, mirroring the CI invocation. Skip if
   the scenario uses the top-level `make test` only.

**Deliverable:** API + CLI coverage drops below 70% in any future
scenario fail CI exactly the way UI does.

### Phase 3 — `server.Deps` nil-handling consistency (Tier 1 #3)

**Goal:** No silent nil-deref traps, no asymmetric defaulting.

**Decision (per greenfield posture):** *Remove* the `Logger` default in
`server.New`; require all of `Pinger`, `Clock`, `Logger`, `Service`,
`Version` to be set. If any zero value is detected, panic with a clear
message.

1. Edit `api/internal/server/server.go::New`:
   ```go
   func New(d Deps) *Server {
       switch {
       case d.Pinger == nil:
           panic("server.New: Deps.Pinger is required")
       case d.Clock == nil:
           panic("server.New: Deps.Clock is required")
       case d.Logger == nil:
           panic("server.New: Deps.Logger is required")
       case strings.TrimSpace(d.Service) == "":
           panic("server.New: Deps.Service is required")
       case strings.TrimSpace(d.Version) == "":
           panic("server.New: Deps.Version is required")
       }
       s := &Server{deps: d, router: mux.NewRouter()}
       s.registerRoutes()
       return s
   }
   ```
2. Update `api/main.go` to pass an explicit logger:
   ```go
   srv := server.New(server.Deps{
       Pinger:  db,
       Clock:   clock.System{},
       Logger:  log.New(os.Stderr, "", log.LstdFlags),
       Service: "{{SCENARIO_ID}}-api",
       Version: "1.0.0",
   })
   ```
3. Update `api/handlers/health/handler_test.go` — already passes
   `log.New(discardWriter{}, "", 0)`, so no change.
4. Update `api/internal/testutil/httpx/server_test.go` — already passes
   `log.New(io.Discard, "", 0)`, so no change.
5. Add `api/internal/server/server_test.go` (new):
   - `TestNew_PanicsWhenPingerNil`
   - `TestNew_PanicsWhenClockNil`
   - `TestNew_PanicsWhenLoggerNil`
   - `TestNew_PanicsWhenServiceEmpty` / `TestNew_PanicsWhenVersionEmpty`
   - `TestNew_HappyPathReturnsServer` (uses fakes from `mocks/`)
   Each panic test uses `defer func() { recover() }()` to assert the
   panic surfaces with the expected substring.
6. Update `docs/internal/SEAMS.md` "How to add a new seam" guidance
   to mention: "if the new dep is required, add a nil check in
   `server.New`'s switch — do not introduce silent defaults."
7. Update `docs/internal/TESTING.md` "Common patterns and anti-patterns"
   table:
   - DO: `server.New` panics on missing required deps
   - DON'T: silent `if d.Logger == nil { d.Logger = log.Default() }` style

**Deliverable:** `server.New` is uniform. Forgetting a dep produces a
loud, immediate error at boot — caught in tests, never in prod.

### Phase 4 — Move `interp()` out of `test-setup.ts` (Tier 1 #4)

**Goal:** `test-setup.ts` is side-effects only; the `interp` helper
lives where consumers expect — `test-utils/`.

1. Create `ui/src/test-utils/interp.ts`:
   - Verbatim move of the function body from `test-setup.ts`
   - Keep the existing JSDoc comment block; update the cross-reference
     to point at `test-utils/index.ts`
2. Add `ui/src/test-utils/interp.test.ts`:
   - Test the happy path (`interp("hello {{name}}", { name: "world" })`)
   - Test number values (`{{count}}` interpolated as `String(count)`)
   - Test the error path (missing variable throws with the expected
     message)
   - Test that an empty template returns the empty string
   - Test that the function is pure (no shared state across calls)
3. Re-export from `ui/src/test-utils/index.ts`:
   ```ts
   export { interp } from "./interp";
   ```
4. Edit `ui/src/test-setup.ts`:
   - Remove the `export const interp = ...` block
   - Remove the JSDoc that documents `interp`
   - Keep the jest-dom import, the cimode `beforeEach`, and the canvas
     `getContext` mock — those are the only side-effects belonging
     here
5. Edit `ui/src/App.test.tsx`:
   - Change `import { interp } from "./test-setup"` to
     `import { interp } from "./test-utils"`
6. `grep -rn "from \"\\./test-setup\"" ui/src` — confirm no other
   importers of the old path exist; if any do, update them in this
   phase.
7. The ESLint `no-restricted-imports` rule already covers
   `**/test-utils/*` for production code; verify `interp.ts` does not
   leak into the production bundle by checking the exclusion list in
   `vite.config.ts::test.coverage.exclude` already covers
   `src/test-utils/**`.
8. Update `docs/internal/TESTING.md` — anywhere it references "interp
   in test-setup.ts" (search for `interp`) update to "interp in
   test-utils/."

**Deliverable:** `test-setup.ts` only configures the test runtime.
`interp` is a normal test-utils helper with its own self-test, just
like `makeHealthResponse`, `renderWithProviders`, and the spatial
mock builders.

### Phase 5 — Lockfile travels with generated scenarios (Tier 1 #5)

**Goal:** First push of a freshly-generated scenario passes CI without
manual lockfile generation.

**Decision:** *Commit the template's `pnpm-lock.yaml`* in
`templates/scenarios/react-vite/ui/`. Rationale: the template's
dependency set is small, stable, and curated. Shipping a lockfile is
the lowest-friction option (no postHook coordination, no README warning
for generator humans) and locks all generated scenarios to a known-good
dep tree until they intentionally diverge.

1. From the template's `ui/` directory, regenerate the lockfile against
   the current `package.json`:
   ```bash
   cd templates/scenarios/react-vite/ui
   corepack pnpm install --ignore-workspace
   ```
   This produces `pnpm-lock.yaml`.
2. **Verify the lockfile substitutes cleanly**: lockfiles store package
   names. The template's `package.json::name` is `{{SCENARIO_ID}}-ui`,
   which means the lockfile's `importers.<root>.name` field will
   contain the literal string `{{SCENARIO_ID}}-ui`. Confirm this is
   either:
   - acceptable as-is (pnpm replays installs from `dependencies` sections,
     not from the package name), **or**
   - handled by the existing template substitution pass that walks the
     template tree replacing `{{SCENARIO_ID}}` etc.
   If neither holds, add `pnpm-lock.yaml` to the substitution-pass file
   list in the generator (`scripts/scenarios/generate.go` or wherever
   `vrooli scenario generate` resolves placeholders).
3. Commit `templates/scenarios/react-vite/ui/pnpm-lock.yaml`. **Do not**
   add it to `.gitignore`.
4. Update `template.json::postHooks`:
   - Keep the `corepack pnpm install --ignore-workspace` postHook for
     local dev (it now reconciles the lockfile against any generated
     name changes; usually a no-op).
5. Update `templates/scenarios/react-vite/README.md`:
   - Replace the "Setup Workflow" `corepack pnpm install --ignore-workspace`
     line with a note that dependencies are pre-locked and a fresh
     install is only needed when adding a new dep.
6. Update `.github/workflows/test.yml`:
   - Confirm `pnpm install --frozen-lockfile --ignore-workspace` is
     still the install command. **Do not** change to
     `--no-frozen-lockfile`; if the lockfile is stale, the right fix is
     to update the template, not loosen CI.

**Deliverable:** `vrooli scenario generate react-vite --id foo ...`
produces a scenario whose first `git push` triggers a CI run that
reaches "tests" without exploding on `pnpm install`.

### Phase 6 — Final cleanup & full-gate verification

1. Regenerate or hard-sync `scenarios/smoke-tier1/` from the updated
   template:
   ```bash
   rm -rf scenarios/smoke-tier1
   vrooli scenario generate react-vite \
     --id smoke-tier1 \
     --display-name "Smoke" \
     --description "Tier 1 smoke"
   ```
   (If `smoke-tier1` carries deliberate divergence from the template,
   selectively `cp` the changed files instead. The investigation found
   none.)
2. Inside `scenarios/smoke-tier1/`, run the full local gate exactly as
   CI would:
   - `(cd api && go vet ./... && CGO_ENABLED=0 go build ./... && go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1)`
   - `(cd cli && go vet ./... && CGO_ENABLED=0 go build ./... && go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1)`
   - `(cd ui && pnpm install --frozen-lockfile --ignore-workspace && pnpm strings:check && pnpm type-check && pnpm lint && pnpm test:coverage && pnpm build)`
3. **Fix every failure surfaced by the gate, including any pre-existing
   issues.** Per project policy, "pre-existing" is not an excuse — if
   the test surface trips it now, the agent fixes it now.
4. Optional but recommended: also run `vrooli scenario test smoke-tier1`
   to exercise the test-genie phased gate (lint, structure, unit,
   business, performance) which is what `make test` invokes.
5. `vrooli scenario restart smoke-tier1` and confirm both the API and
   UI health endpoints respond:
   - `curl -fsS http://localhost:$(vrooli scenario port smoke-tier1 API_PORT)/health`
   - `curl -fsSI http://localhost:$(vrooli scenario port smoke-tier1 UI_PORT)/`
6. Update `docs/PROGRESS.md` inside the template (and the smoke
   scenario) with a one-line entry for this work.

## 9. Contract Decisions

| Surface | Decision | Why |
|---|---|---|
| `store.TaskStore` interface | Methods: `Create`, `Get`, `List`. No `Delete` / `Update` in v1. | Smallest set that demonstrates a non-trivial repository. Mutation API can be added by the first scenario that needs it. |
| `Task.CreatedAt` source | Caller-supplied. Store does not call `time.Now()`. | Keeps the worked example focused on the storage seam; doesn't accidentally introduce a clock dependency in a repo example. |
| Coverage floor | 70% lines, applied to both api/ and cli/. | Honest baseline: clears with the tests shipped today. |
| `server.New` nil policy | Panic on any zero-value required dep. | Greenfield. No silent defaults. Matches "explicit deps" doctrine in unit-testing-architecture-steer. |
| `interp` location | `ui/src/test-utils/interp.ts`, exported from `test-utils/index.ts`. | `test-setup.ts` is side-effects only; consumers expect `test-utils/` for helpers. |
| Lockfile shipping | Committed in template at `ui/pnpm-lock.yaml`. | Lowest-friction; pnpm `--frozen-lockfile` works on first push. |
| Lockfile substitution | Verify or extend the generator's substitution pass. | `package.json::name` is templated; lockfile may carry literal placeholder. |

## 10. Testing Plan

All verification is automated. No manual checklists.

### Phase 1 — Repository worked example
- New: `api/internal/store/sqlite_test.go` covers the four cases listed
  in §8 Phase 1.4
- Updated: `api/internal/store/schema_test.go::TestEnsureSchema_AppliesTasksTable`
  replaces the placeholder no-op test
- Race-detector clean: `go test -race ./internal/store/...`

### Phase 2 — Coverage gate
- CI workflow runs the gate; verify by intentionally breaking it
  locally (delete one test, observe `floor` failure), then restoring.
- The gate failure path is covered by CI's own behaviour, not a unit
  test (CI is the test harness).

### Phase 3 — Deps nil-handling
- New: `api/internal/server/server_test.go` covers all five panic paths
  + happy path. Uses `defer recover()` to assert panic messages.

### Phase 4 — `interp` move
- New: `ui/src/test-utils/interp.test.ts` covers happy-path
  interpolation, number coercion, missing-variable error, empty template.
- Existing: `ui/src/App.test.tsx` continues to pass (proves the new
  import path works in real consumers).
- ESLint check: `pnpm lint` passes with the new import.

### Phase 5 — Lockfile shipping
- CI's `pnpm install --frozen-lockfile --ignore-workspace` step is the
  test. A passing UI job in `scenarios/smoke-tier1/` after regeneration
  proves the lockfile rides correctly through `vrooli scenario generate`.

### Phase 6 — Whole-gate regression
- Full local gate per §8 Phase 6.2 must pass with **zero** lint, type,
  test, coverage, or build failures.
- `vrooli scenario test smoke-tier1` reports green across phases.
- API + UI health endpoints respond 200 after `vrooli scenario restart`.

## 11. Rollout & Validation Checklist

```
[ ] Phase 1: schema.sql adds tasks table; sqlite.go + sqlite_test.go land
[ ] Phase 1: schema_test.go updated; SEAMS.md + TESTING.md cross-ref
[ ] Phase 2: test.yml api/cli jobs run -coverprofile + floor gate
[ ] Phase 2: TESTING.md documents the floor and graduation policy
[ ] Phase 3: server.New panics on missing deps; main.go passes Logger
[ ] Phase 3: server_test.go covers all panic paths + happy path
[ ] Phase 3: SEAMS.md + TESTING.md updated with the new contract
[ ] Phase 4: interp.ts + interp.test.ts in test-utils
[ ] Phase 4: test-setup.ts no longer exports interp
[ ] Phase 4: App.test.tsx imports from test-utils; full grep clean
[ ] Phase 5: pnpm-lock.yaml committed at templates/.../ui/
[ ] Phase 5: generator substitution pass handles lockfile (verified or extended)
[ ] Phase 5: README.md setup workflow updated
[ ] Phase 6: scenarios/smoke-tier1/ regenerated from updated template
[ ] Phase 6: full local gate passes (api + cli + ui jobs locally)
[ ] Phase 6: `vrooli scenario test smoke-tier1` green
[ ] Phase 6: vrooli scenario restart smoke-tier1; both health checks pass
[ ] Phase 6: PROGRESS.md updated in template + smoke scenario
```

## 12. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Lockfile carries `{{SCENARIO_ID}}` placeholder values that break post-generation install | Medium | High — defeats the whole phase | Phase 5 step 2 verifies pnpm tolerates the placeholder (or extends the substitution pass). If pnpm rejects it, fall back to: do **not** ship the lockfile; instead extend `template.json::postHooks` to commit the generated lockfile via `git add ui/pnpm-lock.yaml` and document the requirement in README. |
| 70% coverage floor is currently below actual coverage; adding the gate hides regressions until they breach the floor | Low | Medium | After Phase 2, run the gate locally and capture the actual % in `TESTING.md`. Add a follow-up note to graduate the floor toward the actual when next release ships. |
| `server.New` panic on missing deps is a behaviour change for any forked scenario whose tests omit Logger | High for forked scenarios | Low — tests fail loudly, not silently | Greenfield posture (per §3) makes this an explicit non-goal. Forked scenarios update independently. |
| `EnsureSchema` going from no-op-when-empty to applying a real schema breaks any forked scenario relying on the empty-placeholder no-op | Low — unlikely anyone shipped that branch | Medium | The placeholder no-op test was a contract gate for *the template only*; it's replaced by an actual table-applies test. Forked scenarios that already have their own schema are unaffected. |
| Coverage floor varies between local Go versions and CI Go version | Low | Low | CI pulls Go version from go.mod (already), and `-covermode=atomic` is version-stable. Run Phase 6 with the same Go version CI uses. |
| `interp` import path change breaks any other test files I missed | Low | Low — caught by `pnpm test` | Phase 4 step 6 includes a tree-wide grep before commit. |

## 13. Non-goals & Prohibited Patterns

- **No** Tier 2 cleanups in this plan. They get their own pass after
  Tier 1 lands.
- **No** introduction of dependency-injection frameworks (wire, fx, etc.).
  Constructor injection through `server.Deps` is the contract.
- **No** weakening of existing seam contracts (Pinger stays minimal,
  Clock stays Now-only).
- **No** removal of any existing test (the tests this plan replaces are
  for genuinely-obsolete contracts: empty-placeholder schema and
  silent Logger default).
- **No** addition of compatibility shims, `// removed` markers, or
  re-exports from old paths (e.g., do not keep `interp` exported from
  `test-setup.ts` for "backwards compatibility"). Greenfield only.
- **No** lowering of the coverage floor to accommodate any specific
  test that's hard to write — fix the test, or document the
  exclusion in `vite.config.ts` style with a one-line rationale.
- **No** silent defaults anywhere in `server.New`. If a future dep
  (e.g., a tracer) is genuinely optional, document that explicitly
  and have the panic-switch tolerate `nil` for that field only with
  a comment naming the optionality.

## 14. Definition of Done

The plan is complete when **all** of the following hold:

1. A fresh `vrooli scenario generate react-vite --id <new> ...` produces
   a scenario that, with **zero manual edits**, passes:
   - `(cd api && go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | awk '/^total:/ { exit ($3+0 < 70) }')`
   - `(cd cli && go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | awk '/^total:/ { exit ($3+0 < 70) }')`
   - `(cd ui && pnpm install --frozen-lockfile --ignore-workspace && pnpm strings:check && pnpm type-check && pnpm lint && pnpm test:coverage && pnpm build)`
2. `scenarios/smoke-tier1/` is regenerated from the updated template,
   `vrooli scenario test smoke-tier1` passes, and a `vrooli scenario
   restart smoke-tier1` followed by health checks on the API and UI
   ports both return 200.
3. `templates/scenarios/react-vite/api/internal/store/sqlite_test.go`
   exists, exercises a real (per-test temp) SQLite handle through the
   `TaskStore` interface, and is referenced from
   `docs/internal/TESTING.md` as a canonical example.
4. `.github/workflows/test.yml` enforces a Go-side coverage floor for
   both api and cli jobs, and the floor is documented in
   `docs/internal/TESTING.md`.
5. `server.New` panics on every required-dep nil/zero, all five panic
   paths have unit-test coverage, and the contract is reflected in
   `SEAMS.md` and `TESTING.md`.
6. `interp` lives in `ui/src/test-utils/interp.ts`, has a self-test,
   is re-exported from `test-utils/index.ts`, and `test-setup.ts`
   contains no exports.
7. `templates/scenarios/react-vite/ui/pnpm-lock.yaml` is committed and
   travels through `vrooli scenario generate` to produce a generated
   scenario that passes `pnpm install --frozen-lockfile`.
8. `docs/PROGRESS.md` (template) carries a one-line entry naming this
   plan and the date.
9. **No pre-existing lint, type, or test failures remain in any file
   touched by this plan, in either the template or the smoke scenario,
   even if those failures predate this work.**
