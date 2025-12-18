# Export Package (Handler Layer)

This package provides a thin HTTP-layer wrapper around `services/export` for workflow execution exports (replay movies). All business logic lives in `services/export`; this package only provides:
1. HTTP request types (`Request`)
2. Type aliases for backward compatibility
3. Function wrappers that delegate to `services/export`

## Architecture

### Boundary Enforcement

Per the architectural boundary enforcement principles, handlers should only handle HTTP request/response mapping. This package follows that principle by:

- **Keeping only HTTP concerns**: The `Request` type defines JSON payload structure for API endpoints
- **Delegating business logic**: All functions (`BuildSpec`, `Apply`, `Clone`, etc.) delegate to `services/export`
- **Type aliases for compatibility**: `Overrides`, `ThemePreset`, `CursorPreset` are aliases to `services/export` types

### Files

| File | Purpose |
|------|---------|
| `types.go` | HTTP request type + type aliases for backward compatibility |
| `builder.go` | Thin wrappers delegating to `services/export.BuildThemeFromPreset`, `BuildCursorSpec` |
| `overrides.go` | Thin wrapper delegating to `services/export.Apply` |
| `spec_builder.go` | Thin wrappers delegating to `services/export.BuildSpec`, `Clone`, `Harmonize` |

### Business Logic (in `services/export`)

All actual implementation lives in `services/export`:
- `preset_builder.go` - Theme and cursor preset application logic
- `spec_overrides.go` - Override application and field synchronization
- `spec_harmonizer.go` - Movie spec validation and harmonization
- `presets.go` - Preset definitions (`ChromeThemePresets`, `BackgroundThemePresets`, etc.)
- `exporter.go` - Domain types (`ReplayMovieSpec`, `ExportTheme`, etc.)

## Usage Example

```go
import (
    "github.com/vrooli/browser-automation-studio/handlers/export"
    exportservices "github.com/vrooli/browser-automation-studio/services/export"
)

// Build a complete movie spec from client and server data
spec, err := export.BuildSpec(baselineSpec, clientSpec, executionID)
if err != nil {
    return err
}

// Apply client overrides (themes, cursors, etc.)
overrides := &export.Overrides{
    ThemePreset: &export.ThemePreset{
        ChromeTheme:     "aurora",
        BackgroundTheme: "nebula",
    },
    CursorPreset: &export.CursorPreset{
        Theme: "aura",
        Scale: 1.2,
    },
}
export.Apply(spec, overrides)

// Use the validated and harmonized spec for rendering
```

## Migration Notes

As of 2025-12-17, business logic was moved from this package to `services/export` as part of the boundary enforcement phase. Existing code using `handlers/export` will continue to work via the wrapper functions and type aliases.

For new code, prefer importing directly from `services/export` to access the canonical types and functions.

## Testing

- Unit tests for the public API wrappers are in this package (`*_test.go`)
- Unit tests for internal business logic are in `services/export/*_test.go`

## Related Files

- `handlers/executions.go` - HTTP handlers that use this package
- `handlers/execution_export_helpers.go` - Compatibility shim with local type aliases
- `services/export/` - **Canonical business logic location**
- `services/replay_renderer.go` - Consumes the built movie specs for rendering
