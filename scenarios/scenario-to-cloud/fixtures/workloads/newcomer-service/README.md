# newcomer-service

A scenario that did not exist when the cloud ramp was built. It proves phase 24
generality: bringing a conforming scenario onto the ramp needs declarations and
configuration only, never a cloud source edit.

- `declarations/` is a catalog layout the closure resolver reads directly
  (`scenarios/<id>/.vrooli/service.json`, `resources/<id>/resource.json`).
  `newcomer-service` depends on the scenario `ledger-service` and the resource
  `ledger-store`; each declares its own credential descriptor. The resource
  publishes an amd64 artifact only, so resolving the closure for `linux/arm64`
  names it as `missing_platform_artifact` and the plan refuses with
  `unsupported_capability` (P24-A03).
- The `deployment` block declares one `public_via_edge` listener (`api`), one
  `private` listener (`callback`, the provider webhook receiver), persistent
  data owned by `ledger-store`, and a recovery contract.
- `seed/` holds the deterministic seed; the oracle is recomputed by
  `api/fixtures` from the bytes.

The in-process journey lives in `api/generality`: closure → plan → admission →
execution against a fake `cloud-target` owner → health observation → evidence
cell, for one environment, for two environments of this scenario on two
targets, and across an update that must keep the declared routes.
