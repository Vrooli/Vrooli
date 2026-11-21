# Templates — Phase Capability Contract

## North Star

Template standing is self-describing across the fleet: every scenario records the template lineage that produced or adopted it, exposes honest orientation state, has bounded drift and migration lag, and carries no unresolved inherited template debt.

## The rungs and their gates

L0 means template provenance is missing or unreadable. The next move is to stamp adopted provenance for the latest default scenario template.

L1 means template lineage exists and resolves to a known template. The next move is to make orientation standing machine-readable.

L2 means orientation standing can be inspected cheaply. The next move is to resolve migration lag and static drift against the registered template.

L3 means the scenario is current enough for governance, with only inherited-debt cleanup remaining. The next move is to retire open inherited template debt.

L4 means provenance, orientation, drift, migration, and inherited-debt standing are clean.

## What each finding means

- `PROVENANCE_MISSING` caps provenance at L0 and is an ERROR. The scenario lacks `.vrooli/service.json::generation.template` metadata. It is auto-fixable.
- `TEMPLATE_UNKNOWN` caps provenance at L1 and is an ERROR. The provenance references a template that template-manager does not know.
- `ORIENTATION_STATE_MISSING` caps orientation at L1 and is a WARNING. The scenario has template lineage but no readable orientation state for static standing.
- `TEMPLATE_VERSION_LAG` caps drift and migration at L2 and is a WARNING. The scenario records an older template version than the registry's latest version.
- `TEMPLATE_MANIFEST_DRIFT` caps drift and migration at L2 and is a WARNING. Static manifest drift cannot be bounded from provenance.
- `INHERITED_DEBT_OUTSTANDING` caps inherited debt at L3 and is a WARNING. The template-manager debt ledger still has open debt for the scenario's source template.

## The canonical fix

For `PROVENANCE_MISSING`, preview and apply the deterministic fix. It writes an adopted provenance block for the latest default template; it does not claim the scenario was freshly generated.

For unknown templates, version lag, drift, orientation, and inherited debt findings, inspect the scenario and template-manager registry. These need scenario-specific judgment: register or correct lineage, apply changelog migrations, run or finalize orientation, and burn down ledger debt.

## How to verify

Run `test-genie provider-contract check templates template-manager --json` to verify the provider contract and live probe.

Run the templates phase against a provenance-bearing scenario and confirm the response contains a `template-manager` / `templates` maturity assessment.

For legacy scenarios without provenance, run the deterministic fix preview first, then apply it only to a disposable copy or when the operator has opted into writing.
