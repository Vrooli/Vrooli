# Notification Hub proto sources

This directory is the canonical wire-contract source for notification-hub.
Generated Go, TypeScript, and Python bindings are produced from these files by
the governed proto generator.

## Domains

- `v1/recipients/` — owners, devices, addresses, push subscriptions, quiet
  windows, and escalation policy.
- `v1/notifications/` — notification intake, lifecycle, and history.
- `v1/routing/` — channel selection, sensitivity policy, and suppression.
- `v1/delivery/` — attempts, receipts, timelines, and analytics.
- `v1/conversations/` — questions, answers, waits, and escalation state.
- `v1/shared/` — common types, errors, and health contracts.

After changing a schema, regenerate bindings through the package-owned command:

```bash
cd packages/proto
make generate SCENARIO=notification-hub
```

The generated artifacts under `packages/proto/gen/` are part of the checked-in
contract surface and must remain synchronized with these sources.
