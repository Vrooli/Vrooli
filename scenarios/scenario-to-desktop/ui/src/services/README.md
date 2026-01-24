# Services Layer

Services contain **pure utility functions** for data transformation, validation, and formatting.

## Architecture Role

```
Components (UI) → Hooks (React State) → Controllers (Business Logic) → Services (Pure Functions)
```

## Key Principles

1. **Pure functions** - No side effects, same input always produces same output
2. **Single responsibility** - Each function does one thing well
3. **No API calls** - Services don't make HTTP requests (that's in `lib/api/`)
4. **Highly testable** - Pure functions are trivial to unit test

## Files

### `pipeline.service.ts`
Pipeline-related utilities:
- `isTerminalState()` - Check if status is terminal (completed/failed/cancelled)
- `getRecoverySuggestions()` - Get suggestions for error categories
- `categorizeError()` - Categorize an error for recovery suggestions
- `formatPipelineProgress()` - Format progress for display

### `generator.service.ts`
Generator form utilities:
- `buildPipelineConfigFromForm()` - Transform form state to pipeline config
- `buildValidationParams()` - Build validation parameters from form state
- `getSelectedPlatforms()` - Extract selected platforms list

### `preflight.service.ts`
Preflight validation utilities:
- `buildPreflightDisplayState()` - Build display state from result
- `buildPreflightPayload()` - Build export payload
- `getMissingSecrets()` - Filter required secrets without values
- `isPreflightComplete()` - Check if preflight is fully complete
- `filterValidSecrets()` - Remove empty secret values
- `getValidationStatus()`, `getSecretsStatus()`, etc. - Step status helpers

### `signing.service.ts`
Code signing utilities:
- Signing configuration parsing and formatting

## Usage Example

```typescript
// In a controller
import { getRecoverySuggestions, categorizeError } from '../services/pipeline.service';

const category = categorizeError(error);
const suggestions = getRecoverySuggestions(category);
// Use suggestions in error recovery UI
```

## Testing

Services have comprehensive test coverage. Run tests with:

```bash
npm test -- --run src/services/
```

All service functions should have corresponding tests that verify:
- Normal operation with valid inputs
- Edge cases (null, undefined, empty)
- Error conditions
