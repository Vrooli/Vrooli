# Testing — Compute Manager

> **Status: no test in this document exists yet, except the scaffold's
> own.** The scenario was generated from the `react-vite` template on
> 2026-09-03. The only tests in the tree are the ones the template
> shipped: the health handler, the middleware, the module registry, the
> testutil self-tests, and the UI and CLI smoke gates. Not one line of
> this scenario's behaviour has been tested, because not one line of it
> has been written. Every requirement in `requirements/` is `planned`
> and no automated validation entry is implemented, which is the honest
> state and is recorded in [`PROBLEMS.md`](PROBLEMS.md). The test paths
> named below come from the `notes` field of each requirement's
> validation entries: they are where each test will live and what it
> will have to prove, not a description of a suite that runs.

How to write tests against this scenario's shape. Read this before your
first non-trivial test. The patterns below are load-bearing for the
gates documented in [`SEAMS.md`](SEAMS.md), in `eslint.config.js`, and
in `.github/workflows/test.yml`.

**What makes testing this scenario different.** Most scenarios test
that the right thing happens. This one mostly tests that the wrong
thing cannot, because every failure mode it guards against spends real
money on a real cloud account and keeps spending it until somebody
notices. A missing test here is not a coverage gap, it is an hourly
charge. Four properties carry that weight, and each has a test whose
absence would be the defect:

1. **Ordering.** The intent row and its idempotency key are committed
   before any provider call, so a create that succeeds with a lost
   response is recoverable rather than invisible.
2. **Refusal.** A reservation is obtained before the provider is
   reachable, and a refusal short-circuits with zero provider calls.
3. **Termination.** Expiry is enforced twice, by the sweeper and by a
   timer inside the instance, and destroy is the only stop that exists.
4. **Detection.** Reconciliation compares both directions and reports
   without resolving.

Every one of those is provable against a fake provider, with no API key
and no money spent. That is the point of the fake, and it is why the
first slice of work is testable in full before the first real adapter.

## TL;DR — the canonical examples

**What exists today** are the template's own tests, and they are still
the right shape to copy for mechanics:

- **API**: `api/handlers/health/handler_test.go` is table-driven, runs
  real middleware via `httpx.NewLiveServer`, substitutes the fake
  pinger from `mocks/`, and decodes with
  `assertx.MustUnmarshalProto[healthv1.Response]` (the wire shape lives
  in `packages/proto/schemas/compute-manager/v1/shared/health.proto`;
  assert on typed proto fields, not `map[string]any` chains). For
  endpoints whose wire shape is not in proto yet, `MustDecodeJSON[T]`
  is the fallback, but adding the proto first is the right move.
- **UI composition**: `ui/src/App.test.tsx` is a smoke-only composition
  test. App composes shell plus features; feature behaviour belongs
  beside the feature.
- **UI feature**: `ui/src/features/health/HealthCard.test.tsx` shows
  `renderWithProviders`, factory data, an inline `vi.mock` factory
  closure, cimode assertions, and real-locale assertions.
- **UI a11y**: `ui/src/layout/AppShell.a11y.test.tsx` and
  `ui/src/features/health/HealthCard.a11y.test.tsx` test shell and
  feature accessibility at their ownership boundary.
- **CLI**: `cli/app_test.go` is the smoke gate (NewApp, `--version`,
  `--help`). When domain commands arrive, extend it with
  `clitest.NewAPIServer` and `clitest.CaptureStdout` from
  `cli/internal/testutil/`.

**What does not exist yet** are the six tests this scenario is actually
for. Each is described in its own section below, each is named in a
requirement's validation notes, and each must carry its requirement tag
so the traceability matrix can link it:

| Test | Proves | Requirement |
|---|---|---|
| Lifecycle against the fake provider | Request to destroyed runs end to end with every transition persisted in order | `COMPUTEM-P0-001` |
| Intent before the provider call | The durable write happens first, and a replayed key creates no second machine | `COMPUTEM-P0-002` |
| Bidirectional reconciliation | Both directions produce findings, and the sweep mutates nothing | `COMPUTEM-P0-003` |
| Expiry enforced twice | The sweeper destroys, and the first-boot timer powers off without the control plane | `COMPUTEM-P0-004` |
| Reservation settle and release | Success settles, failure releases, and neither path can do both or neither | `COMPUTEM-P0-006` |
| No stop, pause or suspend anywhere | The structural assertion that the affordance cannot be added by accident | `COMPUTEM-P0-007` |

If a new test does not look like one of the shapes below, ask why
before shipping it.

## What the tests must prove

### Testing against the fake provider

`api/internal/provider/mocks::FakeAdapter` is the substitution that
makes this scenario testable at all. It implements the same four
methods as a real adapter, holds an in-memory inventory, and exposes
the levers that turn each cost-safety property into a deterministic
test:

- a per-method error knob, so a create failure, a describe failure and
  a destroy failure are each independently reachable,
- a **lost response** lever, which is a create that succeeds at the
  provider and returns an error to the caller. This is the single
  failure the intent-before-action rule exists for, and without a fake
  that can produce it on demand the rule is untestable,
- a settable create latency and a controllable ready transition, so a
  test can hold an instance in `creating` and assert nothing downstream
  ran early,
- an inventory that can be mutated behind the scenario's back, which is
  how a destroyed-out-of-band instance and an unaccounted instance are
  staged for the reconciliation test,
- its own declared billing facts, so rounding and minimum-billable-unit
  behaviour can be asserted without a provider account.

Rules for using it:

- **The fake is the default. A real provider is never the default.**
  No test in `go test ./...` may reach a cloud API. A test that needs
  a real account is a separate, tagged, attended test, and it is not
  part of the coverage gate.
- **The fake is not a stub that returns success.** A fake that only
  succeeds proves nothing here, because every property in this
  scenario is about what happens when something fails halfway.
- **The fake declares billing facts, so the fake can be wrong on
  purpose.** Give it hourly rounding in one test and per-second
  billing in another; the point of `OT-P1-004` is that no caller
  changes when the facts do.
- **Assert on the fake's call log, not only on its return values.**
  Most of the properties in this scenario are ordering properties, and
  ordering lives in the call log. "The provider was never called" is
  the assertion that proves a refusal, and no return value can carry
  it.

### The bidirectional reconciliation test

The failure this test exists to catch is that most implementations of
reconciliation build only one direction and read as complete. The
integration test at `api/internal/reconcile/sweep_integration_test.go`
must assert both, in the same run, and it is not a passing test if it
only stages an orphan:

| Direction | Staging | Expected finding | Why the direction matters |
|---|---|---|---|
| At the provider, absent locally | Put an instance into the fake's inventory that the scenario never created | An `unaccounted_at_provider` finding | Cost with no owner. This is the direction everyone builds. |
| Local, absent at the provider | Remove an instance from the fake's inventory while the local record still says running | A `destroyed_out_of_band` finding | The usage window has to close. Without this direction, the scenario keeps metering a machine that no longer exists, and the error is silent and biased towards over-billing. |
| Both present, states differ | Local says running, the fake says the instance is gone or in another state | A state-divergence finding | Divergence is a fact worth recording even when neither side is missing. |
| Both present, cost differs | The fake's billing surface disagrees with metered usage beyond the threshold | A cost-divergence alarm; inside the threshold, a record and no alarm | `COMPUTEM-P1-003`. The threshold is the whole behaviour, so both sides of it need a case. |

Two assertions carry more weight than the findings themselves:

