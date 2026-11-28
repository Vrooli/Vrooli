# Scenario API test support

Scenario APIs keep domain-specific fakes and fixtures under `api/internal/testutil/`.
Use `api-core` test packages for reusable infrastructure:

- `apihttptest` for fake HTTP clients and request helpers
- `databasetest` for isolated database handles
- `scheduletest` for deterministic clocks
- `servertest` for live HTTP tests through the production handler stack
- `testguard` for source-tree rules that must be enforced by multiple APIs

Owner-local `mocks/` and generated bridges can use test-only dependencies because
production source must not import `internal/testutil`. The canonical guard in
`testguard` scans the full API root, including handlers and commands, and
excludes test files, generated output, mocks, and vendored/build artifacts. Keep
testutil directories out of the production scan as well. Keep domain-specific
fake behavior next to the owner; move shared protocol and infrastructure
behavior into `api-core` when multiple owners consume it.
