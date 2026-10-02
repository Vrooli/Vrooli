# Export Package (Handler Layer)

This package provides HTTP request decoding and the one-way projection from the generated replay contract into renderer input. Business logic lives in `services/export`; this package provides:
1. HTTP request types (`Request`)
2. Type aliases for backward compatibility
3. Generated `ReplaySpec` validation and direct rendering

## Architecture

### Boundary Enforcement

Per the architectural boundary enforcement principles, handlers should only handle HTTP request/response mapping. This package follows that principle by:

- **Keeping only HTTP concerns**: The `Request` type defines JSON payload structure for API endpoints
- **Generated wire contract**: `Request.MovieSpec` uses the generated protobuf `ReplaySpec`
- **Direct rendering**: HTML and bitmap renderers consume the generated `ReplaySpec`
- **Type aliases for compatibility**: `Overrides`, `ThemePreset`, `CursorPreset` are aliases to `services/export` types

### Files

| File | Purpose |
|------|---------|
| `types.go` | HTTP request type + type aliases for backward compatibility |
| `builder.go` | Thin wrappers delegating to theme and cursor preset helpers |
| `overrides.go` | Thin wrapper delegating to `services/export.Apply` |

### Business Logic (in `services/export`)

All actual implementation lives in `services/export`:
- `preset_builder.go` - Theme and cursor preset application logic
- `spec_overrides.go` - Override application and field synchronization
- `spec_harmonizer.go` - Movie spec validation and harmonization
- `presets.go` - Preset definitions (`ChromeThemePresets`, `BackgroundThemePresets`, etc.)
- `exporter.go` - Domain types (`ReplayMovieSpec`, `ExportTheme`, etc.)

## Migration Notes

Business logic lives in `services/export`. Replay HTTP input is validated as generated `ReplaySpec`, and the same generated message flows through rendering.

For new code, prefer importing directly from `services/export` to access the canonical types and functions.

## Testing

- Unit tests for request decoding and renderer projection are in this package (`*_test.go`)
- Unit tests for internal business logic are in `services/export/*_test.go`

## Related Files

- `handlers/executions.go` - HTTP handlers that use this package
- `handlers/execution_export_helpers.go` - Compatibility shim with local type aliases
- `services/export/` - **Canonical business logic location**
- `services/replay_renderer.go` - Consumes the built movie specs for rendering
