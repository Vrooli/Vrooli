# typescript-code-graph sidecar

Node sidecar for the `typescript-code-graph` scenario. Hosts `ts-morph` and speaks line-delimited JSON over stdio to the Go API supervisor. Not invoked directly; spawned by the API process. See `scenarios/typescript-code-graph/api/internal/sidecar/` for the supervisor and `src/protocol.ts` for the IPC contract.

The owner-built bundle also exposes a bounded offline boundary-facts command for
repository tooling. Send one JSON request to
`node sidecar/dist/index.js --boundary-facts` with `protocol_version: 1`, an
absolute `root_dir`, and an explicit list of production source files. It writes
one deterministic JSON response to stdout and does not contact the scenario
API. The AST extractor reports only supported executable facts and leaves
unproven target provenance unknown; it does not infer across resolver aliases
or imported wrappers. Resolver navigation and other non-fetch sinks remain
open coverage. Quality Health consumes this owner interface; it does not
maintain a second TypeScript parser. Before use, Quality Health asks the control
plane to verify the bundle's lifecycle freshness manifest and fails with the
owner setup command when the bundle is stale.
