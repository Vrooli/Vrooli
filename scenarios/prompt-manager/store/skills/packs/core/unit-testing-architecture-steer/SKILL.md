## Steer focus: Unit Testing Architecture

Prioritize **the test infrastructure that makes unit tests trustworthy, fast, and easy to add** in `scenarios/{{TARGET}}/`. Treat test architecture as a precondition for test content: co-located test files, a centralized `testutil` package, injectable seams (clock, HTTP, env, logger), and drift-gates that fail when production code leaks untestable dependencies.

This skill owns *what test infrastructure must exist*. The `test` skill owns *what behaviors are asserted given that infrastructure*. Architecture maturity ≥ L3 is a precondition for `test` running productively — without injectable seams, behavior tests degrade into integration smoke or brittle mocks-of-concrete-types.

> **Programmatic signal:** this is a *finding-specific remediation* skill. The test-architecture findings it remediates (`TEST_HELPER_FROM_PRODUCTION`, `TEST_UTIL_MISSING`, `TEST_NOT_COLOCATED`, `MISSING_INJECTABLE_SEAM`, …) are produced by the **unit-health** scenario — the provider behind Test Genie's `unit` phase. Discover the current test-architecture maturity and the exact drift items with:
>
> ```bash
> unit-health validate scenario {{TARGET}}
> ```
>
> Work the architecture-category findings it reports; the ladder below is the diagnostic vocabulary for *why* each finding matters, not a competing scorer.

> **Policy projection contract:** Unit Health also emits policy/profile drift
> findings for template-derived unit infrastructure. When you see
> `UNIT_POLICY_PROJECTION_DRIFT`, align native config with the resolved policy
> (`vite.config.ts`, `package.json` scripts/dependencies, `src/test-utils`, Go
> `internal/testutil/no_prod_import_test.go`, etc.). When you see
> `UNIT_POLICY_PROFILE_INVALID`, `UNIT_POLICY_WEAKENED`,
> `UNIT_REQUIRED_ROLE_MISSING`, or `UNIT_SURFACE_UNGOVERNED`, repair the
> `.vrooli/testing.json` `unit.policy_profile` or the missing discovered
> surface. Do not reimplement these checks in Test Genie.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.

Cross-link skills:
- `prompt-manager skill read seam-discovery-and-enforcement` — when production code lacks the interfaces the testutil mocks should implement.
- `prompt-manager skill read boundary-of-responsibility-enforcement` — when test code and production code blur (e.g., test helpers imported from non-test packages).
- `prompt-manager skill read test` — successor skill; runs after L≥3 is reached.

