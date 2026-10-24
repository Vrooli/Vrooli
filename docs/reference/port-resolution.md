# Port resolution reference

`LookupScenarioPort` resolves a scenario target in this order:

1. The lifecycle peer record in `~/.vrooli/peers/<target>.json`.
2. The read-only runtime registry, using the bound claim row.
3. `vrooli scenario port`, only when the local authorities miss.

The first two rungs are local authorities and avoid a process launch. The
shared cache coalesces concurrent lookups, but each caller retains its own
freshness policy. A peer record is accepted only when its schema, scenario,
port, and owner process are valid.

Port-key vocabulary follows the published record and registry claim: API
claims use `api` and UI claims use `ui`. Callers may use the conventional
environment spelling (`API_PORT`, `UI_PORT`); the resolver normalizes that
spelling before checking local authorities. This keeps lifecycle storage
claim-shaped while preserving the public environment vocabulary.

The CLI is a recovery fallback, not the normal operation. A missing or stopped
scenario is classified and cached only for the short negative window, so a
scenario starting up can be discovered promptly.

Implementation references:

* [CODE: packages/cli-core/cliutil/port_detector.go#LookupScenarioPort]
* [CODE: packages/cli-core/cliutil/port_detector.go#lookupPeerRecord]
* [CODE: packages/cli-core/cliutil/port_detector.go#lookupRuntimeRegistry]
* [CODE: packages/api-core/discovery/resolve.go#ResolveScenarioPort]
