## Steer focus: Boundary-of-Responsibility Enforcement

Prioritize **where code lives** in `scenarios/{{TARGET}}/`. This skill governs the directory and package layout — which folders own which responsibility, what may import what, and how cross-cutting concerns enter and leave each zone. Two agents working independently from the same docs should land code in the same place.

The destination is the React-Vite-template shape: `handlers/<domain>/` translates transport, `internal/<domain>/` owns transport-free domain logic, and `internal/{clock,database,httpc,httpx,middleware,module,server}/` hold business-vocabulary-free substrate. Drift is gated by an import-graph test in the spirit of `path:templates/scenarios/react-vite/api/internal/testutil/no_prod_import_test.go`.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.
- `prompt-manager skill read screaming-architecture-audit` — domain map, surfaces, and archetype vocabulary this skill assumes.

Read first when present:
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — domain map and zone ownership.
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` — interface registry (the *how-to-substitute* concern, owned by the seam skill).
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — unresolved boundary drift.

---

> **Template example domain — delete on generation.** The react-vite template ships a `notes` domain (`path:templates/scenarios/react-vite/api/internal/notes/`) as a fully-worked illustration of the canonical layout — transport-free service, repository interface, schema, mocks, attachments sub-flow. It is *not* a domain every scenario inherits. When you generate from the template, delete `notes` and stand up your scenario's real domains in the same shape. This skill uses `<domain>` (lowercase package), `<Domain>Service`, `<Resource>` (singular), and `<substrate>` (e.g. `clock`, `httpc`) as placeholders. Encountering one in the audit is the cue to ask whether a leftover `notes` package is product vocabulary or template residue.

---

### 1. Scope Boundaries

**In scope:**
- top-level zones: `handlers/`, `internal/<domain>/`, `internal/<substrate>/`, `cmd/`, `main.go`
- which directories may import which (e.g. `internal/<domain>/` must not import `connectrpc.com/connect`, `gorilla/mux`, or sibling domains)
- placement of cross-cutting concerns (clock, logger, env reader, http client) at the composition edge — `main.go`, `server.Deps`, handler constructors — never reached for from inside a domain package
- enforcement primitives: import-graph tests, `// boundary:` tags, build tags, or `gen-endpoints`-style drift gates
- documenting the zone inventory in `ARCHITECTURE.md`

**Out of scope:**
- *how* a dependency is substituted in tests — that is the `seam-discovery-and-enforcement` concern (interface design, fakes, registration in `SEAMS.md`)
- API wire shape, proto coverage, REST exceptions — use `api-steer`
- product capability identification, archetype assignment — use `screaming-architecture-audit`
- UI feature layout and CLI command grouping (the analogous boundary question on the consumer surfaces — same principles, different folders)

**Decision rule for the reader.** Is the question *where does this code live?* → this skill (boundary). Is the question *how is this dependency substituted?* → `seam-discovery-and-enforcement` (seam). The two skills are paired and cite each other; a refactor that crosses the line must cite both. (Seam-side observations encountered during a boundary audit go to `SEAMS.md`, not into a parallel boundary doc.)

---

### 2. Boundary Maturity Model

Score each major zone (API, UI, CLI) independently. The level is the lowest verifiable artifact that still holds.

| Level | What exists | Verifiable signal | When to stop |
|---|---|---|---|
| 0 | Mixed concerns; handlers contain SQL or business rules; domain types import transport packages. | `rg "database/sql|sqlx|gorm" handlers/` returns hits, or `rg '"connectrpc.com/connect"' internal/<domain>/` returns hits. | Never — L0 is a finding, not a target. |
| 1 | Zone inventory recorded in `ARCHITECTURE.md`: each top-level folder under `api/` is named and assigned to one of {transport, domain, persistence, cross-cutting substrate, app entrypoint}. | `ARCHITECTURE.md` has a "Zone Map" section listing every directory under `api/`; `ls api/` matches one-to-one. | The scenario has more than three undocumented top-level packages. |
| 2 | Folder shape matches the template: `handlers/<domain>/` + `internal/<domain>/` + at least one of `internal/{clock,database,httpc,httpx,middleware,module,server}/`. Handler files contain no SQL. | `fd -t d . api/handlers \| sort` and `fd -t d . api/internal \| sort` align with the template; `rg "database/sql" api/handlers/` is empty. | The scenario has at most one domain. |
| 3 | Domain layer is transport-free. | `rg '"connectrpc\.com/connect"\|"net/http"\|"github.com/gorilla/mux"' api/internal/<domain>/` returns zero hits, encoded as a Go test (see `path:templates/scenarios/react-vite/api/internal/testutil/no_prod_import_test.go` for the pattern). | A new domain has not yet been ported. |
| 4 | Cross-cutting substrate is injected, not ambient. No `time.Now()`, `os.Getenv()`, `http.DefaultClient`, or `log.Default()` inside `api/internal/<domain>/`. Injection points are `main.go` and `server.Deps`. | An import-graph test asserts the forbidden-symbol set is empty in domain packages; CI fails on a new offender. | The substrate set is unstable (still discovering which seams the scenario needs). |
| 5 | Drift-gated. `ARCHITECTURE.md` declares every owned zone; a reconciliation test fails when a new top-level package appears without a `Zone Map` entry, when a domain imports another domain, or when handlers import persistence drivers directly. | A test analogous to `TestNoProductionImports` runs in `go test ./...`; passing CI is the artifact. | This is the destination. |