Read first when present:
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — the "Testing infrastructure" section is this skill's primary write target.
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` — seam registry; test substitutions must appear here.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — testing gaps and deferred refactors.

---

> **Template example domain — delete on generation.** The react-vite template ships a `notes` domain (`path:templates/scenarios/react-vite/api/internal/notes/`) as a worked example of every test surface — domain mocks, sqlite-backed repository tests, service tests, error-mapping tests. It is starter scaffolding, not a domain every scenario inherits; delete it when generating from the template. Examples below use `<domain>` placeholders. If a `notes` folder lingers in your scenario, that is template residue, not real product vocabulary.

---

### 1. Scope Boundaries

**In scope**
- test file co-location and naming conventions
- centralized testutil package layout (`fixtures/`, `mocks/`, `db/`, `assertx/`, `httpx/`)
- production-code seams that make domain packages testable: clock, HTTP client, env reader, injected logger
- testcontainers / sqlite harness wiring for repository tests
- mock organization (per-domain `mocks/` plus shared `testutil/mocks/`)
- drift-gates: `no_prod_import_test.go`, registry tests asserting every domain owns a test file
- documenting testing infrastructure in `ARCHITECTURE.md`, recording seams in `SEAMS.md`, recording gaps in `PROBLEMS.md`

**Out of scope**
- individual test quality, assertion strength, coverage gaps → `test` skill
- E2E / integration test workflows → `e2e-testing` skill
- proto/Connect contract testing → `api-steer` / `interoperability-steer`
- formal temporal workflow conformance → `temporal-flow-audit`
- performance testing setup

---

### 2. File Organization

Tests live next to the code they test. Test helpers live in a single `testutil` tree per surface.

| Surface | Source file | Test file | Helper root |
|---|---|---|---|
| Go API | `internal/<domain>/service.go` | `internal/<domain>/service_test.go` | `internal/testutil/` |
| TS UI  | `src/features/<domain>/Card.tsx` | `src/features/<domain>/Card.test.tsx` | `src/test-utils/` |
| Python | `pkg/<domain>/service.py` | `pkg/<domain>/test_service.py` (or `tests/test_service.py` if `conftest.py` is rooted there) | `tests/conftest.py` |

Python convention: `pytest` + `conftest.py` per directory. Scenarios are Go-API + TS-UI by default; treat Python as a secondary case.

Decision table for where a new test artifact goes:

| Artifact | Lives in | Anti-location |
|---|---|---|
| Unit test for `foo.go` | `foo_test.go` (same dir) | `tests/` sibling, `__tests__/` |
| Domain-local mock used only by one domain | `internal/<domain>/mocks/` | shared `testutil/mocks/` |
| Mock for a shared seam (clock, doer, pinger) | `internal/testutil/mocks/` (see `path:templates/scenarios/react-vite/api/internal/testutil/mocks/`) | inline in each test file |
| Cross-domain fixture factory | `internal/testutil/fixtures/` | per-domain copy-paste |
| DB harness | `internal/testutil/db/` (see `path:templates/scenarios/react-vite/api/internal/testutil/db/sqlite.go`) | repeated container boilerplate |
| Custom assertions | `internal/testutil/assertx/` | local `helpers.go` per package |
| HTTP test client | `internal/testutil/httpx/` | inline `httptest.NewServer` calls |

---

### 3. Test Isolation

Every test must pass alone, with others, in any order, on any machine, in CI. Isolation concerns and their verifiable patterns:

| Concern | Pattern | Anti-pattern (greppable) |
|---|---|---|
| Shared state | Fresh fixtures per test | `var .* = .*` at package scope mutated by tests |
| File system | `t.TempDir()` | `os.MkdirTemp` with manual cleanup, fixed `/tmp/...` paths |
| Time | `clock.Clock` injection (`path:templates/scenarios/react-vite/api/internal/clock/clock.go`) | `time.Now()` in `internal/<domain>/` |
| Randomness | Seeded RNG injection | bare `rand.Int()`, `crypto/rand` in domain code |
| Environment | Injected env reader | `os.Getenv` in `internal/<domain>/` |
| Database | testcontainers or `testutil/db` sqlite harness | shared singleton DB, in-memory map repos in tests |
| External HTTP | injected `httpc.Client` mocked at boundary | `http.DefaultClient`, `http.Get` in domain code |
| Logger | injected logger | `log.Printf`, package-level `slog.Default()` use |

---

### 4. Mock vs Real Decision

| Dependency shape | Approach | Why |
|---|---|---|
| Pure function / value object | Use real | Mocking adds noise, no flake risk |
| External API / 3rd-party service | Mock at the interface boundary | Determinism, no network |
| Database | Testcontainers or sqlite harness — **not** an in-memory mock map | Mock-maps miss SQL bugs, constraints, transactions |
| Time | Inject `clock.Clock` and use `mocks.Clock` | Determinism, fast tests |
| Filesystem | Real with `t.TempDir()`, unless cross-platform edge | `t.TempDir` is already isolated |
| Slow CPU work | Mock or extract pure core | Keeps unit tests fast |
| Code owned by another team/service | Mock | Don't couple to their drift |

Domain archetype → testing approach:

| Archetype | Primary test shape | Substrate needed |
|---|---|---|
| Pure logic / policy | Table-driven tests | None beyond `testing` |
| HTTP handler / Connect handler | Error-mapping tests with `httptest` or Connect test client | `testutil/httpx` |
| Repository | testcontainers or sqlite harness against real schema | `testutil/db` |
| External integration / client | Mock at the interface seam | `testutil/mocks` |
| Time-dependent (TTL, retry, schedule) | Inject `clock.Clock`; use fake clock | `clock`, `testutil/mocks/clock.go` |
| Orchestrator / multi-domain | Mock the domain seams, real orchestration | per-domain `mocks/` |

---

### 5. Production-Code Testability

Domain packages must accept dependencies, not construct them. The react-vite template anchors the canonical seam set:

| Seam | Template anchor | Required in domain code |
|---|---|---|
| Clock | `path:templates/scenarios/react-vite/api/internal/clock/clock.go` | yes, for any TTL/retry/scheduled/aged logic |
| HTTP client | `path:templates/scenarios/react-vite/api/internal/httpc/` | yes, for any outbound HTTP |
| Env reader | injected at module boundary | yes, for any config read |
| Logger | injected via constructor | yes, never package-global |
| Repository | interface per domain (`path:templates/scenarios/react-vite/api/internal/notes/repository.go`) | yes; the sqlite/postgres impl is wired at module composition |

Construction rule: `New<Domain>Service(deps Deps)` where `Deps` is a struct of interfaces. Domain code never calls `time.Now()`, `os.Getenv`, `http.DefaultClient`, or a package-level logger directly.

---

### 6. Edge-Case Coverage Contract

Every function handling external input gets table-driven coverage of:

| Category | Cases |
|---|---|
| Null/empty | `nil`, `""`, `[]`, `{}` |
| Boundaries | min, max, min-1, max+1 |
| Type edges | `0`, `-1`, `MAX_INT`, `NaN`, `Infinity` |
| Strings | unicode, control chars, very long, whitespace-only |
| Collections | empty, single, many, duplicates |
| Dates | leap years, DST, timezone boundaries |
| Concurrency | parallel calls, cancellation, retry idempotency (where applicable) |

This is a checklist for *what* the suite must cover, not a template to inline. The `test` skill enforces presence of these cases; this skill ensures the harness (table-driven scaffolding, fake clock, fake env) exists to make writing them cheap.

Assertion-content anti-patterns — tombstone/absence tests after feature removals, change-detector tests, permanent characterization tests — are governed by the `test` skill (§6 "Test the Specification, Not the Diff"). Apply them here too when a refactor tempts you to pin current behavior: scaffolding characterization tests are fine, but delete them when the refactor lands.

---

### 7. Helper Organization Contract

A scenario passes the helper-organization bar when:

- exactly one `internal/testutil/` (Go) or `src/test-utils/` (TS) tree exists per surface
- shared seam mocks live in `testutil/mocks/`, each with its own `_test.go` proving the mock satisfies the interface
- domain-local mocks live in `internal/<domain>/mocks/`, not in the shared tree
- fixture factories accept `Partial<T>` / functional-option overrides; no fixed-value singletons
- there is no test helper imported from a non-test package (enforced by `no_prod_import_test.go`)

For full layout reference: `path:templates/scenarios/react-vite/api/internal/testutil/`.

---

### 8. Test Architecture Maturity Ladder

Use this ladder as the primary diagnostic. Each level is verifiable; don't claim a level you can't grep.

| Level | Name | Verifiable artifact |
|---|---|---|
| L0 | Scattered | No `internal/testutil/` or `src/test-utils/`; tests in random `tests/` or `__tests__/` directories. |
| L1 | Co-located | `find scenarios/{{TARGET}}/api -path '*/tests/*' -name '*_test.go' \| wc -l` returns `0`; every `*_test.go` lives beside a source file. |
| L2 | Centralized testutil | `internal/testutil/` exists with at least `fixtures/` and `mocks/` subdirs; `src/test-utils/` exists on UI side. |
| L3 | Testable domain code | Every external dependency in `internal/<domain>/` is accessed via interface; `rg 'time\.Now\(\)\|os\.Getenv\|http\.DefaultClient' scenarios/{{TARGET}}/api/internal/ -g '!*_test.go' -g '!testutil/**'` returns zero hits. |
| L4 | Real-substrate repository tests | Repository `*_test.go` files use `testutil/db` (sqlite) or testcontainers, not stubbed maps. `rg 'map\[string\].*Repository\|InMemoryRepo' scenarios/{{TARGET}}/api/internal/ -g '*_test.go'` returns zero hits. |
| L5 | Drift-gated | `internal/testutil/no_prod_import_test.go` enforces forbidden imports (see `path:templates/scenarios/react-vite/api/internal/testutil/no_prod_import_test.go`); a registry test asserts every `internal/<domain>/` package has a `*_test.go`; mocks expose `var _ Iface = (*MockIface)(nil)` interface-satisfaction guards. |

Levels are per-surface (API, UI, CLI). A scenario can be L4 on API and L1 on UI. Pick the next concrete step from the lowest surface, not a global score.

---

### 9. Audit Commands

Run these against `scenarios/{{TARGET}}/`; each maps directly to a ladder level.

```bash
# L1: co-location — tests not living in a sibling 'tests/' dir
find scenarios/{{TARGET}}/api -path '*/tests/*' -name '*_test.go'
find scenarios/{{TARGET}}/ui  -type d \( -name '__tests__' -o -name 'tests' \)

