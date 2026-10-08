# Investigation catalog

system-monitor ships a portable catalog in its API binary. It does not discover
product entries by walking the Vrooli source tree.

The built-in catalog contains six entries:

- `container-health` and `master-system-sweep` are the two shell-gated escape hatches.
- `service-health-monitor`, `service-config-validator`, `network-anomaly-detector`, and `resource-leak-detector` use typed native collectors.

Operator-authored entries are JSON files under the scenario state directory,
resolved through `api-core/storage` at `state/vrooli/system-monitor/investigations`.
An operator entry with the same id replaces the built-in entry for that machine.

Use the API or CLI instead of running a source-tree script directly:

```bash
system-monitor investigations catalog --json
system-monitor investigations run master-system-sweep --json
system-monitor investigations runs --limit 5 --json
system-monitor investigations runs-prune --dry-run --json
```

Every execution persists in `investigation_runs` and its findings in
`investigation_findings`. The seven-day retention policy is enforced by the
database-backed pruner. Stdout is one JSON document, progress belongs on stderr,
and persisted/emitted timestamps are RFC 3339 UTC.