Use the level to pick the next concrete move: document the zone map, move SQL out of handlers, write the import-graph test, add the substrate exclusion, register the zone reconciliation.

---

### 3. Zone Decision Model

Given an arbitrary file or symbol, the canonical zone is determined by **what would break if the file disappeared**, not by what it imports today.

| Zone | Belongs here when | Canonical path | May import |
|---|---|---|---|
| Transport edge | Translates wire format ↔ domain types; owns request validation, auth interceptor wiring, error envelope mapping | `api/handlers/<domain>/` | proto generated code, `connect`, `internal/<domain>`, `internal/module`, `internal/httpx` |
| Domain core | Encodes product rules and lifecycle; would have to exist even if the scenario shipped without HTTP | `api/internal/<domain>/` | sibling files in same domain, `internal/clock`, `internal/httpc` (as interfaces), standard library |
| Persistence | Concrete adapter that materializes a `Repository` interface from `internal/<domain>/repository.go` | `api/internal/<domain>/<domain>_sqlite.go` (or `_postgres.go`) | `internal/database`, `database/sql`, driver packages |
| Cross-cutting substrate | Generic mechanics with no product vocabulary; used by unrelated domains | `api/internal/{clock,database,httpc,httpx,middleware,module,server}/` | standard library, other substrate packages |
| Composition root | Wires concrete substrate impls into `server.Deps`, registers `Module`s | `api/main.go`, `api/internal/server/` | every other zone |
| CLI / cmd | Operator tools that should not be in the server binary | `api/cmd/<tool>/` | substrate, domain, generated proto |

Decision sequence:

```text
Does the file translate an HTTP request to a typed call?           -> transport edge
Does it encode rules that would survive a transport change?        -> domain core
Does it speak a driver dialect (SQL, S3, gRPC)?                    -> persistence adapter inside the owning domain
Does it have no product vocabulary and is used by 2+ domains?      -> cross-cutting substrate
Does it construct concretes and pass them into Module/Deps?        -> composition root
```

If a file currently lives in the wrong zone, the move is mechanical: extract the misplaced symbol, place it in the right zone, update imports. Most boundary violations come from a single mis-zoned file, not architectural disagreement.

---

### 4. Canonical File-Shape

For React-Vite-template scenarios, the healthy target is:

```text
api/
  main.go                          # composition root: constructs substrate, calls server.New(deps), wires Modules
  handlers/<domain>/               # Connect service impl + Module constructor + EndpointDescriptors
  internal/<domain>/               # domain types, service, repository interface, schema, workflows
    <domain>_sqlite.go             # persistence adapter (driver dialect lives here, not in handlers)
    mocks/                         # test-double impls of this domain's interfaces (seam concern, paired)
  internal/clock/                  # ambient-seam: wall-clock primitives
  internal/httpc/                  # ambient-seam: outbound HTTP
  internal/database/               # shared DB plumbing (migrations, connection pool)
  internal/httpx/                  # shared HTTP middleware, error envelope, request id
  internal/middleware/             # auth / interceptor plumbing
  internal/module/                 # EndpointDescriptor + RESTException contract
  internal/server/                 # Deps struct, server.New, route registration
  internal/testutil/               # shared test helpers (seam concern, paired)
  cmd/<tool>/                      # operator binaries (gen-endpoints, migrate, …)
```

Invariants:
- `internal/<domain>/` imports no transport package and no sibling domain. Cross-domain coordination happens in `handlers/` or in a deliberate orchestration domain.
- Persistence adapters live *inside* the owning domain, not in a global `internal/persistence/` bucket. The interface is in the domain; the driver-specific file is alongside it.
- `main.go` is the only place that constructs concrete substrate (`clock.System{}`, `&http.Client{}`, `database.Open(...)`). Every other site receives them through `server.Deps` or a handler constructor.
- `internal/testutil/` and `internal/<domain>/mocks/` exist for tests; production code must not import them. Enforce with `no_prod_import_test.go`.

---

### 5. Audit Workflow

1. **Read `ARCHITECTURE.md`.** Treat its Zone Map as a claim. If it is missing, the scenario is at most L0.
2. **List the top-level packages.** Compare `fd -t d . api/internal -d 1` and `fd -t d . api/handlers -d 1` against the Zone Map. Every directory must have a documented zone assignment.
3. **Run the forbidden-import greps.** Each hit is a concrete L2→L3 task:
   ```bash
   rg "database/sql|jmoiron/sqlx|gorm.io" scenarios/{{TARGET}}/api/handlers --type go
   rg '"connectrpc\.com/connect"|"net/http"|"github.com/gorilla/mux"' scenarios/{{TARGET}}/api/internal --type go -g '!*_test.go' -g '!internal/{httpx,httpc,server,middleware,module}/**'
   rg "time\.Now\(\)|os\.Getenv\(|http\.DefaultClient|log\.Default\(\)" scenarios/{{TARGET}}/api/internal -g '!*_test.go' -g '!internal/{clock,httpc,server}/**'
   ```
