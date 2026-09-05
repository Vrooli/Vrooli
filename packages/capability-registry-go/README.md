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
