# Cache

Cache keys are planned to include:

- Code Facts analyzer version.
- Provider analyzer version.
- Canonical target root.
- Request options and requested fact families.
- Project/config hashes such as `go.mod`, `go.sum`, `tsconfig.json`, and package metadata.
- Source content hash or provider graph hash.

Cache responses should expose hit, miss, stale, bypassed, and invalidated states with reasons. Derived facts can be cleared; source code is never owned by Code Facts.
