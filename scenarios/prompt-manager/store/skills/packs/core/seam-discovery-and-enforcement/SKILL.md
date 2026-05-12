## Steer focus: Seam Discovery & Enforcement

Prioritize **how variation is substituted** in `scenarios/{{TARGET}}/`. A *seam* is a named interface declared at the point of use whose production implementation is wired once and whose test double lives in a known catalog. This skill governs the seam itself — its shape, its fakes, and the registry that lets a future agent find it — not the directory layout that holds it.

The destination is the React-Vite-template shape: every dependency a domain has on the outside world (time, network, env, log, persistence) is a Go interface in the domain's own package; production wires the concrete in `main.go` / `server.Deps`; tests substitute a fake from `internal/<domain>/mocks/` or `internal/testutil/mocks/`. Drift is gated by a seam-registry test that reconciles `// seam:`-tagged interfaces with `SEAMS.md`.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.
- `prompt-manager skill read boundary-of-responsibility-enforcement` — the directory layout this skill assumes seams live inside.

Read first when present:
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` — the seam registry this skill is the source of truth for.
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — domain map; seams are owned by domains.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — unresolved seam drift.

---

> **Template example domain — delete on generation.** The react-vite template ships canonical seam examples: `path:templates/scenarios/react-vite/api/internal/clock/clock.go` (ambient time), `path:templates/scenarios/react-vite/api/internal/httpc/doer.go` (outbound HTTP), and `path:templates/scenarios/react-vite/api/internal/notes/repository.go` (domain persistence). The `notes` domain is starter scaffolding to study, not boilerplate to keep — when a scenario is generated, replace it with real domains and apply the same seam shape. This skill uses `<domain>`, `<Domain>Service`, `<Seam>` (interface name), and `<Fake><Seam>` (test double) as placeholders. Substituting them is the prompt to ask whether a leftover `notes` seam is your scenario's real shape or template residue.

---

### 1. Scope Boundaries

**In scope:**
- the interface itself: name, method surface (narrow, single-purpose), where it is declared
- the production implementation: how it is constructed once and threaded through `server.Deps` / handler constructors
- the test double: where the fake lives, what shape it has, whether it satisfies the same compile-time check as the production impl (`var _ Seam = (*Fake)(nil)`)
- the four canonical ambient seams every scenario needs: clock, outbound HTTP, env reader, logger
- domain-level seams: `Repository`, integration clients, blob stores, scheduler clients
- the registry: keeping `SEAMS.md` and the code in lockstep, and gating drift with a registry test

**Out of scope:**
- *where* the interface file lives in the directory tree — that is the `boundary-of-responsibility-enforcement` concern (zone map, what may import what)
- proto-defined RPC contracts and Connect handler wiring — use `api-steer`
- product capability identification — use `screaming-architecture-audit`
- the contents of the fake's behavior beyond "it satisfies the seam" — domain-specific test fixtures live in `internal/testutil/fixtures/` and are tuned per test

**Decision rule for the reader.** Is the question *where does this code live?* → `boundary-of-responsibility-enforcement` (boundary). Is the question *how is this dependency substituted?* → this skill (seam). The two skills are paired and cite each other; a refactor that crosses the line must cite both. (Boundary-side observations encountered during a seam audit go to `ARCHITECTURE.md`, not into a parallel seam doc.)

---

### 2. Seam Maturity Model

Score each seam independently. A scenario has many seams; some may be L5 while others are L1.

| Level | What exists | Verifiable signal | When to stop |
|---|---|---|---|
| 0 | Side effects inline: `time.Now()`, `http.DefaultClient.Do(...)`, `os.Getenv(...)`, raw SQL in handlers; tests skip or mutate globals. | `rg "time\.Now\(\)\|http\.DefaultClient\|os\.Getenv\(" scenarios/{{TARGET}}/api/internal/<domain>/` returns hits. | Never — L0 is a finding, not a target. |
| 1 | `SEAMS.md` exists with a flat list naming each seam, its production file path, and its fake path. | The file is present and lists at least the four ambient seams (clock, http client, env, log). | Fewer than three seams are documented. |
| 2 | Each seam is a Go interface declared in the package that consumes it; a `// seam:` comment tags the declaration; a compile-time check (`var _ <Seam> = (*<Impl>)(nil)`) anchors production and fake. | `rg "// seam:" --type go` lists every interface declared as a seam; each location has a matching `var _ <Seam>` assertion in the production file and the fake file. | A seam is informal (a function value, an exported global) rather than an interface. |
| 3 | The four ambient seams are replaced: no domain package contains `time.Now()`, `os.Getenv()`, `http.DefaultClient`, or `log.Default()`. Each ambient is injected via `server.Deps`. | `rg "time\.Now\(\)\|os\.Getenv\(\|http\.DefaultClient\|log\.Default\(\)" scenarios/{{TARGET}}/api/internal -g '!*_test.go' -g '!internal/{clock,httpc,server}/**'` returns zero hits. | Not all four ambients have been migrated. |
| 4 | Every seam has at least one production impl AND one test double, both with compile-time `var _` assertions. `SEAMS.md` lists every seam with: declaration site, production impl file, test-double file, why it exists. | `rg "// seam:" --type go \| wc -l` equals the row count in `SEAMS.md`'s seam table. | The registry has not been reconciled this cycle. |
| 5 | Drift-gated. A registry test (analogous to `path:templates/scenarios/react-vite/api/internal/testutil/no_prod_import_test.go`) walks the AST, finds every `// seam:`-tagged interface, and asserts each appears in `SEAMS.md` (and vice versa). New ambient calls in domain code fail CI. | `go test ./internal/testutil/... -run TestSeamRegistry` passes; CI breaks when a new `time.Now()` lands in a domain file. | This is the destination. |

