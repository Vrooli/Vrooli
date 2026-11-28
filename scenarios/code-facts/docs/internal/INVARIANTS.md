# Invariants

- Code Facts does not keep language parser implementations. Its Go provider
  invokes the shared `api-core/gograph` extraction owner in-process; its
  TypeScript and JavaScript providers invoke their shared graph owner.
- Graph providers emit generic language/framework facts; Code Facts adapters
  interpret supported framework evidence.
- Proto-health and other consumers must consume proof statuses and evidence,
  not graph-provider packages or source-language parsing.
- Fact-family filtering must not fabricate unsupported facts as success.
- Provider failures must be visible as `unsupported` or `unknown`.
- Unsupported proof is not success.
- Endpoint-level `proven` must not hide a contradicted, missing, or unknown
  required proto role. Status precedence is `contradicted`, then `missing`,
  then `unknown`, then `unsupported`, then `proven`.
- Non-proto roles declared as `none`, `transport_only`, or `external_shape` do
  not prevent endpoint-level `proven` status.
- Cache keys must be deterministic and inspectable.
- CLI JSON output must match API response shapes.
- Search results are provenance-preserving pointers into the authoritative
  `DescribeCodeFacts` report; lexical ranking must never discard evidence
  status, analyzer, or source path.
- Page size and page token affect transport shape only; the cached report and
  its fact ordering remain unchanged.
- Index job state is the sole durable cancellation signal. Job creation rejects
  unknown states, and SQLite updates enforce the `JobState` transition graph
  with a source-state compare-and-swap; no parallel cancellation flag is stored.
