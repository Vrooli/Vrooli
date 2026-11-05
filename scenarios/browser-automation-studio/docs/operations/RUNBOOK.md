# Operations Runbook

Start with `make start` or `vrooli scenario start browser-automation-studio`, inspect with `make logs`, and stop with `make stop`. Do not run API or driver binaries directly.

For validation, invoke `vrooli scenario test browser-automation-studio`, record the returned run ID, and wait once with `test-genie runs wait --json browser-automation-studio <run-id>`. Cancellation does not abort a run.

If the driver is unhealthy, restart the scenario through lifecycle commands, then collect API/driver diagnostics. Do not bypass the sidecar supervisor by binding a replacement process to its port.
