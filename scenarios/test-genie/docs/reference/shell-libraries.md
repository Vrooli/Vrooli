# Shell Libraries (Deprecated)

> **Status**: Deprecated - Superseded by Go-native orchestrator
> **Removal**: Legacy bash libraries have been removed from active use

## Summary

The bash shell libraries that previously lived in `/scripts/scenarios/testing/` have been fully replaced by Go-native implementations within test-genie's API.

## Migration

All testing functionality is now provided by:

| Legacy Bash | Go Replacement |
|-------------|----------------|
| `shell/suite.sh` | `api/internal/orchestrator/` |
| `shell/phase-helpers.sh` | `api/internal/orchestrator/phases/` |
| `shell/requirements-sync.sh` | `api/internal/requirements/` |
| `unit/run-all.sh` | Phase runners in `phases/phase_unit.go` |

## Current Usage

Use the Go orchestrator via CLI or API:

```bash
# Via CLI
test-genie execute my-scenario --preset comprehensive

# Via API
curl -X POST "http://localhost:${API_PORT}/api/v1/executions" \
  -H "Content-Type: application/json" \
  -d '{"scenarioName": "my-scenario", "preset": "comprehensive"}'
```

## See Also

- [Architecture](../concepts/architecture.md) - Go orchestrator design
- [Test Runners](test-runners.md) - Language-specific runners
- [API Endpoints](api-endpoints.md) - REST API reference
- [Go Migration](../internal/go-migration.md) - Historical migration notes
