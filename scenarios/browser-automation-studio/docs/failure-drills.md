# Controlled failure drills

Browser Automation Studio exposes a small development-only recovery
qualification surface. It is for proving capture/session failure behavior; it
is not a general driver administration or chaos-testing API.

Run the catalog and a drill through BAS:

```bash
browser-automation-studio drills list --json
browser-automation-studio drills run --name DRILL_NAME_DRIVER_UNAVAILABLE --json
```

Supported drills are `DRILL_NAME_DRIVER_UNAVAILABLE`,
`DRILL_NAME_PARTIAL_INITIALIZATION`, `DRILL_NAME_CAPACITY`, and
`DRILL_NAME_EXPIRY`. A verdict preserves the primary controlled outcome,
assertions, cleanup result, and redacted pre/post fault snapshots.

## Safety contract

The driver test-control routes accept only loopback requests carrying the
sidecar's administrative secret, and reject `NODE_ENV=production`. BAS keeps
that secret in memory only; the CLI never receives it. Every arm uses a new
opaque token, a bounded TTL, and a remaining-use limit. Tokens never appear in
snapshots or verdict evidence.

Fault injection is limited to these lifecycle seams:

| Fault | Expected effect | Cleanup guarantee |
|---|---|---|
| Driver unavailable | A session admission returns an honest failure. | No session is created. |
| Partial initialization | A post-registration failure returns after reconciliation. | The registered session is force-closed. |
| Capacity | A token-scoped admission returns `429`. | Global `MAX_SESSIONS` is unchanged. |
| Expiry | An unused arm becomes unavailable after TTL. | The fault is removed. |

The drill runner disarms in a deferred cleanup path even if execution fails.
Normal observability configuration, including `MAX_SESSIONS` and pool size,
cannot create these faults.

## Troubleshooting

`FAULT_CONTROL_FORBIDDEN` means the driver is not local, lacks its managed
administrative secret, or is production-gated. Restart BAS through its normal
scenario lifecycle and retry. A failing verdict includes its evidence JSON;
inspect `browser-automation-studio observability sessions --json`, then run a
normal `capture` afterward to prove recovery before retrying a drill.

Implementation: [CODE: playwright-driver/src/fault-control/controller.ts],
[CODE: api/handlers/drills/module.go], and [CODE: cli/drills/register.go].
