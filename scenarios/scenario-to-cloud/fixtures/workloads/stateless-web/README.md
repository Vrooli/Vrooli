# Fixture `stateless-web`

Minimal stateless web scenario: one UI listener, one API listener, no persistent data bindings. The launch-baseline fixture for both architectures.

- Shape: `minimal_stateless_web` — classification **certified** (see `docs/reference/support-policy.md`).
- Seed version `1`, 4 records across 1 file(s); seed checksum `sha256:4a30b6bed38a758ada012f38b81649a7c7753d564c9e979f356d0fbade8d0da7`.
- Oracle `fixture-oracle/v1` checksum `sha256:60c8337acb70955ab0eda78f18a486a2cfc47df1f8c80774bb71ab0234e3b532`. The Go package `api/fixtures` recomputes both from the seed bytes; a mismatch fails `go test ./fixtures/...`.
- Lease scope `fixture:stateless-web`, target ownership `owned_ephemeral`. All writes made under the lease belong to it; cleanup retains failed evidence and removes owned ephemeral resources only.

Expected state: Every seeded page slug serves HTTP 200 on the ui listener with the declared body digest; the api listener answers /api/v1/health with a typed readiness body.

Seed data is synthetic. It contains no credential values and no product names.
Setup and teardown commands are returned by the fixture owner at lease time and are not recorded here (no owner runtime exists yet).
