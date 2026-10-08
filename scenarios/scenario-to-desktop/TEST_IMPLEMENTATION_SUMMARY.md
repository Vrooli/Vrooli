# Test Implementation Summary

Scenario-to-desktop is validated by Test Genie and scenario-owned suites.

## Authoritative commands

- `vrooli scenario test scenario-to-desktop` starts the server-owned full suite.
- `test-genie runs wait --json scenario-to-desktop <run-id>` waits once for the terminal record.
- `make lint` and `make fmt` exercise the scenario quality surfaces.

## Surface ownership

| Surface | Primary validation |
|---|---|
| API | `api/go test ./...` and API Health |
| CLI | `cli/go test ./...` plus generated primitive evidence |
| Runtime | `runtime/go test ./...` |
| UI | `ui/pnpm test:coverage` |

Coverage floors and Test Genie role declarations are defined in
`.vrooli/testing.json`; they are intentionally enforced rather than documented
as achieved when a current run is below its floor.