- **The sweep mutated no instance.** Snapshot every instance row before
  the sweep and compare after. A reconciler with a defect that destroys
  a running node is the failure this scenario is most afraid of, and
  the decision that it reports rather than resolves is only real if a
  test enforces it.
- **The sweep settled, released and adjusted no reservation.** The
  reconciler never writes money. A `destroyed_out_of_band` finding is
  drained by the meter domain, which is where the usage window closes,
  so the sweep's own output is a row and nothing else.

### The no-stop structural test

`COMPUTEM-P0-007` is unusual: it is a requirement that a thing does not
exist. `api/internal/provider/no_pause_test.go` walks the scenario's Go
AST and fails if any exported method, handler or domain function is
named `pause`, `stop`, `suspend`, `halt` or `shutdown`. A companion
assertion in `api/internal/provider/interface_surface_test.go` pins the
adapter interface at exactly four methods, so a fifth cannot arrive
quietly alongside a new provider.

Why it is a structural test rather than a review rule: a stopped
instance still bills at the full rate on five of the seven providers
surveyed, so a pause control charges full price for no value. It is
also the most natural feature request anyone will make, and the most
natural thing for a future contributor to add in good faith. A review
catches it if the reviewer happens to know. The AST assertion catches
it every time, and it fails with a reason attached rather than a style
complaint.

Two practical notes. The walk must cover the CLI and the UI too, not
just `api/internal/`, because the affordance can appear as a command
verb or a button before it appears as a method; the CLI half of this
lives in `cli/domains/instance/verbs_test.go`, which asserts the
manifest declares no `pause`, `stop` or `suspend` verb and that
`destroy` is the only lifecycle terminator. And the test needs an
allowance for genuinely unrelated names, such as a server's own
graceful-shutdown path, so scope it to domain and handler packages and
say in the failure message which names are reserved and why.

### The intent-before-provider ordering test

The property is an ordering guarantee, not a latency budget, so it
cannot be tested by timing. `api/internal/intent/intent_test.go` and
`api/internal/intent/lost_response_integration_test.go` prove it in
two different ways.

The unit test asserts the write happened first by making the provider
unreachable: with the fake adapter configured to fail on any call, a
create request must still leave a committed intent row carrying its
idempotency key. If the row is absent, the code called out before it
recorded, which is exactly the defect. The same test replays the
idempotency key and asserts the original intent is returned and the
fake's create count is still one, because a replay that races into a
second create is a duplicate hourly bill.

The integration test stages the real failure: the fake's lost-response
lever makes the create succeed at the provider and return an error to
the scenario. The assertions are that the intent survives in a state
the reconciler can act on, that a subsequent sweep matches the
unaccounted provider instance back to that intent, and that no second
create was issued anywhere along the way.

Ordering assertions belong in the fake's call log rather than in
timestamps. Record the sequence of "intent committed" and "provider
called" events and assert on their order directly; a test that
compares two `time.Now()` readings is measuring the machine, not the
code.

The same shape covers the reservation ordering in
`api/internal/meter/reserve_test.go`: with the business suite fake set
to refuse, the assertion is that the provider's call count is zero and
the outcome is a distinguishable out-of-credit refusal that names the
ceiling. A refusal must cost nothing.

### The expiry double-enforcement test

One guarantee, two enforcement points, and they are tested separately
because they fail separately.

**The sweeper**, in `api/internal/expiry/sweeper_test.go`, is tested
entirely through the Clock seam. Advance the fake clock to one second
before an instance's expiry and assert nothing was destroyed; advance
past it and assert exactly one destroy was issued to the fake provider.
The same test covers an extension granted before expiry, which must
move the deadline, and an extension attempted on an already destroyed
instance, which must be refused. No test in this file sleeps.

**The instance-side timer**, in
`api/internal/expiry/firstboot_timer_test.go`, is tested by asserting
on the rendered first-boot configuration, because the whole value of
this enforcement point is that it needs nothing from us at runtime.
The assertions are that the rendered configuration contains a
power-off timer, that its timestamp is the same expiry the sweeper
would use, and that it reaches that timestamp with no call back to the
control plane. The two enforcement points agreeing on one timestamp is
what makes expiry immutable after creation unless explicitly extended,
so a test that lets them drift has removed the guarantee.

The third layer is manual and stays manual: with this scenario stopped
entirely, a real instance past its expiry still powers itself off. That
is recorded against
`docs/operations/RUNBOOK.md#verify-the-instance-side-drain` and cannot
be automated here, because the thing under test is what happens when
the automation is not running.

### Reservation settle and release

Both paths, in the same suite, because the bug is always that only one
of them was built.

`api/internal/meter/settle_integration_test.go` covers the success
path: an instance that ran and was destroyed settles measured usage
against its reservation, and the quantity settled is derived from the
`running_at` and `destroyed_at` transitions this scenario caused rather
than from anything observed. Assert the settled quantity, not just that
settle was called, because a settle with the wrong quantity is a
billing error that looks like a pass.

The same file covers the failure path: a provision that reserves and
then fails at the provider must **release** the reservation rather than
leave it to expire. Leaving it is tempting, because the upstream window
is only ten minutes and it will expire on its own. It is still wrong.
An expired hold and a released hold look different to the tenant, and a
scenario that relies on expiry to clean up has no way to tell a
released hold from one it forgot about.

Three further cases belong here:

- **The heartbeat.** The upstream reservation window is ten minutes and
  an instance lives for hours, so re-reservation is not an optimisation.
  Advance the fake clock across several windows and assert a renewal
  row per window, recorded rather than mutated, so the history survives.
- **Exactly one terminal outcome per reservation.** A reservation that
  is both settled and released, or neither, is the defect. Assert on
  the fake's counters, not on the last call.
- **No refund path.** Upstream refunds silently do nothing for
  app-scoped charges, which is recorded in [`PROBLEMS.md`](PROBLEMS.md).
  Settle the real measured quantity rather than over-reserving and
  correcting downwards, and do not write a test that asserts a refund
  works, because it will pass while doing nothing.
- **Not the convenience helper.** The upstream `ReserveAndCharge`
  helper is not a reservation despite its name: it holds no identifier
  and has no release path, so an hour charged for a provision that
  failed three minutes in cannot be compensated. Reserve, then finalize
  or release. A test that exercises the helper is testing a path this
  scenario must not take.

## Requirement traceability

Every P0 and P1 requirement in `requirements/` declares at least two
automated validation layers, and the layers are deliberately different
in kind rather than the same test twice. A unit layer pins the rule, an
integration layer proves it survives contact with the rest of the
scenario, and a structure layer asserts something about the shape of
the code that no runtime test can reach.

Each test carries its requirement tag. Until the test exists, the
requirement's validation entry stays `not_implemented` with no `ref`.
Never hand-set a status; let the test earn it.

