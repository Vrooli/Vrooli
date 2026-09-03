# Vrooli Events platform contract

Vrooli Events is an optional, best-effort observation platform. It never
blocks, retries, or changes a business response.

## Automatic adoption

Every scenario served by `api-core/server.Run` receives the receipt runtime.
The runtime derives its target from `VROOLI_SCENARIO`, resolves Events away from
the request path, and emits only after the handler completes. A missing or
stale policy snapshot, an Events outage, or a failed delivery simply leaves an
action unobserved.

## Canonical receipt

The only receipt event type is `vrooli.events.receipt.v1`. Its
`vrooli.events.v1.domain.EventEnvelope` has universal `event_id`, `event_type`,
`occurred_at`, `source`, `target`, `correlation`, `attribution`, and typed
`data` fields. Receipt data is packed as `ReceiptData`; it contains outcome,
status, duration, policy version, idempotency key, and a `Struct` projection.

No metadata bag, JSON-string projection, compatibility envelope, or inferred
identifier is part of this contract. Policy-selected projection keys are exact
descriptor paths such as `plan.id`; an empty policy projection remains empty.

## Trust, policy, and observations

Only verified Agent Manager identity claims may set the receipt subject and
agent correlation. Verified claims can also carry task and workflow execution,
node, and attempt. Invocation headers are request annotations, not proof.

Vrooli Events declares eligible operations with `ReceiptCapturePolicy`: target,
operation, protocol, event type, response type, explicit projection paths,
retention, and read access. A policy snapshot is replaced atomically by API
Core. The Plan Manager proof policy selects only:

```text
POST /vrooli.plan_manager.v1.plans.PlansService/CreatePlan
plan.id
```

Agent Manager exposes receipts as observation data alongside normal run and
workflow results. Unavailable Events is reported as degraded or unavailable;
the absence of a receipt never means the agent action failed. Swarm Manager
uses verified, exact run ownership and canonical Plan Manager state for durable
links; it does not retain an independent receipt/evidence ledger.

## Incident notification ownership

Incident lifecycle events are a separate fact stream carried through the same
event envelope. `vrooli-autoheal` publishes transition facts only: incident
identity, source check, severity, status, and remediation facts when an
operator decision is required. It does not publish recipients, channel names,
sensitivity labels, or human notification copy. Re-sighting an unchanged open
incident does not publish another lifecycle transition.

`notification-hub` owns the consumer-side renderer, severity-to-sensitivity
mapping, recipient selection, channel routing, and durable delivery evidence.
The approval-request event is rendered into a blocking ask with server-owned
`approve`/`reject` answers. Autoheal verifies the durable answer through the
notification-hub wait surface and binds the one-time authorization to the
incident fingerprint and remediation candidate before executing a generated
artifact. A caller-supplied approval boolean is not an authorization.
