# Error Handling

## Error Categories

| Category | Handling |
|---|---|
| Contract errors | Return clear validation findings with stable codes |
| Missing scenario | Return not found at API boundaries |
| Unsupported doc operation | Return a client error naming the unsupported operation |
| Resource outage | Surface degraded capability without hiding dependency failure |
| Agent job failure | Persist terminal failure state and expose it through status endpoints |

## Recovery Paths

Operators should use health/audit output first, then inspect scenario logs and
resource health if the failure is integration-related.
