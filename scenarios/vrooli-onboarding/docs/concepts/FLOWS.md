# Flows

## Purpose Of This Document

Record the lifecycle of V2 onboarding decisions.

## Flow Inventory

The operator selects scenarios, reviews derived resources, provisions required
credentials, acknowledges deferred integrations, chooses operating mode, and
commits the resulting operator state.

## Flow Details

The read model is recomputed from manifests on each entry. Credential provision
accepts a value only for the request boundary and sends it to the control plane
on stdin; the UI clears the field after the request.

## State Machines

Operator state transitions from absent to committed through atomic replace.
Re-entry reads the committed record and derives a fresh readiness report.

## Maturity Ladder

The integration step remains explicitly deferred until Integration Hub exists.
All other V2 steps are implemented and actionable.

## Production Shape

The API is lifecycle-managed and the UI can run directly, embedded, or inside
the generated desktop host.

## Deferred / Unmodeled Flows

OAuth and connector binding lifecycles belong to Integration Hub and are not
simulated by onboarding.

## Cross-References

- [Domains](DOMAINS.md)
- [Data](DATA.md)
- [Wizard flow](../WIZARD_FLOW.md)
