# Fixture `sql-uploads`

Durable SQL rows plus file uploads: one postgres binding and one object-store binding. Backup, restore and acknowledged-write ledger cases run here.

- Shape: `durable_sql_plus_file_uploads` — classification **certified** (see `docs/reference/support-policy.md`).
- Seed version `1`, 14 records across 5 file(s); seed checksum `sha256:495555dbe87c0a47cbd1707194e1c40e5eff9293ce1fa31d548107d48c90b2e7`.
- Oracle `fixture-oracle/v1` checksum `sha256:930049cc068509a5b7a85a03af2b8b68e2ba641b167315962475afd0ef331417`. The Go package `api/fixtures` recomputes both from the seed bytes; a mismatch fails `go test ./fixtures/...`.
- Lease scope `fixture:sql-uploads`, target ownership `owned_ephemeral`. All writes made under the lease belong to it; cleanup retains failed evidence and removes owned ephemeral resources only.

Expected state: records table holds exactly the seeded rows (count and per-row digest), uploads mount holds exactly the seeded objects with matching sha256; after restore the acknowledged-write ledger equals the restored rows within the 5-minute recovery point.

Seed data is synthetic. It contains no credential values and no product names.
Setup and teardown commands are returned by the fixture owner at lease time and are not recorded here (no owner runtime exists yet).