Use the level to pick the next concrete move: name the seam, write the interface, ship the fake, migrate the ambient, register it, gate it.

---

### 3. Seam Archetype Decision Model

| Archetype | Use when | Canonical interface shape | Production impl | Test double |
|---|---|---|---|---|
| Ambient substrate | A platform primitive (time, network, env, log) the domain depends on but does not own | Single-method interface in `internal/<substrate>/` (e.g. `clock.Clock { Now() time.Time }`) | Struct in the same package (`clock.System{}`) | `internal/testutil/mocks/<substrate>.go` |
| Outbound integration | The domain calls an external HTTP/gRPC service | Narrow interface in `internal/<domain>/` exposing only the operations the domain uses | Adapter struct that wraps an `httpc.Doer` | `internal/<domain>/mocks/<integration>.go` |
| Persistence | The domain reads or writes durable state | `Repository` interface in `internal/<domain>/repository.go` | `<domain>_sqlite.go` or `<domain>_postgres.go` in the same package | `internal/<domain>/mocks/repository.go` |
| Process-out side effect | The domain enqueues background work, emits events, or writes blobs | Narrow interface (`EventPublisher`, `BlobStore`) in `internal/<domain>/` | Concrete adapter in `internal/<substrate>/` | Per-domain mock; recorded calls in tests |
| Policy / decision | A decision the scenario wants to vary per environment or per tenant | Pure function-shaped interface (`Authorizer`, `RateLimiter`) | Production rule struct | Programmable fake returning canned decisions |
| Clock-shaped derivative | Tickers, sleeps, jitter — anything `time`-derived beyond `Now()` | Extend the `clock.Clock` interface rather than adding a parallel seam | Extend `clock.System` | Extend `mocks.FakeClock` |

Decision rule:

```text
Does the call cross a process boundary (network, disk, syscall)?
  YES -> it needs a seam.
Does the call read a global (env, default client, default logger, wall clock)?
  YES -> it needs a seam.
Does the dependency vary per environment or per tenant?
  YES -> it needs a seam.
Is the dependency a pure stdlib data transform?
  NO  -> no seam.
Does an existing seam already cover this concern?
  YES -> extend it (add a method) rather than introduce a parallel one.
```

The interface is declared **in the package that consumes it**, not in the package that implements it. The consumer owns the seam; the producer satisfies it. This is the Go idiom and the precondition for narrow surfaces.

---

### 4. Canonical Seam Shape

A healthy seam in the React-Vite template looks like:

```go
// internal/<domain>/repository.go
package <domain>

import "context"

// seam: Repository persists <Resource> rows. Production wires
// <domain>Sqlite from <domain>_sqlite.go; tests wire FakeRepository
// from mocks/repository.go.
type Repository interface {
    Create(ctx context.Context, r <Resource>) (<Resource>, error)
    Get(ctx context.Context, id string) (<Resource>, error)
}
```

