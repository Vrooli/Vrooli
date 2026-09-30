# Owner cleanup contract

Owner-delegated providers keep deletion authority inside the scenario that owns
the data. Storage Manager reaches the owner through three local HTTP endpoints:

- `GET /api/v1/cleanup/estimate` with `min_age_seconds`, `max_bytes`, and
  `keep_count` query parameters.
- `POST /api/v1/cleanup/preview` with an `estimate` object.
- `POST /api/v1/cleanup/apply` with a `preview`, `idempotency_key`, and
  `approval_mode` (`owner` or `operator`).

Estimate responses identify the provider and report estimated bytes, item count,
the observed time, and the effective filters. Preview items include an `id`,
owner path, byte count, age in seconds, and `protected` flag. An owner must not
return protected items for deletion and must never remove a protected item even
if a caller submits one manually.

Apply is idempotent: repeating an `idempotency_key` returns the first result and
does not delete the items again. An owner that does not implement cleanup returns
HTTP 404 from the estimate endpoint; Storage Manager distinguishes that from an
unreachable owner and from an unconfigured client.

The shared executable contract checks are in
`api/internal/cleanup/ownerconformance.go`. Owner scenario tests can call
`cleanup.RunOwnerConformance(t, baseURL)` against their test server.
