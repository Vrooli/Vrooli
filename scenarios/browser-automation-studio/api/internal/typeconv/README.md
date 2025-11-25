# typeconv - Type Conversion Utilities

This package provides safe type conversion utilities for the browser-automation-studio API. It handles conversion between untyped `any` values (often from JSON/database) to strongly-typed Go structures.

## Purpose

When working with JSON payloads, database JSONB columns, or other untyped data sources, we frequently need to convert `map[string]any` or `any` values to specific types. This package centralizes that logic with consistent error handling.

## Architecture

The package is organized into three modules:

### `primitives.go`
Basic type conversions for primitive Go types:
- `ToString(any) string` - Convert to string with fallback to empty string
- `ToInt(any) int` - Convert to int with fallback to 0
- `ToBool(any) bool` - Convert to bool with fallback to false
- `ToFloat(any) float64` - Convert to float64 with fallback to 0.0
- `ToTimePtr(any) *time.Time` - Convert to time pointer with nil on failure
- `ToStringSlice(any) []string` - Convert to string slice with empty slice fallback

### `contracts.go`
Conversions for automation contract types:
- `ToAssertionOutcome` - Convert to assertion result structures
- `ToHighlightRegion(s)` - Convert to element highlight regions
- `ToMaskRegion(s)` - Convert to privacy mask regions
- `ToElementFocus` - Convert to element focus metadata
- `ToBoundingBox` - Convert to element bounding box
- `ToPoint(Slice)` - Convert to coordinate points

### `timeline.go`
Timeline-specific conversions and types:
- `ToRetryHistory(Entry)` - Convert to retry attempt records
- `ToTimelineScreenshot` - Convert database artifacts to screenshot metadata
- `ToTimelineArtifact` - Convert database artifacts to timeline artifacts
- Type definitions: `RetryHistoryEntry`, `TimelineScreenshot`, `TimelineArtifact`

## Design Principles

1. **Fail-Safe**: All conversions return zero values on failure rather than panicking
2. **Flexible Input**: Accept multiple input types (native Go types, JSON unmarshaled maps, etc.)
3. **No External Dependencies**: Only depends on stdlib and our internal contracts
4. **Consistent Naming**: All functions use `To*` prefix for clarity

## Usage Example

```go
import "github.com/vrooli/browser-automation-studio/internal/typeconv"

// Convert untyped map fields safely
data := map[string]any{
    "width": 1920,
    "height": "1080",  // string instead of int
    "enabled": "true",
}

width := typeconv.ToInt(data["width"])    // 1920
height := typeconv.ToInt(data["height"])  // 1080 (parsed from string)
enabled := typeconv.ToBool(data["enabled"]) // true (parsed from string)

// Convert complex structures
bbox := typeconv.ToBoundingBox(map[string]any{
    "x": 100.5,
    "y": 200.3,
    "width": 50,
    "height": 75,
})
// Returns *contracts.BoundingBox{X: 100.5, Y: 200.3, Width: 50, Height: 75}
```

## Migration Notes

This package was extracted from `services/timeline.go` to improve code organization and reusability. The original functions used lowercase names (e.g., `toString`, `toInt`) but have been renamed to exported names (e.g., `ToString`, `ToInt`) following Go conventions for public APIs.

## Future Considerations

- Add optional error returns for cases where callers need to distinguish between zero values and conversion failures
- Consider adding validation-focused variants that return errors instead of zero values
- Add benchmark tests if performance becomes a concern
