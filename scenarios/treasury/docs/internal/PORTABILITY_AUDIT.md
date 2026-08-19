# Treasury Portability Audit

Date: 2026-08-19

## Support posture

Treasury is portable across Vrooli's supported Linux, macOS, and Windows hosts
for its core API, CLI, UI, and SQLite state. The API uses pure-Go
`modernc.org/sqlite`, builds with `CGO_ENABLED=0`, derives storage from
`SCENARIO_DATA_DIR`, and uses Go path APIs rather than fixed Unix paths. The UI
and generated Connect clients have no platform-specific runtime dependency.

The optional x402 facilitator runs as a digest-pinned amd64/arm64 container.
Its availability therefore depends on the host's governed container runtime;
Treasury remains fail-closed when it is absent or unconfigured. Live payment
support also depends on network and chain configuration and is not implied by
core cross-platform compatibility.

## Platform-sensitive boundaries

- Lifecycle start, stop, ports, and data directories are owned by the Vrooli
  control plane; Treasury does not implement host repair.
- SQLite file paths are normalized by the shared database layer. Tests cover
  a real SQLite handle rather than OS-specific mocks.
- Credentials are logical references resolved at use time and are never stored
  in the SQLite database or shell-specific files.
- Shell snippets exist only in governed lifecycle/test orchestration. The API
  and CLI binaries themselves do not invoke Unix-only process utilities.
- The facilitator image supports amd64 and arm64; other architectures degrade
  by leaving the optional dependency unavailable and all x402 writes refused.

## Validation and remaining attended checks

Go unit/race suites and the TypeScript production build run without CGO. The
comprehensive Test Genie suite is the repository-owned portability and lifecycle
gate. Native attended smoke tests on macOS and Windows remain release evidence,
not a prerequisite for local implementation completeness; any platform defect
found there must be remediated in the control plane when it concerns host state.

## Evidence

- `.vrooli/service.json`
- `api/main.go`
- `api/internal/database/`
- `docs/reference/configuration.md`
- `docs/guides/troubleshooting.md`
- `resources/x402-facilitator/resource.json`