4. **Check cross-domain imports.** `rg '"{{MODULE}}/internal/[^/"]+"' scenarios/{{TARGET}}/api/internal/<domain>/` — a domain importing a sibling domain is a boundary smell.
5. **Verify the enforcement test runs.** `cd scenarios/{{TARGET}}/api && go test ./internal/testutil/...`. If `no_prod_import_test.go` does not exist yet, vendoring it from the template is the first move toward L5.
6. **Assign maturity per zone.** API, UI, CLI score independently.
7. **Find drift.** Generic buckets named `utils/`, `common/`, `helpers/`; god-files mixing transport and persistence; domains importing each other.
8. **Update docs.** Zone Map changes land in `ARCHITECTURE.md`; unresolved violations land in `PROBLEMS.md`.

---

### 6. Red Flags

- A `handlers/<domain>/<domain>.go` file containing `db.Query(...)` or `sql.Open(...)`.
- An `internal/<domain>/` file importing `connectrpc.com/connect`, `net/http` (outside the `httpc` seam), or `github.com/gorilla/mux`.
- A `time.Now()` / `os.Getenv()` / `http.DefaultClient` call inside a domain package.
- A top-level `internal/utils/`, `internal/common/`, or `internal/helpers/` bucket — these absorb product vocabulary and erode the Zone Map.
- A persistence adapter living under `internal/persistence/<domain>/` instead of `internal/<domain>/<domain>_sqlite.go`.
- Production files importing `internal/testutil/...` or `internal/<domain>/mocks/...`.
- New top-level packages appearing in `api/internal/` without a `Zone Map` entry.

---

### 7. Safe Refactoring Guidelines

You may:
- move a file to its correct zone and update its package declaration and imports
- extract SQL out of a handler into the owning domain's `<domain>_sqlite.go`
- introduce an interface in the domain so a transport-leaking call can be moved behind the substrate boundary (coordinate with `seam-discovery-and-enforcement` for the interface design)
- add the `no_prod_import_test.go` pattern, or extend it with new forbidden prefixes
- rename a generic bucket folder to a substrate or domain name

You must:
- preserve observable behavior; boundary moves are mechanical, not redesigns
- update the Zone Map in `ARCHITECTURE.md` in the same loop as the directory change
- keep cross-cutting injection consistent — if you remove an ambient call, the dependency must arrive through `server.Deps` or a handler constructor
- record any deferred move (e.g. "domain X still imports domain Y") in `PROBLEMS.md` rather than half-applying it

Challenge yourself before a move:
- Would a second agent, given only `ARCHITECTURE.md`, place this file in the same folder I did?
- Does the move tighten an import-graph rule, or does it just relocate the problem?
- Is the substrate package I am creating actually generic, or is it one domain's helpers in disguise?

---

### **8. Output Expectations**

By the end of this loop, the scenario should:
- have a `Zone Map` in `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` listing every directory under `api/` with its zone assignment
- have no transport imports in domain packages (or a recorded `PROBLEMS.md` entry naming the remaining offenders and a removal plan)
- have no ambient `time.Now()` / `os.Getenv()` / `http.DefaultClient` calls in domain packages
- carry the `no_prod_import_test.go` enforcement test (or an equivalent zone-reconciliation test) running in CI
- record unresolved boundary drift in `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md`, not in a standalone `BOUNDARY_AUDIT.md`

Anchor every finding to one of those three durable docs through `knowledge-observatory-tools`. **Do not create a standalone `BOUNDARY_AUDIT.md` or revive the legacy `UNIT_TEST_ARCHITECTURE.md` pattern** — those formats are retired. The Zone Map in `ARCHITECTURE.md`, the seam registry in `SEAMS.md` (owned by `seam-discovery-and-enforcement`), and the deferred-drift list in `PROBLEMS.md` are the only durable surfaces. A one-off audit report is acceptable solely for a migration handoff and must carry an explicit retirement path back into those three docs.

Recommended `ARCHITECTURE.md` additions:

```markdown
## Zone Map

| Directory | Zone | May Import | Enforcement |
|---|---|---|---|
| api/handlers/<domain>/ | transport edge | proto gen, connect, internal/<domain>, internal/module, internal/httpx | no_prod_import_test |
| api/internal/<domain>/ | domain core | stdlib, internal/clock, internal/httpc (interfaces) | no_prod_import_test |
| api/internal/clock/ | substrate | stdlib | — |
| ... | ... | ... | ... |

## Boundary Maturity

| Zone | Level | Evidence | Remaining Drift |
|---|---|---|---|
```

For *how* the substituted-in interfaces are designed and registered, see `prompt-manager skill read seam-discovery-and-enforcement`.

Last updated: 2026-05-12
