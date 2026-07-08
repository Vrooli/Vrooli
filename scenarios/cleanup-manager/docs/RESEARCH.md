# Research — Cleanup Manager

## Fit Summary

Cleanup Manager fills the cleanup policy/orchestration gap between
existing adjacent capabilities:

- `system-monitor` observes disk pressure and system metrics.
- `vrooli-autoheal` detects health failures and may request remediation.
- `storage-health` validates storage architecture and test isolation.
- `workspace-sandbox` owns sandbox-specific lifecycle and garbage
  collection.

Cleanup Manager should not duplicate those responsibilities. It owns the
central cleanup contract: provider metadata, preview-first planning,
policy gates, approval, replay-safe apply, and immutable audit.

## Ecosystem Role

- Role: meta-scenario / interface-enabler.
- Interfaces: programmatic CLI and Connect API, direct operator UI, and
  future agentic/action discovery.
- Done obligations: stable CLI/API contracts for other scenarios, a
  production-usable UI for operators, and provider/action metadata that
  makes cleanup capabilities discoverable without broad shell recipes.
- Compound-value seam: scenario-owned providers let future scenarios
  expose private cleanup safely while Cleanup Manager keeps policy and
  audit centralized.

## Safety Notes

- Preview-first is mandatory: providers without Estimate and Preview are
  rejected.
- Default policy is conservative. Docker volumes, live databases, model
  stores, and scenario-private data are disabled unless an owner/provider
  contract and explicit operator policy allow them.
- Tests must use injected seams, fake executors, fake Docker/journal
  clients, and temp roots. No test may run host cleanup commands.

## References

- Source plan: `/home/matthalloran8/.vrooli/plans/cleanup-manager-scenario-implementation.md`
- `scenarios/system-monitor/PRD.md`
- `scenarios/vrooli-autoheal/docs/reference/checks/system-disk.md`
- `scenarios/workspace-sandbox/docs/reference/configuration.md`
- `scenarios/storage-health/PRD.md`
