# Testing — Personal Planner

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

**DESIGN-STAGE.** The strategy below is the *planned* verification approach
from the implementation plan §25. No product domain tests exist yet; only
the `health` infrastructure domain and the fenced `notes` worked example
carry real tests today. The fixtures (F01–F16) and acceptance cases
(T01–T36) named here are the **target evidence map** — the desired
behaviors these tests must eventually prove — not tests that currently
run. Every test asserts DESIRED/EXPECTED behavior (the honest-planning
invariants), not whatever the implementation happens to do. This scenario
is workflow-heavy; see [`../concepts/FLOWS.md`](../concepts/FLOWS.md) for
the flows under test and their maturity ladder, and
[`../internal/SEAMS.md`](SEAMS.md) for the substitution boundaries every
layer below relies on.

**No real user data is ever touched.** All fixtures use recorded or
synthetic data; no real calendar account, provider connection, or
identity is exercised. Read-only provider adapters are tested against
canned responses, never a live account.

### Layer 1 — Pure domain / property tests (fast, no I/O)

The scheduling/capacity/forecast kernel is pure and deterministic, so its
strongest evidence is property tests over the §25.3 invariants. These run
with an injected fixed clock/timezone (the determinism seam) and no
database:

- **Interval algebra** — normalization is idempotent; union duration
  never exceeds the sum of component durations; adjacent half-open
  intervals do not overlap; subtraction never yields a negative duration.
- **Effort conservation** — a split/move conserves total active demand
  except through an explicit effort revision; excess over remaining
  effort is flagged as overallocated (fixtures F02, F04).
- **Recurrence identity** — routine exceptions keep a stable
  original-occurrence key after rescheduling; a moved instance carries
  that key alongside its new start (fixtures F09, F12).
- **Dependency acyclicity** — cycles are rejected; edges stay scoped to
  accessible same-workspace records or authorized source projections.
- **Deterministic proposals** — every accepted change satisfies hard
  constraints under the proposal's validated snapshot; repeated
  projection ingestion converges to the same authoritative state;
  candidate selection follows the documented tie-break order (retain
  feasible placement → meet constraints → reduce fragmentation → fit
  focus windows → avoid unnecessary movement), seeded for reproducibility.
- **Forecast non-mutation** — forecast generation leaves authoritative
  rows unchanged; it never rewrites `promised_finish`, protected
  availability, or accepted session placement (fixtures F05, F06, F15).
- **Correction, sharing, and session invariants** — a correction changes
  effective totals without deleting the original measurement (except via
  the explicit deletion workflow); removing a permission can only reduce
  or maintain visible data; any successful concurrent start sequence
  leaves at most one current exclusive session per person.

### Layer 2 — Real-DB integration tests

Repository and command tests run against a real SQLite handle (per the
per-domain repository seams) to pin behavior the fakes cannot:

- **Transactions / atomic apply** — a split/move, session transition,
  commitment revision, or proposal apply commits its allocation changes,
  conservation checks, revision bump, history, and outbox record
  together, or not at all (INV-11).
- **Unique constraints** — including the database-enforced
  **one-exclusive-session guard** (partial unique on person where state
  ∈ {running, paused} and attention = exclusive under Postgres; an
  equivalent transactional guard under SQLite); a preflight `SELECT`
  alone is insufficient (fixture F10).
- **Tenant isolation** — every read/write/job/projection/cache is scoped
  to the authorized workspace and subject (INV-01); a foreign subject
  cannot read another's records.
- **Idempotency** — a retried command (scoped by caller + command type +
  workspace, fingerprinted) returns the prior outcome and never a
  duplicate human-facing effect (INV-10); reuse of a key with a different
  payload is an error.

### Layer 3 — Adapter contract tests

Source and provider adapters are tested against recorded/synthetic
responses through their seam interfaces:

- **Source-app adapter** — normalized `SchedulingIntent` ingestion,
  revisioned projections, replay/out-of-order convergence, and stale
  source-constraint detection (fixtures F08, F16 replay paths).
- **External-calendar provider adapter (read-only)** — initial +
  incremental reads, deletion reporting, moved recurring instance keeping
  its original-instance key, invalidated-cursor forced scoped resync, and
  reauth-required surfacing (fixtures F07, F09). Read-only is asserted at
  both requested scope and adapter behavior.
- **notification-hub delivery** — dedup/idempotent retry with quiet hours
  and no private reason in shared delivery (fixture F11 revocation, T25).

### Layer 4 — End-to-end suite

A small e2e suite covers the load-bearing loops end to end:

- the **planning → focus → review** loop (T01, T09, T14–T17), and
- **sharing security** (recipient binding, exact field mask, ID-guessing
  blocked, nested-leak blocked, expiry, revocation — fixture F11, T24).

### Evidence map: fixtures and acceptance cases

The canonical fixtures **F01–F16** (capacity/deductions, demand
conservation, fragmentation, timer-vs-actual, forecast shift, competing
initiatives, unknown/passive work, source-constraint change, provider
reset/recurrence, two-device sessions, selective sharing, timezones/DST,
honest learning, visual geometry, commitment revision, forgotten timer)
are the deterministic inputs shared across layers. The acceptance cases
**T01–T36** map those fixtures to end-to-end expectations across
foundation/persistence (T01–T02), capacity/conservation (T03–T05),
dependencies/proposals (T06–T10), forecasts/shared capacity (T11–T13),
focus/actuals (T14–T17), learning/source integration (T18–T21),
provider/temporal (T22–T23), sharing/notifications (T24–T25), what-if
side-effect isolation (T26), export/import lifecycle (T27–T28),
visual/accessibility/responsive (T29–T31), reliability/CLI parity
(T32–T34), and migration/optional-assistance (T35–T36). Each T-case is
the acceptance target the layers above must satisfy; link real local
tests to their fixture and T-case as they are added.

Record the required external services and any exceptions to the shared
harness here as they appear. Link to real local tests as they are added.
Keep execution receipts with their owning plan.

## Binary startup

`api/main_e2e_test.go` uses `api-core/boottest` to build this API, verify its
health identity in isolated storage, and check shutdown. Keep the expected
service name in the scenario test. Shared process machinery and failure
regressions live in `packages/api-core/boottest`; see its package README section
for configuration and evidence limits. The existing E2E gate runs this test.
