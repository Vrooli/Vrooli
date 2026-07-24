# Test-Isolation Contract

This is the canonical, durable contract for how a scenario's database and
file storage are isolated during test-genie's destructive (playbooks / E2E) phase. It is
owned by `storage-health` and migrated here from the former
`docs/agent-system/routed-test-db.md`.

The contract is **static** and the gate is **fail-closed**: isolation is
proven by static seam-wiring analysis, not by a runtime probe or a scenario
restart. When isolation cannot be statically proven, test-genie **refuses**
destructive playbooks (loud, instructive skip) rather than risk mutating a
real database. There is no restart-based fallback.

## The routed path (the only path)

A scenario routes test traffic to per-run isolated SQL and file stores by wiring
the `api-core` seams. workflow-health installs the lease on the **running**
process at runtime via a Connect-RPC call (`InstallTestPool`); the scenario
is never restarted. Static proof that all four seams are wired ⟹ the routed
path is eligible ⟹ the test pool installs on the live process ⟹ isolation
by construction.

If isolation is **not** statically proven (a seam is unwired, or the API is
non-Go and therefore unverifiable), the destructive playbooks are skipped
fail-closed. Read-only playbooks remain allowed where declared.

## How a scenario qualifies — the four-seam cookbook

Turning a `*sql.DB`-based scenario into a routed-eligible (isolation-safe)
one is four changes. `storage-health`'s L2 analyzers check exactly these:

1. **`database.Open → *RoutedDB`.** Replace `database.Connect` with
   `database.Open`; the returned type is `*database.RoutedDB`.
2. **No captured raw handles.** Change struct fields, package-level vars,
   and constructor parameters from `*sql.DB` to `*database.RoutedDB` or a
   narrow consumer-owned `SQLExecutor`-shaped interface that both `*sql.DB`
   and `*RoutedDB` satisfy (the interface keeps test fixtures on `*sql.DB`).
3. **`apihttp.TestModeMiddleware`.** In `main`, mount the middleware around
   the API handler. It honors the test-mode header and self-disables in
   production mode.
4. **`devrouting.RegisterWithFileRoots(rootMux, db, roots)`.** Construct
   `filerouting.RoutedRoots` from startup class roots and register the dev-only
   `RoutingService` against both routing seams. It self-disables in production
   mode.
5. **Context-aware file paths.** File-persisting stores resolve their class root
   through `RoutedRoots.Pick(ctx, class)` at read and write time; they do not
   capture a startup root string.

`database.EnsureSchemas` applies the embedded schema; it is the schema-side
companion to these seams.

No UI changes are needed — the test-mode header is attached at the browser
context (Playwright `extraHTTPHeaders`), so the scenario's existing fetch /
Connect-RPC client picks it up automatically. No `.vrooli/service.json`
change is needed unless deploying to production: `mode` defaults to
`"development"`, which already enables the dev-only surfaces. The
`react-vite` template ships in this shape — new scenarios are born
compliant.

## Header semantics

The test-mode header is exactly:

```
X-Vrooli-Test-Mode: 1
```

Only the literal value `1` opts in. Anything else (`true`, `yes`, empty,
missing) is ignored. `TestModeMiddleware` reads the header, calls
`database.WithTestMode(ctx)` on the request context, and `*RoutedDB.pick(ctx)`
serves the request from the test pool.

The header name and value are exported as shared constants from
`packages/api-core/apihttp` (`apihttp.TestModeHeader`,
`apihttp.TestModeValue`). All callers — middleware, test-genie, anything
else — reference those constants rather than re-stringing the literal.

### How the header reaches the API during a routed run

1. workflow-health installs the target lease, then builds
   `{apihttp.TestModeHeader: apihttp.TestModeValue}` for every BAS execution,
   including observer-labeled cases.
2. The runner forwards it as `ExecutionParams.ExtraHeaders`, which the BAS
   execution client serializes into the request as
   `parameters.browser_profile.extra_headers`.
3. BAS's `context-builder` merges those into the Playwright browser context
   as `extraHTTPHeaders`. Playwright attaches the header to **every** HTTP
   request the page makes — page loads, `fetch`, XHR, same- or cross-origin
   — for the lifetime of the context.
4. Each UI→API call carries `X-Vrooli-Test-Mode: 1`; `TestModeMiddleware`
   flips the context flag; `RoutedDB` serves from the installed test pool.
   The UI is unmodified and the scenario does not restart.

## Mode flag

`.vrooli/service.json` may carry an optional top-level `mode` field. **It
defaults to `"development"` when omitted** — so a new scenario gets the
dev-only surfaces (TestModeMiddleware + RoutingService) automatically and
doesn't need to touch `service.json` to be routed-eligible.

| Value | Effect |
|---|---|
| `"development"` *(default — also used when the field is absent, unparseable, or any unrecognized value)* | `TestModeMiddleware` honors `X-Vrooli-Test-Mode: 1`; `RoutingService` is mounted. |
| `"production"` | Both surfaces self-disable; the `RoutingService` route returns 404. |

`projectmeta.Mode()` (in `packages/api-core/projectmeta`) resolves the
value by walking up from the running binary's cwd to the nearest
`.vrooli/service.json`. **Deploy guidance:** explicitly set
`mode: "production"` in production; the "safe default" keeps local dev and
`vrooli scenario start` working, but production deployments must opt into
the locked-down mode.

