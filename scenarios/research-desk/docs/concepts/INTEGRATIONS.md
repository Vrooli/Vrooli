# Integrations — Research Desk


Pointer. `.vrooli/service.json` declares five optional scenario dependencies:
`web-console`, `vrooli-events`, `data-backup-manager`, `agent-manager`, and
`notification-hub`. Data Backup Manager uses `startup_policy: try_start`; the
other four use `ignore`. The capability registry reports runtime reachability
for its supported capabilities; declaration alone does not mean a peer is
available. Purpose and degradation behavior are in [`SYSTEM.md`](SYSTEM.md)
and [`../DECISIONS.md`](../DECISIONS.md). No external resource is declared.

## Cross-References

- [`SYSTEM.md`](SYSTEM.md)
- [`../DECISIONS.md`](../DECISIONS.md)
