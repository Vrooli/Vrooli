# Decision Boundaries

This document explains how to identify, extract, and document decision boundaries in the Playwright Driver codebase. Following these patterns makes the code easier to understand, test, and evolve.

## What is a Decision Boundary?

A **decision boundary** is a place in code where the system chooses between different behaviors based on input. Extracting these decisions into explicit, named structures:

1. **Makes decisions findable** - Search for `DECISION BOUNDARY` to find all decision points
2. **Makes criteria explicit** - Each option is documented with its behavior
3. **Makes testing easier** - Test each decision path independently
4. **Makes evolution safer** - Adding new options is systematic

## Model Pattern: NetworkHandler Operations

The `src/handlers/network.ts` file demonstrates the ideal pattern for decision boundaries:

```typescript
/**
 * Canonical network operations supported by this handler.
 */
type NetworkOperation = 'mock' | 'block' | 'modifyRequest' | 'modifyResponse' | 'clear';

/**
 * DECISION BOUNDARY: Network Operations
 *
 * Each operation has distinct behavior and side effects:
 * - mock: Intercept requests and return synthetic responses
 * - block: Abort matching requests entirely
 * - modifyRequest: Alter request headers/body before sending
 * - modifyResponse: Alter response headers/body before returning
 * - clear: Remove all active route handlers for the session
 */
const NETWORK_OPERATIONS: Record<NetworkOperation, { description: string }> = {
  mock: { description: 'Mock response for matching requests' },
  block: { description: 'Block requests matching pattern' },
  modifyRequest: { description: 'Modify request headers/body before sending' },
  modifyResponse: { description: 'Modify response headers/body before returning' },
  clear: { description: 'Clear all mocks for session' },
} as const;
```

### Why This Pattern Works

| Aspect | Benefit |
|--------|---------|
| **Type union** | TypeScript enforces valid operations |
| **`DECISION BOUNDARY` comment** | Searchable marker in codebase |
| **Description per option** | Self-documenting behavior |
| **`as const`** | Enables exhaustive switch checking |
| **Record type** | Compile-time verification of completeness |

---

## Template for New Decision Boundaries

When you encounter implicit decision logic (typically `if/else` chains or `switch` statements), consider extracting it using this template:

```typescript
/**
 * Canonical [thing] types supported by this [module].
 */
type [Thing]Type = 'option1' | 'option2' | 'option3';

/**
 * DECISION BOUNDARY: [Thing] Selection
 *
 * [Brief explanation of what this decision controls]
 *
 * Each type has distinct behavior:
 * - option1: [description of what happens]
 * - option2: [description of what happens]
 * - option3: [description of what happens]
 */
const [THING]_TYPES: Record<[Thing]Type, {
  description: string;
  // Add other metadata as needed:
  // sideEffects?: boolean;
  // requiresAuth?: boolean;
  // etc.
}> = {
  option1: { description: '...' },
  option2: { description: '...' },
  option3: { description: '...' },
} as const;

/**
 * Type guard to check if a value is a valid [Thing]Type.
 */
function isValid[Thing]Type(value: string): value is [Thing]Type {
  return value in [THING]_TYPES;
}
```

---

## When to Extract a Decision Boundary

Extract a decision boundary when you see:

### 1. Magic Strings in Switch/If Statements

**Before:**
```typescript
if (action === 'click') {
  // ... click logic
} else if (action === 'hover') {
  // ... hover logic
} else if (action === 'type') {
  // ... type logic
}
```

**After:**
```typescript
type InteractionType = 'click' | 'hover' | 'type';

const INTERACTION_TYPES: Record<InteractionType, { description: string }> = {
  click: { description: 'Click on element' },
  hover: { description: 'Hover over element' },
  type: { description: 'Type text into element' },
} as const;

// Switch now has exhaustiveness checking
switch (action as InteractionType) {
  case 'click': return handleClick(ctx);
  case 'hover': return handleHover(ctx);
  case 'type':  return handleType(ctx);
}
```

### 2. Strategy Selection Based on Input

