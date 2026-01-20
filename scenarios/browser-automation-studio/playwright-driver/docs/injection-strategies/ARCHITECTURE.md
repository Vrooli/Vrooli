# Injection Strategy Architecture

This document explains the dependency injection architecture for recording script injection.

## Overview

The injection system uses a strategy pattern with dependency injection to provide swappable injection mechanisms. This allows the recording system to adapt to different browser providers and environments.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                  RecordingContextInitializer                    │
│                                                                 │
│  ┌──────────────────┐    ┌──────────────────────────────────┐  │
│  │  injectionStrategy │◄───│  InjectionStrategyFactory        │  │
│  │  (InjectionStrategy)│    │  - createByName()               │  │
│  └──────────────────┘    │  - create(options)                │  │
│                          │  - selectStrategyForProvider()    │  │
│                          └──────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    InjectionStrategy Interface                   │
├─────────────────────────────────────────────────────────────────┤
│  + name: InjectionStrategyName                                  │
│  + initialize(context, options): Promise<void>                  │
│  + injectScript(page, script): Promise<InjectionResult>         │
│  + verify(page): Promise<boolean>                               │
│  + getStats(): InjectionStrategyStats                           │
│  + resetStats(): void                                           │
│  + cleanup(): Promise<void>                                     │
│  + supportsProvider(providerName): boolean                      │
└─────────────────────────────────────────────────────────────────┘
                    ▲           ▲           ▲
                    │           │           │
        ┌───────────┴───┐   ┌───┴───┐   ┌───┴────────────┐
        │  InitScript   │   │  CDP  │   │    Route       │
        │  Injection    │   │  Injec│   │    Injection   │
        │  Strategy     │   │  tion │   │    Strategy    │
        │               │   │       │   │    (Legacy)    │
        │ RECOMMENDED   │   │FALLBK │   │    DEPRECATED  │
        └───────────────┘   └───────┘   └────────────────┘
```

## File Structure

```
playwright-driver/src/recording/injection/
├── index.ts                      # Public exports
├── types.ts                      # Interface definitions
├── factory.ts                    # InjectionStrategyFactory
├── auto-detector.ts              # Runtime strategy detection
└── strategies/
    ├── index.ts                  # Strategy exports
    ├── init-script-injection.ts  # RECOMMENDED
    ├── cdp-injection.ts          # Fallback
    └── route-injection.ts        # Legacy
```

## Core Components

### InjectionStrategy Interface

The core interface that all strategies implement:

```typescript
interface InjectionStrategy {
  readonly name: InjectionStrategyName;
  initialize(context: BrowserContext, options: InjectionStrategyOptions): Promise<void>;
  injectScript(page: Page, script: string): Promise<InjectionResult>;
  verify(page: Page): Promise<boolean>;
  getStats(): InjectionStrategyStats;
  resetStats(): void;
  cleanup(): Promise<void>;
  supportsProvider(providerName: string): boolean;
}
```

### InjectionStrategyFactory

Creates strategies based on configuration:

```typescript
const factory = new InjectionStrategyFactory();

// Auto-select based on provider
const strategy = factory.create({ providerName: 'rebrowser-playwright' });

// Explicit selection
const cdpStrategy = factory.createByName('cdp-injection');
```

### InjectionAutoDetector

Runtime detection of working strategies (expensive - use sparingly):

```typescript
const detector = new InjectionAutoDetector();
const result = await detector.detect(context, strategyOptions);

if (result.strategy) {
  // Use result.strategy
} else {
  // All strategies failed
  console.error(result.attempts);
}
```

## Strategy Selection Logic

The factory selects strategies in this priority order:

1. **Environment Variable** (`INJECTION_STRATEGY`)
2. **Explicit Option** (`strategyName` parameter)
3. **Provider-Based Auto-Selection**

```typescript
// Priority 1: Check environment variable
const envStrategy = getStrategyFromEnv();
if (envStrategy && envStrategy !== 'auto') {
  return createByName(envStrategy);
}

// Priority 2: Check explicit option
if (options.strategyName && options.strategyName !== 'auto') {
  return createByName(options.strategyName);
}