| Requirement | Layer | What that layer actually checks |
|---|---|---|
| `COMPUTEM-P0-001` provider-agnostic lifecycle | unit | The fake satisfies all four methods, and a request for a size the provider does not offer is refused before any call is made. |
| | integration | A full request-to-destroyed cycle against the fake, with every state transition persisted in order. |
| | structure | The adapter interface declares exactly four methods, and no caller package imports a concrete provider implementation. |
| `COMPUTEM-P0-002` intent before the provider call | unit | The intent row and its idempotency key are committed before the provider client is reachable; a replayed key returns the original intent. |
| | integration | A create that succeeds with a lost response leaves an intent the reconciler can match to the instance that actually exists. |
| `COMPUTEM-P0-003` bidirectional reconciliation | unit | Provider-present-local-absent is an orphan finding; local-present-provider-absent closes the usage window instead of metering on. |
| | integration | A full sweep against the fake emits findings in both directions and mutates no instance. |
| | manual | An operator can quarantine a reported unaccounted instance, inspect it, and then destroy it. Recorded against the runbook, not automated. |
| `COMPUTEM-P0-004` expiry enforced twice | unit | The sweeper destroys past expiry, honours an extension granted before expiry, and refuses to extend one already destroyed. |
| | integration | The rendered first-boot configuration contains a timer that powers the instance off at its expiry with no control-plane contact. |
| | manual | With this scenario stopped, an instance past its expiry still powers itself off. |
| `COMPUTEM-P0-005` unattended enrollment | unit | The rendered first-boot configuration carries the bridge onboarding public key and no password, token or private key of any kind. |
| | integration | Against a bridge fake, the machine record and its locators are created and onboarding starts, with no credential value in any request field or log line. |
| | manual | A real provisioned instance reaches online with no interactive step. Blocked upstream until bridge publishes its onboarding key. |
| `COMPUTEM-P0-006` credit reserved before boot | unit | A reservation is obtained before the provider client is reachable, and a refusal short-circuits with zero provider calls and a distinguishable out-of-credit outcome. |
| | integration | Teardown settles measured usage; a provision that fails after reserving releases rather than burns the reservation. |
| | business | A client asserting a higher tier than its subscription is still refused, because the tier is resolved server-side and never read from the request. |
| `COMPUTEM-P0-007` destroy is the only stop | structure | No exported method, handler or domain function named pause, stop, suspend, halt or shutdown exists anywhere in the scenario. |
| | cli | The CLI manifest declares no pause, stop or suspend verb, and destroy is the only lifecycle terminator. |
| `COMPUTEM-P1-001` adopt an existing machine | unit | Adoption records no instance, no intent and no reservation, so a machine the operator already owns is never metered. |
| | integration | An adopted host reaches online through the same bridge onboarding contract as a provisioned one. |
| `COMPUTEM-P1-002` per-tenant ceiling | unit | A request that would cross the computed ceiling is refused with the ceiling named, rather than accepted and alarmed afterwards. |
| | integration | Concurrent requests against one tenant cannot jointly exceed the ceiling. This is a race test; run it under `-race` with real concurrency, not sequential calls. |
| `COMPUTEM-P1-003` daily cost reconciliation | unit | Divergence beyond the threshold alarms; divergence inside it records and does not alarm. |
| | integration | The daily comparison runs against the fake's billing surface and never feeds its result back into a charge. |
| `COMPUTEM-P1-004` a second provider changes no caller | unit | Two registered adapters are selected by identifier alone, each declaring its own rounding, minimum billable unit and stopped-instance behaviour as data. |
| | structure | No domain or handler package references a provider by name. |
| `COMPUTEM-P1-005` operator inventory surface | unit | The table renders loading, empty, populated, expiring-soon and unaccounted states, and distinguishes each without relying on colour alone. |
| | integration | Live inventory, elapsed cost and remaining lifetime render from the generated client against a stubbed API, with figures in tabular numerals. |

## API testing

### Layout

```
api/
├── internal/
│   ├── clock/clock.go            # Clock interface + clock.System
│   ├── database/pinger.go        # Pinger interface
│   ├── middleware/logging.go     # Uses clock.Clock — no time.Now()
│   ├── server/                   # Server wires cross-cutting Clock + Logger
│   └── testutil/
│       ├── assertx/              # AssertStatus, MustDecodeJSON[T]
│       ├── db/                   # NewSQLite(t) — modernc.org/sqlite
│       ├── fixtures/             # NewHealthResponse(opts...) — functional options
│       ├── httpx/                # NewLiveServer(t, *Server) over real socket
│       ├── mocks/                # FakeClock, FakePinger
│       ├── no_prod_import_test.go  # AST guardrail (see below)
│       └── testutil.go           # Package contract
└── handlers/health/
    ├── handler.go                # Production REST handler
    └── handler_test.go           # Canonical test
```

### The five primitives every test uses

1. **`mocks.FakeClock`** — substitutes `clock.Clock`. Construct with
   `mocks.NewFakeClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))`,
   advance with `.Advance(d)`. Tests that touch duration logging or
   timestamp output start here.
2. **`mocks.FakePinger`** — substitutes `database.Pinger` (cross-domain
   mock under `internal/testutil/mocks/`). Construct with
   `&mocks.FakePinger{PingErr: errors.New("connection refused")}` to
   exercise the unhealthy branch; default `PingErr: nil` is the happy
   path. Atomic `Calls` counter is for "the handler called Ping exactly
   once" assertions.
3. **`httpx.NewLiveServer(t, srv)`** — wraps your `*server.Server` in a
   real `httptest.Server` listening on a real socket. Returns a struct
   with a `Do(t, method, path, body) (*http.Response, []byte)` method.
   **Use this, not `httptest.NewRecorder`.** Recorder fakes `Flusher`
   and `Hijacker`, masking SSE-flush bugs that workspace-sandbox shipped
   in production on 2026-04-28. The cost of a real socket is measured
   in microseconds; the cost of the bug class it catches is measured
   in incidents.
