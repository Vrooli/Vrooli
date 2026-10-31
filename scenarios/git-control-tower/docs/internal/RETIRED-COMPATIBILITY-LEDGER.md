# Retired compatibility ledger

This ledger records which older paths are replaced and which remain because
they have distinct ownership. Deletion is deferred until callers and runtime
receipts prove replacement.

| Older path | Current authority | Disposition |
|---|---|---|
| Real-Git validation fixtures | Fake Git runners and filesystem-only fixtures | Converted; static admission gate covers API tests |
| Caller-supplied human headers | Verified principal and resource-bound human intent | Header values are advisory; forged/absent values deny |
| In-memory review job map | SQLite `reviewjobs.Store` plus GCT hydration | Replaced for new advisory lifecycle |
| Latest Test Genie execution fallback in review jobs | Exact execution ID captured at dispatch | Removed from durable job aggregation; direct legacy summary remains explicitly latest-oriented |
| Workspace Sandbox provenance presentation | Owner receipt model plus GCT uncertainty-preserving presentation | Retained as consumer; no private duplicate index |
| Remaining legacy REST repository writers | Typed RepoService Connect-RPC plus human intent | Repository registry, credentials, remote URL, SSH, file, grouping rules, gitignore remediation, tracked-binary remediation, discard, ignore, push, pull, upstream, and precommit writers were retired in favor of generated RepoService clients; remaining independent REST writers are outside this repository/configuration family and retain their own documented owner or migration slice |
| Visual capture routes | Existing visual capture owner and descriptor-based baseline domain | Both retained: capture is producer, baseline is comparison authority |
| Provider SDKs and secret lifecycle | Integration Hub | Not implemented in GCT |

Every retained compatibility path has an owner and a documented reason. A
future retirement must add caller migration evidence before removing it.
