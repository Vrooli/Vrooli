# Export Package

This package provides configuration, validation, and building utilities for workflow execution exports (replay movies). It was extracted from the monolithic `handlers/execution_export_helpers.go` file (969 lines) to improve code organization, maintainability, and testability.

## Architecture

The package is organized into five focused modules:

### 1. `presets.go` - Theme & Cursor Presets
- **Purpose**: Defines visual configuration presets for chrome themes, background themes, and cursor styles
- **Key Types**: `ChromeThemePreset`, `BackgroundThemePreset`, `CursorThemePreset`
- **Key Functions**: `ClampCursorScale()`
- **Data**: Predefined preset maps for various visual themes (aurora, chromium, midnight, etc.)

### 2. `types.go` - Request/Response Types
- **Purpose**: Defines JSON payload structures for export API endpoints
- **Key Types**: `Request`, `Overrides`, `ThemePreset`, `CursorPreset`
- **Usage**: Request parsing and validation in HTTP handlers

### 3. `builder.go` - Theme & Cursor Builders
- **Purpose**: Constructs export themes and cursor specs by applying preset configurations
- **Key Functions**:
  - `BuildThemeFromPreset()` - Applies chrome and background presets to build an ExportTheme
  - `BuildCursorSpec()` - Applies cursor preset configurations and defaults

### 4. `overrides.go` - Override Application
- **Purpose**: Applies client-provided overrides to movie specs
- **Key Functions**:
  - `Apply()` - Main entry point for applying all overrides
  - `applyDecorOverrides()` - Applies preset names for provenance tracking
  - `syncCursorFields()` - Synchronizes cursor-related fields across spec structures

### 5. `spec_builder.go` - Movie Spec Construction
- **Purpose**: Validates and harmonizes ReplayMovieSpec structures for export
- **Key Functions**:
  - `BuildSpec()` - Main entry point for building validated specs
  - `Clone()` - Deep copies movie specs via JSON marshaling
  - `Harmonize()` - Validates execution IDs and fills in defaults
  - `ensureTheme()`, `ensureDecor()`, `ensureCursor()`, etc. - Fills in nested structures

## Usage Example

```go
import (
    "github.com/vrooli/browser-automation-studio/handlers/export"
    "github.com/vrooli/browser-automation-studio/services"
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

## Design Principles

1. **Separation of Concerns**: Each module has a single, well-defined responsibility
2. **Immutability**: Builder functions create new values rather than modifying inputs
3. **Defensive Defaults**: All fields have sensible fallback values
4. **Validation**: Execution IDs are validated to prevent spec mismatches
5. **Documentation**: All exported types and functions are documented

## Benefits of This Refactoring

- **Reduced Complexity**: 969-line file → 5 focused modules
- **Improved Testability**: Each module can be tested independently
- **Better Documentation**: Clear module boundaries and responsibilities
- **Easier Maintenance**: Changes are localized to specific modules
- **Type Safety**: Type aliases maintain backwards compatibility while improving organization

## Testing

Currently, the export package relies on integration tests in `handlers/executions_test.go`. Consider adding unit tests for:
- Preset application logic
- Cursor scale clamping
- Movie spec harmonization edge cases
- Theme and cursor builder functions

## Related Files

- `handlers/executions.go` - HTTP handlers that use this package
- `handlers/execution_export_helpers.go` - Thin compatibility shim (41 lines)
- `services/replay_renderer.go` - Consumes the built movie specs for rendering
