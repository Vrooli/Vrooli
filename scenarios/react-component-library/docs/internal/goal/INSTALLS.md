# External installs

## 2026-10-01 — Browser Automation Studio CLI projection refresh

- Tool and size: `browser-automation-studio` CLI at `/home/matthalloran8/.vrooli/bin/browser-automation-studio` (36 MiB).
- Reason: orchestrator acceptance read of the retained desktop/mobile E1 workflow executions detected an outdated CLI projection; its owner refreshed the command from current BAS sources during the read.
- Authorization: orchestrator action within the already operator-authorized E1 acceptance work; no package/dependency install and no external process started.
- Cleanup: keep the managed CLI projection in place; removal is owned by the BAS CLI lifecycle and could break other consumers.