`VROOLI_TEST_MODE_FORCE_ENABLE=1` is the documented escape hatch: it opens
the dev-only surfaces regardless of `service.json` mode (intended for CI
where toggling the file isn't ergonomic; not for production).

## RoutingService contract

The dev-only service lives in proto schema
`packages/proto/schemas/dev-routing/v1/routing/routing.proto`:

```proto
service RoutingService {
  rpc InstallTestPool(InstallTestPoolRequest) returns (InstallTestPoolResponse);
  rpc ClearTestPool(ClearTestPoolRequest)     returns (ClearTestPoolResponse);
}
```

`packages/api-core/devrouting.RegisterWithFileRoots(mux, db, roots)` mounts
the handler against both `*database.RoutedDB` and `*filerouting.RoutedRoots`.
workflow-health calls `InstallTestPool(dsn=…, lease_id=<runID>)` at the start
of each routed workflow run and
`ClearTestPool(lease_id=<runID>)` in its defer block; the scenario is never
restarted.

### Lease contract (concurrency guard)

`InstallTestPoolRequest.lease_id` / `ClearTestPoolRequest.lease_id` carry
the test-genie run UUID. `RoutedDB` rejects:

- `InstallTestPool` with `ErrLeaseConflict` (Connect-RPC `AlreadyExists`,
  `active_lease_id` populated) when a pool is already installed under a
  different lease.
- `ClearTestPool` / `HeartbeatTestPool` with `ErrLeaseMismatch`
  (`FailedPrecondition`) when the lease does not match the owner.
- Same-lease `InstallTestPool` is idempotent, so a retried RPC after a
  transient transport failure does not break.

The authoritative concurrency check lives in test-genie's
`internal/playbooksclaims` package (DB-backed lease + heartbeat);
`RoutedDB`'s lease check is the defensive secondary so a direct RPC caller
cannot bypass the orchestrator.

### Lease TTL + heartbeat

`RoutedDB` records an absolute `expiresAt` per install.
`InstallTestPoolRequest.lease_ttl_ms` sets the initial TTL; zero means "use
the scenario default" (currently 90s). `pick(ctx)` enforces the TTL: a
request arriving after expiry is served from the primary pool and the lease
is cleared. test-genie heartbeats at TTL/3 via `HeartbeatTestPool`. If the
orchestrator crashes between Install and Clear, the lease expires within one
TTL and routing reverts to the primary pool without operator intervention.

### Transactions

`(r *RoutedDB).BeginTx(ctx, opts)` binds the returned `*sql.Tx` to whichever
pool was picked at that moment. A transaction cannot span the primary and
test pools.

## In-run defense-in-depth: the leak counter

Static proof makes the routed path eligible; the routed leak-counter is the
*runtime* secondary check that the path actually fired. After every routed
run, `ClearTestPool` returns a `LeaseStats` snapshot:

| Counter | Meaning |
|---|---|
| `test_pool_requests` | Requests served from the installed test pool — proof routing actually fired. |
| `primary_during_test_mode_requests` | Requests that carried `X-Vrooli-Test-Mode: 1` but were served from the **primary** pool — a sign some code path holds a raw `*sql.DB` instead of going through `RoutedDB`. |
| `test_root_writes` | Successful file writes routed into the leased temporary roots. |
| `primary_root_writes_during_test_mode` | Successful file writes that carried test mode but reached primary roots. This is a hard leak. |

test-genie promotes both to hard failures by default:

- `primary_during_test_mode_requests > 0` or
  `primary_root_writes_during_test_mode > 0` — always a real defect (some call
  site holds a raw `*sql.DB`). This is exactly what `storage-health`'s L3
  `SQL_DB_HANDLE_CAPTURE` / `RAW_SQL_OPEN` analyzers catch statically, ahead
  of the run.
- `test_pool_requests == 0` — the run never exercised the test pool (header
  didn't round-trip, playbooks never touched the API, or they're
  intentionally read-only).

A scenario opts out with `playbooks.allow_empty_test_pool: true` in
`.vrooli/testing.json` (both signals downgrade to warnings). The opt-out is
for legitimately UI-only playbooks or scenarios mid-migration; new scenarios
should leave it unset.

## Why static, not a probe

The previous design had a restart-based fallback: stop the scenario, restart
it with env pointing at a test DB, run playbooks, restart back. It restarted
the scenario with **no verification** that it actually connected to the test
DB, the worst case (a non-cooperating scenario) silently mutated real data
with zero detection, and it was slow. It is **deleted**, not deprecated. The
contract is now: prove isolation statically, or fail closed.

## Cross-References

- `storage-model.md` — engines, namespace seams, schema convention, ladder.
- `overview.md` — concept index.
- `.vrooli/maturity.json` — L2 findings (`ROUTED_SEAMS_UNWIRED`,
  `STORAGE_NAMESPACE_HARDCODED`, `STORAGE_ISOLATION_UNVERIFIED`).
- `../../PRD.md` — OT-P0-002 (static isolation proof + fail-closed gate),
  OT-P0-006 (`prove-isolation` CLI).
