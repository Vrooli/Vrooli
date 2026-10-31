# Agent runtime contract

Agent Manager signs a workspace and run identity. Child identities inherit
the parent workspace and can only narrow scopes and expiry. Secrets Manager
rejects a workspace mismatch and evaluates the active grant at every use.

The typed credential contract has four stages:

1. `RequestAccess` creates a durable, digest-bound request.
2. `GetAccessRequest` or `WaitAccessRequest` returns status metadata only.
3. `CreateBrokerSession` and `ExecuteBrokerOperation` perform a bounded,
   origin-pinned operation.
4. `RevokeBrokerSession` or the run terminal cleanup seam invalidates the
   capability.

Run-bound browser, SSH, and broker handles are revoked when the Agent Manager
run reaches a terminal state. A copied external machine enrollment token is
one-time and becomes unusable after principal revocation.

The broker response is labeled `brokered`. The Hermes-compatible process
hydration path is a separate, explicit `runtime_injection` mode: the receiving
process gets the raw value in its ephemeral environment and the caller receives
a time-bounded hydration lease. The control-plane lifecycle owner must revoke
that lease on stop, failure, and restart. Revocation reports two facts
separately: `future_delivery_stopped` means the authority will not hydrate a
new process, while `process_exposure: already_running_process` records that an
already-started process may still hold its inherited environment until it
exits. Metadata and capability discovery never include credential values or
unauthorized item metadata.

Deployment credentials use the same stable `logical_id` plus `field` address
across resources, scenarios, local services, desktop packages, and remote
targets. `generate`, `prompt`, `delegate`, and `strip` remain deployment
strategies: generated values are stored through the authority and included in
inventory and recovery coverage, prompt values remain operator supplied,
delegated values are fetched at runtime, and stripped values are omitted from
the package manifest so startup cannot request them. A provider outage is
reported as unavailable or absent; it is never converted into a request to
recreate an unknown value.
