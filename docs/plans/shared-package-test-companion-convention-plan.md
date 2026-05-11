# Shared-Package Test-Companion Convention (api-core, cli-core)

**Plan file**: `path:docs/plans/shared-package-test-companion-convention-plan.md`
**Authored**: 2026-05-05
**Owner**: meta-optimization team (handed off to executing agent)
**Status**: implemented; companion-package validation green; generated-template gate has unrelated blockers
**Related (do not modify)**: `~/.claude/plans/thanks-for-the-postgres-wise-hearth.md` — the active template-fitness iteration plan; this convention is a prerequisite for that program's iteration 3

**Implementation note (2026-05-05)**: Phases A-G have landed in the worktree:
`databasetest`, `connectxtest`, and `cliapptest` exist; `cliutiltest` was
skipped because `cliutil` currently exposes concrete injection points rather
than a fake-worthy exported interface; the react-vite notes tests consume the
new companions. Package-level `go test -race ./...` validation is green for
`api-core` and `cli-core`. The Phase G end-to-end gate is now green on a
generated throwaway `shared-pkg-testkit-smoke` scenario: api `vet`/`build`/
`go test -race`, cli `vet`/`build`/`go test -race`, and ui `strings:check`/
`type-check`/`lint`/`test:coverage`/`build` all pass; smoke residue cleanup
zero. Earlier gate failures (broader template standards, missing requirement
modules, UI bundle build) were resolved out-of-band before this measurement.

---

## 1. Purpose

### 1.1 What this plan is

Establish a single canonical convention for **where test-side fakes and helpers live for shared Vrooli Go packages**, and apply it as a pilot to `api-core` and `cli-core`. The convention follows Go stdlib precedent: every shared package `<pkg>` may ship a sibling `<pkg>test` package containing the canonical fakes, server harnesses, and assertion helpers for code that consumes `<pkg>`. After this plan lands:

