# repo-contract-go

Typed adapter for `.vrooli/repo-contract.json` (Go package `repocontract`): layout, target
kinds, scenario well-known paths, runtime-home entries, and path resolution helpers. Code
resolves repository paths here rather than carrying literals; `TestNoManifestPathLiteralJoins`
enforces that for manifest paths. `Contract.WalkSource` is the shared walker for
repository-wide source audits. The contract itself is described in
[docs/repo-contract.md](../../docs/repo-contract.md).

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