```go
// internal/<domain>/<domain>_sqlite.go
package <domain>

type SqliteRepository struct{ db *sql.DB }

func (r *SqliteRepository) Create(...) { ... }
func (r *SqliteRepository) Get(...)    { ... }

var _ Repository = (*SqliteRepository)(nil)
```

```go
// internal/<domain>/mocks/repository.go
package mocks

type FakeRepository struct{ ... }

func (f *FakeRepository) Create(...) { ... }
func (f *FakeRepository) Get(...)    { ... }

var _ <domain>.Repository = (*FakeRepository)(nil)
```

Invariants:
- The interface is declared in the consumer package, not in a separate `interfaces/` bucket.
- Both production and fake carry the `var _ <Seam> = (*Impl)(nil)` compile-time check. Renaming a method on the interface fails the build everywhere it must.
- The fake's package name is `mocks`, lacks the `_test.go` suffix (so sibling `_test.go` files in other packages can import it), and is exempt from `no_prod_import_test.go` via the `mocks/` directory rule.
- The interface surface is narrow — only the methods the consumer actually calls. Resist exposing the full `*sql.DB` surface through a `Repository` interface.
- The four ambient seams (clock, httpc, env, log) live in `internal/<substrate>/`. Domain seams live in `internal/<domain>/`.

---

### 5. The Four Ambient Seams

Every scenario inherits these from the template. Migrating L0→L3 is mostly the work of replacing the ambient call with the injected seam.

| Seam | Interface | Production | Fake |
|---|---|---|---|
| Wall clock | `clock.Clock { Now() time.Time }` | `clock.System{}` in `internal/clock/clock.go` | `mocks.FakeClock` in `internal/testutil/mocks/clock.go` |
| Outbound HTTP | `httpc.Doer { Do(*http.Request) (*http.Response, error) }` | `*http.Client` (satisfies via assertion) | `mocks.FakeDoer` in `internal/testutil/mocks/doer.go` |
| Env reader | `envx.Reader { Get(key string) string }` | `envx.OS{}` | `mocks.FakeEnv` (programmable map) |
| Structured logger | `logx.Logger` (project-specific surface) | `slog.Logger` adapter | `mocks.FakeLogger` (records calls) |

The template ships `clock` and `httpc` as concrete examples and `httpx` middleware as a consumer of both. `envx` and `logx` follow the same shape; if the template has not landed them yet, framing them as the L3 destination for new scenarios is the right move.

---

### 6. Audit Workflow

1. **Read `SEAMS.md`.** Treat each row as a claim: does the interface exist at the listed path? Does the fake satisfy it?
2. **Inventory every interface.** `rg "^type \w+ interface" scenarios/{{TARGET}}/api/internal --type go` — each one is a seam candidate. Tag with `// seam:` or remove if it is internal collaboration.
3. **Check ambient leaks.**
   ```bash
   rg "time\.Now\(\)" scenarios/{{TARGET}}/api/internal -g '!*_test.go' -g '!internal/clock/**'
   rg "http\.DefaultClient\|http\.Get\(\|http\.Post\(" scenarios/{{TARGET}}/api/internal -g '!*_test.go' -g '!internal/httpc/**'
   rg "os\.Getenv\(" scenarios/{{TARGET}}/api/internal -g '!*_test.go' -g '!main\.go'
   rg "log\.Default\(\)\|log\.Print" scenarios/{{TARGET}}/api/internal -g '!*_test.go'
   ```
   Each hit is a concrete L0→L3 task.
4. **Verify compile-time assertions.** `rg "var _ \w+\.\w+ = " --type go` — every seam should have one in both the production and the fake file.
5. **Verify fakes exist.** For each `// seam:` interface, `fd "<seamname>.go" internal/<domain>/mocks internal/testutil/mocks` should hit.
6. **Reconcile with `SEAMS.md`.** Every `// seam:` interface appears in the registry; every registry row has a `// seam:` interface.
7. **Run the enforcement tests.** `go test ./internal/testutil/...` — `no_prod_import_test.go` for production-side cleanliness, plus the seam-registry test if it exists. Where it does not yet, vendor the pattern from the template.
8. **Update `SEAMS.md`.** Land discovered seams; record unresolved drift in `PROBLEMS.md`.

