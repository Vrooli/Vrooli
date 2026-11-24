# CLI Commands — Measures Health

The command surface is declared in [`cli/manifest.json`](../../cli/manifest.json) and loaded through `cliapp.LoadFromManifest`. Domain handlers are in `cli/domains/`; generated Connect clients carry proto-typed requests. `--json`, `--api-base`, and other global behavior come from cli-core.

## Commands

| Command | Purpose | Handler binding |
|---|---|---|
| `measures-health status` | Report API health | API health endpoint |
| `measures-health validate scenario <name> [--probe]` | Grade one scenario; `--probe` executes its declared measures | `ScenarioValidationService.ValidateScenario` |
| `measures-health validate coverage` | Roll up coverage across scenarios | `ValidationService.ListFleetCoverage` |
| `measures-health search query <question> [--limit N]` | Find and execute a matching measure | `SearchService.Search` |
| `measures-health search status` | Report measure index/provider status | `SearchService.Status` |
| `measures-health measures failed [--window token]` | Count failed validations in a time window | `MeasuresService.CountFailedValidations` |
| `measures-health measures coverage [--window token]` | Count passed validations in a time window | `MeasuresService.CountValidationCoverage` |

The default time window is `this_week`; supported tokens are declared on the CLI manifest and the measures API contract. The `--probe` flag is local-only input mapped to `ValidateScenarioRequest.IncludeExecution`.

## Testing and changes

`cli/manifest.json` is the source of truth for command names, arguments, bindings, and governance metadata. Update it and the corresponding handler binding together. The domain registration tests verify RPC coverage; handler tests should exercise argument parsing and request mapping through cli-core's test context.

Run `go test ./...` from `cli/` for CLI behavior and the scenario's endpoint/test gates for cross-layer changes. See [`../internal/TESTING.md`](../internal/TESTING.md).
