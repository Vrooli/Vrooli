# CLI Reference

The vrooli-events CLI is the governed local command surface for event
ingress, querying, live subscriptions, store inspection, and receipt-capture
reconciliation.

## Ingest an event

```bash
vrooli-events ingest --event-id "example.audit.001" --type "example.audit.v1" --source "example-scenario" --payload '{}'
```

Required flags are `--event-id`, `--type`, and `--source`. Optional flags are
`--target`, `--correlation-id`, and `--payload`.

## Query durable events

```bash
vrooli-events query --type "swarm-manager.**" --limit 10
vrooli-events query --source "agent-manager" --since "24h"
vrooli-events query --correlation-id "trace-abc123"
```

Available filters are `--type`, `--source`, `--correlation-id`, `--since`, and
`--limit`.

## Subscribe to live events

```bash
vrooli-events subscribe --type "**"
vrooli-events subscribe --type "swarm-manager.backlog.*"
vrooli-events subscribe --source "agent-manager"
```

Optional filters are `--type`, `--source`, and `--target`. The command keeps an
SSE connection open and writes matching events as they arrive.

## Inspect store health

```bash
vrooli-events stats
```

## Receipt-capture governance

Preview declarations and reconcile them with the active policy store:

```bash
vrooli-events capture-preview
vrooli-events capture-reconcile
```

The HTTP API provides policy snapshots and persistent subscription CRUD. See
[Managing Policies](../guides/managing-policies.md) and
[Creating Subscriptions](../guides/creating-subscriptions.md) for those
surfaces.
