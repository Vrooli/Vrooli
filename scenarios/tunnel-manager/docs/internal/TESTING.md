# Testing — Tunnel Manager

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

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/features/health/HealthCard.test.tsx](../../ui/src/features/health/HealthCard.test.tsx)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## TL;DR — the canonical examples

These files are the source of truth. When in doubt, copy their shape:

- **API**: `api/handlers/health/handler_test.go` — table-driven, real
  middleware via `httpx.NewLiveServer`, fake pinger from `mocks/`,
  typed-proto decode via
  `assertx.MustUnmarshalProto[healthv1.Response]` (the wire shape lives
  in `packages/proto/schemas/tunnel-manager/v1/health/health.proto`;
  assert on typed proto fields, not `map[string]any` chains). For
  endpoints whose wire shape isn't in proto yet, `MustDecodeJSON[T]`
  is the fallback — but adding the proto first is the right move.
- **UI composition**: `ui/src/App.test.tsx` — smoke-only composition
  test. App composes shell + features; feature behaviour belongs beside
  the feature.
- **UI feature**: `ui/src/features/health/HealthCard.test.tsx` —
  `renderWithProviders`, factory data, inline `vi.mock` factory
  closure, cimode assertions, and real-locale assertions.
- **UI a11y**: `ui/src/components/AppShell.a11y.test.tsx` and
  `ui/src/features/health/HealthCard.a11y.test.tsx` — shell and feature
  accessibility are tested at their ownership boundary.
- **CLI**: `cli/app_test.go` — smoke gate (NewApp, --version, --help).
  When domain commands arrive, extend with `clitest.NewAPIServer` +
  `clitest.CaptureStdout` from `cli/internal/testutil/`.

If your test doesn't look like one of those three, ask why before
shipping.

## Product Test Strategy

> **Status: implemented and evolving.** Product tests now cover the seven
> domains ([`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)), scheduler
> behavior, config readiness, UI operator workflows, and CLI manifest
> parity. The inventory below is the durable strategy for maintaining and
> extending that coverage; every entry reuses a canonical pattern above
> rather than introducing a new test primitive.

### Per-layer plan

| Layer | Pattern (reused from above) | What the product domains will test |
|---|---|---|
| Go repository | Real sqlite via `db.NewSQLite(t)` + `apidb.EnsureSchemas` (compose pattern) | `routes`/`exposure`/`config`/`tunnel`/`probes`/`recovery` SQL semantics: manifest CRUD + one-route-per-subdomain, lease lifecycle rows, metrics time-series insert/query, probe-history append, recovery-event append. |
| Go service | Domain fakes + `FakeClock` where time matters | Exposure tier policy (CORE never-expire, LEASED TTL/extend/revoke), reaper eligibility, route validation (DNS-label subdomain, port-matches-fixed-port), failure classification rules. |
| Go service (seam-driven) | Domain seam fakes (`FakeCloudflareAPI`/ingress client, `FakeCommandRunner`, `FakeProber`, metrics/client fakes, core-set/lifecycle/audit readers) | Cloudflare ingress sync (idempotent re-push, remote-unavailable setup path), tunnel health + degraded-mode thresholds, probe scheduling + classification, ensure-running delegation, port-compliance findings. |
| Connect handler | `connectxtest.StartTestServer` + `FakeService` + buffer logger | Per-domain RPC routing, request/response proto shapes, Connect error envelope (codes per [`ERROR-HANDLING.md`](ERROR-HANDLING.md)), 500-path operator-log assertion. |
| CLI | `clitest.NewAPIServer` + `clitest.CaptureStdout` | `tunnel`/`routes`/`exposure`/`probes`/`audit`/`recovery`/`config` — success and error paths, both human-render and `--json` proto contract. |
| UI vitest | `renderWithProviders` + inline `vi.mock("./api/<domain>")` + factories | Overview, Settings/Setup, Exposure, Recovery, Metrics/Diagnostics, and Audit: render, loading/error/setup states, filtering, sync/reconcile feedback, and safe recovery guidance. |
| UI a11y | `expectNoA11yViolations(container)` at the feature boundary | Each dashboard feature's accessibility at its ownership boundary; `App.test.tsx` stays composition-smoke only. |
| Integration | E2E binary smoke gate (`//go:build e2e`) + seam-driven flow tests | Lease lifecycle (request → ensure-running → ingress → expiry → reap) and recovery flow (probe-fail → classify → restart-via-seam → backoff → circuit-break), all behind fakes. |

### [REQ:ID] tagging convention

Every product test ties back to a requirement under `requirements/`.
Tag the test name or a leading comment with the requirement id so the
test↔requirement mapping is greppable and the requirements suite can
report coverage:

```go
// [REQ:03-exposure-tiers] CORE-tier routes are never reaped.
func TestReaper_SkipsCoreRoute(t *testing.T) { ... }
```

```tsx
// [REQ:08-... ] lease extend updates the expiry shown on the Exposure surface
it("[REQ:08] extends a lease", async () => { ... });
```

Use the requirement file's id (e.g. `01-exposure-manifest`,
`03-exposure-tiers`, `04-port-compliance`, `05-tunnel-health`,
`06-liveness-probes`, `07-auto-recovery`) as listed in the domain table
in [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md). A requirement
with no tagged test is the signal to write one — not to lower the gate.

### Coverage expectations

The thresholds in the "Coverage thresholds" section above apply
unchanged to product code: UI 85%, API 75%, CLI 75%, with
`internal/testutil/...` excluded from the Go denominator. Raise toward
80%/85% as the domains stabilise.

### Auto-recovery must be tested behind the seam — never against real cloudflared

This is the load-bearing testing rule for this scenario. Auto-recovery
is **LIVE from day one** ([`DECISIONS.md`](DECISIONS.md)) and Tunnel
Manager is the single authoritative owner of cloudflared restart. Tests
of the recovery engine MUST drive the `FakeCommandRunner` and
`FakeCloudflareAPI` seams (see [`SEAMS.md`](SEAMS.md)) and assert on
recorded invocations:

- ✅ Assert "recovery issued `vrooli resource restart cloudflared` exactly
  once, then backed off, then circuit-broke after N attempts" against
  the fake's recorded argv log.
- ✅ Drive failure classification with `FakeProber` / `FakeMetricsSource`
  canned results so every classification → action mapping is reached.
- ❌ **Never** let a test invoke a real managed-resource restart, push real
  Cloudflare ingress, or probe a real public URL. Restarting the managed
  cloudflared resource from a test would take down remote access for the whole
  Vrooli instance — the exact blast radius the circuit breaker exists to
  bound (see [`SECURITY.md`](SECURITY.md)). The seams make the live path
  unreachable from tests by construction.
