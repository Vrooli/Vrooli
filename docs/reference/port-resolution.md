# Port resolution reference

On a full installation, `LookupScenarioPort` resolves a scenario target in this order:

1. The lifecycle peer record in `~/.vrooli/peers/<target>.json`, where live
   targets use the bare scenario name and non-live targets use
   `<scenario>@<variant>`.
2. The read-only runtime registry, using the bound claim row.
3. `vrooli scenario port`, only when the local authorities miss.

The first two rungs are local authorities and avoid a process launch. The
shared cache coalesces concurrent lookups, but each caller retains its own
freshness policy. A peer record is accepted only when its schema, bare scenario,
requested instance variant, port, and owner process are valid. New records may
include a `version` and `provides` list. When `provides` is present, it lists the
published port capabilities and a lookup must match one of them; older records
without the field remain readable and use their port map. Writers cannot replace
a record owned by another live process, and removers must identify the recorded
owner (stale dead-owner records can be cleaned up). Legacy records without an
`instance` value are interpreted as live only; a shadow record cannot satisfy a
bare live lookup.

Port-key vocabulary follows the published record and registry claim: API
claims use `api` and UI claims use `ui`. Callers may use the conventional
environment spelling (`API_PORT`, `UI_PORT`); the resolver normalizes that
spelling before checking local authorities. This keeps lifecycle storage
claim-shaped while preserving the public environment vocabulary.

In a desktop bundle, `LookupScenarioPort` calls the runtime's authenticated
loopback `/scenario-port` endpoint using the bundle's IPC metadata. It never
falls back to a host registry command or the CLI. The runtime tries a compatible
full peer first, then its declared embedded service. Root-scenario requests
resolve to the root service. The shared `runtime/peerlookup` owner applies the
same selection and version rules to native discovery and the legacy shim.

Desktop peer edges carry `dependencies.scenarios.*.versionRange` into
`bundle.json`. An empty range imposes no version constraint; malformed,
unsupported, or unsatisfied constraints fail closed. The runtime owns that
consumer contract, so callers do not infer constraints from repository state.

The CLI is a recovery fallback, not the normal operation. A missing or stopped
scenario is classified and cached only for the short negative window, so a
scenario starting up can be discovered promptly.

Implementation references:

* [CODE: packages/cli-core/cliutil/port_detector.go#LookupScenarioPort]
* [CODE: packages/cli-core/cliutil/port_detector.go#lookupPeerRecord]
* [CODE: packages/cli-core/cliutil/port_detector.go#lookupRuntimeRegistry]
* [CODE: packages/api-core/discovery/resolve.go#ResolveScenarioPort]
