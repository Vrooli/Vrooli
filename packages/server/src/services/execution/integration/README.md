# ExecutionArchitecture Integration

This module provides the central ExecutionArchitecture factory that wires together all three tiers of Vrooli's execution system.

## Overview

The ExecutionArchitecture class is the main orchestration point that:
- Instantiates all three tiers with proper dependency injection
- Wires up the tier delegation chain (Tier 1 → Tier 2 → Tier 3)
- Initializes shared services (EventBus, StateStores, etc.)
- Provides a unified entry point for external systems
- Implements proper lifecycle management

## Architecture

```
┌─────────────────────────────────────────────────┐
│           ExecutionArchitecture                  │
│                                                  │
│  ┌───────────────────────────────────────────┐  │
│  │  Tier 1: Coordination Intelligence        │  │
│  │  - SwarmStateMachine                      │  │
│  │  - Depends on: Tier 2                     │  │
│  └───────────────────────────────────────────┘  │
│                      ↓                           │
│  ┌───────────────────────────────────────────┐  │
│  │  Tier 2: Process Intelligence             │  │
│  │  - RunStateMachine                        │  │
│  │  - Depends on: Tier 3                     │  │
│  └───────────────────────────────────────────┘  │
│                      ↓                           │
│  ┌───────────────────────────────────────────┐  │
│  │  Tier 3: Execution Intelligence           │  │
│  │  - UnifiedExecutor                        │  │
│  │  - No tier dependencies                   │  │
│  └───────────────────────────────────────────┘  │
│                                                  │
│  Shared Services:                                │
│  - EventBus (Redis-based)                        │
│  - SwarmStateStore (Redis/InMemory)             │
│  - RunStateStore (Redis/InMemory)               │
└─────────────────────────────────────────────────┘
```

## Usage

### Basic Initialization

```typescript
import { getExecutionArchitecture } from './executionArchitecture';

// Get or create the architecture instance
const architecture = await getExecutionArchitecture({
    useRedis: true,          // Use Redis for production
    telemetryEnabled: true,  // Enable monitoring
});

// Access individual tiers
const tier1 = architecture.getTier1();
const tier2 = architecture.getTier2();
const tier3 = architecture.getTier3();
```

### Configuration Options

```typescript
interface ExecutionArchitectureOptions {
    useRedis?: boolean;        // Use Redis (true) or in-memory (false)
    telemetryEnabled?: boolean; // Enable telemetry emission
    logger?: Logger;           // Custom logger instance
    config?: {                 // Configuration overrides
        tier1?: Record<string, unknown>;
        tier2?: Record<string, unknown>;
        tier3?: Partial<UnifiedExecutorConfig>;
    };
}
```

### Tier Communication

All tiers implement the `TierCommunicationInterface`:

```typescript
interface TierCommunicationInterface {
    execute<TInput, TOutput>(
        request: TierExecutionRequest<TInput>
    ): Promise<ExecutionResult<TOutput>>;
    
    getExecutionStatus(executionId: ExecutionId): Promise<ExecutionStatus>;
    cancelExecution(executionId: ExecutionId): Promise<void>;
    getCapabilities?(): Promise<TierCapabilities>;
}
```

### Example: Swarm Coordination (Tier 1)

```typescript
const tier1 = architecture.getTier1();

const request: TierExecutionRequest<SwarmCoordinationInput> = {
    context: {
        executionId: generatePk(),
        swarmId: generatePk(),
        userId: 'user123',
        timestamp: new Date(),
        correlationId: generatePk(),
    },
    input: {
        goal: 'Research and summarize AI trends',
        availableAgents: [...],
    },
    allocation: {
        maxCredits: '1000',
        maxDurationMs: 300000,
        maxMemoryMB: 512,
        maxConcurrentSteps: 10,
    },
};

const result = await tier1.execute(request);
```

### Lifecycle Management

```typescript
// Start the architecture (called automatically by create())
await architecture.start();

// Get current status
const status = architecture.getStatus();
// {
//   initialized: true,
//   tier1Ready: true,
//   tier2Ready: true,
//   tier3Ready: true,
//   eventBusReady: true,
//   stateStoresReady: true
// }

// Get capabilities of all tiers
const capabilities = await architecture.getCapabilities();

// Graceful shutdown
await architecture.stop();
```

## Development vs Production

The architecture automatically adapts based on the environment:

### Development (useRedis: false)
- Uses in-memory state stores
- No external dependencies required
- State is lost on restart
- Ideal for testing and local development

### Production (useRedis: true)
- Uses Redis for state persistence
- Requires Redis connection
- State survives restarts
- Supports distributed deployments

## Error Handling

The architecture includes comprehensive error handling:
- Initialization failures clean up partially created components
- Each tier handles its own errors and propagates them appropriately
- Failed executions return structured error results
- Graceful degradation when optional services are unavailable

## Monitoring and Telemetry

When telemetry is enabled:
- Execution metrics are emitted via EventBus
- Cross-tier events are tracked
- Performance data is collected
- Resource usage is monitored

## Testing

For testing, use the singleton management functions:

```typescript
import { resetExecutionArchitecture } from './executionArchitecture';

// Reset between tests
await resetExecutionArchitecture();

// Create a fresh instance with test config
const testArchitecture = await getExecutionArchitecture({
    useRedis: false,
    telemetryEnabled: false,
});
```