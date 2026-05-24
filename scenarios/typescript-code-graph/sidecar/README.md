# typescript-code-graph sidecar

Node sidecar for the `typescript-code-graph` scenario. Hosts `ts-morph` and speaks line-delimited JSON over stdio to the Go API supervisor. Not invoked directly; spawned by the API process. See `scenarios/typescript-code-graph/api/internal/sidecar/` for the supervisor and `src/protocol.ts` for the IPC contract.
