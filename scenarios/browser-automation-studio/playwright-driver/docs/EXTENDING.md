# Extending the Playwright Driver

This guide documents the primary extension points for the Playwright Driver, making it easy to add new capabilities without modifying core infrastructure.

## Change Axes Overview

The Playwright Driver is designed around two primary change axes:

| Change Axis | Primary Files | Extension Mechanism |
|-------------|--------------|---------------------|
| **Instruction Handlers** | `src/handlers/` | Handler Registry Pattern |
| **Recording Action Types** | `src/recording/` | Action Type Registry + Executor Registry |
| **Selector Strategies** | `src/recording/selector-config.ts` | Shared Configuration Module |

---

## 1. Adding a New Instruction Handler

Instruction handlers process commands from the API (e.g., `navigate`, `click`, `screenshot`).

### Steps

1. **Create the handler file**

```typescript
// src/handlers/my-handler.ts
import { BaseHandler, type HandlerResult, type HandlerContext } from './base';
import type { Instruction } from '../types/instruction';

export class MyHandler extends BaseHandler {
  readonly supportedTypes = ['my-action'] as const;

  async handle(instruction: Instruction, context: HandlerContext): Promise<HandlerResult> {
    // Your implementation
    return {
      success: true,
      data: { /* result data */ },
    };
  }
}
```

2. **Register the handler**

```typescript
// src/server.ts - in registerHandlers()
import { MyHandler } from './handlers/my-handler';

handlerRegistry.register(new MyHandler());
```

3. **Add tests**

```typescript
// tests/unit/handlers/my-handler.test.ts
```

### Files Changed
- `src/handlers/my-handler.ts` (new)
- `src/handlers/index.ts` (add export)
- `src/server.ts` (register)

---

## 2. Adding a New Recording Action Type

Recording action types capture user interactions during Record Mode.

### Steps

1. **Add to action types** (`src/recording/action-types.ts`)

```typescript
// Add to ACTION_TYPES array
export const ACTION_TYPES: readonly ActionType[] = [
  // ... existing types
  'my-action',
] as const;

// Add alias mapping if needed
const ACTION_TYPE_ALIASES: Record<string, ActionType> = {
  // ... existing
  'my-raw-event': 'my-action',
};

// Add proto kind mapping
const ACTION_KIND_MAP: Record<ActionType, RecordedActionKind> = {
  // ... existing
  'my-action': 'RECORDED_ACTION_TYPE_CLICK', // or appropriate kind
};
```

2. **Add type to union** (`src/recording/types.ts`)

```typescript
// The ActionType is derived from SelectorStrategyType, but if adding
// new action types, update the source
```

3. **Register executor** (`src/recording/action-executor.ts`)

```typescript
// At the bottom of the file
registerActionExecutor('my-action', async (action, context, baseResult) => {
  if (isSelectorMissing(action)) {
    return {
      ...baseResult,
      error: { message: 'My action missing selector', code: 'UNKNOWN' },
    };
  }

  const selector = action.selector!.primary;
  const validation = await context.validateSelector(selector);
  if (!validation.valid) {
    return selectorError(baseResult, validation);
  }

  // Execute the action
  await context.page.myPlaywrightMethod(selector, { timeout: context.timeout });
  return { ...baseResult, success: true };
});
```

4. **Update injector for capture** (`src/recording/injector.ts`)

Add event listener in the stringified script to capture the action:

```javascript
// Inside getRecordingScript() template literal
document.addEventListener('my-event', function(e) {
  captureAction('my-action', e.target, e, {
    // payload data
  });
}, true);
```

5. **Add tests**

```typescript
// tests/unit/recording/action-types.test.ts - add test cases
// tests/unit/recording/action-executor.test.ts - add executor tests
```

### Files Changed
- `src/recording/action-types.ts`
- `src/recording/action-executor.ts`
- `src/recording/injector.ts`
- `tests/unit/recording/action-types.test.ts`
- `tests/unit/recording/action-executor.test.ts`

---

## 3. Adding a New Selector Strategy

Selector strategies determine how elements are identified in the DOM.

### Steps

1. **Update shared configuration** (`src/recording/selector-config.ts`)

