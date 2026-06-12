# Cache

Code Facts caches derived graph and report payloads. Source code is never owned by Code Facts.

Cache keys include:

- Code Facts analyzer/schema version.
- Provider analyzer version.
- Canonical target root.
- Parse unit id, root, and config path for graph entries.
- Request options and requested fact families for report entries.
- Config hashes such as `go.mod`, `go.sum`, `tsconfig.json`, `package.json`, and lockfiles.
- Source content fingerprints for bounded Go/TypeScript/JavaScript parse-unit files.
- Provider graph hash when a graph result is available.

Cache scopes:

- `graph`: parse unit plus provider options to provider graph payload.
- `report`: selected fact families plus target/options to `CodeFactsReport`.

Callers can pass `use_cache=false` on describe/proof/surface requests to bypass lookup and force fresh extraction. Fresh results refresh cache entries with the latest source/config hash evidence.

Cache metadata exposes key, scope, state (`hit`, `miss`, `bypassed`, or `stored`), reason, analyzer version, provider version, schema version, source hash, config hash, graph hash, age, and hit count. `code-facts cache clear <target> --dry-run` reports matches without deleting derived entries.
