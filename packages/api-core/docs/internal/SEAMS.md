# api-core seam registry

Seams declared in `packages/api-core/`. Drift is gated by `// seam:` tags in
the source files; if you add a new tag, add a row here.

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `database.RoutedDB` | `database/routed.go` | `database.Open` wraps `*sql.DB` | tests construct against in-memory drivers | Persistence substrate seam: test-genie installs and clears a runtime test pool here. |
| `database.Clock` | `database/routed.go` (`Clock` interface) | `database.SystemClock()` (`systemClock`) | `routed_lease_test.go` `fakeClock` | Ambient time used for lease-TTL expiry. Injected via `OpenWithClock`. |
| `database.RoutedDB.PinTestPool` | `database/routed.go` | caller supplies the expected active lease ID | routed lease tests | Returns a context bound to the exact installed `*sql.DB` and owner. Pass it to all later RoutedDB calls; stale, expired, replaced, or foreign-owner pins fail closed and never select a replacement or primary pool. `PoolForContext` honors the same pin. |
| `filerouting.RoutedRoots.PinTestRoots` | `filerouting/routed.go` | caller supplies the DB lease ID expected by the operation | routed root tests | Returns a context bound to the current root owner, lease ID, and install generation. A same-ID reinstall is a new generation. `Pick`, `PickRequired`, and `RecordWrite` honor the binding; stale/expired/foreign bindings cannot fall through. |
| `apihttp.TestModeMiddleware` | `apihttp/testmode.go` | wraps any `http.Handler`; no-op in production mode | tests can wrap a stub handler | Marks an in-flight request as test-mode for `RoutedDB.pick`. Dev/prod is frozen at wrap time. |
| `devrouting.Register` | `devrouting/service.go` | mounts the Connect-RPC handler on dev mode | tests use `httptest.NewServer` + a `*RoutedDB` over in-memory sqlite | Dev-only admin surface for runtime test-pool install/clear/heartbeat. Returns false in production. |
| `devrouting.Mux` | `devrouting/service.go` | `*http.ServeMux` or `*gorilla/mux.Router` | tests pass `*http.ServeMux` | Minimal mux interface so callers don't have to commit to a specific router. |

## Related docs

- [`scenarios/storage-manager/docs/concepts/test-isolation-contract.md`](../../../../scenarios/storage-manager/docs/concepts/test-isolation-contract.md)
  — The canonical test-isolation contract: how test-genie's playbooks phase uses these seams end-to-end.
