---
title: "Error Semantics"
description: "How the API signals failure modes to clients and operators"
category: "internal"
order: 102
audience: ["developers"]
internal: true
---

# Error Semantics

How the HTTP API signals failure to clients, and how those signals translate to UI state + operator alerts.

## HTTP status conventions

| Status | When it's used |
|--------|----------------|
| `200 OK` | Successful read or idempotent write that mutated nothing surprising |
| `201 Created` | New resource created (used sparingly — most write paths return `200` with the resource) |
| `202 Accepted` | Async job accepted (e.g. `/api/v1/customize`) |
| `400 Bad Request` | Caller-supplied JSON failed `json.Decoder.Decode` or domain validation |
| `401 Unauthorized` | Missing or invalid admin session / user JWT / service bearer token |
| `403 Forbidden` | Authenticated but not authorized for this resource (e.g. user requesting another user's account) |
| `404 Not Found` | Resource ID is well-formed but does not exist |
| `409 Conflict` | Idempotency key collision, duplicate webhook delivery, race on a write |
| `422 Unprocessable Entity` | **Not used.** Validation failures use `400`. |
| `429 Too Many Requests` | Rate limiter (`api/rate_limit.go`) tripped — magic-link flow uses 5 / 15 min per email |
| `500 Internal Server Error` | Unexpected panic, DB error, or Stripe API error that we cannot map to a 4xx |
| `502 / 503` | Reserved for upstream LLM provider failures from the metered inference |

## Body shape

All error responses currently use `http.Error(w, msg, code)`, which produces a `text/plain` body. UI clients should not parse the body — they read the status code. Some newer endpoints (metered inference, deploy-readiness) return structured JSON `{"error": "...", "code": "..."}`; these are documented per-endpoint.

## Logging contract

- Successful requests emit a single `request_completed` line via `loggingMiddleware` (see `api/main.go`).
- Handlers log structured errors via `logStructuredError(event, fields)` *before* returning a non-2xx status. The `event` slug is the canonical name used by ops dashboards (e.g. `admin_reset_failed`, `variant_space_write_failed`, `stripe_webhook_invalid_signature`).
- `level` is implied by the function chosen (`logStructured` = info, `logStructuredError` = error). There is no `warn` level.

## Webhook idempotency

- Stripe webhook handler (`/api/v1/webhooks/stripe`) deduplicates by `stripe_event_id` against `credit_transactions` and the unified `payment_anomaly_log`. A duplicate delivery returns `200 OK` *without* re-applying side effects — this is the contract Stripe expects to stop retrying.

## Anomaly dispatch

- Detected payment anomalies write to `payment_anomaly_log`. The dispatcher (`anomaly_alert_dispatcher.go`) polls `dispatch_status = 'pending'` rows and sends to the configured webhook. Failed dispatches retry with backoff and surface via `dispatch_attempts` + `dispatch_error`. The HTTP request that *created* the anomaly always returns its own success/failure independently of dispatch outcome.

## What clients should not assume

- Error bodies are stable. They are not.
- That a `500` means data was not written. Some webhook paths write *then* fail to encode a response.
- That a `429` body tells you the retry-after window. Use the `Retry-After` header (where set) or back off exponentially.