```typescript
// Add to SELECTOR_STRATEGIES
export const SELECTOR_STRATEGIES = [
  // ... existing
  'my-strategy',
] as const;

// Add confidence score
export const CONFIDENCE_SCORES = {
  // ... existing
  myStrategy: 0.75,
} as const;

// Add specificity score
export const SPECIFICITY_SCORES = {
  // ... existing
  myStrategy: 70,
} as const;
```

2. **Implement in selectors.ts** (`src/recording/selectors.ts`)

```typescript
function generateMyStrategySelector(element: Element): SelectorCandidate | null {
  // Your selector generation logic
  const myAttr = element.getAttribute('my-attr');
  if (myAttr) {
    const selector = `[my-attr="${escapeCssAttributeValue(myAttr)}"]`;
    if (isUniqueSelector(selector)) {
      return {
        type: 'my-strategy',
        value: selector,
        confidence: CONFIDENCE_SCORES.myStrategy,
        specificity: SPECIFICITY_SCORES.myStrategy,
      };
    }
  }
  return null;
}

// Add call in generateSelectors()
export function generateSelectors(element: Element, options: SelectorGeneratorOptions = {}): SelectorSet {
  // ... existing candidates

  // Strategy N: My strategy
  const myCandidate = generateMyStrategySelector(element);
  if (myCandidate) {
    candidates.push(myCandidate);
  }

  // ... rest
}
```

3. **Update injector.ts** (`src/recording/injector.ts`)

Add the same logic in the stringified JavaScript (note: config values are automatically injected via `serializeConfigForBrowser()`):

```javascript
// Inside getRecordingScript() template literal
function generateMyStrategySelector(element) {
  const myAttr = element.getAttribute('my-attr');
  if (myAttr) {
    const selector = '[my-attr="' + escapeCssAttr(myAttr) + '"]';
    if (isUniqueSelector(selector)) {
      return {
        type: 'my-strategy',
        value: selector,
        confidence: CONFIG.CONFIDENCE.myStrategy,
        specificity: CONFIG.SPECIFICITY.myStrategy,
      };
    }
  }
  return null;
}
```

4. **Add tests**

```typescript
// tests/unit/recording/selector-config.test.ts - verify config
// tests/integration/injector-selectors.test.ts - verify browser behavior
```

### Files Changed
- `src/recording/selector-config.ts`
- `src/recording/selectors.ts`
- `src/recording/injector.ts`
- `tests/unit/recording/selector-config.test.ts`
- `tests/integration/injector-selectors.test.ts`

---

## Architecture Notes

### Single Source of Truth

- **Selector Configuration**: All selector-related configuration lives in `selector-config.ts`
  - Patterns, confidence scores, specificity scores
  - Exported via `serializeConfigForBrowser()` for injection

### Testing Strategy

| Test Type | Purpose | Location |
|-----------|---------|----------|
| Unit | Test individual functions | `tests/unit/` |
| Integration | Test browser-context code | `tests/integration/` |
| E2E | Test full workflow | `tests/e2e/` |

### Key Design Patterns

1. **Handler Registry**: Decouples instruction routing from implementation
2. **Action Executor Registry**: Decouples action replay from controller
3. **Config Serialization**: Maintains single source of truth for browser-context code

---

## Checklist for Extensions

### New Handler
- [ ] Create handler class extending `BaseHandler`
- [ ] Export from `handlers/index.ts`
- [ ] Register in `server.ts`
- [ ] Add unit tests
- [ ] Update this documentation if needed

### New Action Type
- [ ] Add to `ACTION_TYPES` in `action-types.ts`
- [ ] Add alias mapping if needed
- [ ] Add proto kind mapping
- [ ] Register executor in `action-executor.ts`
- [ ] Update `injector.ts` for capture
- [ ] Add unit tests for action-types
- [ ] Add unit tests for executor
- [ ] Update this documentation if needed

### New Selector Strategy
- [ ] Add to `SELECTOR_STRATEGIES` in `selector-config.ts`
- [ ] Add confidence score
- [ ] Add specificity score
- [ ] Implement in `selectors.ts`
- [ ] Mirror implementation in `injector.ts`
- [ ] Add config test in `selector-config.test.ts`
- [ ] Add integration test in `injector-selectors.test.ts`
- [ ] Update this documentation if needed