4. **`assertx`** — `AssertStatus(t, resp, want)` for status code
   checks (dumps body on mismatch); `MustUnmarshalProto` for proto-typed JSON
   decoding (use this whenever the
   endpoint's wire shape lives in `packages/proto/schemas/`);
   `MustDecodeJSON` for ad-hoc JSON when no proto exists yet. Resist
   over-generalising; add helpers when the third caller appears.
5. **Generated proto types** — every endpoint's wire shape lives in
   `packages/proto/schemas/compute-manager/v1/<domain>/<file>.proto`.
   Tests import the generated Go type directly
   (`healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/shared"`)
   and decode wire bodies into it via `MustUnmarshalProto`. The
   `fixtures` package re-exports the proto type as a short alias
   (`fixtures.HealthResponse = healthv1.Response`) so test code reads
   cleanly.

### Canonical test pattern

```go
package health_test

import (
    "errors"
    "io"
    "log"
    "net/http"
    "testing"

    "github.com/gorilla/mux"
    "github.com/stretchr/testify/require"
    healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/shared"

    "compute-manager/handlers/health"
    "compute-manager/internal/clock"
    "compute-manager/internal/module"
    "compute-manager/internal/server"
    "compute-manager/internal/testutil/assertx"
    "compute-manager/internal/testutil/httpx"
    "compute-manager/internal/testutil/mocks"
)

func TestHealthHandler(t *testing.T) {
    cases := []struct {
        name           string
        pingErr        error
        wantCode       int
        wantStatus     string
        wantConnected  bool
    }{
        {"ok", nil, http.StatusOK, "healthy", true},
        {"db_unreachable", errors.New("connection refused"), http.StatusServiceUnavailable, "unhealthy", false},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            pinger := &mocks.FakePinger{PingErr: tc.pingErr}
            h := health.NewHandler(health.Deps{
                Pinger:  pinger,
                Service: "compute-manager",
                Version: "1.0.0",
            })
            mod := module.Module{
                Name: "health",
                Mount: func(r *mux.Router) {
                    r.HandleFunc("/health", h).Methods(http.MethodGet)
                },
            }
            srv := server.New(
                server.Deps{Clock: clock.System{}, Logger: log.New(io.Discard, "", 0)},
                mod,
            )

            live := httpx.NewLiveServer(t, srv)
            resp, body := live.Do(t, http.MethodGet, "/health", nil)

            assertx.AssertStatus(t, resp, tc.wantCode)
            got := assertx.MustUnmarshalProto[healthv1.Response](t, body)
            require.Equal(t, tc.wantStatus, got.Status)
            require.Equal(t, tc.wantConnected, got.Dependencies["database"].Connected)
            require.Equal(t, int64(1), pinger.Calls.Load())
        })
    }
}
```

The proto schema in `packages/proto/schemas/compute-manager/v1/shared/health.proto`
mirrors `api-core/health.Response` field-for-field, so `MustUnmarshalProto`
round-trips the wire shape directly into the generated Go type — no
`map[string]any` chains, no per-test `interface{}` casts, no parallel
hand-written struct mirror to drift against. `DiscardUnknown:true` is
wired in `MustUnmarshalProto` so the test keeps passing when the wire
grows fields the proto hasn't caught up to.

### The first vertical slice

The scaffold's worked example domain was removed when this scenario was
detemplated, so there is no in-tree reference to copy. The first
vertical slice is `intent`, and it is the right one to build first for
a reason that is not arbitrary: it is the smallest domain, it is the
first durable write on every provisioning path, and its whole contract
is an ordering rule that can be proven with no provider and no money.

Build it one layer at a time, and write the test for each layer before
the next one exists:

| Layer | File | What its test proves |
|---|---|---|
| Wire contract | `packages/proto/schemas/compute-manager/v1/intent/intent.proto` | Nothing on its own. It is a contract, not a seam. |
| Domain types | `api/internal/intent/types.go` | Typed sentinels exist and are the values the handler maps from. |
| Repository interface | `api/internal/intent/repository.go` | The persistence seam. Substituted in service tests. |
| Repository impl | `api/internal/intent/sqlite.go` and `sqlite_test.go` | Against real SQLite via the compose helper below: an intent round-trips, and a duplicate idempotency key does not create a second row. |
| Schema | `api/internal/intent/schema.{sql,go}` | An embed tripwire, so the DDL and the Go code cannot drift. |
| Service | `api/internal/intent/service.go` and `service_test.go` | Against a fake repository: validation, and the ordering rule that the commit happens before anything downstream is reachable. |
| Handler | `api/handlers/intent/connect_handler_test.go` | Against a fake service: transport, error mapping, and that a replayed key returns the original rather than a new intent. |
| Mocks | `api/internal/intent/mocks/{repository,service}.go` | Per-method error knobs and atomic counters, co-located so deleting the domain takes them along. |

The three-layer split matters more here than it does in a CRUD app.
Each layer's test substitutes exactly one thing, which is what lets the
ordering assertions be about ordering rather than about SQLite:

```
handler → Service (validates, enforces ordering) → Repository (persists)
             ↑                                          ↑
             FakeService (handler tests)                 FakeRepository (service tests)
                                                         Real sqlite (repository tests)
```

`provider` is the natural second slice, because the fake adapter it
introduces is what every later domain's tests are written against.

#### Compose pattern: schema-applied repository test

`db.NewSQLite(t)` returns a blank handle. Repository tests apply the
production schema before the first query so the test exercises the
same shape `main.go` ships:

```go
func newSchemaDB(t *testing.T) *sql.DB {
    t.Helper()
    d := db.NewSQLite(t)
    require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
        apidb.SchemaProviderFunc(localdb.SystemSchema),
        apidb.SchemaProviderFunc(<domain>.Schema),
    ))
    return d
}
```

That helper is the canonical entry point for every new domain's
`*_sqlite_test.go`. Don't reach for migrations frameworks or in-test
`CREATE TABLE` literals — the per-domain `schema.sql` files (collected
by `internal/modules/registry.go::AllSchemas()` in production) are the
source of truth for both production and tests.

### Temporal workflow tests

The canonical workflow inventory lives in
[`FLOWS.md`](../concepts/FLOWS.md). Tests prove the state/event
contracts documented there.

Use temporal workflow tests when a domain has lifecycle states where
some events are allowed and others are forbidden. Do not use coverage
percentages as proof that the state space is complete; a suite can
touch every line while never testing "retry after success" or
"complete after cancel."

Three state machines in this scenario earn that treatment, and
[`FLOWS.md`](../concepts/FLOWS.md) is where their states and events are
defined:

- **Instance.** `requested`, `creating`, `running`, `draining`,
  `destroyed`, plus `orphaned` and `unknown`, which reconciliation
  assigns and nothing requests. The forbidden transitions are the
  interesting ones: nothing leaves `orphaned` except through an
  operator, and there is no transition out of `running` that is not
  eventually destruction, because there is no pause.
- **Reservation.** `held`, with a self-transition for heartbeat
  re-reservation, then exactly one of `settled`, `released` or
  `expired`. The invariant worth checking after every transition is
  that a reservation reaches exactly one terminal state.
- **Intent.** `reserving`, then `reserved`, `refused`, `fulfilled` or
  `abandoned`. A `reserving` intent left behind by a crash is a
  recoverable hold rather than a leak, and that is the whole reason the
  state exists.

These are domain policy, so production and tests call the same
transition function. The provider call, the reservation call and the
bridge call around them are the seams.

The canonical API shape is:

```
api/internal/<domain>/
  <flow>_workflow.flow.json     # hand: source of truth
  <flow>_workflow.go            # hand: wrapper
  <flow>_workflow_test.go       # hand: thin replay delegation
  generated/<foldername>/
    model.qnt
    artifact.json
    runtime.go
    replay.go
```

`workflow.go` defines:

- status and event types used by the generated topology declarations,
- a pure `Transition(state, event)` wrapper around generated
  status-transition helpers,
- `CheckInvariants(state)` for rules that must hold after every
  transition.

`model_conformance_test.go` uses
`api/internal/testutil/modeltest` to prove:

- every production status is represented,
- every production event is represented,
- every status/event pair has exactly one expected row,
- duplicate, missing, and unknown rows fail loudly,
- traces replay step-by-step against the production transition
  function.
- the generated formal artifact is fresh against the `*.flow.json`
  contract, generated `.qnt` model, generator source, and checked
  invariants.
- the generated transition-table check is present as generated-check
  metadata, not as a fake verified invariant.

The canonical UI shape is:

```
ui/src/features/<domain>/
  <Domain>Workflow.flow.json    # hand: source of truth
  <Domain>Workflow.ts           # hand: wrapper
  <Domain>Workflow.fixtures.ts  # hand: replay fixtures
  <Domain>Workflow.test.ts      # hand: thin replay delegation (~5-8 lines)
  generated/<foldername>/
    model.qnt
    artifact.json
    runtime.ts
    replay.helper.ts
```

Use TypeScript discriminated unions so impossible UI states are not
representable. For example, an upload should not be able to hold both
`{ status: "uploading" }` and a success payload through parallel
booleans. Components dispatch events to the workflow and render the
returned state; they do not duplicate transition rules in event
handlers. Generated formal replay helpers build replay transitions with
the shared `transitionFromReplayAdapter` helper plus generated fixture
map types and generated `*ReplayFixtureContract` constants, so adding a
generated status/event creates a type error until the runtime fixture
exists.

Workflow maturity is incremental:

| Level | Name | Validation expectation |
|---|---|---|
| 1 | Inventory | Flow listed in `docs/concepts/FLOWS.md`. |
| 2 | Workflow model | Pure transition and invariant checks exist. |
| 3 | Matrix + traces | Every state/event pair and representative trace is executable. |
| 4 | Declarative contract | A domain-local `*.flow.json` declares states, events, transitions, invariants, and named traces. |
| 5 | Checked formal model | Quint/TLA+ or equivalent is generated from the contract, checked, and replayed by production tests. |

This scenario is at level 1: the four flows are listed in
[`FLOWS.md`](../concepts/FLOWS.md) and nothing executes. The scaffold's
worked Level 5 flow was removed at detemplate time, so there is no
in-tree reference to copy from. The instance lifecycle is the first
flow that should climb the ladder, because its illegal transitions are
the ones that cost money. The generic file layout per flow is:

- The `flow-verifier` scenario CLI (`flow-verifier verify check|run`, `flows list|validate|explain`)
- `api/internal/<domain>/flow/flow.json`
- `api/internal/<domain>/flow/transition.go` (package `flow`)
- `api/internal/<domain>/flow/flow_test.go` (thin replay delegation, package `flow`)
- `api/internal/<domain>/flow/generated/{model.qnt,artifact.json,runtime.go,replay.go}` (package `generated`)
- `ui/src/features/<domain>/flow/flow.json`
- `ui/src/features/<domain>/flow/transition.ts`
- `ui/src/features/<domain>/flow/fixtures.ts`
- `ui/src/features/<domain>/flow/flow.test.ts` (thin replay delegation)
- `ui/src/features/<domain>/flow/generated/{model.qnt,artifact.json,runtime.ts,replay.helper.ts}`

`make temporal-models` invokes `flow-verifier verify check --root .`, which
runs `quint typecheck`, `quint test`, `quint verify`, and deterministic MBT
trace generation through the flow-verifier pipeline. It fails if the checked-in
artifacts, generated declarations, or generated replay files are stale. The
generated declarations provide state/event topology and formal freshness
expectations, including concrete hashes for the contract, model, and generator.
They also expose pure generated status-transition helpers derived from
`*.flow.json`, so production code does not maintain a second abstract
transition matrix. Generated Go and TypeScript replay tests load those
artifacts through `modeltest` and replay generated transitions/traces against
production transition functions. UI replay keeps the hand-authored runtime
fixture map in `flow/fixtures.ts`; the generated
`replay.helper.ts` owns freshness, matrix replay, and trace replay, and
the hand-authored `.test.ts` is a ~5-line module that imports the helper
and the fixtures and calls `runFormalReplay({ transition, fixtures })`
at top level. An AST-level lint in `flow-verifier verify check` rejects any
file that imports the helper without calling it.

Formal artifacts use schema v5 coverage metadata. `transitionMatrixComplete`
and `terminalTransitionsChecked` describe the generated matrix. `namedTraces`
describes required hand-authored trace coverage. `generatedTraces` reports
what Quint MBT traces visited, including `coveredPairs` and
`allPairsCovered`; that field is informational and may be false.

Schema v5 `*.flow.json` files no longer declare any output paths; the
generated subpackage location is derived from the flow ID. The contract's
`replay` block carries only `fixtureModule`, `fixtureExport`, and
`transition` metadata.
`flow-verifier flows validate`, `verify run`, and `verify check` validate each
contract against the embedded flow schema before semantic validation, so
unknown fields, missing required fields, old marker-based `replay.bindings`,
and invalid enum values fail with contract-path context before Quint runs.
`check` then compares the generated replay files byte-for-byte, which makes a
missing production replay test a generator failure instead of a later review
catch. Use `flow-verifier flows explain --flow <flow-id>` to inspect generated
files, runtime typing, fixture contracts, topology, generated replay paths,
fixture module expectations, coverage, and the exact commands to run next.

A Quint/TLA+ model is only accepted when this full loop exists.
Documentation-only formal specs are drift-prone and should not be
added. Plain CRUD should stay plain; copy the Level 5 pattern only for
flows with lifecycle states and illegal transitions.

When adding or changing a Level 5 state/event:

1. Edit the flow contract.
2. Regenerate that flow with `flow-verifier verify run --root . --flow <flow-id>`.
3. Update only runtime payload logic that the abstract model cannot own
   (file handles, attempt ids, repository side effects, user-facing
   messages).
4. Update UI replay fixture modules; missing keys should be compile-time
   failures via the generated formal replay fixture interface.
5. Run `make temporal-models` before the regular scenario tests.

### Buffer-backed logger pattern

The production `*log.Logger` shouldn't write to stderr during tests —
it pollutes the runner's output and makes failure messages harder to
read. Connect handler tests should use the shared helper:

```go
logger, logBuf := connectxtest.NewLogger(t)
client := new<Domain>Client(t, fakeService, logger)
```

For scenario-local helpers that do not consume `api-core/connectx`, the same
shape is a `bytes.Buffer`-backed logger:

```go
logBuf := &bytes.Buffer{}
srv := server.New(server.Deps{
    Logger: log.New(logBuf, "", 0),
    // …other deps
})
```

Discard-only sinks (`log.New(io.Discard, "", 0)`) work for tests that
don't need to inspect log output; reach for the buffer when the test
asserts on what was logged — e.g. a 500-path handler test that checks
the underlying error reaches operator logs.

### Testing context cancellation

The template ships no streaming endpoints today, but the test
infrastructure supports them. When a scenario adds an SSE / long-poll
/ background-work endpoint, the canonical cancellation test is:

```go
live := httpx.NewLiveServer(t, srv)
ctx, cancel := context.WithCancel(context.Background())
req, _ := http.NewRequestWithContext(ctx, http.MethodGet, live.URL+"/stream", nil)
resp, err := live.Client.Do(req)
require.NoError(t, err)
defer resp.Body.Close()

// Read enough bytes to confirm the handler started writing.
buf := make([]byte, 64)
_, _ = resp.Body.Read(buf)

cancel()
// Expect the handler to observe r.Context().Done() and abort cleanly.
// The fake (Pinger / FakeService / future StreamProducer) records the
// cancellation; assert on the recorded state.
```

`httpx.NewLiveServer` runs over a real socket (not `httptest.NewRecorder`),
so `http.Flusher` and the request `Context()` plumbing match production
behavior. Recorder-based tests would silently pass while production
hangs on a never-cancelled handler — the same class of bug
workspace-sandbox shipped on 2026-04-28.

### Production-import quarantine (`no_prod_import_test.go`)

The test in `api/internal/testutil/no_prod_import_test.go` walks every
non-test `.go` file under `api/` and fails if any imports anything
starting with `<module>/internal/testutil/`. The module name is read
dynamically from `go.mod`, so it works after `compute-manager`
substitution.

This is the load-bearing rule that makes it safe to put `time.Sleep`,
process-wide globals, and `testing.T` references in the testutil
package. If the rule fires:

- ✅ **Move the helper out of testutil** into a non-test package.
- ❌ **Don't add `// nolint`** — the production code path will then
   carry the test-only dep into the binary on every build.

### Outbound HTTP — `httpc.Doer`

Production callers consuming external services depend on the `Doer`
interface declared at `internal/httpc/doer.go`. `*http.Client` satisfies
it directly (compile-time-asserted in the same file); tests substitute
`mocks.FakeDoer`. Reference test:
`internal/httpc/doer_test.go::TestDoer_TestPath` exercises both the
production-side and test-side wiring through one tiny inline caller —
the canonical substitution shape.

`mocks.FakeDoer` queues canned `*http.Response` (or errors) via
`AddResponse(status, body) []byte` and records every inbound request
into `.Requests` for after-the-fact assertions. The test fake is the
same shape every scenario reaches for; resist hand-rolling per-feature
HTTP fakes when this surface fits.

The seam ships *unwired* in production by intent — there's no
`server.Deps.Doer` field until the first scenario actually needs one.
When you wire it, follow the canonical pattern:

```go
// main.go
deps := server.Deps{
    Doer: &http.Client{Timeout: 10 * time.Second},
    // …other deps
}
```

## UI testing

### Layout

```
ui/src/
├── test-setup.ts                # vitest setupFiles entry
├── test-utils/
│   ├── index.ts                 # re-exports
│   ├── a11y.ts                  # expectNoA11yViolations(container)
│   ├── factories.ts             # makeHealthResponse(overrides?)
│   ├── renderWithProviders.tsx  # QueryClient + i18n wrapper
│   └── mocks/
│       └── spatial.ts           # builders for @vrooli/iframe-bridge/spatial
├── components/
│   ├── AppShell.test.tsx
│   └── AppShell.a11y.test.tsx
└── features/<name>/             # feature tests and feature a11y live here
```

The `mocks/` directory holds shared mock-shape builders for external
SDKs (today: spatial-nav). Each hook test still calls
`vi.mock("@vrooli/iframe-bridge/spatial", ...)` inline (Vitest hoisting
is non-negotiable), but the factory closure invokes the builders in
`mocks/spatial.ts` so the contract for each SDK lives in one file.
Adding a new SDK: drop a `mocks/<sdk>.ts` builder beside it, add a
`mocks/<sdk>.test.ts` self-test, re-export from `test-utils/index.ts`.

### The four primitives every test uses

1. **`renderWithProviders(<Component />, opts?)`** — wraps the tree in
   `QueryClientProvider` (retries disabled — tests should fail fast,
   not paper over flakes) and `I18nextProvider` bound to the same
   singleton production uses. Returns a `RenderResult` plus the
   `queryClient` for tests that need to seed cache state. The helper
   has its own self-test at `ui/src/test-utils/renderWithProviders.test.tsx`
   pinning retries-disabled, queryClient identity, custom-client
   override, and singleton i18n wiring — mirrors the API-side
   `internal/testutil/httpx/server_test.go` pattern.
2. **`make<Domain>(overrides?: Partial<Domain>)`** — typed factory for
   stable test data. `makeHealthResponse()` is the worked example;
   add new factories alongside it as new shapes appear. Defaults
   should make the most common test path `make<Domain>()` with no
   args.
3. **Inline `vi.mock("./api/health", async (importOriginal) => …)`** —
   the canonical mocking shape. **Do not** wrap this in a helper
   function. Vitest hoists `vi.mock(...)` calls before any imports
   resolve; a wrapper function imported from `test-utils` would be in
   the temporal dead zone at hoist time. `make<Domain>()` calls *are*
   safe inside the factory because the closure runs after imports
   initialise.
4. **`expectNoA11yViolations(container)`** — shared axe-core assertion
   for component-level accessibility tests. Render and wait for the
   component's stable state in the owning test file, then call this
   helper. Do not put feature-specific waits in app-composition tests.

### Canonical UI test pattern

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { makeApiMocks, renderWithProviders } from "../../test-utils";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});

