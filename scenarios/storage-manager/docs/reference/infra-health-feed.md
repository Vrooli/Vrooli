# Storage infra-health feed

Storage-manager exposes the typed host storage feed at:

```text
GET /api/v1/infra-health/storage?root=/
```

The response is read-only and uses the latest persisted census, recovery
ledger, retention receipts, and writer snapshots. It does not start a
filesystem walk. Byte fields are integer byte counts.

The feed supports these headroom cells:

| Cell | Meaning |
|---|---|
| H1 | Device census availability and freshness |
| H2 | Growth slope and trust of the measurement |
| H3 | Declared and enforced ceiling coverage |
| H4 | Recovery efficacy |
| H5 | Budget truth and enforcement state |
| H6 | Hot governed-root writers |

When storage-manager is unavailable, consumers must report an unavailable
source and the reason. They must not convert missing data to zero growth or
zero usage.
