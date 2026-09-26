# Fixture `dependent-chain`

Scenario depending on another scenario that owns its own resource and credential. Proves the closure includes transitive requirements with reasons and that credentials materialise only to the authorised consumer.

- Shape: `scenario_with_scenario_dependency` — classification **certified** (see `docs/reference/support-policy.md`).
- Seed version `1`, 5 records across 2 file(s); seed checksum `sha256:0b91fffd4e9a0110de2444601437e116daf13f2b23dcabd5463723f8c6622412`.
- Oracle `fixture-oracle/v1` checksum `sha256:5b70fbe6739d8564e4f78b98ae95136812be62324b555276a12d738584052350`. The Go package `api/fixtures` recomputes both from the seed bytes; a mismatch fails `go test ./fixtures/...`.
- Lease scope `fixture:dependent-chain`, target ownership `owned_ephemeral`. All writes made under the lease belong to it; cleanup retains failed evidence and removes owned ephemeral resources only.

Expected state: The computed closure equals seed/closure.json (same members, same reasons); the dependency's events table holds exactly the seeded rows; the primary can read them through the dependency's API using only the dependency-owned credential.

Seed data is synthetic. It contains no credential values and no product names.
Setup and teardown commands are returned by the fixture owner at lease time and are not recorded here (no owner runtime exists yet).
