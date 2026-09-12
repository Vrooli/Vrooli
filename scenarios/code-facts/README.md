# Code Facts

Code Facts is Vrooli's evidence service for answering questions about the
project's implementation. It resolves a caller-declared target, discovers
language parse units and runtime surfaces, brokers Go and TypeScript graph
providers, and returns facts with analyzer, status, file-range, and target
provenance.

It is deliberately evidence-first: an unavailable analyzer is reported as a
typed warning, unsupported work is not silently treated as a path scan, and a
derived fact retains its evidence status rather than being flattened into
unattributed prose. Search consumers should use the `code-facts.code` leaf for
symbols, files, packages, and file domains, and `code-facts.contracts` for
proto adoption and service-contract evidence.

## Targets

The API and CLI accept these target forms:

- `scenario:<name>` — one scenario, including its API, CLI, runtime, and UI
  surfaces.
- `project:<repo-root>` — the governed `scenarios/`, `packages/`, control-plane
  command, and control-plane internal trees, with stable target-qualified facts.
- `repo:<repo-root>` — the repository root for callers that explicitly want the
  broader filesystem target.
- `package:<name>` — one `packages/<name>` subtree.
- `control-plane:<repo-root>` — `cmd/vrooli` plus the governed `internal/` tree.
- an absolute or relative path — generic path discovery.

`MODULE`, `RESOURCE`, `TOOL`, `SAFEGUARD`, `DOCS`, and `TEAM` are explicit
typed-unsupported kinds. They never silently degrade to a path target.

## Evidence families

The service exposes surfaces, parse units, imports, symbols, references, calls,
proto adoption, endpoint proofs, CLI/UI proofs, and architecture-owned
`file_domain` evidence. `DescribeCodeFacts` is selective and cache-aware;
`facts surfaces` and the proof methods reuse the same target resolver so their
answers cannot disagree about the requested root.

The cache is content/config/analyzer scoped and bounded. Project-level facts
are qualified by their owning scenario or source root, making cross-scenario
answers safe to merge and cite.

## Running and testing

Use the lifecycle entrypoints from the repository root:

```bash
make start
make test
```

For local package feedback, `go test ./...` may be run from `api/` or `cli/`.
The scenario-owned test-genie suite remains the validation authority for
runtime behavior and integration seams.

## Search registration

The committed `.vrooli/search.json` is the provider contract. It separates
indexed implementation evidence from indexed contract/proof evidence, carries
scenario/path scope plus explicit stage budgets, and uses the typed generation
status surface for freshness, counts, drift, and degraded stages. Both leaves
implement the shared token-gated reindex protocol; their tuning is intentionally
pinned, so they advertise no config-write endpoint. Search registration is best-effort and
must never prevent Code Facts from serving its own API; a missing Search Hub is
reported as degraded rather than hidden.
