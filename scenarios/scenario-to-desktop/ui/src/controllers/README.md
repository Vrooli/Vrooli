# Controllers Layer

Controllers contain **pure business logic functions** that orchestrate operations across services and APIs.

## Architecture Role

```
Components (UI) → Hooks (React State) → Controllers (Business Logic) → Services (Pure Functions)
```

## Key Principles

1. **No React imports** - Controllers should be framework-agnostic pure functions
2. **Orchestration** - Combine multiple services and API calls to implement business operations
3. **Testable** - Pure functions with explicit inputs/outputs are easy to unit test
4. **Reusable** - Can be called from any hook or component

## Files

### `pipelineController.ts`
Pipeline-related business logic:
- `buildPipelineConfig()` - Construct pipeline config from form state
- `validateBeforeRun()` - Validate prerequisites before running a stage
- `canProceedToGeneration()` - Check if generation can proceed
- `shouldAutoStartPolling()` - Determine if polling should auto-start
- `filterNonEmptySecrets()` - Filter out empty secret values

### `generatorController.ts`
Generator page orchestration:
- `loadGeneratorPageData()` - Load all data for the Generator page
- `submitGeneratorForm()` - Validate and submit the generator form
- `prepareFormSubmission()` - Prepare form data for submission
- `testProxyConnection()` - Test proxy connectivity
- `findScenarioByName()` - Look up scenario by name

### `preflightController.ts`
Preflight validation orchestration:
- `buildPreflightPipelineConfig()` - Build config for preflight run
- `buildPreflightSectionState()` - Build complete preflight UI state
- `resolveAllStepStatuses()` - Calculate status for all preflight steps
- `calculatePreflightStatus()` - Get comprehensive preflight status
- `isPreflightReadyForGeneration()` - Check if preflight is complete

### `signingController.ts`
Code signing orchestration:
- `loadSigningPageData()` - Load signing configuration and status
- `saveSigningConfig()` - Persist signing configuration

## Usage Example

```typescript
// In a hook
import { prepareFormSubmission } from '../controllers/generatorController';

const result = prepareFormSubmission({
  scenarioName,
  selectedTemplate,
  // ... other params
});

if (result.errors.length > 0) {
  setValidationErrors(result.errors);
  return;
}

pipelineActions.generateDesktop(result.pipelineConfig);
```

## Testing

Controllers have comprehensive test coverage. Run tests with:

```bash
npm test -- --run src/controllers/
```
