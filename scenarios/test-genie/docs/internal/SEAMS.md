# test-genie seam registry

Seams declared in `scenarios/test-genie/api/internal/`. Drift is gated by
`// seam:` tags; if you add a new tag, add a row here.

## Playbooks phase

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `routingChecker` | `phases/phase_playbooks.go` (`eligibilityChecker` interface) | `*eligibility.Checker` | tests reassign the package var to a stub satisfying the interface | Routes the routed-vs-fallback decision; tests inject canned `Eligibility`. |
| `resolveScenarioRoutingClient` | `phases/phase_playbooks.go` | `routing_v1connect.NewRoutingServiceClient` against the scenario's discovered base URL | tests override to return a stub `RoutingServiceClient` | Connect-RPC client for the scenario's `RoutingService`. |
| `probeRoutingServiceEnabled` | `phases/phase_playbooks.go` | HTTP GET against the install procedure URL; 404 → `errRoutingServiceDisabled` | tests override to return `nil` or `errRoutingServiceDisabled` directly | Detects production-mode targets (route unmounted → fallback). |
| `extractTestDSN` | `phases/phase_playbooks.go` | strict per-driver DSN selection | unit-tested via `extractTestDSN_test.go` | Refuses to guess when `primaryDriver` is empty or the env doesn't carry the matching DSN. |
| `isolationManagerFactory` | `phases/phase_playbooks.go` | `isolation.NewManager` | tests inject a `fakeIsolation` | Stub isolation without requiring Docker. |

## Eligibility

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `eligibility.ResolveBaseURL` | `eligibility/auditor.go` | `discovery.ResolveScenarioURLDefault(ctx, "scenario-auditor")` | tests override the package var | scenario-auditor discovery indirection. |
| `eligibility.HTTPClient` | `eligibility/auditor.go` | `&http.Client{Timeout: 10s}` | tests can override | HTTP client used by auditor calls. |
| `eligibility.FetchRegisteredRules` | `eligibility/path_decision.go` | `GET /api/v1/rules` via `RequestJSON` | tests override to return a canned `map[string]struct{}` | Used by `AssertRulesObserved` to verify the routing rules are registered + enabled in the auditor. |
| `eligibility.Checker.Invalidate` | `eligibility/router.go` | per-scenario cache eviction | unit-tested in `path_decision_test.go` | Called from the playbooks defer so a successive run in the same test-genie process picks up code fixes. |

## Related docs

- [`docs/agent-system/routed-test-db.md`](../../../../docs/agent-system/routed-test-db.md)
  — End-to-end routed-vs-fallback path documentation.
- [`packages/api-core/docs/internal/SEAMS.md`](../../../../packages/api-core/docs/internal/SEAMS.md)
  — Substrate-level seams (`RoutedDB`, `Clock`, `TestModeMiddleware`,
  `devrouting.Register`).