import { HealthCard } from "./HealthCard";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

describe("HealthCard rendering (cimode — copy-independent)", () => {
  afterEach(() => { cleanup(); });

  it("renders the card via test ID", () => {
    renderWithProviders(<HealthCard />);
    expect(screen.getByTestId(selectors.health.card)).toBeInTheDocument();
  });

  it("renders translation keys in cimode", async () => {
    renderWithProviders(<HealthCard />);
    expect(await screen.findByText(strings.health.title)).toBeInTheDocument();
  });
});
```

### Two-layer test pattern (cimode + real locale)

`test-setup.ts` puts vitest into `cimode` before every test — `t('app.title')`
returns the literal key `"app.title"`. Most tests run there: assertions
go through `selectors.*` (test IDs) and `strings.*` (the typed key
registry), so they survive any wording change in any locale.

A second `describe` block opts into real locales with
`beforeEach(async () => { await setLocale("en"); })` and asserts on the
canonical English copy via raw `en.json` references. These tests *should*
update when canonical English copy changes — that's what they verify.

See `features/health/HealthCard.test.tsx` for the full pattern and the
CLDR plural variants (`refreshCount_one`,
`notifications.summary_zero` / `_one` / base). Keep `App.test.tsx`
smoke-only so deleting a feature does not require rewriting the app
composition test.

### Mock builders for `api/*` surfaces

`vi.mock(path, factory)` is hoisted before any user import resolves;
a wrapper imported from `test-utils` would be in the temporal dead
zone at hoist time. The escape hatch is to keep the `vi.mock` call
inline at the top of each test file, but move the *factory body* into
a builder function that runs when the closure executes — which is
*after* imports initialise.

`@/test-utils` exports shared, cross-feature mock builders such as
`makeApiMocks()`. Feature-specific builders live beside the feature so
deleting the feature takes its mocks with it (import
`make<Domain>Mocks()` from `features/<domain>/mocks/<domain>`).

Canonical shape:

```tsx
import { makeApiMocks } from "@/test-utils";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});
```

Defaults are picked so the most common test paths work no-args:
`makeApiMocks().fetchHealth` resolves to a healthy response. The
`...actual` spread keeps non-mocked exports (the `ApiError` class,
re-exported proto types) intact — only network-touching functions are
substituted.

A feature with its own API surface adds a second `vi.mock` for its
client and a feature-local builder; per-test overrides use vitest's
standard pattern *after* the mock is wired
(`vi.mocked(client.method).mockResolvedValueOnce(...)`). The inventory
feature is where this scenario will first need the two-mock shape: one
mock for the shared health surface and one for `api/inventory`.

When a third lib/* surface lands (e.g., `lib/users.ts`), follow the
same pattern: builder in `ui/src/test-utils/mocks/<surface>.ts`, self-
test alongside, re-export from `test-utils/index.ts`.

### Accessibility tests

Accessibility tests follow the same ownership rule as production UI:

- **Shell**: `components/AppShell.a11y.test.tsx` renders
  `<AppShell>` with stable placeholder children. It covers page layout,
  headings, locale controls, and shell-level semantics.
- **Feature**: `features/<name>/<Name>Card.a11y.test.tsx` renders the
  feature directly, owns its API mocks, waits for each state it scans,
  and calls `expectNoA11yViolations(container)`.
- **App**: `App.test.tsx` stays composition smoke. Do not make it the
  default a11y gate; a full-`App` a11y test couples shell coverage to
  every async feature query and becomes fragile as features change.

Before running axe, wait for the state the test owns. For example,
`HealthCard.a11y.test.tsx` waits for `selectors.health.statusValue`
for the success state and `selectors.health.error` for the error state.
This keeps React Query updates inside the awaited test boundary and
prevents `act(...)` warnings.

`test-setup.ts` fails tests that write unexpected `console.error` or
`console.warn` output. If a test intentionally exercises a noisy React
path, suppress that warning locally and assert the user-visible
contract. `ErrorBoundary.test.tsx` is the reference: it suppresses
React's intentional boundary logging while asserting `onError` and the
fallback UI.

### ErrorBoundary tests

`ui/src/components/ErrorBoundary.test.tsx` is the reference for testing
React error boundaries. The patterns it pins:

- **Controlled-throw fixture** (`Throw({ when, message })`). One
  component shared across cases keeps the surface narrow — the
  boundary's contract is "if a child throws, swap to the fallback,"
  and that's what each case exercises.
- **`console.error` suppression** in `beforeEach` / `afterEach`. React
  intentionally logs the caught error; without the spy, the suite's
  output drowns real failure messages. The `onError` test still
  asserts the prop fired, so coverage of the error path is preserved.
- **Mutable-control recovery test**. To prove `setState` re-renders
  children, the test flips a shared object's `value` field rather
  than re-mounting the boundary — boundary identity has to survive
  the retry click for `setState` to take effect.
- **cimode key assertions**. The default test setup runs i18next in
  cimode, so `t(strings.errorBoundary.title)` returns the literal
  key path. Asserting on the key (not the English copy) proves the
  fallback consulted the registry — the test stays green when copy
  changes and breaks loudly if a key is renamed.

App-level wrap point lives in `ui/src/main.tsx`; the boundary nests
inside `<QueryClientProvider>` (and after the `./i18n` side-effect
init) so `useTranslation` works inside the localised fallback.

### Test-utils quarantine

ESLint's `no-restricted-imports` rule (in `eslint.config.js`) bans
imports from `**/test-utils/*` and `@/test-utils/*` in production
files. The `*.test.{ts,tsx}` and `*.spec.{ts,tsx}` override block
turns the rule off so tests import freely.

This mirrors the Go AST guardrail. If the rule fires:

- ✅ **Confirm the importing file is a test.** If your test file is
   named `something.tsx` instead of `something.test.tsx`, ESLint
   correctly treats it as production. Rename it.
- ✅ **Move the helper out of test-utils** if it genuinely needs to
   ship in production.
- ❌ **Don't disable the rule** for a "one-off" production import. There
   is no path back from a test-utils leak — every future build carries it.

## CLI testing

### Layout

```
cli/
├── app.go                              # NewApp wires cli-core's StandardScenarioApp
├── app_test.go                         # canonical smoke test
├── domains/                            # domain-specific command groups
└── internal/
    └── testutil/
        ├── server.go                   # NewAPIServer, NewHTTPServer, WithAPIBase, CaptureStdout
        └── no_prod_import_test.go      # AST guardrail (mirrors the API side)
```

### Smoke test (always present)

`cli/app_test.go` is the canonical smoke gate every scenario inherits.
It catches regressions in cli-core wiring before any domain command
exists:

```go
func TestNewAppConstructs(t *testing.T) {
    app, err := NewApp()
    if err != nil { t.Fatalf("NewApp() error: %v", err) }
    if app == nil || app.core == nil || app.core.CLI == nil {
        t.Fatal("NewApp() returned an incomplete app")
    }
}

func TestRunVersion(t *testing.T) {
    app, _ := NewApp()
    if err := app.Run([]string{"--version"}); err != nil {
        t.Fatalf("--version: %v", err)
    }
}
```

`--version` and `--help` are NeedsAPI=false code paths in cli-core, so
they don't try to reach the configured API base. Tests for API-backed
commands need the httptest pattern below.

### Testing API-backed commands

When a scenario adds its first API-backed command (in
`cli/domains/<domain>/`), the canonical pattern is:

```go
package instance_test

import (
    "encoding/json"
    "net/http"
    "testing"

    clitest "compute-manager/cli/internal/testutil"
)

func TestInstanceList(t *testing.T) {
    server := clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/api/v1/instances" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        _ = json.NewEncoder(w).Encode([]map[string]any{
            {"id": "i-1", "state": "running"},
        })
    }))
    _ = server // server.URL is wired into API_BASE_URL by NewAPIServer

    app, err := NewApp()
    if err != nil { t.Fatalf("NewApp: %v", err) }

    out := clitest.CaptureStdout(t, func() error {
        return app.Run([]string{"instance", "list"})
    })

    if !strings.Contains(out, "running") {
        t.Fatalf("expected instance state in output, got: %s", out)
    }
}
```

The CLI carries one assertion no other surface can make. The verb list
for a lifecycle domain is itself a cost-safety contract: `create`,
`describe`, `list` and `destroy`, and nothing that reads as a pause.
`cli/domains/instance/verbs_test.go` asserts that, per
`COMPUTEM-P0-007`, and it is the reason the CLI has a required
validation layer on a requirement about a missing method.

Why each piece:

1. **`NewAPIServer`** wraps `httptest.NewServer` and sets `API_BASE_URL`
   to the test server's URL via `t.Setenv` — cli-core's APIBase
   resolver picks it up. Auto-restored at end-of-test.
2. **Real httptest.Server**, not a Recorder. Same reasoning as the API
   side: Recorder fakes `Flusher`/`Hijacker`; a real socket catches
   SSE/streaming bugs that have shipped before.
3. **`CaptureStdout`** captures the human output written by
   `RenderProtoList`, `RenderProtoMutation`, or the `RunContext`
   report helpers. Use it instead of `--json` when you want to assert
   on the contract scenario users actually see.
4. **`go test -race`** is enforced by CI (`.github/workflows/test.yml`).
   `CaptureStdout` swaps `os.Stdout` and the test server runs in a
   separate goroutine; race coverage keeps both honest.

### CLI test-utils quarantine

`cli/internal/testutil/no_prod_import_test.go` walks every non-test
`.go` file under `cli/` and fails if any imports
`<module>/cli/internal/testutil/...`. Same rule, same rationale as the
API side: the testutil package is allowed to depend on `testing`,
mutate `os.Stdout`, and run real listeners, because production code
provably can't see it. If the test fires:

- ✅ **Move the helper out of testutil** into a non-test package.
- ❌ **Don't add `// nolint`** — every future build would carry the
   test-only dep into the binary.

### When to add a CLI test

| Change | Test |
|---|---|
| New API-backed command | One success-path test + one error-path test, both via `NewAPIServer` + `CaptureStdout`. |
| New non-API command (config write, fingerprint print, etc.) | Direct `app.Run([...])` with `CaptureStdout`. No fake server needed. |
| Change to `app.go` wiring | The smoke gate (`TestNewAppConstructs`) catches most regressions automatically. Add a focused test only when the wiring touches a non-default code path. |
| New env var that affects API resolution | Extend or wrap `clitest.WithAPIBase` rather than calling `t.Setenv` inline. Keeps the env-var name in one place. |

## How to add a new proto

Wire shapes for new endpoints belong in proto, not in hand-written Go
structs or TS interfaces. After generation, the canonical source lives
under `packages/proto/schemas/compute-manager/`.

Steps:

1. **Author the schema.** In a generated scenario, add
   `packages/proto/schemas/compute-manager/v1/<domain>/<name>.proto`.
   Use snake_case in the proto package directive
   (`package vrooli.compute_manager.v1.<domain>;`) and add a
   `go_package` option pointing at the per-scenario gen path:

   ```protobuf
   option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/<domain>;<domain>_v1";
   ```

2. **Regenerate.** From the repo root:

   ```bash
   cd packages/proto && make generate && make lint
   ```

   New artifacts land under the language-specific generated trees:
   `packages/proto/gen/go/compute-manager/v1/<domain>/`,
   `packages/proto/gen/typescript/compute-manager/v1/<domain>/`, and
   `packages/proto/gen/python/compute_manager/v1/<domain>/`.
   Commit them alongside the schema — generated code is checked in so
   downstream scenarios don't have to re-run codegen.

3. **Wire it on the API side.** Import the generated Go type in your
   handler test and decode via `assertx.MustUnmarshalProto`:

   ```go
   import domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/<domain>"

   got := assertx.MustUnmarshalProto[domainv1.ListResponse](t, body)
   ```

   For fixtures, follow the `fixtures/health.go` pattern — re-export
   the proto type as a short alias and provide functional-options
   builders:

   ```go
   type ListResponse = domainv1.ListResponse
   func NewListResponse(opts ...ListOpt) *domainv1.ListResponse { /* ... */ }
   ```

4. **Wire it on the UI side.** Import the generated TS schema and use
   `fromJson` for decode + `create` for fixtures:

   ```ts
   import { fromJson, create } from "@bufbuild/protobuf";
   import { ListResponseSchema } from "@vrooli/proto-types/compute-manager/v1/<domain>/<domain>_pb";

   // production
   return fromJson(ListResponseSchema, json, { ignoreUnknownFields: true });

   // tests
   const fixture = create(ListResponseSchema, { items: [{ id: "n-1" }] });
   ```

5. **Tests follow.** Connect handler tests call the generated client;
   fixture tests assert on the typed shape via `proto.Equal`. UI tests
   mock `api/<domain>` and return generated response objects from the
   factory.

Don't add a new `mocks/Fake*` interface for the proto type — the proto
isn't a seam, it's a contract. Seams are interfaces; protos are
payload shapes. See `SEAMS.md::Wire contracts live in proto, not seams`.

## E2E binary smoke gate

`api/main_e2e_test.go` is a build-tag-isolated test (`//go:build e2e`)
that:

