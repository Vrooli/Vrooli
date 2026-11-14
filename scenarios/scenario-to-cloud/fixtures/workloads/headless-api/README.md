# Fixture `headless-api`

Headless API scenario with no UI surface: one API listener and a bounded in-memory cache. Proves the no-UI declaration path and that edge routing publishes only declared endpoints.

- Shape: `headless_api` — classification **certified** (see `docs/reference/support-policy.md`).
- Seed version `1`, 6 records across 1 file(s); seed checksum `sha256:b288d0b5c7bfdc8bfce93087bb525cc4a6f59e9b8c714c674753ef72035ae0bf`.
- Oracle `fixture-oracle/v1` checksum `sha256:58ea1915b2e2526de91ba26914f733469ae7361c99866cf861d8a65fea24a6c8`. The Go package `api/fixtures` recomputes both from the seed bytes; a mismatch fails `go test ./fixtures/...`.
- Lease scope `fixture:headless-api`, target ownership `owned_ephemeral`. All writes made under the lease belong to it; cleanup retains failed evidence and removes owned ephemeral resources only.

Expected state: Each public route answers with its expected status from outside the target; every non-public route is unreachable from the public ingress (EDGE-03/EDGE-04).

Seed data is synthetic. It contains no credential values and no product names.
Setup and teardown commands are returned by the fixture owner at lease time and are not recorded here (no owner runtime exists yet).