- `path:packages/api-core/database` has a sibling `path:packages/api-core/databasetest` shipping `FakeExecer`.
- `path:packages/api-core/connectx` has a sibling `path:packages/api-core/connectxtest` shipping `StartTestServer`, `NewLogger`, and request-capture helpers.
- `path:packages/api-core/blobstore` keeps `Memory` inline (it's a dual-purpose alt impl, not a test-only fake) but optionally gains `path:packages/api-core/blobstoretest` for assertion helpers.
- `path:packages/cli-core/cliapp` keeps its existing `NewTestRunContext` exports for back-compat AND ships the same surface from the new sibling `path:packages/cli-core/cliapptest`. Future code uses `cliapptest`; existing call sites continue to work.
- `path:packages/cli-core/cliutil` gains a sibling `cliutiltest` if and only if its exported surface has consumers worth faking (decided in Phase E).
- `path:docs/agent-system/SHARED_PACKAGE_TESTING.md` is the canonical convention reference, linked from `path:packages/api-core/README.md`, `path:packages/cli-core/README.md`, and `path:docs/agent-system/REFERENCE_PATTERN_FITNESS.md`.
- The react-vite template's notes-domain test code (`path:api/internal/notes/`, `path:api/handlers/notes/`) consumes the new test-companion packages instead of hand-rolling equivalents.

### 1.2 What this plan is NOT

This plan is **scoped narrowly** so the executing agent does not over-reach.

- **Not** an iteration of the template-fitness program. The fitness harness program is tracked separately at `~/.claude/plans/thanks-for-the-postgres-wise-hearth.md` and at `path:scenarios/prompt-manager/store/teams/meta-optimization/template-fitness-audit/react-vite-template/2026-05-04/`. After this plan lands, the fitness program runs iteration 3 against the new convention.
- **Not** introducing a generic `Repository[T]` interface in api-core. That opinion-package is deferred under "duplicate before extracting" — proves the pattern template-side first, lifts to api-core only after a second consumer.
- **Not** TypeScript shared packages. `@vrooli/api-base`, `@vrooli/proto-types`, and the UI test-utils path (`@vrooli/ui-test-utils` if added) follow the *same convention spirit* via TS subpath exports (`@vrooli/api-base/testing`), but landing those is a separate plan written after this one's pilot succeeds.
- **Not** migrating non-react-vite scenarios that hand-rolled fakes. Those catch up per scenario when next touched, per the project's standing per-scenario greenfield rule. Specifically out of scope: `path:scenarios/workspace-sandbox/api/internal/testutil/mocks/` and any other downstream hand-rolls.
- **Not** removing the template's `path:internal/testutil/` umbrella. That umbrella holds *scenario-local* helpers (sqlite handle, fixtures, http server, fake clock) that are too scenario-specific to lift; it stays.
- **Not** designing a generic `SliceRepo[T]` for in-memory repository fakes. That is a separate concern (the iter-3 fitness work) and lives template-side first.

### 1.3 Why this matters

The reference-pattern-fitness audit and the iteration-2 measurement against the post-Connect template (see `RESULTS.md` "Iteration 2 — Results") both flagged that **per-domain test scaffolding is now the dominant remaining cost** in adding a domain to a generated scenario:

- ~400 lines per domain in `mocks/{repository,service}.go` plus `mocks/{repository,service}_test.go` (test-of-mock).
- ~10 lines per domain of duplicated `newClient` helper in connect handler tests.
- ~50 lines per domain of duplicated sqlite-test setup (already partially mitigated by `path:internal/testutil/db/sqlite.go`).
- ~5 lines per call site of duplicated logger / buffer / err-injector setup in handler tests.

These costs aggregate over scenarios × domains. The fix is structural: **shared packages own the canonical test asset for their own surface**, and scenarios consume rather than re-author. Without a convention, the next agent who needs to fake `database.SchemaExecer` writes their own — drift accumulates, behaviors diverge, and bugs caught in one scenario stay buried in another.

This plan is the convention pilot. After it lands, every shared Go package gets a `<pkg>test` sibling on the same shape, and the convention generalizes mechanically (cli-core's other packages, future shared packages, eventual TS subpaths).

---

## 2. Hard rules

These constrain every phase. Repeat them in the Definition of Done; failures here invalidate the plan.

### 2.1 Naming and discovery

- **Convention is `<package>test` top-level sibling.** Not `<package>/test/` subpackage; not an umbrella `package:api-core/testing/` directory; not `*kit` / `*util` suffixes. Mirrors Go stdlib (`httptest`, `iotest`, `fstest`).
- **Production-quality alternate implementations stay inline.** `blobstore.Memory` and `blobstore.Filesystem` both stay in `blobstore/` because both are valid runtime choices, not test-only doubles.
- **Pure exported consumer test helpers go in `<pkg>test`.** Recording fakes, error-injection wrappers, server harnesses, assertion helpers. Helpers used only by a shared package's own tests may stay unexported in `_test.go`, `testdata/`, or module-local `path:internal/testutil/` when repeated.
- **The package import path is `github.com/vrooli/api-core/<pkg>test` (or `github.com/vrooli/cli-core/<pkg>test`).** Conventional alias is `<pkg>test`, e.g. `import dbtest "github.com/vrooli/api-core/databasetest"` — alias optional, but encouraged for readability.

### 2.2 Additive only on api-core / cli-core

cli-core and api-core are consumed by other scenarios outside this repo's worktree (not just the template). Therefore:

- **Do not modify exported function signatures** of existing api-core or cli-core surface.
- **Do not remove exported symbols.** Existing exports stay.
- **`cliapp.NewTestRunContext`, `cliapp.NewTestRunContextFromArgs`, `cliapp.TestRunContextOptions` continue to exist** in their current location (`path:packages/cli-core/cliapp/runcontext.go`). Phase D adds copies in `cliapptest/` that delegate to the existing implementation. The convention doc names `cliapptest` as the recommended import going forward, but the old import path is not deprecated, not aliased to a `Deprecated:` comment, and not removed. Two import paths — same behavior — let downstream consumers migrate at their own pace.

### 2.3 Greenfield in template adoption

The template is regenerable, not migrated. So in Phase F:

- **No compatibility shims** in `path:templates/scenarios/react-vite/`. The notes-domain Connect test code uses `connectxtest.StartTestServer` (etc.) directly for shared-package plumbing. Thin per-domain helpers may remain when they only bind domain-specific generated handlers and clients (for example, `newNotesClient` may remain as a small wrapper around `connectxtest.StartTestServer`). What gets deleted is duplicated mux / `httptest.NewServer` / logger plumbing, not domain-specific wiring.
- **No `// Deprecated:`, `// legacy`, `// compat` markers** in template diffs.
- **Failed approaches surface immediately.** If a consumed test-companion API doesn't fit the template's needs, fix the test-companion package; do not work around in template.

### 2.4 Single-source-of-truth for fakes

Once `<pkg>test` ships a fake of an interface in `<pkg>`, scenarios SHOULD NOT hand-roll an equivalent. Enforcement in this plan is convention + documentation only (no AST guardrail in this plan; that's a follow-up if drift surfaces).

### 2.5 Testable rule statements

A `git grep` of the post-plan worktree returns:
- Zero `// Deprecated:` / `// legacy` / `// compat` markers introduced under `path:templates/scenarios/react-vite/`.
- Zero `// Deprecated:` markers introduced under `path:packages/cli-core/cliapp/runcontext.go` for `NewTestRunContext` / `NewTestRunContextFromArgs` / `TestRunContextOptions`. They stay un-decorated.
- Zero new exported symbols *removed* from `path:packages/api-core/` or `path:packages/cli-core/` (no removals; only additions).
- Exactly one canonical `FakeExecer` for `database.SchemaExecer`, in `path:packages/api-core/databasetest/`.
- Exactly one canonical Connect test-server harness, in `path:packages/api-core/connectxtest/`.

---

## 3. Required reading

Run before executing any phase:

```bash
prompt-manager skill read unit-test-architecture test-seam-discovery-and-enforcement utils-unification implementation-plan-authoring
```

Reference files to read top-to-bottom (in this order):

- This file.
- [`path:packages/api-core/database/connect.go`](../../packages/api-core/database/connect.go) — current `Config`, `Connect`, `MustConnect` surface.
- [`path:packages/api-core/database/schemas.go`](../../packages/api-core/database/schemas.go) — `SchemaProvider`, `SchemaProviderFunc`, `SchemaExecer`, `EnsureSchemas`. The first concrete fake target.
- [`path:packages/api-core/blobstore/blobstore.go`](../../packages/api-core/blobstore/blobstore.go) and `memory.go` — the inline-alt-impl precedent. Confirms Memory stays inline.
- [`path:packages/api-core/connectx/connectx.go`](../../packages/api-core/connectx/connectx.go) — `ServiceMount`, `RegisterServices`. The second concrete fake target.
- [`path:packages/cli-core/cliapp/runcontext.go`](../../packages/cli-core/cliapp/runcontext.go) — read end-to-end. Confirm `NewTestRunContext` (line ~196), `NewTestRunContextFromArgs` (line ~241), `TestRunContextOptions` (line ~171) are the exports to mirror.
- [`path:packages/cli-core/cliapp/scenario_app.go`](../../packages/cli-core/cliapp/scenario_app.go) — `*ScenarioApp` type referenced by `RunContext.Core()`.
- [`path:packages/cli-core/cliutil/httpclient.go`](../../packages/cli-core/cliutil/httpclient.go) — surface scan to decide whether `cliutiltest` is justified in Phase E.
- [`path:templates/scenarios/react-vite/api/internal/notes/mocks/repository.go`](../../templates/scenarios/react-vite/api/internal/notes/mocks/repository.go) — handwritten domain fake. Stays for now (Phase F target is *handler-side* testing helpers, not domain-side mocks). Confirm scope.
- [`path:templates/scenarios/react-vite/api/handlers/notes/connect_handler_test.go`](../../templates/scenarios/react-vite/api/handlers/notes/connect_handler_test.go) — reads the `newNotesClient` helper whose low-level server/logger plumbing Phase F replaces with `connectxtest`.
- [`path:templates/scenarios/react-vite/api/internal/notes/sqlite_test.go`](../../templates/scenarios/react-vite/api/internal/notes/sqlite_test.go) — reads `newSchemaDB` helper. Decide in Phase F whether to migrate it to `databasetest` or keep it template-side (recommended: keep, because it composes scenario-specific schemas).
- [`path:scenarios/prompt-manager/store/teams/meta-optimization/template-fitness-audit/react-vite-template/2026-05-04/RESULTS.md`](../../scenarios/prompt-manager/store/teams/meta-optimization/template-fitness-audit/react-vite-template/2026-05-04/RESULTS.md) "Iteration 2 — Results" section — the cost composition that motivates this plan.
- [`path:docs/agent-system/REFERENCE_PATTERN_FITNESS.md`](../../docs/agent-system/REFERENCE_PATTERN_FITNESS.md) — the lens canon. This plan addresses the per-replica cost sub-lens for shared-package consumption.

---

## 4. Problem statement

### 4.1 What today's pattern looks like

Today there is no consistent home for test-side companions of shared packages.

- **`cliapp.NewTestRunContext`** exists *inside* the production package (`path:packages/cli-core/cliapp/runcontext.go`). It works, but it mixes prod + test surface in one file. There is no test-companion package; the test helper happens to live in the same package as the type it doubles for.
- **`blobstore.Memory`** is an inline alt-impl (correct — it's also a runtime choice).
- **`database.SchemaExecer`** has no fake. Tests in scenarios that exercise code calling `EnsureSchemas` either use a real sqlite handle (acceptable but slow) or hand-roll a one-off implementation.
- **`connectx.RegisterServices`** has no test harness. Every Connect-handler test in the template (and in any future scenario that follows the template) repeats a 12-line `newClient` helper that mounts a real Connect handler on a real `httptest.Server`. Today: 1 copy in the template (notes domain). Tomorrow: 1 per Connect domain × 1 per scenario.
- **`cliutil.HTTPClient`-style surfaces** have no fakes. Tests against code that calls api-core CLIs hand-roll http server fixtures.

### 4.2 What replication factor looks like

For one new domain in one new scenario today:

| Cost | Lines | Notes |
|------|------|------|
| `newClient` helper | ~12 | Per-domain Connect-handler test setup |
| `mocks/repository.go` | ~100 | Hand-written FakeRepository (template owns this; not in scope here) |
| `mocks/service.go` | ~100 | Hand-written FakeService (template owns this; not in scope here) |
| `newSchemaDB` helper | ~15 | Per-test-file sqlite-with-schema helper |
| Logger / buffer wiring | ~5 | Per-test setup |

This plan reduces the **shared-package-attributable** part of that — the `newClient` helper, the logger/buffer wiring, and any future fake-of-shared-interface tests — by moving the helpers to `<pkg>test`. The `mocks/{repository,service}.go` files are the iter-3 fitness target; this plan does not touch them.

### 4.3 What the audit established

From the iter-2 results (`RESULTS.md` § "Issues and observations from the iter-2 template" and § "Open questions for iteration 3"):

> "The remaining cost concentrates in: (a) per-domain mocks/ folder (~150 lines/domain — could a `gen-mocks` codegen close this?), (b) per-domain test scaffolding (handlers tests, sqlite tests, service tests — substantial duplicated shape) …"

This plan attacks (b) for the parts attributable to shared-package consumption. (a) is iter-3 fitness work, not in scope here.

### 4.4 The risk we are pricing

Without the convention:
- Every scenario evolves its own `FakeExecer`, its own `newConnectClient`, its own logger setup. Behavior drifts.
- New agents repeatedly relearn how to fake api-core's interfaces because there is no canonical home.
- The "duplicate before extracting" memory becomes harder to honor: when a second domain shows up, where does the extracted version go? Without a convention, the answer is "wherever the agent feels like" — which means inconsistency.

With the convention:
- `<pkg>test` is the obvious home; future agents reach for it before writing their own.
- api-core and cli-core become the single source of truth for their own fakes.
- The convention generalizes: when a third or fourth shared package needs the same treatment, the answer is mechanical.

---

## 5. Scope

### 5.1 In scope

**Phase A — Convention documentation.** New file `path:docs/agent-system/SHARED_PACKAGE_TESTING.md` (≈100–150 lines):
- Statement of the `<pkg>test` sibling convention.
- Naming: top-level sibling, not nested subpackage; no `*kit` / `*util` suffixes.
- Content guidelines: pure exported test helpers belong here; production-quality alternate impls (in-memory stores that are also runtime choices) stay inline; unexported helpers used only by the shared package's own tests may stay in `_test.go`, `testdata/`, or module-local `path:internal/testutil/` when repeated.
- Two-layer test utility rule: exported consumer helpers live in `<pkg>test`; shared-package-internal test helpers stay local and unexported.
- Discovery: linked from each shared package's README; mentioned in `REFERENCE_PATTERN_FITNESS.md` § "Drift surface map".
- TypeScript equivalent: subpath export pattern (`@vrooli/<pkg>/testing`). Marked as "applies when a TS shared package adopts this convention; not landed in this plan."
- Examples: api-core/databasetest, api-core/connectxtest, cli-core/cliapptest.
- Anti-patterns: no umbrella `testing/` directory in api-core; no `*kit`/`*util` suffixes; no inline test-only fakes that aren't dual-purpose.

**Phase B — `path:packages/api-core/databasetest/`.**
- New directory with `execer.go`, `execer_test.go`, `doc.go`.
- `FakeExecer` struct: records queries, supports default `ExecErr` injection and ordered per-call failure injection (`FailOnCall` + `FailErr`), exposes `ExecCalls atomic.Int64`, and provides synchronized accessors such as `SnapshotQueries()` / `SetExecErr(error)`. Implements `database.SchemaExecer` (compile-time guarantee via `var _ database.SchemaExecer = (*FakeExecer)(nil)`).
- Returns a stub `sql.Result` value (e.g. `driver.RowsAffected(0)`) so `EnsureSchemas` doesn't dereference nil.
- `doc.go` contains the package doc with the convention reference.
- Tests (≥90% coverage): records queries in order; all-call err injection preempts append; ordered failure injection fails only the requested call; atomic counter increments under concurrent use.

**Phase C — `path:packages/api-core/connectxtest/`.**
- New directory with `server.go`, `server_test.go`, `logger.go`, `logger_test.go`, `doc.go`.
- `StartTestServer(t *testing.T, services ...connectx.ServiceMount) *httptest.Server` — replicates the existing per-domain `newClient` helper. Includes `t.Cleanup(server.Close)`.
- `NewLogger(t *testing.T) (*log.Logger, *bytes.Buffer)` — returns a logger and the buffer it writes to, so tests can assert on log output. Two-return-value shape preserves the common usage pattern of "give me a logger AND let me read what it logged."
- Optional: `RecordingService` — a tiny `connectx.ServiceMount` whose handler records request paths/headers without forwarding to a real implementation. Defer to the optional list; only land if Phase C's test work surfaces a concrete need.
- Tests (≥90% coverage): server lifecycle (cleanup runs); logger captures expected text; multiple services mount correctly.

**Phase D — `path:packages/cli-core/cliapptest/` (move-with-back-compat).**
- New directory with `runcontext.go`, `runcontext_test.go`, `doc.go`.
- `cliapptest.NewTestRunContext`, `cliapptest.NewTestRunContextFromArgs`, `cliapptest.TestRunContextOptions` — three new exports. Implementation is a one-line forward to the existing `cliapp.NewTestRunContext` etc. (NOT a copy of the body — preserve single-implementation-source).
- `doc.go` explains: "This package mirrors the test-helper exports currently in `cliapp` so consumers can follow the `<pkg>test` convention. The existing `cliapp.NewTestRunContext` etc. continue to work; new code SHOULD prefer `cliapptest`."
- The pre-existing `cliapp.NewTestRunContext` etc. **stay** (no `// Deprecated:` marker, no removal, no signature change). Per §2.2.
- Tests verify the forwards behave identically to the originals (table-driven: same input → same RunContext shape).

**Phase E — `path:packages/cli-core/cliutiltest/` (decision point + conditional landing).**
- Decision criterion: scan `path:packages/cli-core/cliutil/` exported surfaces. If any exported interface (e.g. an HTTP client surface) is consumed by code that scenarios test against, land a fake. If not, skip and document the decision in §15 Deviations.
- If landing: `path:packages/cli-core/cliutiltest/<surface>.go` + tests + `doc.go`.
- If skipping: explicit one-paragraph note in `path:docs/agent-system/SHARED_PACKAGE_TESTING.md` § "Per-package status" stating which packages got companions and why others were skipped.

**Phase F — Template adoption (notes domain only).**
- `path:templates/scenarios/react-vite/api/handlers/notes/connect_handler_test.go`:
  - Keep a thin `newNotesClient` helper if useful, but replace its duplicated mux / `httptest.NewServer` / default-logger implementation with `connectxtest.StartTestServer` and `connectxtest.NewLogger`.
  - The per-domain wiring stays per-domain because it ties a domain-specific `notes.NewConnectHandler` to a domain-specific `notesconnect.NewNotesServiceClient` — it's not generalizable without higher-order generics, deferred to a later iteration.
  - Use `connectxtest.NewLogger` for log-assertion tests.
- `path:templates/scenarios/react-vite/api/handlers/notes/attachments_handler_test.go`: review only. Do not migrate to `connectxtest.StartTestServer` unless the file is actually mounting Connect services; today's multipart upload tests are REST/mux tests and should keep scenario-local REST helpers.
- `path:templates/scenarios/react-vite/api/handlers/notes/adapter_test.go`: review for relevant migrations; apply where applicable.
- `path:templates/scenarios/react-vite/cli/domains/notes/handlers_test.go`: switch from `cliapp.NewTestRunContext` to `cliapptest.NewTestRunContext`. (Optional but demonstrates the new convention.)
- `path:templates/scenarios/react-vite/api/internal/notes/sqlite_test.go`: keep `newSchemaDB` template-side (it composes scenario-specific `notes.Schema` + `localdb.SystemSchema`; not generalizable). Document the decision.
- Update `path:templates/scenarios/react-vite/docs/internal/SEAMS.md` with new rows for the consumed test-companion packages.
- Update `path:templates/scenarios/react-vite/docs/internal/REPLACING-NOTES.md` to point new agents at `connectxtest.StartTestServer` / `cliapptest.NewTestRunContext` / `databasetest.FakeExecer` as the canonical fakes when adding a new domain.

**Phase G — Validation.**
- All gates green on api-core, cli-core, template.
- A throwaway scenario generated from the post-plan template gates green end-to-end.
- Convention doc is discoverable: `path:docs/README.md` (or the agent-system index) links to it.

### 5.2 Out of scope

- **Generic `Repository[T]` opinion in api-core.** Deferred — proves template-side first.
- **`SliceRepo[T]` test primitive.** Iteration-3 fitness work, not this plan.
- **Migrating template-side `mocks/{repository,service}.go`.** Per-domain mocks are the iter-3 target. This plan touches *handler-side* test helpers, not domain-side mocks.
- **TypeScript shared packages.** `@vrooli/api-base/testing` subpath: separate plan after this pilot.
- **AST guardrails enforcing the convention.** No `no_test_helpers_outside_pkgtest_test.go` guardrail in this plan. If drift surfaces post-landing, that's a follow-up.
- **Migrating non-react-vite scenarios.** Workspace-sandbox, swarm-manager, prompt-manager, etc. continue using their hand-rolled helpers; they catch up when next touched per the per-scenario greenfield rule.
- **Any change to api-core's exported surface beyond additions.** Phases B–E are purely additive.
- **Renaming or relocating `blobstore.Memory`.** Stays where it is.

### 5.3 The files this plan creates / modifies

**Created:**
| Path | Phase | Purpose |
|------|-------|---------|
| `path:docs/agent-system/SHARED_PACKAGE_TESTING.md` | A | Convention doc |
| `path:packages/api-core/databasetest/doc.go` | B | Package doc |
| `path:packages/api-core/databasetest/execer.go` | B | `FakeExecer` |
| `path:packages/api-core/databasetest/execer_test.go` | B | FakeExecer tests |
| `path:packages/api-core/connectxtest/doc.go` | C | Package doc |
| `path:packages/api-core/connectxtest/server.go` | C | `StartTestServer` |
| `path:packages/api-core/connectxtest/server_test.go` | C | server tests |
| `path:packages/api-core/connectxtest/logger.go` | C | `NewLogger` |
| `path:packages/api-core/connectxtest/logger_test.go` | C | logger tests |
| `path:packages/cli-core/cliapptest/doc.go` | D | Package doc |
| `path:packages/cli-core/cliapptest/runcontext.go` | D | Forwards to cliapp |
| `path:packages/cli-core/cliapptest/runcontext_test.go` | D | Forward-equivalence tests |
| `path:packages/cli-core/cliutiltest/*` | E | Conditional |

**Modified:**
| Path | Phase | Change |
|------|-------|--------|
| `path:packages/api-core/README.md` | A | Link to SHARED_PACKAGE_TESTING.md; mention databasetest, connectxtest |
| `path:packages/cli-core/README.md` | A | Link; mention cliapptest |
| `path:docs/agent-system/REFERENCE_PATTERN_FITNESS.md` | A | One paragraph in "Drift surface map" referencing the convention |
| `path:templates/scenarios/react-vite/api/handlers/notes/connect_handler_test.go` | F | Adopt `connectxtest.*` |
| `path:templates/scenarios/react-vite/api/handlers/notes/attachments_handler_test.go` | F | Review only; adopt shared helpers only if a true Connect mount or logger duplication exists |
| `path:templates/scenarios/react-vite/api/handlers/notes/adapter_test.go` | F | Adopt where applicable |
| `path:templates/scenarios/react-vite/cli/domains/notes/handlers_test.go` | F | Switch to `cliapptest.NewTestRunContext` |
| `path:templates/scenarios/react-vite/docs/internal/SEAMS.md` | F | New rows for the test-companion seams |
| `path:templates/scenarios/react-vite/docs/internal/REPLACING-NOTES.md` | F | Point to canonical fakes |

**Untouched (intentionally):**
- `path:packages/cli-core/cliapp/runcontext.go` — exports stay; Phase D adds `cliapptest` as an additional path.
- `path:packages/api-core/blobstore/{blobstore,memory,filesystem}.go` — inline alt-impl pattern preserved.
- `path:templates/scenarios/react-vite/api/internal/notes/mocks/*.go` — domain-side mocks; iter-3 target.
- `templates/scenarios/react-vite/api/internal/notes/sqlite_test.go::newSchemaDB` — composes scenario-specific schemas; stays template-side.

---

## 6. Current technical context

### 6.1 api-core surface today

`path:packages/api-core/` is a Go module consumed by scenarios via `replace` directive. Public packages:

- `database/` — `Connect`, `MustConnect`, `Config`, `SchemaProvider`, `SchemaProviderFunc`, `SchemaExecer`, `EnsureSchemas`. Multi-driver (postgres, sqlite, sqlite-legacy). `EnsureSchemas` is fully driver-agnostic via the `SchemaExecer` interface.
- `connectx/` — `ServiceMount`, `RegisterServices`, plus error-mapping helpers. Used by every Connect-RPC handler in scenarios.
- `blobstore/` — `Store` interface, `Memory` and `Filesystem` impls, both production-quality. Memory is also the canonical "fake" for test code that calls a `Store`.
- `secrets/`, `health/`, `discovery/`, `retry/`, `storage/`, `staleness/`, `pathfilter/`, `preflight/`, `server/`, `scenario/`, `scenariocli/` — other packages; surface scan is in scope for Phase A's per-package status note but only `database` and `connectx` get test companions in this plan.

No `<pkg>test` package exists in api-core today.

### 6.2 cli-core surface today

`path:packages/cli-core/` is a Go module. Public packages:

- `cliapp/` — `App`, `ScenarioApp`, `Command`, `RunContext`, `ArgSchema`, `Call[Req,Resp]`, helpers. Includes `NewTestRunContext`, `NewTestRunContextFromArgs`, `TestRunContextOptions` exported from `runcontext.go`. These three exports are the migration target in Phase D.
- `cliutil/` — `APIError`, `ParseAPIError`, `HTTPClient` and friends, `ParseInterspersed`, identity / freshness / sandbox helpers. Phase E decides whether `cliutiltest` is justified.
- `cmd/`, `buildinfo/`, `sandbox-resolve/` — utility packages; not in scope for this plan.

No `<pkg>test` package exists in cli-core today.

### 6.3 Template adoption surface today

`path:templates/scenarios/react-vite/api/handlers/notes/connect_handler_test.go` (~99 lines) contains the `newNotesClient` helper whose low-level server/logger plumbing this plan replaces. The helper:

```go
func newNotesClient(t *testing.T, fake *mocks.FakeService, logger *log.Logger) notesconnect.NotesServiceClient {
    t.Helper()
    if logger == nil {
        logger = log.New(&bytes.Buffer{}, "", 0)
    }
    path, handler := notesconnect.NewNotesServiceHandler(notes.NewConnectHandler(notes.Deps{Service: fake, Logger: logger}))
    router := mux.NewRouter()
    connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
    server := httptest.NewServer(router)
    t.Cleanup(server.Close)
    return notesconnect.NewNotesServiceClient(server.Client(), server.URL)
}
```

Twelve lines. The important duplicate is the mux/server/default-logger plumbing. After Phase F the per-domain client helper may remain, but it becomes a thin wrapper:

```go
import connectxtest "github.com/vrooli/api-core/connectxtest"

func newNotesClient(t *testing.T, fake *mocks.FakeService, logger *log.Logger) notesconnect.NotesServiceClient {
    t.Helper()
    if logger == nil {
        logger, _ = connectxtest.NewLogger(t)
    }
    path, handler := notesconnect.NewNotesServiceHandler(notes.NewConnectHandler(notes.Deps{Service: fake, Logger: logger}))
    server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
    return notesconnect.NewNotesServiceClient(server.Client(), server.URL)
}
```

Saving: ~5 lines per per-domain Connect helper, 1 file per Connect domain × N future domains. Smaller than per-replica savings target — but the structural win is the consolidated owner: future `connectx` changes propagate via one `connectxtest` update, not N domain-by-domain edits. REST-only helpers, such as multipart upload tests that use `httptest.NewRecorder`, remain scenario-local unless another shared-package seam is involved.

### 6.4 Why this plan does not centralize `newSchemaDB`

`templates/scenarios/react-vite/api/internal/notes/sqlite_test.go::newSchemaDB` opens a sqlite handle and calls `apidb.EnsureSchemas` with **scenario-specific** providers (`localdb.SystemSchema`, `notes.Schema`). The first arg is generalizable; the rest is not. A naïve `databasetest.NewSchemaDB(t, providers...)` could replace the open-handle part, but the per-test value comes from the *combined* helper that pre-applies the right schemas.

Decision: leave the helper template-side; document the seam in `SEAMS.md`. If a future audit shows the schema-list composition is itself replicated across scenarios, lift then.

---

## 7. Target end state

After this plan executes:

### 7.1 Convention documented

1. `path:docs/agent-system/SHARED_PACKAGE_TESTING.md` exists with the convention statement, naming rules, content guidelines, examples, and anti-patterns.
2. `path:packages/api-core/README.md` and `path:packages/cli-core/README.md` link to the convention doc.
3. `path:docs/agent-system/REFERENCE_PATTERN_FITNESS.md` references the convention in its drift-surface section.

### 7.2 api-core test companions exist

4. `path:packages/api-core/databasetest/` exists with `FakeExecer` (records queries, injects errors, supports ordered failure/injection, exposes synchronized accessors, atomic counter), passes `var _ database.SchemaExecer` compile-check, has tests covering record/inject/ordered failure/concurrent paths.
5. `path:packages/api-core/connectxtest/` exists with `StartTestServer`, `NewLogger`, tests covering server lifecycle + log capture + multi-service mount.

### 7.3 cli-core test companions exist

6. `path:packages/cli-core/cliapptest/` exists with `NewTestRunContext`, `NewTestRunContextFromArgs`, `TestRunContextOptions`. Each is a one-line forward to the corresponding `cliapp` export.
7. `path:packages/cli-core/cliapp/runcontext.go` is unchanged in exported surface (no `Deprecated:` markers, no removals, no signature changes).
8. `path:packages/cli-core/cliutiltest/` exists *if* Phase E found a justified surface; otherwise the decision is documented.

### 7.4 Template adopts the convention

9. `path:templates/scenarios/react-vite/api/handlers/notes/connect_handler_test.go` no longer mounts `httptest.NewServer` directly for Connect handlers; any retained `newNotesClient` helper delegates server setup to `connectxtest.StartTestServer`.
10. `attachments_handler_test.go` is reviewed and left REST-local unless it contains true Connect-service mounting or logger duplication that a shared helper owns.
11. `path:cli/domains/notes/handlers_test.go` imports `cliapptest` (not `cliapp`) for the test-context constructor.
12. `path:docs/internal/SEAMS.md` and `path:docs/internal/REPLACING-NOTES.md` reflect the new convention.
13. Greenfield rule check: zero `Deprecated:` / `// legacy` / `// compat` markers introduced in template diffs.

### 7.5 Validation

14. `cd packages/api-core && go test -race ./...` green.
15. `cd packages/cli-core && go test -race ./...` green.
16. Throwaway scenario generated from the post-plan template gates green end-to-end (api/cli/ui type-check / lint / test / build).

---

## 8. Implementation strategy

Phases A → G run in order. A gates B/C/D/E (convention before pilot). B/C/D/E can run in parallel after A but the agent should sequence them serially for clarity. F gates on B/C/D/E. G gates on F.

### Phase A — Convention documentation

**Skill alignment**: `unit-test-architecture`, `utils-unification`, `implementation-plan-authoring`.

#### A.1 Author `path:docs/agent-system/SHARED_PACKAGE_TESTING.md`

Contents (suggested headings, ~100–150 lines total):

1. **Statement.** Every shared Go package `<pkg>` MAY ship a sibling `<pkg>test` Go package containing canonical fakes, server harnesses, and assertion helpers for code that *consumes* `<pkg>`.
2. **Naming.** `<pkg>test` top-level sibling. Not `<pkg>/test/` subpackage. Not umbrella `package:api-core/testing/`. No `*kit`/`*util` suffixes.
3. **Content rules.**
   - In `<pkg>test`: pure test helpers — recording fakes (`FakeExecer`), server harnesses (`StartTestServer`), test-only constructors (`NewLogger(t)`).
   - In `<pkg>` (inline): production-quality alternate impls (`blobstore.Memory`). Distinguished by "is this also a runtime choice?" — yes ⇒ inline, no ⇒ companion.
4. **Two-layer test utility rule.**
   - Exported helpers for consumers live in `<pkg>test`.
   - Helpers used only by a shared package's own tests stay close to those tests: unexported helpers in `_test.go`, fixtures in `testdata/`, or a module-local `path:internal/testutil/` only when repeated across several packages.
   - Do not create broad `package:api-core/test-utils`, `package:cli-core/test-utils`, or umbrella testing packages to mirror scenario-local UI layout. The conceptual rule matches the React template's `test-utils` pattern — centralize at the owner boundary — but the Go import-path shape is `<pkg>test`.
5. **Discovery.** Each shared package's README links to this doc. Future agents reading a shared package learn the convention from its README.
6. **TypeScript equivalent.** TS packages use subpath exports: `@vrooli/<pkg>/testing` resolved via `package.json` `exports` field. Pattern shape is identical (test-companion = sibling import-path); the mechanism is package-manager-specific. Not landed in this plan.
7. **Per-package status (table).** Lists each api-core and cli-core package, says whether it has a `<pkg>test` companion, whether one is planned, or whether it's intentionally skipped (with reason).
8. **Anti-patterns.**
   - "Hand-rolling a `Fake<X>` for an api-core/cli-core interface in scenario or template code" — instead, consume `<pkg>test`.
   - "Putting recording fakes in the production package" — they go in `<pkg>test`.
   - "Inventing umbrella packages like `package:api-core/testing/` or `package:api-core/testkit/`" — per-package sibling only.
9. **Examples.** Side-by-side before/after for a hypothetical scenario consuming `databasetest.FakeExecer` and `connectxtest.StartTestServer`.

#### A.2 Update READMEs and fitness doc

- `path:packages/api-core/README.md`: append a "Test companions" section listing `databasetest`, `connectxtest`, and linking to the convention doc.
- `path:packages/cli-core/README.md`: append a "Test companions" section listing `cliapptest`, and (if landed) `cliutiltest`.
- `path:docs/agent-system/REFERENCE_PATTERN_FITNESS.md`: in the drift-surface section, add a paragraph noting that hand-rolled fakes of shared-package interfaces are a drift surface; the canonical mitigation is the `<pkg>test` convention.

#### A.3 Phase A validation

- `path:docs/agent-system/SHARED_PACKAGE_TESTING.md` exists.
- It is linked from both shared-package READMEs and from the fitness doc.
- It is self-contained: an agent reading it cold can apply the convention to a new package without further context.

### Phase B — `path:packages/api-core/databasetest/`

**Gate**: Phase A complete.

#### B.1 Create the package

- `databasetest/doc.go` — package-level doc citing the convention.
- `databasetest/execer.go` — `FakeExecer` type:

```go
package databasetest

import (
    "context"
    "database/sql"
    "database/sql/driver"
    "sync"
    "sync/atomic"

    apidb "github.com/vrooli/api-core/database"
)

// FakeExecer is the canonical test double for database.SchemaExecer.
// It records every successful query passed to ExecContext (in order),
// supports all-call and ordered failure injection, and exposes
// synchronized accessors so `go test -race` stays quiet when callers fan out.
type FakeExecer struct {
    mu sync.Mutex

    queries []string
    execErr error

    // FailOnCall, if >0, makes that 1-based ExecContext call fail before
    // recording. FailErr is returned; if FailErr is nil, sql.ErrConnDone is
    // returned so the failure is never silently nil.
    FailOnCall int64
    FailErr    error

    // ExecCalls counts ExecContext invocations, including those that
    // returned an injected error.
    ExecCalls atomic.Int64
}

func (f *FakeExecer) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
    call := f.ExecCalls.Add(1)
    f.mu.Lock()
    defer f.mu.Unlock()
    if f.execErr != nil {
        return nil, f.execErr
    }
    if f.FailOnCall > 0 && call == f.FailOnCall {
        if f.FailErr != nil {
            return nil, f.FailErr
        }
        return nil, sql.ErrConnDone
    }
    f.queries = append(f.queries, query)
    return driver.RowsAffected(0), nil
}

func (f *FakeExecer) SetExecErr(err error) {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.execErr = err
}

func (f *FakeExecer) SnapshotQueries() []string {
    f.mu.Lock()
    defer f.mu.Unlock()
    return append([]string(nil), f.queries...)
}

// Compile-time guarantee.
var _ apidb.SchemaExecer = (*FakeExecer)(nil)
```

- `databasetest/execer_test.go` — at least five tests:
  - records queries in order
  - returns the error configured by `SetExecErr` before appending
  - supports ordered failure injection (`FailOnCall`) without poisoning earlier calls
  - `ExecCalls` increments under concurrent invocations (`go test -race` clean)
  - integrates with `apidb.EnsureSchemas` end-to-end (apply two providers; assert `SnapshotQueries()` length 2 and content)

#### B.2 Phase B validation

```bash
cd packages/api-core && go test -race ./databasetest/...
```

Coverage ≥90% on the new package.

### Phase C — `path:packages/api-core/connectxtest/`

**Gate**: Phase A complete; Phase B may proceed in parallel.

#### C.1 Create the package

- `connectxtest/doc.go`.
- `connectxtest/server.go`:

```go
package connectxtest

import (
    "net/http/httptest"
    "testing"

    "github.com/gorilla/mux"
    "github.com/vrooli/api-core/connectx"
)

// StartTestServer mounts the provided Connect service mounts on a
// gorilla/mux router and returns a running httptest.Server.
// t.Cleanup is registered to close the server.
func StartTestServer(t *testing.T, services ...connectx.ServiceMount) *httptest.Server {
    t.Helper()
    router := mux.NewRouter()
    connectx.RegisterServices(router, services...)
    server := httptest.NewServer(router)
    t.Cleanup(server.Close)
    return server
}
```

- `connectxtest/logger.go`:

```go
package connectxtest

import (
    "bytes"
    "log"
    "testing"
)

// NewLogger returns a *log.Logger that writes to the returned buffer.
// Tests use this to capture log output for assertion. The buffer is
// safe to read after the test executes the code under test.
func NewLogger(t *testing.T) (*log.Logger, *bytes.Buffer) {
    t.Helper()
    buf := &bytes.Buffer{}
    return log.New(buf, "", 0), buf
}
```

- `connectxtest/server_test.go` — tests: server starts, registered service is reachable, t.Cleanup closes the server (probe by triggering the cleanup early in a sub-test), multi-service mount works.
- `connectxtest/logger_test.go` — tests: buffer captures expected text; logger and buffer share state; `t.Helper` correctly attributed.

Optional (decide during implementation): `connectxtest/recording.go` with a `RecordingService` whose handler captures request paths/headers without forwarding. Add only if a real test motivates it.

#### C.2 Phase C validation

```bash
cd packages/api-core && go test -race ./connectxtest/...
```

Coverage ≥90%.

### Phase D — `path:packages/cli-core/cliapptest/`

**Gate**: Phase A complete.

#### D.1 Create the package

- `cliapptest/doc.go`:

```go
// Package cliapptest is the canonical test-helper companion for
// github.com/vrooli/cli-core/cliapp. It re-exports the test-context
// constructors that previously lived inline in cliapp; new code SHOULD
// prefer importing from this package per the shared-package test-
// companion convention documented at
// docs/agent-system/SHARED_PACKAGE_TESTING.md.
//
// The pre-existing cliapp.NewTestRunContext etc. continue to work; this
// package's exports forward to them. Two import paths, single
// implementation.
package cliapptest
```

- `cliapptest/runcontext.go`:

```go
package cliapptest

import (
    "github.com/vrooli/cli-core/cliapp"
)

// TestRunContextOptions mirrors cliapp.TestRunContextOptions. New code
// SHOULD use this type.
type TestRunContextOptions = cliapp.TestRunContextOptions

// NewTestRunContext mirrors cliapp.NewTestRunContext.
func NewTestRunContext(opts TestRunContextOptions) cliapp.RunContext {
    return cliapp.NewTestRunContext(opts)
}

// NewTestRunContextFromArgs mirrors cliapp.NewTestRunContextFromArgs.
func NewTestRunContextFromArgs(schema cliapp.ArgSchema, args []string, core *cliapp.ScenarioApp, stdout, stderr io.Writer) (cliapp.RunContext, error) {
    return cliapp.NewTestRunContextFromArgs(schema, args, core, stdout, stderr)
}
```

(Note on `TestRunContextOptions = cliapp.TestRunContextOptions`: this is a Go type alias, not a new type. It does not conflict with §2.5's "no `type Old…= New…` aliases" rule — that rule was about template-side compatibility shims. Type aliases for re-export from a sibling test-companion package are a separate idiom and are permitted here because they are the cleanest way to honor §2.2's "additive only" constraint.)

- `cliapptest/runcontext_test.go` — table-driven tests proving `cliapptest.NewTestRunContext(opts)` produces the same RunContext as `cliapp.NewTestRunContext(opts)` for representative options sets (no flags, single flag, repeated positional, --json globals).

#### D.2 Phase D validation

```bash
cd packages/cli-core && go test -race ./cliapptest/... ./cliapp/...
```

Both green. The `cliapp` package tests still pass unchanged.

### Phase E — `path:packages/cli-core/cliutiltest/` (decision point)

**Gate**: Phase A complete; Phases B/C/D may proceed in parallel.

#### E.1 Surface scan

Read `path:packages/cli-core/cliutil/` exports. Identify any exported interface that:
- Is consumed by template or scenario test code.
- Has a non-trivial fake (>5 lines) that scenarios might hand-roll.

Likely candidates: `cliutil.HTTPClient` / `cliutil.Doer` if such a surface exists; the `cliutil/installartifact.go` surface; freshness/sandbox helpers.

#### E.2 Decision

If Phase E.1 finds at least one justified target:
- Create `path:packages/cli-core/cliutiltest/` mirroring the Phase B/C shape.
- Author `<surface>.go` + tests + `doc.go`.

If not:
- Skip Phase E entirely.
- Document the decision in `path:docs/agent-system/SHARED_PACKAGE_TESTING.md` § "Per-package status" with a one-paragraph reason.
- Record in §15 Deviations.

#### E.3 Phase E validation

If landed: `cd packages/cli-core && go test -race ./cliutiltest/...` green.
If skipped: convention doc reflects the decision; no other work.

### Phase F — Template adoption (notes domain only)

**Gate**: Phases B, C, D (and E if landed) all complete and green.

#### F.1 Migrate `connect_handler_test.go`

In `path:templates/scenarios/react-vite/api/handlers/notes/connect_handler_test.go`:

- Add import: `connectxtest "github.com/vrooli/api-core/connectxtest"`.
- Replace the inline `newNotesClient` helper's body to use `connectxtest.StartTestServer`.
- Replace the inline `log.New(&bytes.Buffer{}, "", 0)` calls in tests that need a logger with `connectxtest.NewLogger(t)`.
- Confirm no `bytes`, `httptest`, `mux` imports are now unused.

#### F.2 Review `attachments_handler_test.go`

Do not force `connectxtest.StartTestServer` onto REST/multipart tests. Today's `attachments_handler_test.go` uses `httptest.NewRecorder` against a mux-mounted REST handler; that remains scenario-local. Only apply a shared helper if the review finds a true shared-package seam, such as duplicated logger capture that should use `connectxtest.NewLogger`.

#### F.3 Review `adapter_test.go`

If it has structurally similar helpers, migrate. If not, leave alone.

#### F.4 Migrate `path:cli/domains/notes/handlers_test.go`

- Replace `cliapp.NewTestRunContext` with `cliapptest.NewTestRunContext` (and the same for `NewTestRunContextFromArgs` if used).
- Update import: `cliapptest "github.com/vrooli/cli-core/cliapptest"`.

#### F.5 Update template docs

- `path:docs/internal/SEAMS.md`: add rows for `connectxtest.StartTestServer`, `connectxtest.NewLogger`, `cliapptest.NewTestRunContext`, `databasetest.FakeExecer` (the last one even though no current template test uses it — future scenarios will).
- `path:docs/internal/REPLACING-NOTES.md`: in the section on writing connect-handler tests, point to `connectxtest.StartTestServer`. In the CLI tests section, point to `cliapptest`.

#### F.6 Phase F validation

```bash
cd templates/scenarios/react-vite/api && go test -race ./...
cd templates/scenarios/react-vite/cli && go test -race ./...
```

Both green.

```bash
git diff templates/scenarios/react-vite/ | grep -E '(// Deprecated:|// legacy|// compat|// backwards-compat)'
# Must print nothing.
```

### Phase G — End-to-end validation

**Gate**: Phase F complete and green.

#### G.1 Generate a throwaway scenario

```bash
vrooli scenario generate react-vite \
  --id shared-pkg-testkit-smoke \
  --display-name "Shared Package Testkit Smoke" \
  --description "End-to-end verification for the shared-package test-companion convention plan"

cd scenarios/shared-pkg-testkit-smoke
```

#### G.2 Run all gates

```bash
( cd api && go vet ./... && go build ./... && CGO_ENABLED=0 go build ./... && go test -race ./... )
( cd cli && go vet ./... && go build ./... && CGO_ENABLED=0 go build ./... && go test -race ./... )
( cd ui && pnpm install --ignore-workspace && pnpm strings:check && pnpm type-check && pnpm lint && pnpm test:coverage && pnpm build )
```

All green.

#### G.3 Cleanup

```bash
SCEN_ID="shared-pkg-testkit-smoke"
SCEN_ID_UNDER="${SCEN_ID//-/_}"

vrooli scenario stop "$SCEN_ID" 2>/dev/null || true
rm -rf "scenarios/$SCEN_ID" \
       "packages/proto/schemas/$SCEN_ID" \
       "packages/proto/gen/go/$SCEN_ID" \
       "packages/proto/gen/typescript/$SCEN_ID" \
       "packages/proto/gen/typescript/js/$SCEN_ID" \
       "packages/proto/gen/python/$SCEN_ID_UNDER"
( cd packages/proto && make generate )

# Verify zero residue:
ls -d packages/proto/gen/go/*"$SCEN_ID"* packages/proto/gen/typescript/js/*"$SCEN_ID"* 2>/dev/null
ls -d packages/proto/gen/python/*"$SCEN_ID_UNDER"* 2>/dev/null
git status   # confirm zero residue in tracked or untracked paths
```

#### G.4 Phase G validation

- All gates green on the throwaway.
- Zero residue after cleanup.
- `git status` shows only the intentional plan-side changes (api-core additions, cli-core additions, template diffs, convention doc).

---

## 9. Contract decisions

These are the load-bearing design choices future readers may second-guess.

### 9.1 `<pkg>test` sibling, not `<pkg>/test/` subpackage or umbrella `testing/`

Sibling beats subpackage: import paths read cleanly (`package:api-core/databasetest`) and align with Go stdlib (`package:net/http` + `package:net/http/httptest` is the only stdlib precedent for a *nested* test package, and even there the convention is `httptest`, not `package:http/test`). Top-level siblings (`iotest`, `fstest`, `nettest`) dominate.

Sibling beats umbrella: `package:api-core/testing/database/` would force scenarios to import `package:api-core/testing/database` *and* `package:api-core/database` — two paths to remember. `<pkg>test` is one mental model: "to fake `<pkg>`, import `<pkg>test`."

### 9.2 Production-quality alternate impls stay inline

`blobstore.Memory` is also a runtime choice (in-memory deployments, fast local dev). Putting it in `blobstoretest/` would force production code to import a `*test` package — reads weird and would require a CI-time decision about whether `blobstoretest` is allowed in prod compilation graphs.

The discriminator is "is this also a runtime choice?" — yes (Memory) ⇒ inline, no (FakeExecer) ⇒ `<pkg>test`.

### 9.3 cli-core gets type-alias forwards; no Deprecated marker

§2.2 forbids removing `cliapp.NewTestRunContext`. The cleanest "additive only" path is type-alias re-export from `cliapptest`. No `// Deprecated:` marker is added because:
- Downstream consumers that haven't migrated should not see noisy warnings.
- The convention doc explicitly names `cliapptest` as the new home; that's where the migration nudge lives.
- Adding `Deprecated:` to a stable cross-scenario consumer surface is a scope creep; this plan is not a deprecation campaign.

### 9.4 No `// Deprecated:` markers anywhere in the diff

Honoring the spirit of the iter-1 plan's rule §2.4. The convention is taught via documentation, not via in-code warnings.

### 9.5 Convention doc lives in `path:docs/agent-system/`, not `packages/`

Cross-package conventions are agent-facing concerns. They live next to other agent-facing docs (REFERENCE_PATTERN_FITNESS, SEAMS conventions, etc.) so an agent doing reference-pattern-fitness audits or seam discovery finds them in one place.

### 9.6 Don't introduce a generic `Repository[T]` interface in api-core

Per "duplicate before extracting" memory. The pattern needs at least two consumers (notes + tasks) to validate the shape. An iter-3 fitness work-item proves the shape template-side first; a future plan lifts it to api-core.

### 9.7 Decide-then-document on cliutiltest (Phase E)

The plan does not pre-commit to landing `cliutiltest`. The `cliutil` package's exported test-faking surface is small and may not justify a sibling. The agent decides during execution and records the decision in §15 + the convention doc's per-package status table. Either outcome is acceptable; not deciding is not.

---

## 10. Testing plan

### 10.1 Each new test-companion package self-tests

Coverage targets (`go test -coverprofile`):
- `databasetest` ≥ 90%.
- `connectxtest` ≥ 90%.
- `cliapptest` ≥ 95% (small surface — three forwarding functions; tests are table-driven equivalence checks).
- `cliutiltest` ≥ 90% if landed.

Each package's tests live in `<pkg>test/*_test.go` (the test-companion package tests itself). Style matches the surrounding cli-core / api-core test conventions.

### 10.2 Existing tests still pass

- `cd packages/api-core && go test -race ./...` green (databasetest is additive; nothing existing breaks).
- `cd packages/cli-core && go test -race ./...` green (cliapptest is additive; cliapp tests unchanged).
- `cd templates/scenarios/react-vite/api && go test -race ./...` green (Phase F migration preserves behavior).
- `cd templates/scenarios/react-vite/cli && go test -race ./...` green.

### 10.3 End-to-end gate (§G)

Full smoke scenario gates green. Zero residue post-cleanup.

### 10.4 Convention discoverability check

Manual check: a fresh agent reading `path:packages/api-core/README.md` finds the link to the convention doc. The convention doc is self-contained.

---

## 11. Rollout / Validation Checklist

- [ ] Required-reading skills loaded (§3).
- [ ] **Phase A — convention documentation**:
  - [ ] `path:docs/agent-system/SHARED_PACKAGE_TESTING.md` exists, ≥ 100 lines, covers all sections in §A.1.
  - [ ] `path:packages/api-core/README.md` links to the convention doc.
  - [ ] `path:packages/cli-core/README.md` links to the convention doc.
  - [ ] `path:docs/agent-system/REFERENCE_PATTERN_FITNESS.md` references the convention.
- [ ] **Phase B — databasetest**:
  - [ ] `path:packages/api-core/databasetest/{doc,execer,execer_test}.go` exist.
  - [ ] `var _ database.SchemaExecer = (*FakeExecer)(nil)` compiles.
  - [ ] Tests cover record / all-call inject / ordered failure inject / atomic-counter / EnsureSchemas-integration paths.
  - [ ] `go test -race -coverprofile=cover.out ./databasetest/...` ≥ 90%.
- [ ] **Phase C — connectxtest**:
  - [ ] `path:packages/api-core/connectxtest/{doc,server,server_test,logger,logger_test}.go` exist.
  - [ ] `StartTestServer` registers `t.Cleanup`.
  - [ ] `NewLogger` returns logger + buffer; buffer is shared.
  - [ ] Tests cover lifecycle, capture, multi-service mount.
  - [ ] Coverage ≥ 90%.
- [ ] **Phase D — cliapptest**:
  - [ ] `path:packages/cli-core/cliapptest/{doc,runcontext,runcontext_test}.go` exist.
  - [ ] `cliapp.NewTestRunContext` etc. unchanged in `cliapp/runcontext.go` (no `Deprecated:` marker, no signature change, no removal).
  - [ ] Type alias `TestRunContextOptions = cliapp.TestRunContextOptions` works.
  - [ ] Equivalence tests prove forwards behave identically.
  - [ ] Coverage ≥ 95%.
- [ ] **Phase E — cliutiltest decision**:
  - [ ] Surface scan of `path:packages/cli-core/cliutil/` performed.
  - [ ] Decision (land or skip) recorded in §15 Deviations.
  - [ ] If landed: `cliutiltest` tests green, coverage ≥ 90%.
  - [ ] Convention doc § "Per-package status" reflects the decision.
- [ ] **Phase F — template adoption**:
  - [ ] `connect_handler_test.go` uses `connectxtest.StartTestServer`.
  - [ ] `attachments_handler_test.go` is reviewed; it remains REST-local unless a true shared-package seam exists.
  - [ ] `path:cli/domains/notes/handlers_test.go` uses `cliapptest.NewTestRunContext`.
  - [ ] `path:docs/internal/SEAMS.md` has new rows for the consumed test seams.
  - [ ] `path:docs/internal/REPLACING-NOTES.md` points to canonical fakes.
  - [ ] Greenfield audit: `git diff templates/ | grep -E '(// Deprecated:|// legacy|// compat)'` prints nothing.
  - [ ] All template tests green.
- [ ] **Phase G — end-to-end**:
  - [ ] Throwaway scenario `shared-pkg-testkit-smoke` generated.
  - [ ] api / cli / ui gates all green.
  - [ ] Cleanup performed; zero residue verified.
  - [ ] `git status` shows only the intentional plan-side changes.

---

## 12. Risks + Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Phase D type alias `TestRunContextOptions = cliapp.TestRunContextOptions` is misread as a §2.5-prohibited "type Old = New" alias | medium | §9.3 explicitly distinguishes: type aliases for re-export between packages are NOT compat shims. Compat shims are template-side temporary bridges; this is a permanent re-export. The §2.5 prohibition is about template diffs, not cli-core's permanent surface. |
| Tests of `cliapptest` forwards drift away from the originals over time | low | The forwards are one-line passthroughs; drift requires actively editing them. Equivalence tests catch behavioral drift if the underlying `cliapp.NewTestRunContext` changes shape. |
| `connectxtest.StartTestServer`'s mux dependency couples it to a specific router | medium | Today's `connectx.RegisterServices` already takes a `*mux.Router` (or similar). The test helper inherits that coupling intentionally — switching the helper to a router-agnostic shape is a `connectx` change, not a `connectxtest` change. If `connectx` ever generalizes, `connectxtest` follows. |
| Phase E ships `cliutiltest` without a clear consumer, becoming dead code | low | Decision criterion in §E.2 requires a justified target before landing. If none, skip. The plan documents the decision either way. |
| Convention doc gets stale as more packages add test companions | medium | The "Per-package status" table is the maintenance hotspot. Each future plan that adds a `<pkg>test` updates that table. Plan §A.1.6 names this convention. |
| Discovery: future agents don't know the convention exists | medium | README links + REFERENCE_PATTERN_FITNESS reference + REPLACING-NOTES reference. Three independent discovery surfaces. If still missed, follow-up adds a code-comment hint in `cliapp/runcontext.go` ("see also cliapptest.NewTestRunContext for the canonical home going forward") — defer until first miss surfaces. |
| Generic `SliceRepo[T]` work bleeds into this plan | medium | §1.2 and §5.2 explicitly out-of-scope it. The agent should not author a generic repository fake here; that work happens in iter-3 of the fitness program. |
| Phase F migration breaks an existing test assertion | medium | §10.2 explicitly requires existing template tests pass post-migration. The migration is structural (helper internal change) — test assertions on `out`, `err.Error()`, request-shape stay verbatim. If an assertion fails, fix the helper, not the assertion. |
| `connectxtest.NewLogger` two-return-value shape diverges from existing template usage | low | Most existing template helpers do `log.New(&bytes.Buffer{}, "", 0)` and lose the buffer reference. The new shape gives the buffer back, which is strictly more flexible. Migration is mechanical: replace the inline two-line setup with one call. |
| TS subpath story leaks into this plan | low | §5.2 out-of-scope. Convention doc mentions the TS equivalent in one paragraph and explicitly says "not landed in this plan." |

---

## 13. Non-goals / Prohibited patterns

- **No generic `Repository[T]` interface in api-core.** Iter-3 work; this plan does not commit api-core to a storage opinion.
- **No `*kit` / `*util` suffixes on new packages.** Convention is `<pkg>test`; nothing else.
- **No umbrella `package:api-core/testing/` directory.** Per-package siblings only.
- **No `// Deprecated:` markers.** Migration is taught via documentation, not in-code warnings.
- **No removal of existing exports** in cli-core or api-core. Strictly additive.
- **No new test-companion package without a tested consumer.** If the surface has no fake-able shape worth landing (Phase E), skip and document.
- **No template-side compatibility shims.** Greenfield rule §2.3.
- **No AST guardrails enforcing the convention** (yet). Convention is documentation-enforced; AST guardrail is a follow-up if drift surfaces.
- **No migration of non-react-vite scenarios.** Per-scenario greenfield rule.
- **No TS package work** in this plan.
- **No new architecture-decision-record file.** The convention doc IS the record.

---

## 14. Definition of Done

The plan is done when **all** of the following are true:

1. **Phases A–G checklist (§11) is fully ticked.**
2. **All test gates green** on api-core, cli-core, template api/cli, and the throwaway end-to-end scenario.
3. **Convention doc is discoverable**: linked from `path:packages/api-core/README.md`, `path:packages/cli-core/README.md`, and referenced in `path:docs/agent-system/REFERENCE_PATTERN_FITNESS.md`.
4. **The canonical fakes exist**:
   - `path:packages/api-core/databasetest/FakeExecer` — implements `database.SchemaExecer`, supports synchronized query snapshots, all-call error injection, and ordered failure injection.
   - `path:packages/api-core/connectxtest/StartTestServer`, `NewLogger` — replace template's hand-rolled Connect server/logger plumbing where that shared-package seam exists.
   - `path:packages/cli-core/cliapptest/NewTestRunContext`, `NewTestRunContextFromArgs`, `TestRunContextOptions` — forwarders.
   - `path:packages/cli-core/cliutiltest/...` — landed if Phase E found a justified target; skipped + documented otherwise.
5. **Greenfield self-audit passes**:
   - `git diff` of template tree contains zero `// Deprecated:`, `// legacy`, `// compat`, `// backwards-compat`, `type Old…= New…` markers.
   - api-core / cli-core diffs contain only additions; zero exported-symbol removals or signature changes.
6. **Single-source-of-truth check**:
   - `git grep "log.New(&bytes.Buffer{}" templates/scenarios/react-vite/api/handlers/notes/` returns zero matches in non-test scaffolding (the template's connect handler tests are migrated to `connectxtest.NewLogger`).
   - `git grep "httptest.NewServer" templates/scenarios/react-vite/api/handlers/notes/` returns zero matches.
7. **§15 "Deviations during execution" is populated** with `(none)` if no deviations; otherwise lists each deviation with reason.
8. **The plan is filed** as canonical: this file at `path:docs/plans/shared-package-test-companion-convention-plan.md` is left in place (not deleted post-completion). Future iterations of the fitness program reference it.

If any item above is false, the plan is not done.

---

## 15. Deviations during execution

Each entry: **file or step** — **what changed** — **why**.

(Populated during execution. Agent: do not leave this section empty at completion; if there were no deviations, write `(none)` here so future readers don't wonder.)

---

## 16. Handoff

After this plan completes, the next agent picking up shared-package work:

1. Reads `path:docs/agent-system/SHARED_PACKAGE_TESTING.md` to learn the convention.
2. Looks at `path:packages/api-core/databasetest/` and `path:packages/api-core/connectxtest/` as worked examples.
3. Applies the same pattern to whichever shared package is next on their list. Likely candidates:
   - **`path:packages/api-core/secrets`, `path:packages/api-core/health`, etc.** — review surfaces; some may have nothing fake-worthy (and that's fine; the convention doc records it).
   - **`path:packages/cli-core/cliutil`** — if Phase E skipped it, revisit when a concrete consumer surfaces.
   - **TS shared packages** — `@vrooli/api-base/testing` subpath. Separate plan; references this one as the conceptual precedent.
4. After the convention is well-established (≥3 shared packages have companions), considers adding an AST guardrail that flags hand-rolled fakes of shared-package interfaces in non-`<pkg>test` locations. That guardrail is opt-in and lives in a follow-up plan.

For the template-fitness iteration program specifically:

- The active iteration plan at `~/.claude/plans/thanks-for-the-postgres-wise-hearth.md` is **not** modified by this plan. After this plan lands, iteration 3 of the fitness program runs against the new convention.
- Iteration 3's hypothesis (per the iter-2 RESULTS' "Open questions" section) is around per-domain mocks and test scaffolding. The substrate this convention establishes (canonical fakes for shared-package interfaces) is the *foundation* on which iteration 3 builds — iter-3 layers domain-side mocks (`SliceRepo[T]`-style generic primitives) on top of the pattern this plan codifies.

---

## Appendix A — Why the convention is `<pkg>test` specifically

A short rationale repeating §9.1 in case it's missed.

| Candidate | Pros | Cons | Decision |
|-----------|------|------|----------|
| `<pkg>test` sibling (Go stdlib) | Clean import, single mental model, top-level discoverable | Multi-package modules accumulate top-level dirs | **chosen** |
| `<pkg>/test/` subpackage (Kubernetes) | Co-located with `<pkg>` | Import path reads `package:database/test` — awkward; dir name "test" collides with build tooling expectations | rejected |
| Umbrella `package:api-core/testing/` | Single dir to grep for "where are the fakes" | Two-import pattern (`package:api-core/database` + `package:api-core/testing/database`) doubles cognitive load | rejected |
| `*kit`/`*util` suffix (`package:api-core/databasekit`, `package:cli-core/repokit`) | Distinct from `<pkg>` | No precedent in Go stdlib; conflates "kit" (utility collection) with "fake provider" | rejected |
| `<pkg>fake` suffix (gomock pattern) | Explicit semantics ("this is a fake") | Misleading when the package contains non-fake helpers (server harnesses, assertion libs) | rejected |
| `<pkg>/mocks/` subpackage (gomock-generated) | Familiar to Mock-style users | Implies tools-of-the-trade like gomock; we don't generate; conflates manual + generated | rejected |

The Go stdlib convention wins on every axis except per-module top-level dir count, which is a non-issue at our scale.

---

## Appendix B — File touch summary

For Phase B+C (api-core), the files added:

| Path | Change |
|---|---|
| `path:packages/api-core/databasetest/doc.go` | New |
| `path:packages/api-core/databasetest/execer.go` | New |
| `path:packages/api-core/databasetest/execer_test.go` | New |
| `path:packages/api-core/connectxtest/doc.go` | New |
| `path:packages/api-core/connectxtest/server.go` | New |
| `path:packages/api-core/connectxtest/server_test.go` | New |
| `path:packages/api-core/connectxtest/logger.go` | New |
| `path:packages/api-core/connectxtest/logger_test.go` | New |
| `path:packages/api-core/README.md` | Edit: add Test companions section |

For Phase D+E (cli-core), the files added:

| Path | Change |
|---|---|
| `path:packages/cli-core/cliapptest/doc.go` | New |
| `path:packages/cli-core/cliapptest/runcontext.go` | New |
| `path:packages/cli-core/cliapptest/runcontext_test.go` | New |
| `path:packages/cli-core/cliutiltest/*` | New (conditional on E decision) |
| `path:packages/cli-core/README.md` | Edit: add Test companions section |

For Phase A (convention) + F (template), the files modified:

| Path | Change |
|---|---|
| `path:docs/agent-system/SHARED_PACKAGE_TESTING.md` | New |
| `path:docs/agent-system/REFERENCE_PATTERN_FITNESS.md` | Edit: drift-surface section |
| `path:templates/scenarios/react-vite/api/handlers/notes/connect_handler_test.go` | Edit: adopt connectxtest |
| `path:templates/scenarios/react-vite/api/handlers/notes/attachments_handler_test.go` | Edit: review only; adopt shared helpers only if a true shared-package seam exists |
| `path:templates/scenarios/react-vite/api/handlers/notes/adapter_test.go` | Edit: adopt where applicable |
| `path:templates/scenarios/react-vite/cli/domains/notes/handlers_test.go` | Edit: adopt cliapptest |
| `path:templates/scenarios/react-vite/docs/internal/SEAMS.md` | Edit: new rows |
| `path:templates/scenarios/react-vite/docs/internal/REPLACING-NOTES.md` | Edit: point to canonical fakes |
