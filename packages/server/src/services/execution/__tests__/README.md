# Execution Architecture Tests

This directory contains comprehensive tests for the three-tier execution architecture.

## Test Structure

```
__tests__/
├── swarmExecutionService.test.ts    # Main service entry point tests
├── tier1/                           # Tier 1 - Coordination Intelligence tests
│   ├── swarmStateMachine.test.ts    # Swarm lifecycle and state management
│   └── redisSwarmStateStore.test.ts # Redis persistence for swarm state
├── tier2/                           # Tier 2 - Process Intelligence tests
│   └── runStateMachine.test.ts      # Run orchestration and navigation
├── tier3/                           # Tier 3 - Execution Intelligence tests
│   └── strategies.test.ts           # Conversational, Reasoning, and Deterministic strategies
├── integration/                     # Integration tests
│   └── executionFlow.test.ts        # End-to-end execution flow tests
├── jest.config.js                   # Jest configuration
├── setupTests.ts                    # Test environment setup
└── README.md                        # This file
```

## Running Tests

### Run all execution tests
```bash
cd packages/server
pnpm test src/services/execution/__tests__
```

### Run specific test suites
```bash
# Main service tests
pnpm test src/services/execution/__tests__/swarmExecutionService.test.ts

# Tier 1 tests
pnpm test src/services/execution/__tests__/tier1

# Tier 2 tests
pnpm test src/services/execution/__tests__/tier2

# Tier 3 tests
pnpm test src/services/execution/__tests__/tier3

# Integration tests
pnpm test src/services/execution/__tests__/integration
```

### Run with coverage
```bash
pnpm test:coverage src/services/execution/__tests__
```

### Debug mode
```bash
DEBUG=1 pnpm test src/services/execution/__tests__
```

## Test Categories

### Unit Tests

1. **SwarmExecutionService** - Tests the main service entry point
   - Swarm creation with permissions
   - Run execution with routine loading
   - Status checking
   - Cancellation handling
   - Event coordination

2. **Tier 1 Components**
   - SwarmStateMachine: State transitions, run requests, resource alerts
   - RedisSwarmStateStore: Persistence, indexing, TTL management

3. **Tier 2 Components**
   - RunStateMachine: Navigation, step execution, checkpointing
   - Branch coordination and parallel execution
   - Performance monitoring and optimization

4. **Tier 3 Components**
   - ConversationalStrategy: Natural language processing
   - ReasoningStrategy: Analytical frameworks
   - DeterministicStrategy: Automated execution

### Integration Tests

1. **End-to-End Execution Flow**
   - Complete swarm and run lifecycle
   - Multi-tier event coordination
   - Concurrent run execution
   - Error handling and recovery
   - Resource management

## Test Utilities

### Mocking

- All external dependencies (Redis, LLM services) are mocked
- Event bus is real but isolated per test
- State stores use in-memory implementations

### Common Test Patterns

```typescript
// Setup a test swarm
const swarmConfig = {
    swarmId: "test-swarm",
    name: "Test Swarm",
    // ... configuration
};

// Mock dependencies
sandbox.stub(authService, "checkPermission").resolves(true);
sandbox.stub(routineService, "loadRoutine").resolves(mockRoutine);

// Execute and verify
const result = await service.startSwarm(swarmConfig);
expect(result.swarmId).toBe("test-swarm");
```

## Coverage Goals

- **Target**: 80% coverage for all metrics (branches, functions, lines, statements)
- **Current Focus**: Core execution paths and error handling
- **Future**: Edge cases and performance scenarios

## Debugging Failed Tests

1. Check test logs for detailed error messages
2. Use `DEBUG=1` environment variable for verbose output
3. Examine event flow in integration tests
4. Verify mock setup matches expected behavior
5. Check for timing issues in async operations

## Adding New Tests

When adding new functionality:

1. Create unit tests for individual components
2. Add integration tests for cross-tier interactions
3. Include both success and failure scenarios
4. Test resource limits and edge cases
5. Verify event emissions and handling

## Performance Considerations

- Tests use in-memory stores for speed
- Long-running operations are mocked
- Timeouts are set to 30 seconds for integration tests
- Parallel test execution is supported