# L2: testutil presence
ls scenarios/{{TARGET}}/api/internal/testutil/ 2>/dev/null
ls scenarios/{{TARGET}}/ui/src/test-utils/ 2>/dev/null

# L3: untestable patterns in domain code
rg -n 'time\.Now\(\)|os\.Getenv|http\.DefaultClient|log\.Printf|slog\.Default' \
   scenarios/{{TARGET}}/api/internal/ -g '!*_test.go' -g '!testutil/**' -g '!clock/**' -g '!httpc/**'

# L4: in-memory repo stubs hiding in tests
rg -n 'map\[string\].*\bRepository\b|InMemoryRepo|fakeRepo\s*:=\s*&\w+\{' \
   scenarios/{{TARGET}}/api/internal/ -g '*_test.go'

# L5: drift gates present
test -f scenarios/{{TARGET}}/api/internal/testutil/no_prod_import_test.go && echo OK
rg -n 'var _ \w+ = \(\*\w+\)\(nil\)' scenarios/{{TARGET}}/api/internal/testutil/mocks/

# Inline mocks that should be centralized
rg -n 'type mock\w+ struct' scenarios/{{TARGET}}/api/ --type go
rg -n 'const mock\w+ = ' scenarios/{{TARGET}}/ui/ --type ts
```

A non-empty result on L3/L4 commands is a concrete drift item — record it in `PROBLEMS.md` with the file:line list.

---

### 10. Hand-off Contract

| Owner | Responsibility |
|---|---|
| This skill | What test infrastructure must exist: co-location, testutil layout, injectable seams, drift gates. |
| `test` skill | What behaviors are asserted given the infrastructure: edge cases present, assertions strong, regressions covered. |
| `seam-discovery-and-enforcement` | Identifying which production interfaces should exist in the first place. |
| `boundary-of-responsibility-enforcement` | Stopping test helpers from leaking into production import graphs. |

**Architecture L≥3 is a precondition for `test` running productively.** Concrete examples of work that bounces back to this skill:

- `test` skill wants to assert a TTL expiry, but domain code calls `time.Now()` directly → fix the seam here first.
- `test` skill wants to cover a 429 retry path, but the service uses `http.DefaultClient` → introduce `httpc.Client` injection here first.
- `test` skill wants to verify config-driven branching, but the service calls `os.Getenv` inline → introduce the env reader seam here first.
- `test` skill wants repository-level uniqueness/constraint coverage, but the test uses an in-memory map → swap to `testutil/db` sqlite or testcontainers here first.

---

### 11. Output Expectations

Write findings into the durable docs; do not create a standalone audit report.

| Document | Section | What goes there |
|---|---|---|
| `path:scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` | "Testing infrastructure" | testutil layout summary, current ladder level per surface, evidence, next step. |
| `path:scenarios/{{TARGET}}/docs/internal/SEAMS.md` | one row per seam | clock, httpc, env reader, logger, repository interfaces — production wiring point and test fake. |
| `path:scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` | testing gaps | non-empty L3/L4 grep results, missing drift gates, deferred refactors. |

**Explicitly retired:** the older `docs/internal/UNIT_TEST_ARCHITECTURE.md` template. Do not create it. If one exists in a scenario, fold its content into the three docs above and delete the file (record the move in `PROBLEMS.md`).

You may, in `scenarios/{{TARGET}}/`:
- create or reshape `internal/testutil/` and `src/test-utils/` to match the canonical layout
- introduce missing seams (clock, httpc, env, logger) and refactor domain code to consume them
- add `no_prod_import_test.go` and per-domain registry tests
- move scattered tests to co-located positions, move inline mocks to `mocks/` directories
- wire `testutil/db` sqlite harness or testcontainers for repository tests

You must:
- keep `{{TARGET}}` fully functional; existing tests must still pass
- preserve observable behavior of domain services while introducing seams
- create reusable patterns, not one-off helpers
- record each ladder advancement in `ARCHITECTURE.md` with the verifying command

You must NOT:
- add new behavioral tests (that is the `test` skill's job)
- delete tests to make a move pass
- introduce DI frameworks; prefer plain constructor injection
- over-abstract — interfaces exist only where a real test substitution requires them
- create `UNIT_TEST_ARCHITECTURE.md`

Avoid superficial moves: renaming a folder or shuffling files without advancing a ladder level is drift, not progress.
