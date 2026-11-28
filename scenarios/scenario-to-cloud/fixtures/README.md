# Certification fixtures

Reusable, leased fixtures for the cloud certification matrix (`certification/matrix.json`).

- `workloads/<id>/fixture.json` — one declaration-derived workload fixture: dependencies, listeners, persistent data bindings, expected capacity, deterministic seed and an expected-state oracle. `seed/` holds the synthetic seed bytes.
- `coexistence/*.json` — multi-deployment journeys (two deployments sharing a resource; an existing full install plus an unrelated workload).

Every fixture declares `ownership.lease_scope` and `ownership.target_ownership`; writes made while a fixture is leased belong to that scope, and cleanup removes only owned ephemeral resources. The Go package `api/fixtures` loads this catalog, recomputes every seed and oracle checksum (`fixture-oracle/v1`) and refuses fixtures that omit ownership.

Fixtures are generic: no product names, no credential values, no private paths. Setup and teardown argv are returned by the fixture owner at lease time (typed actions), not recorded here.
