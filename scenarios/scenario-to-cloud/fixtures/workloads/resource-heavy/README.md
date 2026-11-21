# Fixture `resource-heavy`

Resource-heavy profile (GPU plus large memory) certified only for the explicit unsupported or insufficient-capacity refusal path. A successful deployment of this fixture is never a certification claim.

- Shape: `resource_heavy_profile` — classification **compatibility_only** (see `docs/reference/support-policy.md`).
- Seed version `1`, 3 records across 1 file(s); seed checksum `sha256:37b31d94fa63908c2b73b6de4073c06c14d77ab62880505061680ea7871279a8`.
- Oracle `fixture-oracle/v1` checksum `sha256:9c2cc2f1b2fbb47f44d2816db6203a2335ab5021123864d3414cce3d883cc04e`. The Go package `api/fixtures` recomputes both from the seed bytes; a mismatch fails `go test ./fixtures/...`.
- Lease scope `fixture:resource-heavy`, target ownership `owned_ephemeral`. All writes made under the lease belong to it; cleanup retains failed evidence and removes owned ephemeral resources only.

Expected state: For each seeded machine profile the preflight or plan returns the expected typed refusal with its reason and no target effect (PLAN-04); no bundle, staging directory or process is created.

Seed data is synthetic. It contains no credential values and no product names.
Setup and teardown commands are returned by the fixture owner at lease time and are not recorded here (no owner runtime exists yet).
