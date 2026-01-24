# LPBS Integration Guide

This document describes how Browser Automation Studio (BAS) integrates with the Landing Page Business Suite (LPBS) for centralized usage tracking.

## Overview

BAS reports AI credit usage to LPBS for:
- Centralized billing and usage tracking across all apps
- Subscription tier limit enforcement
- Admin dashboards and analytics

## Architecture

```
┌─────────────────────────────┐        ┌─────────────────────────────┐
│  Browser Automation Studio  │        │ Landing Page Business Suite │
│                             │        │                             │
│  ┌─────────────────────┐   │        │   ┌───────────────────┐    │
│  │  Credit Service     │   │  HTTP  │   │   Usage Service   │    │
│  │  ─────────────────  │───┼───────>│   │   ─────────────── │    │
│  │  - Charge locally   │   │  POST  │   │   - Record usage  │    │
│  │  - Report to LPBS   │   │        │   │   - Check limits  │    │
│  │  - Retry on failure │   │        │   │   - Idempotency   │    │
│  └─────────────────────┘   │        │   └───────────────────┘    │
│                             │        │                             │
└─────────────────────────────┘        └─────────────────────────────┘
```

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `LPBS_URL` | LPBS service base URL | `http://localhost:15000` or `https://vrooli.com` |
| `LPBS_SECRET` | Service-to-service auth token | Must match `LPBS_SERVICE_SECRET` on LPBS |
| `APP_BUNDLE_KEY` | App identifier for usage records | `browser-automation-studio` (default) |

## Usage Sync Flow

1. **Local Charge**: BAS charges credits locally in its `credit_usage` table
2. **LPBS Report**: After successful local charge, BAS asynchronously reports to LPBS
3. **Idempotency**: Each report includes an `operation_id` (UUID) to prevent double-counting on retries
4. **Retry Logic**: If LPBS is unreachable, BAS retries with exponential backoff (500ms, 1s, 2s)
5. **Fail-Safe**: Local operations succeed even if LPBS reporting fails

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

1. Start LPBS: `cd scenarios/landing-page-business-suite && make start`
2. Set environment:
   ```bash
   export LPBS_URL=http://localhost:15000
   export LPBS_SECRET=test-secret
   ```
3. Run BAS tests: `cd scenarios/browser-automation-studio && make test`

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
