# Implementation Plan: LPBS Publish Webhook

## Purpose

Add a webhook notification mechanism to LPBS so that when an artifact is published via `apply` or `set-current`, external systems (deployment-manager, etc.) are notified in real time instead of polling.

## Required Reading

```bash
prompt-manager skill read implementation-plan-authoring cli-steer api-steer seam-discovery-and-enforcement
```

## Problem Statement

After `handleAdminApplyDownloadArtifact` or `handleAdminSetArtifactAsCurrent` successfully upserts a download asset, LPBS returns a response to the caller but does not notify any other system. External consumers like deployment-manager must poll to detect new versions. This creates latency, wasted resources, and tight coupling to LPBS internals.

The `download_apps.update_api_key` column already exists but is unused — it was provisioned for this purpose.

## Scope

### In scope
- Database table for webhook subscriptions (URL, secret, per-app or per-bundle)
- Webhook dispatch after successful `apply` and `set-current` operations
- HMAC-SHA256 signature on webhook payloads for verification
- Admin CRUD endpoints for webhook subscriptions
- Async fire-and-forget delivery (goroutine with timeout)
- Delivery logging for debugging
- Tests for webhook registration, dispatch, and signature verification

### Out of scope
- Retry queues or persistent delivery guarantees (can be added later)
- UI for webhook management (API-only for now)
- Webhook subscriptions for non-publish events (delete, upload, etc.)
- Rate limiting on webhook delivery

## Current Technical Context

### Key files
| File | Role |
|------|------|
| `download_hosting_handlers.go:204-276` | `apply` handler — calls `UpsertAsset()`, returns JSON |
| `download_hosting_handlers.go:279-366` | `set-current` handler — calls `UpsertAsset()`, returns JSON |
| `download_service.go` | `DownloadService` with `UpsertAsset()`, `GetAsset()` |
| `download_hosting.go` | `DownloadHostingService` with artifact CRUD |
| `routes.go` | Route registration, handler factory pattern |
| `main.go` | Server init, embedded migrations |
| `stripe_webhook_service.go` | Reference pattern for *receiving* webhooks |

### Existing infrastructure
- Handler factory pattern: closures capture service dependencies at route registration
- `download_apps.update_api_key` column exists but is unused
- No outbound webhook infrastructure exists
- Stripe webhook *receiver* exists as a reference for signature verification patterns

## Target End State

1. New `webhook_subscriptions` table stores per-bundle (optionally per-app) webhook URLs with HMAC secrets
2. After successful `UpsertAsset()` in both `apply` and `set-current`, a webhook fires asynchronously
3. Webhook payload contains: event type, bundle_key, app_key, platform, release_version, artifact_id, timestamp
4. Payload is signed with HMAC-SHA256 using the subscription's secret, delivered in `X-Webhook-Signature` header
5. Admin endpoints: list, create, update, delete webhook subscriptions
6. Delivery is fire-and-forget with a 10s timeout per request
7. Delivery attempts are logged (success/failure + status code) for debugging

## Implementation Strategy

### Phase 1: Data Model
- Add `webhook_subscriptions` table via embedded migration in `main.go`
- Columns: id, bundle_key, app_key (nullable = bundle-wide), url, secret, active, created_at, updated_at
- Unique constraint on (bundle_key, app_key, url)

### Phase 2: Webhook Service
- New `webhook_service.go` with `WebhookService` struct
- Methods: `Create`, `List`, `Update`, `Delete`, `FirePublishEvent`
- `FirePublishEvent` queries active subscriptions matching bundle/app, dispatches in goroutines
- HMAC-SHA256 signing of JSON payload using subscription secret
- 10s HTTP client timeout per delivery
- Log delivery results (structured logging)

### Phase 3: Integration
- Inject `WebhookService` into server
- After successful `UpsertAsset()` in `apply` handler, call `webhookService.FirePublishEvent()`
- Same for `set-current` handler
- Event types: `artifact.applied`, `artifact.set_current`

### Phase 4: Admin Endpoints
- `GET /api/v1/admin/webhook-subscriptions` — list
- `POST /api/v1/admin/webhook-subscriptions` — create
- `PUT /api/v1/admin/webhook-subscriptions/{id}` — update
- `DELETE /api/v1/admin/webhook-subscriptions/{id}` — delete
- All behind `s.requireAdmin()` middleware
- Register in `routes.go` via `registerWebhookAdminRoutes(s *Server)`

### Phase 5: Tests
- Unit tests for HMAC signing
- Integration tests for webhook CRUD endpoints
- Integration test for end-to-end: apply artifact → webhook fires → test HTTP server receives correct payload + signature
- Test that inactive subscriptions don't fire
- Test bundle-wide vs app-specific subscription matching

## Contract Decisions

<!-- TBD — pending workshop decisions -->

## Testing Plan

- `webhook_service_test.go`: Unit tests for signature generation, subscription matching
- `webhook_handlers_test.go`: Integration tests for admin CRUD endpoints using testcontainers
- `webhook_integration_test.go`: End-to-end test with httptest server as webhook receiver
- Verify payload schema, HMAC signature, timeout behavior, inactive subscription filtering

## Rollout/Validation Checklist

- [ ] `go build ./...` passes
- [ ] `go test ./... -timeout 300s` passes
- [ ] `gofumpt -w .` produces no changes
- [ ] `golangci-lint run` passes
- [ ] Manual test: create subscription, apply artifact, verify webhook received

## Risks + Mitigations

| Risk | Mitigation |
|------|-----------|
| Webhook delivery slows down apply/set-current response | Fire-and-forget goroutine — handler returns immediately |
| Webhook target is down | Log failure, don't retry (V1). Consumer can poll as fallback |
| Secret leakage | Secrets stored in DB, never returned in list/get responses (write-only) |
| Goroutine leak on high volume | Use bounded concurrency or context cancellation if needed later |

## Non-goals / Prohibited Patterns

- No polling-based alternatives — this is specifically about push notifications
- No message queue infrastructure (overkill for V1)
- No compatibility shims for the unused `update_api_key` column — it can be repurposed or dropped separately
- No UI components

## Definition of Done

- Webhook subscriptions can be created, listed, updated, and deleted via admin API
- Publishing an artifact via `apply` or `set-current` fires webhooks to all matching active subscriptions
- Webhook payloads are HMAC-SHA256 signed
- All new code has test coverage
- Code passes `go build`, `go test`, `gofumpt`, and `golangci-lint`
