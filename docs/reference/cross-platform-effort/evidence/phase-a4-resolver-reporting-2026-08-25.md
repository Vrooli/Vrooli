# Phase A4 resolver and reporting defects — 2026-08-25

Implemented and exercised the four reporting corrections:

- controls-only capabilities resolve from all required controls; the five
  `process-containment` controls resolve as implemented on Linux and incomplete
  on hosts where controls are absent;
- architecture-specific `platform_policies` are applied per grid cell;
  `hardware-error-telemetry/linux/arm64` now reports `ineligible` with policy
  `no_equivalent_ever`;
- authored scenario capability overrides accept `essential: false` by default,
  degrade for non-essential unsupported capabilities, block only essential
  unsupported capabilities, and name every unsupported capability in the
  reason;
- the portability renderer prints host OS and architecture for every cell;
  fleet desktop bundling classifies real resource manifests and errors when no
  resources exist.

Validation:

```text
$ go test ./internal/deployability/...
PASS
$ cd scenarios/infrastructure-manager/api && go test ./internal/portability/... ./handlers/portability/...
PASS
$ cd scenarios/scenario-dependency-analyzer/api && go test ./internal/deployment/...
PASS
$ vrooli capability ledger --json
hardware-error-telemetry linux/arm64: ineligible, policy=no_equivalent_ever
process-containment linux/amd64: implemented, all 5 control declarers resolve
$ vrooli capability fleet --json
blocked_by_os=15 docker_blocked=7 desktopBundling.resources=29 reason="desktop bundling classified 29 resources"
system-monitor is absent from blocked_by_os
```

The fleet owner scenarios were restarted through `vrooli scenario restart`;
Ollama remained a degraded optional resource during the restart, but the
infrastructure-manager and dependency-analyzer services reported healthy.
