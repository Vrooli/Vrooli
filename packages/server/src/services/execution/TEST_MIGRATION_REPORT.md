# Test Framework Migration Report - Execution Service

## Current State Analysis

### Test Frameworks in Use
The execution service currently uses **THREE different testing frameworks**:

1. **Jest** - Used in `__test__` directories (13 files)
2. **Vitest** - Used in `tier3/strategies/reasoning/__tests__` and `tier3/strategies/__tests__` (7 files)
3. **Mocha/Chai/Sinon** - Used in `__test__` directories (3 files)

### Directory Structure Issues

#### Directories Found:
- `/src/services/execution/__test__/` - Jest-based tests with jest.config.js
- `/src/services/execution/tier3/strategies/__test__/` - Mocha-based tests
- `/src/services/execution/tier3/strategies/__tests__/` - Vitest-based tests
- `/src/services/execution/tier3/strategies/reasoning/__tests__/` - Vitest-based tests
- `/src/services/execution/tier3/strategies/adapters/__tests__/` - Jest-based tests

## Files by Testing Framework

### Jest Files (Need Migration)
```
__test__/
├── conversationalStrategyIntegration.test.ts
├── deterministicAdapter.test.ts
├── deterministicStrategy.enhanced.test.ts
├── deterministicStrategyIntegration.test.ts
├── swarmExecutionService.test.ts
├── jest.config.js
├── setupTests.ts
├── integration/
│   └── executionFlow.test.ts
├── tier1/
│   ├── redisSwarmStateStore.test.ts
│   └── swarmStateMachine.test.ts
├── tier2/
│   └── runStateMachine.test.ts
└── tier3/
    └── strategies.test.ts

tier3/strategies/adapters/__tests__/
└── conversationalAdapter.test.ts

tier3/strategies/__tests__/
└── conversationalStrategy.enhanced.test.ts
```

### Vitest Files (Need Migration)
```
tier3/strategies/reasoning/__tests__/
├── validationEngine.test.ts
├── legacyInputHandler.test.ts
├── legacyOutputGenerator.test.ts
├── fourPhaseExecutor.test.ts
└── legacyCostEstimator.test.ts

tier3/strategies/__tests__/
├── reasoningStrategy.test.ts
└── reasoningStrategy.integration.test.ts
```

### Mocha/Chai/Sinon Files (Already Correct Framework)
```
tier3/strategies/__test__/
├── strategyIntegration.test.ts
├── reasoningStrategy.test.ts
└── strategySelector.test.ts
```

## Migration Plan

### Phase 1: Framework Standardization
1. **Convert all Jest tests** to use Mocha/Chai/Sinon
   - Replace `@jest/globals` imports with `mocha`, `chai`, and `sinon`
   - Convert `jest.fn()` to `sinon.stub()` or `sinon.spy()`
   - Replace `expect().toEqual()` with `expect().to.equal()`
   - Replace `expect().toBe()` with `expect().to.equal()`
   - Replace `expect().toHaveBeenCalled()` with `sinon.assert.called()`
   - Remove Jest configuration files

2. **Convert all Vitest tests** to use Mocha/Chai/Sinon
   - Replace `vitest` imports with `mocha`, `chai`, and `sinon`
   - Convert `vi.fn()` to `sinon.stub()` or `sinon.spy()`
   - Update test syntax to match Mocha patterns

### Phase 2: Directory Consolidation
1. Move all tests from `__tests__` directories to `__test__` directories
2. Create proper directory structure:
   ```
   execution/
   ├── __test__/
   │   ├── integration/
   │   ├── tier1/
   │   ├── tier2/
   │   └── tier3/
   ├── tier1/__test__/
   ├── tier2/__test__/
   └── tier3/
       ├── __test__/
       └── strategies/__test__/
   ```

### Phase 3: Configuration Cleanup
1. Remove `jest.config.js` from `__tests__` directory
2. Remove `setupTests.ts` or convert to Mocha setup file
3. Ensure all tests use the server's main Mocha configuration

## Configuration Changes Needed

### Remove:
- `/src/services/execution/__tests__/jest.config.js`
- `/src/services/execution/__tests__/setupTests.ts`
- All Jest and Vitest dependencies from package.json (if not used elsewhere)

### Update:
- Test scripts in package.json to ensure they pick up all `__test__` directories
- Import statements in all test files
- Mock setup to use Sinon instead of Jest/Vitest mocks

## Key Differences to Address

### Jest to Mocha/Chai/Sinon
```typescript
// Jest
import { describe, it, expect, beforeEach } from "@jest/globals";
const mockFn = jest.fn();
expect(result).toBe(expected);
expect(mockFn).toHaveBeenCalledWith(arg);

// Mocha/Chai/Sinon
import { describe, it, beforeEach } from "mocha";
import { expect } from "chai";
import sinon from "sinon";
const mockFn = sinon.stub();
expect(result).to.equal(expected);
sinon.assert.calledWith(mockFn, arg);
```

### Vitest to Mocha/Chai/Sinon
```typescript
// Vitest
import { describe, it, expect, vi } from "vitest";
const mockFn = vi.fn();

// Mocha/Chai/Sinon
import { describe, it } from "mocha";
import { expect } from "chai";
import sinon from "sinon";
const mockFn = sinon.stub();
```

## Recommended Migration Order

1. **High Priority**: Convert integration tests in `__tests__/integration/`
2. **Medium Priority**: Convert tier-specific tests
3. **Low Priority**: Convert strategy tests (some already use Mocha)

## Estimated Effort

- **Total Files to Migrate**: 20 files
- **Jest Files**: 13 files
- **Vitest Files**: 7 files
- **Estimated Time**: 2-3 days for complete migration

## Benefits of Migration

1. **Consistency**: Single testing framework across the entire codebase
2. **Maintenance**: Easier to maintain with consistent patterns
3. **Configuration**: Simplified test configuration
4. **Learning Curve**: Developers only need to know one testing framework
5. **Tooling**: Consistent tooling and debugging experience