1. `go build -o <tmp> .` from the api directory
2. Boots the binary with `API_PORT` and
   `VROOLI_LIFECYCLE_MANAGED=true` set
3. Polls `/health` over a real socket
4. Sends `SIGTERM` and asserts clean exit within 5s

It catches regressions handler tests can't see:

- main.go forgets to wire a new field into `server.Deps` (handler
  tests construct `server.New` directly with a hand-built Deps;
  main.go's wiring is unverified)
- `preflight.Run` order changes break the boot path
- `apiserver.Run` listener config drifts (SIGTERM cleanup hook,
  port resolution from env)
- `storage.NewResolver(ProfileAuto)` chooses an unwritable path on
  the host

Default `go test ./...` skips it (no `e2e` tag). The CI workflow
runs it as a dedicated step via `go test -tags=e2e -run TestE2E .`.
Local invocation: `cd api && go test -tags=e2e .`.

This is the only test in the suite that exercises the actual binary
entry point. As scenarios add streaming endpoints, websocket upgrades,
or background workers, extend this file with matching `TestE2E_*`
cases — one per top-level boot-path concern. Resist using it for
feature-level coverage; that's what handler tests are for.

## Coverage thresholds

| Module | Floor | Where it's enforced |
|---|---|---|
| UI (`ui/`) | 85% lines / branches / functions / statements | `ui/vite.config.ts` `test.coverage`; CI runs `pnpm test:coverage` |
| API (`api/`) | 75% total | `.github/workflows/test.yml` `api` job |
| CLI (`cli/`) | 75% total | `.github/workflows/test.yml` `cli` job |

### UI

Configured in `vite.config.ts` at the bottom of the `test.coverage`
block. The `coverage.exclude` list covers test scaffolding and codegen
only — production source under `src/` is exhaustively included. The
default position is: every new `src/` file ships with its own
`*.test.{ts,tsx}` and lands inside the include set. If a scenario adds
genuinely-untestable code, prefer a narrow file exclusion with a
one-line rationale comment in `vite.config.ts` over loosening the
thresholds.

### Go (API + CLI)

Both Go modules gate on a 75% total floor in CI. The threshold is
intentionally lower than the UI's 85% because Go has more
declaration-only surface (interfaces, generated proto types, struct
types) that doesn't carry executable lines — 75% in Go is roughly
equivalent to 85% in TypeScript by lines-per-meaningful-coverage.

`internal/testutil/...` is excluded from the denominator. Those
packages exist to support tests; including them would create the wrong
incentive (writing tests *of* test helpers to inflate the gate). Each
test-utils package has its own self-test (see
`internal/testutil/db/sqlite_test.go`,
`internal/testutil/httpx/server_test.go`, etc.) so the substrate
itself is still verified — it's just not what the production-coverage
number tracks.

Raise toward 80%/85% as scenarios stabilise. Tighten the threshold
rather than loosening it when a new file lands without tests; that's
the signal that drives the test-first habit.

Coverage is the weakest gate in this scenario, and it is worth saying
so plainly. A suite can hold 85 percent while never staging a lost
response, never advancing the clock past an expiry, and never checking
the second direction of a sweep. The percentage is a floor against
untested new files, not evidence that the cost-safety properties hold.
The evidence is the named tests above and the requirement tags they
carry.

### CI failure mode

A drop below floor fails the relevant CI job immediately with the
actual percentage in the error message (`::error::API coverage 71.4%
< 75%`). The fix is to raise coverage in the missing file, not to
lower the gate.

## Common patterns and anti-patterns

| ✅ DO | ❌ DON'T |
|---|---|
| `mocks.FakeClock` for time-dependent assertions | `time.Sleep(150 * time.Millisecond)` then assert on a fuzzy match |
| `httpx.NewLiveServer` for handler tests | `httptest.NewRecorder` (hides SSE-flusher bugs) |
| `getByTestId(selectors.x.y)` for stable selectors | `getByText("Save")` (breaks the moment copy changes) |
| `vi.mock("./api/health", async (importOriginal) => …)` inline at top of file | Helper-wrapped `vi.mock` (TDZ at hoist time) |
| `makeHealthResponse({ status: "degraded" })` for variants | Hardcoded literal payload in three different tests |
| Per-method error knob (`PingErr error`) on fakes | Single global "fail mode" boolean across the fake |
| `var _ Pinger = (*sql.DB)(nil)` to lock the contract at compile time | Runtime "does this satisfy" check in init |
| Adding new seams to [`SEAMS.md`](SEAMS.md) at the same commit | "We'll document it later" |
| `FakeAdapter` for every provider interaction | A test that reaches a cloud API, tagged or not, inside `go test ./...` |
| Asserting the provider was never called, to prove a refusal | Asserting only on the returned error, which cannot prove the call did not happen |
| Recording an ordered call log and asserting on the order | Comparing two `time.Now()` readings to prove one thing happened before another |
| Staging both directions of divergence in one reconciliation test | A sweep test that only stages an orphan and reads as complete |
| Asserting the settled quantity | Asserting only that settle was called, which passes with the wrong number |
| An AST assertion for a thing that must not exist | A review convention for the same thing, which works until the reviewer changes |
| A fake that can fail halfway | A fake that always succeeds, which proves nothing this scenario cares about |

## Cross-references

- **Seams definition + adding new seams**: [`SEAMS.md`](SEAMS.md).
- **The flows and state machines these tests prove**: [`FLOWS.md`](../concepts/FLOWS.md).
- **The domains each test belongs to**: [`DOMAINS.md`](../concepts/DOMAINS.md).
- **Why the provider adapter has four methods, and why the reconciler never resolves**: [`DECISIONS.md`](DECISIONS.md).
- **The upstream defects a test must not assume away**: [`PROBLEMS.md`](PROBLEMS.md).
- **The error taxonomy the failure-path tests assert on**: [`ERROR-HANDLING.md`](ERROR-HANDLING.md).
- **What wave one of measurement can capture against the fake alone**: [`PERFORMANCE.md`](PERFORMANCE.md).
- **Skill bundle for testing-related work** (load before substantial test changes):
  ```bash
  prompt-manager skill read seam-discovery-and-enforcement test unit-testing-architecture-steer
  ```
- **Test runner used by CI and `vrooli scenario test`**: see
  `.github/workflows/test.yml` and `packages/cli-core/cmd/scenario_test.go`.
- **Why no inline mocks in `*_test.go` files**: the testutil package
  is the single source of fake behavior. Inline mocks in tests
  fragment the contract; when the interface grows a method, every
  inline mock has to be updated. One mock in `mocks/`, one update.