// Priority 3: Auto-select based on provider
const selectedStrategy = selectStrategyForProvider(options.providerName);
return createByName(selectedStrategy);
```

## Strategy Implementation Details

### InitScriptInjectionStrategy

Uses `context.addInitScript()` for injection.

**Lifecycle:**
1. `initialize()` - Registers script via `context.addInitScript()`
2. `injectScript()` - No-op (script already registered)
3. `verify()` - Checks verification markers via CDP
4. `cleanup()` - Clears internal references (script persists)

**Key Code:**
```typescript
async initialize(context: BrowserContext, options: InjectionStrategyOptions): Promise<void> {
  const initScript = generateRecordingInitScript(options.bindingName);
  await context.addInitScript(initScript);
}
```

### CDPInjectionStrategy

Uses Chrome DevTools Protocol for injection.

**Lifecycle:**
1. `initialize()` - Creates CDP sessions, registers script per page
2. `injectScript()` - Executes script via `Runtime.evaluate`
3. `verify()` - Checks verification markers via CDP
4. `cleanup()` - Removes script registrations, detaches sessions

**Key Code:**
```typescript
async setupPageInjection(page: Page): Promise<void> {
  const session = await page.context().newCDPSession(page);
  await session.send('Page.addScriptToEvaluateOnNewDocument', {
    source: this.initScript,
    runImmediately: true,
  });
}
```

### RouteInjectionStrategy

Wraps existing `setupHtmlInjectionRoute()` for backward compatibility.

**Lifecycle:**
1. `initialize()` - Sets up route handler via `setupHtmlInjectionRoute()`
2. `injectScript()` - No-op (injection happens via route handler)
3. `verify()` - Checks verification markers
4. `cleanup()` - Clears references (route handler persists)

## Integration Points

### RecordingContextInitializer

The main integration point with the injection system:

```typescript
class RecordingContextInitializer {
  private injectionStrategy: InjectionStrategy | null = null;

  async initialize(context: BrowserContext): Promise<void> {
    // Determine strategy to use
    const strategyToUse = getStrategyFromEnv() ?? this.requestedStrategy;

    // Create and initialize strategy
    this.injectionStrategy = createInjectionStrategy({
      strategyName: strategyToUse,
      providerName: playwrightProvider.name,
    });

    await this.injectionStrategy.initialize(context, {
      bindingName: this.bindingName,
      logger: this.logger,
      diagnosticsEnabled: this.diagnosticsEnabled,
      onFirstInjection: () => this.triggerSanityCheck(),
    });
  }
}
```

### Backward Compatibility

The system maintains backward compatibility:

1. Existing `setupHtmlInjectionRoute()` function still works
2. Legacy `InjectionStats` format is supported
3. New `InjectionStrategyStats` provides richer data

## Testing

### Unit Tests

```typescript
describe.each([
  ['init-script', InitScriptInjectionStrategy],
  ['cdp-injection', CDPInjectionStrategy],
  ['route-injection', RouteInjectionStrategy],
])('%s strategy', (name, StrategyClass) => {
  it('should have correct name');
  it('should implement all required methods');
  it('should return proper stats');
  it('should support appropriate providers');
});
```

### Integration Tests

Integration tests require a real browser and should:
1. Create browser context
2. Initialize strategy
3. Navigate to test page
4. Verify injection markers

## Performance Considerations

### Auto-Detection Cost

`InjectionAutoDetector.detect()` is expensive:
- Creates test pages
- Tries multiple strategies
- Waits for verification

**Recommendation:** Cache the result and reuse for subsequent contexts.

### Strategy Overhead

| Strategy | Overhead | Notes |
|----------|----------|-------|
| init-script | Low | Single script registration |
| cdp-injection | Medium | CDP session per page |
| route-injection | High | Network interception per request |

## Future Considerations

### Adding New Strategies

To add a new injection strategy:

1. Create implementation in `strategies/`
2. Implement `InjectionStrategy` interface
3. Add to factory's `createByName()` method
4. Add to `InjectionStrategyName` type
5. Add provider support check
6. Write tests
7. Update documentation

### Strategy Deprecation

When deprecating a strategy:

1. Add deprecation warning in strategy
2. Add JSDoc `@deprecated` tag
3. Update documentation
4. Consider auto-fallback to recommended strategy
