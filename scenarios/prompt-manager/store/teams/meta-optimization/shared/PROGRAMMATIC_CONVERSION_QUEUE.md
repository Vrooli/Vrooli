# Programmatic Conversion Queue - Retired

This queue has been retired. `ACTION_CONVERSION_QUEUE.md` is the authoritative register for deterministic-operation promotion.

The old "thin wrapper over scenario CLI" endpoint is no longer the final conversion target. The current pipeline is:

```text
skill prose -> Vrooli-controlled CLI implementation -> Action contract -> skill collapse or retirement when appropriate
```

Historical adjacent note moved forward: `swarm-manager-backlog-tools` remains a trim candidate in `SKILL_AUDIT.md`; it is not an Action candidate until one stable CLI command owns the deterministic operation.