---

### 7. Red Flags

- A test that calls `time.Sleep` to wait for a clock-driven branch (the seam is missing or unused).
- A test that sets an env var to drive production behavior (env reader is not a seam).
- A `*http.Client` constructed inside a domain function.
- An interface in `internal/<domain>/` whose method set mirrors `*sql.DB` verbatim — the seam is leaking the implementation.
- Two seams with overlapping surfaces (e.g. `Clock` and a separate `TimeProvider`) — collapse them.
- A fake in `internal/testutil/mocks/` that does not carry a `var _ <Seam> = ...` assertion.
- `SEAMS.md` rows that point to deleted files, or `// seam:` interfaces missing from `SEAMS.md`.
- A "test mode" boolean threaded through production code in lieu of a seam.

---

### 8. Safe Refactoring Guidelines

You may:
- introduce a new interface to replace an ambient call (`time.Now()` → `clock.Clock.Now()`), wire it through `server.Deps`, and ship a fake
- narrow an existing seam by removing an unused method
- extend an existing seam (add `Sleep` to `clock.Clock`) rather than introduce a parallel one
- move a fake from `internal/testutil/mocks/` to `internal/<domain>/mocks/` when it serves only one domain (coordinate with `boundary-of-responsibility-enforcement` for the zone change)
- add the seam-registry test, or extend it with new tag patterns

You must:
- preserve observable behavior; a seam introduction is a refactor, not a feature change
- update both `SEAMS.md` and the `// seam:` tag in the same loop
- ship the production impl and the fake together — a seam without a fake is L2 at best
- keep compile-time `var _ <Seam>` assertions on every impl
- record any seam that *should* exist but requires broader redesign in `PROBLEMS.md`

Challenge yourself before a move:
- Is this interface narrow enough that the fake is trivial to write?
- Would a second agent, reading `SEAMS.md` alone, find this seam and its fake?
- Is the seam declared in the consumer's package, or did I put it in the implementer's?
- Does the registry test fail if I delete the `// seam:` tag without updating `SEAMS.md`?

---

### 9. Output Expectations

By the end of this loop, the scenario should:
- have a `Seam Registry` table in `scenarios/{{TARGET}}/docs/internal/SEAMS.md` listing every seam with its declaration site, production impl, fake, and reason for existing
- have all four ambient seams (clock, httpc, env, log) replaced in domain code, or a `PROBLEMS.md` entry naming the remaining offenders
- carry compile-time `var _ <Seam>` assertions on every production impl and every fake
- carry a seam-registry test that reconciles `// seam:` tags with `SEAMS.md` (or a `PROBLEMS.md` entry pointing at it as the next step toward L5)
- record unresolved seam drift in `PROBLEMS.md`, not in a standalone `SEAM_AUDIT.md`

Anchor every finding to those durable docs through `knowledge-observatory-tools`. **Do not create a standalone `SEAM_AUDIT.md` or revive the legacy `UNIT_TEST_ARCHITECTURE.md` pattern** — those formats are retired. The Seam Registry in `SEAMS.md`, the Zone Map in `ARCHITECTURE.md` (owned by `boundary-of-responsibility-enforcement`), and the deferred-drift list in `PROBLEMS.md` are the only durable surfaces. A one-off audit report is acceptable solely for a migration handoff and must carry an explicit retirement path back into those three docs.

Recommended `SEAMS.md` additions:

```markdown
## Seam Registry

| Seam | Declaration | Production Impl | Test Double | Why it exists |
|---|---|---|---|---|
| clock.Clock | internal/clock/clock.go | clock.System | testutil/mocks/clock.go (FakeClock) | Wall-clock primitives for deterministic tests |
| httpc.Doer | internal/httpc/doer.go | *http.Client | testutil/mocks/doer.go (FakeDoer) | Outbound HTTP substitution |
| <domain>.Repository | internal/<domain>/repository.go | <domain>.SqliteRepository | <domain>/mocks/repository.go | Persistence substitution |
| ... | ... | ... | ... | ... |

## Seam Maturity

| Seam | Level | Evidence | Remaining Drift |
|---|---|---|---|
```

For *where the file lives* and the import-graph rules that protect it, see `prompt-manager skill read boundary-of-responsibility-enforcement`.

Last updated: 2026-05-12
