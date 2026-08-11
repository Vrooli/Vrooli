# LPBS Integration Guide

This document describes the BAS integration with Landing Page Business Suite
(LPBS) for subscription identity, entitlement checks, metered AI, and usage
reporting.

## Overview

BAS uses LPBS as the authority for:

- the consumer identity and subscription tier;
- atomic credit authorization and charging for subscription-backed AI; and
- centralized usage reporting and administration.

Local model execution and operator BYOK remain available without an LPBS
request. Only the LPBS path is metered against the consumer subscription.

## Architecture

```
┌─────────────────────────────┐        ┌─────────────────────────────┐
│  Browser Automation Studio  │        │ Landing Page Business Suite │
│                             │        │                             │
│  ┌─────────────────────────┐ │      │ ┌────────────────────────┐ │
│  │ Shared credential       │ │      │ │ LPBS consumer authority │ │
│  │ authority               │─┼──────┼>│ RS256 access + refresh  │ │
│  │ refresh token only      │ │      │ │ JWKS + entitlements     │ │
│  └────────────┬────────────┘ │      │ └────────────┬───────────┘ │
│               │ access token │                   │               │
│  ┌────────────▼────────────┐ │      │ ┌───────────▼────────────┐ │
│  │ BAS / AI Gateway         │─┼──────┼>│ Metered inference       │ │
│  │ memory-only bearer       │ │      │ │ identity + atomic debit │ │
│  └─────────────────────────┘ │      │ └────────────────────────┘ │
│                             │        │                             │
└─────────────────────────────┘        └─────────────────────────────┘
```

## Identity and single sign-on

All deployment tiers use the same logical credential identity:
`vrooli/lpbs-account` / `refresh-token`.

- Tier 1 (local Vrooli server): a scenario provisions the native credential
  authority; every scenario on that machine resolves the same identity.
- Tier 2 (desktop bundle): the desktop runtime stores the refresh token in its
  native credential store and exposes an authenticated, declared-credential IPC
  surface to bundled services. BAS and AI Gateway use the same logical ID.
- Tier 3 (remote/VPS): the server keeps the refresh token in its deployment
  credential provider; LPBS is still the issuer and the access token is bound to
  the configured LPBS authority.

The browser stores only a short-lived access token in `sessionStorage`. Desktop
safeStorage protects the local profile copy, while the native credential
authority is the durable source of truth. Refresh rotation is serialized and a
replayed refresh token revokes its family.

The web callback requires the stored state value, removes the URL fragment
before making the provisioning request, and sends the refresh token only to the
same-origin BAS session endpoint. BAS APIs derive identity from verified LPBS
claims and reject mismatched query/header identities.

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `BAS_ENTITLEMENT_SERVICE_URL` | LPBS service base URL for hosted or local authority | `http://localhost:15000` or `https://vrooli.com` |
| `BAS_LPBS_SERVICE_SECRET` | Optional service credential for legacy usage reporting | Must match the LPBS service secret; never a consumer JWT |
| `LPBS_METERED_INFERENCE_URL` | AI Gateway's LPBS metered inference authority | LPBS base URL |
| `APP_BUNDLE_KEY` | App identifier for usage records | `browser-automation-studio` (default) |

## Usage Sync Flow

1. **Resolve**: BAS or AI Gateway resolves the shared refresh token through the
   credential authority and rotates it at LPBS when the in-memory access token
   is near expiry.
2. **Verify**: LPBS verifies the RS256 access token against its public key set,
   derives the consumer identity from claims, and checks the subscription.
3. **Charge**: LPBS authorizes and charges metered inference atomically.
4. **Report**: BAS may separately report operational usage with an idempotent
   service-to-service operation ID.
5. **Degrade safely**: a missing/expired subscription token cannot silently
   become another user or a paid request; local/BYOK providers may still be
   selected according to policy.

## Idempotency

To prevent double-counting when retries occur (network errors, timeouts), BAS generates a unique `operation_id` for each charge operation. This ID is:
- Generated once when the charge occurs
- Reused across all retry attempts
- Stored in LPBS to detect duplicate reports

