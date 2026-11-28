# capability-registry-go

This governed Go package is owned by the Vrooli repository. Its package
manifest is the machine-readable adoption and lifecycle record.
# Capability Registry Contract

This package is the shared operational integration model used by scenario
adopters. A `Def` is a manifest-projected dependency enriched by an optional
scenario-owned `Overlay`; `State` is the checked, time-qualified result.

Lifecycle availability and feature compatibility are separate fields. A
reachable process may still report a missing feature. Recovery metadata is
allowlisted (`scenario_start`, `scenario_restart`, `owner_guidance`, or an
operator command) and must never carry an arbitrary upstream business
mutation.

`ProjectManifest` reads `.vrooli/service.json` and preserves that file as the
authority for enabled dependencies, requiredness, startup policy, and
dependency kind. It skips disabled entries and sorts the projection by stable
ID. Overlays can add presentation labels, feature IDs, probes, and recovery
policy without creating undeclared dependencies.

The registry has two check tiers: `ResolveLiveness` is cheap and safe for
health summaries; `Resolve` performs bounded feature/readiness checks. Cached
states retain their checked timestamp so consumers can show stale evidence
instead of presenting it as a fresh observation.

`DescribeHandler` is the standard HTTP adapter for `Registry.Describe`. API
modules keep ownership of the route path, allowed method, and endpoint metadata,
but share the nil-registry response, unavailable response on projection errors,
and JSON success response.

A missing liveness checker yields `unknown`; it never falls back to an expensive
readiness checker. Both tiers execute checks outside the registry mutex, so a
blocked readiness refresh cannot serialize liveness behind model work. Keep
separate bounded clients where liveness and readiness call the same provider.
Checkers that implement `ScenarioTargetChecker` declare the scenario they
observe; `Registry.Validate` rejects a mismatch with the dependency slug.
The checker map must also match the definition IDs exactly: validation rejects
both declared capabilities without checkers and orphan checkers without a
capability definition. The registry copies checker maps when constructed and
when liveness checkers are set, so later caller map mutation cannot change
validation or in-flight resolution. The optional liveness-checker map may be a
subset of definitions; validation rejects orphan entries and checks scenario
target parity for entries that declare a target.

`ScenarioStatusChecker` is the shared bounded checker for local,
Vrooli-managed scenario dependencies whose status contract is the control-plane
`vrooli scenario status <slug> --json` command. It uses the caller context, a
default timeout equal to the registry's liveness-check budget, and an injectable
`CommandRunner`; scenario modules only declare the target slug. Use a
scenario-specific checker when the peer requires protocol health or when the
consumer is packaged, remote, or deployed without the local control-plane CLI.
Lifecycle status reports that the scenario is running; it does not prove the
peer's API is reachable. For deployed targets, resolve a configured static or
Bridge address through the owning discovery path and probe the peer's health
contract, keeping target addressing with the consumer.
