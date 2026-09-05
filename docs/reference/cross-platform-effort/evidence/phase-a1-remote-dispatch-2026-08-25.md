# Phase A1 remote-dispatch evidence — 2026-08-25

The `hello-plugin` CLI manifest already had `binding.kind: local` on both
commands when Phase A1 started. The repair was therefore preserved rather
than duplicated.

## Automated evidence

- `go test ./internal/cli/scenariohandlers ./internal/cli/vroolicli ./internal/cli/scenariocli` — passed.
- `go test ./scopecatalog ./discovery` from `packages/api-core` — passed.
- `go test ./internal/dispatch` from `scenarios/vrooli-bridge/api` — passed.
- The focused remote-dispatch tests cover `status`, `start`, `restart`,
  `stop`, `wait`, and `port`, including variant and argument forwarding and
  human triage blocks for node failures.
- `BuildResilient` quarantines an invalid scenario manifest with its path and
  decode/validation reason while preserving valid scenario scopes. Bridge
  dispatch and its grant validator consume this resilient catalog.

## Live Bridge transcript

The Bridge was already healthy at `127.0.0.1:18767`. Running:

```text
vrooli scenario status minimouse/system-monitor --json
```

reached `minimouse` and returned a node-sourced `system-monitor` response:

```text
scenario.name=system-monitor
scenario.path=/Users/matthalloran8/vrooli/scenarios/system-monitor
runtime.status=stopped
```

Running:

```text
vrooli scenario start minimouse/system-monitor
```

returned a non-empty failure report after the remote relay deadline:

```text
✗ Failed to start 'minimouse/system-monitor'
  Error: remote scenario start system-monitor: relay scenario start to "minimouse": ... deadline_exceeded ...
  Full log: /home/matthalloran8/.vrooli/logs/minimouse/system-monitor.log
  Tail:     vrooli scenario logs minimouse/system-monitor --tail 100
```

The node was reachable for status but did not complete the start within the
relay deadline. This is valid typed remote failure evidence; it is not a
silent local fall-through.