```go
// BAS generates operation_id ONCE
operationID := uuid.New().String()

// Same ID used for all retries
for attempt := 0; attempt < maxRetries; attempt++ {
    report := LPBSUsageReport{
        OperationID: operationID, // Same ID every retry
        // ... other fields
    }
    err := sendToLPBS(report)
    if err == nil {
        break
    }
    time.Sleep(backoff)
}
```

LPBS checks for duplicate `operation_id` before recording:
```sql
-- If operation_id exists, return success without incrementing
SELECT EXISTS(SELECT 1 FROM usage_records WHERE operation_id = $1)
```

## Health Checks

### BAS Health Check
```bash
# Check if BAS can reach LPBS
curl http://localhost:16000/api/v1/credits/lpbs-health
```

Response:
```json
{
  "configured": true,
  "reachable": true,
  "last_sync": "2026-01-21T10:30:00Z",
  "last_error": ""
}
```

### LPBS Health Check
```bash
# Check LPBS usage service health
curl http://localhost:15000/api/v1/usage/health
```

Response:
```json
{
  "healthy": true,
  "database_connected": true,
  "last_record_at": "2026-01-21T10:30:00Z",
  "records_this_period": 1234
}
```

## API Reference

### Report Usage (BAS -> LPBS)

**Endpoint**: `POST /api/v1/usage/report`

**Headers**:
- `Authorization: Bearer {LPBS_SECRET}`
- `Content-Type: application/json`

**Request**:
```json
{
  "user_identity": "user@example.com",
  "limit_key": "ai_credits",
  "amount": 500000,
  "app_bundle_key": "browser-automation-studio",
  "operation_id": "550e8400-e29b-41d4-a716-446655440000",
  "metadata": {
    "operation": "ai.workflow.generate",
    "model": "gpt-4-turbo",
    "prompt_tokens": 1500,
    "is_byok": false
  }
}
```

**Response**:
```json
{
  "success": true
}
```

## Testing

### Unit Tests

BAS includes mock reporters for testing without real LPBS:

```go
reporter := &mockLPBSReporter{}
svc := credits.NewService(credits.ServiceOptions{
    LPBSReporter: reporter, // Use mock instead of HTTP
})

// After operations, verify reports
assert.Len(t, reporter.reports, 1)
assert.Equal(t, "test@example.com", reporter.reports[0].UserIdentity)
```

### Integration Tests

To test end-to-end:

1. Start LPBS: `make start` in `scenarios/landing-page-business-suite`.
2. Set environment:
   ```bash
   export BAS_ENTITLEMENT_SERVICE_URL=http://localhost:15000
   export LPBS_METERED_INFERENCE_URL=http://localhost:15000
   ```
3. Provision the consumer session through the BAS web or desktop sign-in flow;
   do not put a refresh token in shell history.
4. Run BAS tests with `vrooli scenario test browser-automation-studio`.

### Verifying Idempotency

```bash
# Send same operation_id 3 times
for i in 1 2 3; do
  curl -X POST http://localhost:15000/api/v1/usage/report \
    -H "Authorization: Bearer $LPBS_SECRET" \
    -H "Content-Type: application/json" \
    -d '{
      "user_identity": "test@example.com",
      "limit_key": "ai_credits",
      "amount": 100000,
      "app_bundle_key": "bas",
      "operation_id": "test-uuid-123"
    }'
done

# Verify: usage_amount should be 100000, NOT 300000
curl "http://localhost:15000/api/v1/usage/summary?user_identity=test@example.com"
```

## Troubleshooting

### "LPBS report failed after retries"

Check:
1. LPBS is running: `curl http://localhost:15000/api/v1/usage/health`
2. Secret matches: `LPBS_SECRET` == `LPBS_SERVICE_SECRET`
3. Network connectivity between services

### Usage not appearing in LPBS

Check:
1. BAS logs for "lpbs: usage report sent" (debug level)
2. LPBS logs for "usage_recorded"
3. Operation completed successfully locally (check BAS `operation_log` table)

### Double-counted usage

This shouldn't happen with idempotency, but if it does:
1. Check that `operation_id` is being generated (look for UUID in logs)
2. Verify LPBS has the `operation_id` column (run schema migration)
3. Check for LPBS version mismatch