**Before:**
```typescript
let strategy;
if (useCDP) {
  strategy = new CDPScreencast();
} else {
  strategy = new PollingCapture();
}
```

**After:**
```typescript
type StreamingStrategy = 'cdp-screencast' | 'polling';

/**
 * DECISION BOUNDARY: Frame Streaming Strategy
 *
 * - cdp-screencast: Push-based from Chrome compositor (30-60 FPS)
 * - polling: Pull-based screenshot capture (10-15 FPS)
 */
const STREAMING_STRATEGIES: Record<StreamingStrategy, {
  description: string;
  fpsRange: [number, number];
}> = {
  'cdp-screencast': { description: 'CDP push-based', fpsRange: [30, 60] },
  'polling': { description: 'Screenshot polling', fpsRange: [10, 15] },
} as const;
```

### 3. Configuration-Driven Behavior

**Before:**
```typescript
if (config.telemetry.screenshot.enabled) {
  // collect screenshot
}
if (config.telemetry.dom.enabled) {
  // collect DOM
}
```

**After:**
```typescript
type TelemetryType = 'screenshot' | 'dom' | 'console' | 'network';

/**
 * DECISION BOUNDARY: Telemetry Collection
 *
 * Each type can be independently enabled/disabled via config.
 */
const TELEMETRY_TYPES: Record<TelemetryType, {
  description: string;
  configPath: string;
  collector: (ctx: Context) => Promise<unknown>;
}> = {
  screenshot: {
    description: 'Page screenshot capture',
    configPath: 'telemetry.screenshot.enabled',
    collector: collectScreenshot,
  },
  // ...
} as const;
```

---

## Existing Decision Boundaries

| Location | Boundary | Purpose |
|----------|----------|---------|
| `handlers/network.ts:23-39` | `NETWORK_OPERATIONS` | Network interception operations |
| `session/state-machine.ts` | Phase transitions | Session lifecycle states |
| `recording/selector-config.ts:22-30` | `SELECTOR_STRATEGIES` | Selector generation priority |
| `recording/selector-config.ts:193-232` | `CONFIDENCE_SCORES` | Selector confidence scoring |
| `frame-streaming/strategies/` | Strategy interface | Frame capture methods |
| `outcome/types.ts` | `FailureKind` | Error classification |

---

## Decision Boundaries vs Configuration

Not every choice needs a decision boundary. Use this guide:

| Use Decision Boundary | Use Configuration |
|-----------------------|-------------------|
| Fixed set of behaviors | Numeric/boolean tuning |
| Different code paths | Same code path, different values |
| Needs documentation | Self-explanatory |
| Affects system design | Affects performance |

**Example:**
- "Which telemetry types are supported?" → Decision Boundary
- "How many console logs to keep?" → Configuration (`CONSOLE_MAX_ENTRIES`)

---

## Testing Decision Boundaries

Each decision boundary should have tests that:

1. **Cover all options** - Test each value in the type union
2. **Test the type guard** - Verify invalid values are rejected
3. **Document edge cases** - What happens at boundaries?

```typescript
describe('NETWORK_OPERATIONS', () => {
  it('should have all operations documented', () => {
    const operations: NetworkOperation[] = ['mock', 'block', 'modifyRequest', 'modifyResponse', 'clear'];
    for (const op of operations) {
      expect(NETWORK_OPERATIONS[op]).toBeDefined();
      expect(NETWORK_OPERATIONS[op].description).toBeTruthy();
    }
  });
});
```

---

## Adding a New Decision Boundary: Checklist

- [ ] Define a type union for valid options
- [ ] Create a `const` record with metadata for each option
- [ ] Add `DECISION BOUNDARY` comment with explanation
- [ ] Create a type guard function if needed
- [ ] Update any switch statements to use the type
- [ ] Add the boundary to this document's "Existing Decision Boundaries" table
- [ ] Write tests covering all options

---

## See Also

- `ARCHITECTURE.md` - System architecture and change axes
- `SEAMS.md` - Architectural seams and extension points
- `CONFIG-TIERS.md` - Configuration organization
- `src/handlers/network.ts` - Model implementation
