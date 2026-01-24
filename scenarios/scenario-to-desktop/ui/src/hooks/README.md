# Hooks Layer

Hooks manage **React state and side effects**, serving as thin orchestrators that compose store access, controller calls, and React lifecycle.

## Architecture Role

```
Components (UI) → Hooks (React State) → Controllers (Business Logic) → Services (Pure Functions)
```

## Key Principles

1. **Thin orchestrators** - Delegate business logic to controllers
2. **React-specific** - Handle useEffect, useCallback, useMemo, etc.
3. **State composition** - Combine multiple stores and state sources
4. **Component interface** - Provide a clean API for components to consume

## Files

### Page Hooks (Main Orchestrators)

#### `useGeneratorPage.ts`
Main hook for the Generator page:
- Composes form state, pipeline actions, server sync, and signing
- Handles form submission via `prepareFormSubmission` controller
- Manages scenario selection and defaults

#### `usePreflightSection.ts`
Hook for the Preflight section:
- Composes preflight state from store
- Uses `buildJobStepMap` and `resolveAllStepStatuses` from controller
- Manages view mode and copy/download actions

### State Hooks

#### `useFormState.ts`
Form state composition:
- Aggregates form store selectors
- Provides form actions and validation

#### `usePipelineActions.ts`
Pipeline action wrapper:
- Wraps store actions with additional logic
- Manages generate and connection test mutations

#### `useScenarioSync.ts`
Server state synchronization:
- Handles form state persistence to server
- Manages preflight seed loading
- Auto-saves form changes

### Utility Hooks

#### `useGeneratorModals.ts`
Modal state management for Generator page

#### `useSigningConfig.ts`
Signing configuration state

#### `usePipelineButton.ts`
Shared logic for pipeline action buttons

## Usage Example

```typescript
// In a component
import { useGeneratorPage } from '../hooks/useGeneratorPage';

function GeneratorPage() {
  const {
    formState,
    pipelineActions,
    handleSubmit,
    scenarios,
  } = useGeneratorPage({
    scenarioName,
    selectedTemplate,
    // ... other props
  });

  return <form onSubmit={handleSubmit}>...</form>;
}
```

## Testing

Hooks should be tested with React Testing Library's `renderHook`:

```typescript
import { renderHook } from '@testing-library/react';
import { useGeneratorPage } from './useGeneratorPage';

test('handles form submission', () => {
  const { result } = renderHook(() => useGeneratorPage({...}));
  // Test the hook's behavior
});
```
