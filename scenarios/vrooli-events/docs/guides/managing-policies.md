# Managing Policies

vrooli-events exposes receipt-capture policies as its public policy resource.
These policies decide which API receipts are projected into the durable event
stream. Inter-scenario access, rate-limit, and circuit-breaker enforcement is
implemented by the discovery integration and local policy caches; it is not a
vrooli-events CLI CRUD surface.

Implementation: [CODE: internal/policy/policy.go] | API handlers: [CODE: api/handlers_capture_policy.go] and [CODE: api/handlers_policy.go]

## Inspect the active policy snapshot

```bash
curl "http://localhost:${API_PORT}/api/v1/policies/snapshot"
```

The snapshot is versioned and is the source used by API Core policy consumers.
Connected consumers can receive updates over the policy stream:

```bash
curl -N "http://localhost:${API_PORT}/api/v1/policies/subscribe"
```

## Create or reconcile a receipt-capture policy

The same `policy_id` is idempotent: posting it again updates the existing
declaration instead of creating a duplicate rule.

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/receipt-capture-policies" \
  -H 'Content-Type: application/json' \
  -d '{"policy_id":"example.receipt","enabled":true,"selector":{"target_scenario":"example-scenario","operation":"example.v1.run","protocol":"http","event_type":"vrooli.events.receipt.v1"},"response_type":"example.response.v1","response_projection_paths":["status.code"],"retention_days":30,"access":{"read_principals":["agent-manager"]}}'
```

The selector requires a target scenario, a stable operation pattern, either
`http` or `connect`, and the canonical receipt event type. Projection paths
must be descriptor-style names such as `status.code`.

List enabled declarations:

```bash
curl "http://localhost:${API_PORT}/api/v1/receipt-capture-policies"
```

Delete by declaration identity:

```bash
curl -X DELETE "http://localhost:${API_PORT}/api/v1/receipt-capture-policies/example.receipt"
```

## Reconcile scenario declarations

Scenarios can declare receipt policies in their metadata. Reconciliation
validates the complete declaration set and applies it atomically:

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/receipt-capture-policies/reconcile" \
  -H 'Content-Type: application/json' \
  -d '{"scenario":"example-scenario"}'
```

Use `vrooli-events capture-preview` to inspect the local declaration before
reconciling it, and `vrooli-events capture-reconcile` to apply the governed
declaration through the CLI.

## Enforcement model

Discovery-integrated callers use a local policy cache before making a network
call. Receiver middleware performs a second check using the source-scenario
identity header. The two caches receive versioned updates from the policy SSE
stream, so request traffic does not depend on a round trip to the policy
service.

The policy endpoints documented here are deliberately limited to receipt
capture. Do not use older `vrooli-events policy ...` or `vrooli-events
subscriptions ...` commands; subscriptions are managed through the HTTP API
documented in [Creating Subscriptions](creating-subscriptions.md).
