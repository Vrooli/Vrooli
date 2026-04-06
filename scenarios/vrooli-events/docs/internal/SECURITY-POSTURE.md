# Security Posture

## Last Updated
2026-04-05

## Threat Model

vrooli-events operates in a **trusted internal network** within the Vrooli ecosystem. The PRD explicitly states that policy enforcement is governance, not adversarial isolation. All scenarios trust each other.

**In-scope threats**: Accidental misconfiguration, runaway event producers, resource exhaustion.
**Out-of-scope threats**: Malicious actors, credential theft, network-level attacks.

## Security Controls by Category

### Input Validation
| Control | Status | Location |
|---------|--------|----------|
| Request body size limit (1MB) | Active | `handlers.go:25` — `io.LimitReader(r.Body, 1<<20)` |
| Event type format validation | Active | `handlers.go` — regex check for `{a}.{b}.{c}.{d}` |
| Required fields check | Active | `handlers.go` — rejects empty `event_type`, `source_scenario` |
| Proto JSON unmarshal with `DiscardUnknown` | Active | Ignores unexpected fields instead of rejecting |
| Query param validation | Active | `limit` and `since` parsed as integers with error handling |

### Resource Protection
| Control | Status | Location |
|---------|--------|----------|
| Subscriber buffer cap (64) | Active | `broker.go` constant `subscriberBufSize` |
| Background pruning | Active | `pruner.go` — time-based and size-based retention |
| SSE write timeout disabled | Active | `main.go` — required for long-lived SSE connections |
| Query result limit | Active | Default limit of 100 events per query |

### Authentication & Authorization
| Control | Status | Notes |
|---------|--------|-------|
| API authentication | Not implemented | No auth on any endpoint. Relies on network-level isolation. |
| SSE authentication | Not implemented | `EventSource` API doesn't support custom headers. |
| Policy enforcement | Not implemented | Policy engine is a P1 feature (OT-P1-004 through OT-P1-009). |

### Data Protection
| Control | Status | Notes |
|---------|--------|-------|
| Encryption at rest | Not implemented | SQLite file is unencrypted. Relies on OS-level file permissions. |
| Encryption in transit | Not implemented | HTTP only (no TLS). Relies on internal network. |
| Payload sanitization | Not implemented | Event payloads stored as-is. No XSS filtering (payloads are machine-consumed). |

## Known Gaps

1. **No authentication**: Any process that can reach the API can ingest, query, and subscribe. Acceptable for internal use; would need auth for any external exposure.
2. **No rate limiting on ingestion**: A runaway producer could flood the store. Policy engine (P1) will address this.
3. **No TLS**: All communication is plaintext HTTP. Acceptable on localhost; would need TLS for cross-host deployment.
