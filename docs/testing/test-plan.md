# Test Plan

This document details the Test Plan for the Vrooli platform. It specifies the scope, objectives, resources, and criteria for testing activities, aligning with the overall [Test Strategy](./test-strategy.md).

## Quick Reference

### Common Test Commands
```bash
# Run all tests (⚠️ 5+ min timeout needed)
pnpm test

# Package-specific tests
cd packages/server && pnpm test          # Server tests (5+ min)
cd packages/ui && pnpm test              # UI tests (3+ min)
cd packages/jobs && pnpm test            # Jobs tests (3+ min)

# Watch mode for development
cd packages/server && pnpm test-watch

# Coverage reports
cd packages/server && pnpm test-coverage # (5+ min)

# Type checking (use specific files when possible)
cd packages/server && tsc --noEmit src/file.ts  # Single file
cd packages/server && pnpm run type-check        # Full package (4+ min)
```

### Critical Testing Rules
1. **Always use `.js` extensions** in TypeScript imports
2. **Never mock Redis/PostgreSQL** - use testcontainers
3. **Use `__test` directories** (not `__tests`)
4. **Wrap database tests** with `withDbTransaction()`
5. **Set extended timeouts** for long-running operations

## 1. Introduction

The purpose of this Test Plan is to provide a clear and comprehensive guide for all testing activities related to the Vrooli project. It ensures that testing is conducted in a systematic and effective manner, contributing to the delivery of a high-quality product.

This plan integrates with the [AI Maintenance System](../ai-maintenance/README.md) for continuous test quality improvement and coverage tracking.

## 2. Scope of Testing

### 2.1. Features to be Tested

Testing will cover all major functional and non-functional aspects of the Vrooli platform, including but not limited to:

-   **Core Platform Functionality:** (As defined in `docs/roadmap.md` Phase 1)
    -   User Authentication and Authorization
    -   AI Chat Functionality (including message storage, context handling)
    -   Routines Implementation (creation, execution, triggers, visualization)
    -   Commenting, Reporting, and Moderation Features
-   **Three-Tier AI System:**
    -   Tier 1: Coordination Intelligence
    -   Tier 2: Process Intelligence
    -   Tier 3: Execution Intelligence
    -   Event Bus: Redis-based communication
-   **UI/UX:**
    -   Navigation and User Flows
    -   Responsive Design and Mobile Experience
    -   Component Library (Storybook integration and component behavior)
    -   MSW (Mock Service Worker) integration for isolated testing
-   **API Endpoints:** All publicly exposed and internal APIs
-   **Integrations:** Interactions with any third-party services or internal microservices
-   **Data Integrity:** Verification of data storage, retrieval, and consistency

### 2.2. Features Not to be Tested (Initially)

-   Advanced AI agent swarm behaviors beyond defined routine execution (emergent capabilities)
-   Specific third-party integrations not yet implemented
-   Exhaustive performance and load testing under extreme conditions (phased approach)

### 2.3. Current Test Coverage Baseline

**Last Updated: 2025-06-19**

Current test coverage status:
- **Shared Package**: **92.73%** coverage (16,400/17,684 statements)
  - Branches: 91.97% (2,293/2,493)
  - Functions: 86.22% (651/755)
  - Lines: 92.73% (16,400/17,684)
- **Server Package**: ~151 test files (coverage report needed)
- **UI Package**: ~66 test files (coverage report needed)
- **Jobs Package**: ~17 test files (coverage report needed)

Target: **80%+ coverage across all packages by Phase 2 (T+12-24 weeks)**

*Note: Run `pnpm test-coverage` in each package directory for up-to-date metrics*

## 3. Testing Objectives

-   **Identify Defects:** Discover and document software defects as early as possible
-   **Verify Requirements:** Ensure that the application meets the specified functional and non-functional requirements
-   **Validate User Experience:** Confirm that the application is intuitive, usable, and meets user expectations
-   **Assess Quality:** Provide an overall assessment of the application's quality and readiness for release
-   **Ensure Stability:** Verify the stability and reliability of the application under normal operating conditions
-   **Maintain Coverage:** Achieve and maintain 80%+ test coverage across all packages
-   **Support AI Development:** Ensure emergent capabilities have proper test infrastructure

### 3.1. Future Test Categories

These testing categories are not currently implemented but should be considered for future phases:

#### Snapshot Testing
- **Purpose**: Detect unexpected UI changes by comparing component output
- **Tools**: Vitest snapshot support, React Testing Library
- **Implementation**: Phase 3+ when UI stabilizes
- **Example**:
  ```typescript
  test('Button renders correctly', () => {
      const { container } = render(<Button>Click me</Button>);
      expect(container).toMatchSnapshot();
  });
  ```

#### Visual Regression Testing
- **Purpose**: Catch visual bugs by comparing screenshots
- **Tools**: Storybook with Chromatic, Percy, or similar
- **Implementation**: After Storybook coverage expands
- **Use Cases**: Design system components, critical UI flows

#### Accessibility Testing
- **Purpose**: Ensure platform is usable by everyone
- **Tools**: jest-axe, @testing-library/jest-dom, Pa11y
- **Implementation**: Should be integrated into component tests
- **Example**:
  ```typescript
  import { axe } from 'jest-axe';
  
  test('form is accessible', async () => {
      const { container } = render(<LoginForm />);
      const results = await axe(container);
      expect(results).toHaveNoViolations();
  });
  ```

## 4. Test Resources

### 4.1. Testing Framework and Tools

-   **Primary Framework:** Vitest (NOT Mocha/Chai)
-   **Test Containers:** For Redis and PostgreSQL (never mock these)
-   **UI Testing:** Vitest + React Testing Library + MSW
-   **E2E Testing:** Cypress (installed, implementation pending)
-   **Coverage:** Vitest coverage with C8
-   **Component Testing:** Storybook for visual testing

### 4.2. Test Environment

-   **Local Development:** Docker-based test containers
-   **CI Environment:** GitHub Actions with containerized services
-   **Test Data:** Fixtures using real application shape functions (see [fixtures-and-data-flow.md](./fixtures-and-data-flow.md))

### 4.3. Test Helpers and Infrastructure

-   **Database Helpers:** `/packages/server/src/__test/helpers/`
-   **Transaction Wrapper:** `withDbTransaction()` for test isolation
-   **Mock Services:** `/packages/jobs/src/__test/mocks/services.ts`
-   **UI Test Setup:** `/packages/ui/src/__test/setup.vitest.ts`

### 4.4. Detailed Test Infrastructure

#### Database Test Helpers
```typescript
// packages/server/src/__test/helpers/db.ts
export async function withDbTransaction<T>(
    fn: (tx: PrismaTransaction) => Promise<T>
): Promise<T> {
    // Automatically rolls back after test
}

// packages/server/src/__test/helpers/testingServer.ts
export function testingServer() {
    // Returns Express app configured for testing
}
```

#### Mock Services
```typescript
// packages/jobs/src/__test/mocks/services.ts
export const mockLlmService = {
    generateResponse: vi.fn(),
    // Other LLM methods
};

export const mockEmailService = {
    sendEmail: vi.fn(),
    // Other email methods
};
```

#### Logger Mocking
```typescript
// packages/server/src/__test/logger.mock.ts
vi.mock('../logger', () => ({
    logger: {
        error: vi.fn(),
        warn: vi.fn(),
        info: vi.fn(),
        debug: vi.fn(),
    }
}));
```

#### Session/Auth Testing Utilities
```typescript
// packages/ui/src/__test/fixtures/sessionFixtures.ts
export const testUser = {
    id: "123456789012345678",
    roles: ["User"],
    permissions: ["Read", "Write"],
    // Complete session data
};

export function createMockSession(overrides?: Partial<Session>) {
    return { ...testUser, ...overrides };
}

## 5. Test Writing Guidelines

### 5.1. Import Requirements

```typescript
// ✅ CORRECT - Always use .js extension for relative imports
import { UserModel } from "../models/User.js";
import { generatePK } from "@vrooli/shared";  // Package imports (@vrooli/*) don't need extension

// ❌ WRONG - Missing .js extension on relative import
import { UserModel } from "../models/User";
import { generatePK } from "@vrooli/shared/id/index.js";  // Never use deep imports from packages
```

### 5.2. Database Test Pattern

```typescript
import { withDbTransaction } from "../__test/helpers/db.js";

describe("UserModel", () => {
    it("should create user", async () => {
        await withDbTransaction(async (tx) => {
            // Your test code here - transaction auto-rollbacks
            const user = await UserModel.create(tx, userData);
            expect(user).toBeDefined();
        });
    });
});
```

### 5.3. Architectural Patterns

-   **Use Real Functions:** Always use actual shape functions from the codebase
-   **Avoid Mocks:** Prefer integration over unit tests where practical
-   **Round-Trip Testing:** Test full data flow (UI → API → DB → API → UI)
-   **Fixture Reuse:** Leverage existing fixtures from test helpers

See [fixtures-and-data-flow.md](./fixtures-and-data-flow.md) for comprehensive patterns.

### 5.4. MSW (Mock Service Worker) Integration

MSW is used in the UI package to intercept and mock API calls during testing:

#### Basic Setup
```typescript
// packages/ui/src/__test/setup.vitest.ts
import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);

// Start server before all tests
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));

// Reset handlers after each test
afterEach(() => server.resetHandlers());

// Clean up after all tests
afterAll(() => server.close());
```

#### Creating Handlers
```typescript
// packages/ui/src/__test/handlers/bookmark.ts
import { http, HttpResponse } from 'msw';
import { endpointsBookmark } from "@vrooli/shared";

export const bookmarkHandlers = [
    // Create bookmark
    http.post(`/api/v2${endpointsBookmark.createOne.endpoint}`, async ({ request }) => {
        const body = await request.json();
        return HttpResponse.json({
            data: {
                __typename: "Bookmark",
                id: "123456789012345678",
                ...body
            }
        });
    }),
    
    // Get bookmarks
    http.get(`/api/v2${endpointsBookmark.findMany.endpoint}`, () => {
        return HttpResponse.json({
            data: {
                edges: [{ node: bookmarkFixture }],
                pageInfo: { hasNextPage: false }
            }
        });
    })
];
```

#### Using in Tests
```typescript
// Override handlers for specific tests
import { server } from '../__test/setup.vitest.ts';
import { http, HttpResponse } from 'msw';

test('handles API error', async () => {
    // Override handler for this test only
    server.use(
        http.post('/api/v2/bookmark', () => {
            return HttpResponse.json(
                { error: 'Server error' },
                { status: 500 }
            );
        })
    );
    
    // Test error handling
    render(<BookmarkButton />);
    // ... test implementation
});
```

#### Best Practices
- Use real endpoint paths from `@vrooli/shared`
- Return realistic response structures matching API types
- Test both success and error scenarios
- Use `server.use()` for test-specific overrides

### 5.5. Test Naming Conventions

#### File Naming
- **Test files**: Use `.test.ts` suffix (NOT `.spec.ts`)
- **Test directories**: Use `__test` (NOT `__tests`)
- **Location**: Place tests close to the code they test
  ```
  src/
    models/
      User.ts
      User.test.ts
    services/
      auth.ts
      auth.test.ts
  ```

#### Test Description Naming
```typescript
// ✅ GOOD - Clear, specific descriptions
describe('UserModel', () => {
    describe('create', () => {
        test('should create user with valid data', async () => {});
        test('should throw error when email is invalid', async () => {});
        test('should hash password before saving', async () => {});
    });
});

// ❌ BAD - Vague or redundant descriptions
describe('User', () => {
    test('works', async () => {});
    test('test create', async () => {});
    test('it should work correctly', async () => {});
});
```

#### Naming Patterns
- **Unit tests**: `should [expected behavior] when [condition]`
- **Integration tests**: `[feature] should [behavior] when [scenario]`
- **E2E tests**: `user can [action] in [context]`

Examples:
```typescript
// Unit test
test('should return formatted date when valid timestamp provided', () => {});

// Integration test
test('bookmark creation should update user stats when successful', async () => {});

// E2E test
test('user can complete registration flow with email verification', async () => {});
```

#### Test Data Naming
```typescript
// Use descriptive names for test data
const validUserData = { /* ... */ };
const userWithoutEmail = { /* ... */ };
const adminUser = { /* ... */ };

// Avoid generic names
const data = { /* ... */ };  // ❌
const obj = { /* ... */ };    // ❌
const test1 = { /* ... */ };  // ❌
```

## 6. AI Tier Testing Guidance

### 6.1. Three-Tier Architecture Overview

The AI system consists of three distinct tiers that communicate via Redis event bus:
- **Tier 1**: Coordination Intelligence (strategic planning, resource allocation)
- **Tier 2**: Process Intelligence (task orchestration, routine navigation)
- **Tier 3**: Execution Intelligence (direct task execution, tool integration)

### 6.2. Testing Each Tier in Isolation

#### Tier 1 Testing (Coordination)
```typescript
import { describe, test, expect, beforeEach } from 'vitest';
import { TierOneCoordinator } from '../services/execution/tier1/tierOneCoordinator.js';
import { createTestEventBus } from '../__test/helpers/eventBus.js';

describe('Tier 1 - Coordination Intelligence', () => {
    let coordinator: TierOneCoordinator;
    let eventBus: TestEventBus;
    
    beforeEach(() => {
        eventBus = createTestEventBus();
        coordinator = new TierOneCoordinator({ eventBus });
    });
    
    test('should allocate resources based on task priority', async () => {
        const task = { id: '123456789012345678', priority: 'high', type: 'routine' };
        await coordinator.allocateResources(task);
        
        expect(eventBus.published).toContainEqual({
            channel: 'tier2:task:assign',
            data: expect.objectContaining({ taskId: task.id })
        });
    });
});
```

#### Tier 2 Testing (Process)
```typescript
describe('Tier 2 - Process Intelligence', () => {
    test('should orchestrate routine execution steps', async () => {
        const routine = { id: '123456789012345678', steps: ['input', 'process', 'output'] };
        const orchestrator = new TierTwoOrchestrator({ eventBus });
        
        await orchestrator.executeRoutine(routine);
        
        // Verify each step is delegated to Tier 3
        expect(eventBus.published).toHaveLength(3);
        expect(eventBus.published[0].channel).toBe('tier3:execute:step');
    });
});
```

#### Tier 3 Testing (Execution)
```typescript
describe('Tier 3 - Execution Intelligence', () => {
    test('should execute tool with proper context', async () => {
        const executor = new TierThreeExecutor({ eventBus });
        const task = { tool: 'llm:prompt', input: 'Test prompt' };
        
        const result = await executor.executeTool(task);
        
        expect(result).toMatchObject({
            success: true,
            output: expect.any(String)
        });
    });
});
```

### 6.3. Event Bus Testing Patterns

#### Testing Event Communication
```typescript
import { withRedisContainer } from '../__test/helpers/redis.js';

test('tiers should communicate via event bus', async () => {
    await withRedisContainer(async (redisClient) => {
        const eventBus = new EventBus(redisClient);
        const received = [];
        
        // Subscribe to events
        await eventBus.subscribe('tier2:*', (event) => {
            received.push(event);
        });
        
        // Publish from Tier 1
        await eventBus.publish('tier2:task:assign', { taskId: '123' });
        
        // Verify receipt
        await waitFor(() => {
            expect(received).toHaveLength(1);
            expect(received[0].data.taskId).toBe('123');
        });
    });
});
```

#### Testing Event Bus Isolation
```typescript
test('event bus channels should be isolated', async () => {
    const tier1Events = [];
    const tier2Events = [];
    
    await eventBus.subscribe('tier1:*', (e) => tier1Events.push(e));
    await eventBus.subscribe('tier2:*', (e) => tier2Events.push(e));
    
    await eventBus.publish('tier1:status', { status: 'ready' });
    await eventBus.publish('tier2:task', { id: '123' });
    
    expect(tier1Events).toHaveLength(1);
    expect(tier2Events).toHaveLength(1);
    expect(tier1Events[0].channel).not.toBe(tier2Events[0].channel);
});
```

### 6.4. Integration Testing Between Tiers

```typescript
describe('Multi-Tier Integration', () => {
    test('complete task flow through all tiers', async () => {
        const swarm = await createTestSwarm();
        
        // Submit task to Tier 1
        const task = await swarm.submitTask({
            type: 'routine',
            routineId: '123456789012345678',
            input: { data: 'test' }
        });
        
        // Wait for completion
        await waitFor(() => {
            expect(task.status).toBe('completed');
        }, { timeout: 30000 });
        
        // Verify each tier participated
        const events = await swarm.getEventHistory(task.id);
        expect(events).toContainEqual(
            expect.objectContaining({ tier: 1, action: 'allocated' })
        );
        expect(events).toContainEqual(
            expect.objectContaining({ tier: 2, action: 'orchestrated' })
        );
        expect(events).toContainEqual(
            expect.objectContaining({ tier: 3, action: 'executed' })
        );
    });
});
```

### 6.5. Mocking Tier Dependencies

When testing individual tiers, mock other tier interactions:

```typescript
// Mock Tier 3 responses for Tier 2 tests
const mockTier3 = {
    execute: vi.fn().mockResolvedValue({ success: true, output: 'mocked' })
};

// Inject mock into Tier 2
const tier2 = new TierTwoOrchestrator({
    eventBus,
    tier3Client: mockTier3
});
```

### 6.6. Performance Testing for AI Tiers

```typescript
test('tier should handle concurrent requests', async () => {
    const requests = Array.from({ length: 100 }, (_, i) => ({
        id: `task_${i}`,
        type: 'prompt'
    }));
    
    const start = Date.now();
    const results = await Promise.all(
        requests.map(req => tier3.execute(req))
    );
    const duration = Date.now() - start;
    
    expect(results).toHaveLength(100);
    expect(results.every(r => r.success)).toBe(true);
    expect(duration).toBeLessThan(5000); // 5 seconds for 100 requests
});
```

### 6.7. Error Handling and Resilience

```typescript
test('tier should handle downstream failures gracefully', async () => {
    // Simulate Tier 3 failure
    eventBus.on('tier3:execute:step', () => {
        throw new Error('Tier 3 unavailable');
    });
    
    const result = await tier2.executeRoutine({
        id: '123456789012345678',
        steps: ['failing-step']
    });
    
    expect(result.status).toBe('partial_failure');
    expect(result.completedSteps).toHaveLength(0);
    expect(result.error).toContain('Tier 3 unavailable');
});
```

## 7. Package-Specific Testing Guidance

### 7.1. Server Package (`/packages/server`)

-   **Endpoint Tests:** Located in `/src/endpoints/logic/*.test.ts`
-   **Model Tests:** Test database operations with transactions
-   **Service Tests:** Test business logic with real dependencies
-   **Execution Tiers:** Test each tier independently with event bus

Example structure:
```typescript
// endpoint test
describe("UserEndpoints", () => {
    const app = testingServer();
    
    it("should create user", async () => {
        const response = await app.request
            .post("/api/user")
            .send(userData);
        expect(response.status).toBe(201);
    });
});
```

### 7.2. UI Package (`/packages/ui`)

-   **Component Tests:** Use React Testing Library
-   **Hook Tests:** Test custom hooks in isolation
-   **Storybook Tests:** Visual regression testing
-   **MSW Integration:** Mock API responses for component testing

Example:
```typescript
import { render, screen } from "@testing-library/react";
import { TestWrapper } from "../__test/TestWrapper.js";

test("renders user profile", () => {
    render(
        <TestWrapper>
            <UserProfile userId="123" />
        </TestWrapper>
    );
    expect(screen.getByRole("heading")).toBeInTheDocument();
});
```

### 7.3. Jobs Package (`/packages/jobs`)

-   **Schedule Tests:** Test cron job execution
-   **Queue Tests:** Test job processing with Bull
-   **Service Mocks:** Use provided mock services

### 7.4. Shared Package (`/packages/shared`)

-   **Utility Tests:** Pure function testing
-   **Type Tests:** TypeScript compilation tests
-   **Validation Tests:** Schema validation testing

## 7. AI Maintenance Integration

### 7.1. Test Coverage Tracking

Tests are tracked using the AI maintenance system:

```typescript
// AI_CHECK: TEST_COVERAGE=15 | LAST: 2024-11-20
// Tests added for user authentication flow
```

### 7.2. Test Quality Reviews

Regular AI-driven reviews check for:
- Missing test cases
- Poor test descriptions
- Lack of edge case coverage
- Flaky test patterns

### 7.3. Maintenance Tasks

- **TEST_COVERAGE**: Improve test coverage in specific modules
- **TEST_QUALITY**: Enhance test reliability and clarity
- **TEST_PERFORMANCE**: Optimize slow-running tests

## 8. Schedule & Timeline

Testing activities are integrated into the development sprints:

-   **Unit/Integration Testing:** Continuous during development
-   **Component Testing:** During UI development sprints  
-   **E2E Testing:** Before major releases (Phase 2+)
-   **Performance Testing:** Phase 3 (T+24 weeks)
-   **Security Testing:** Via emergent AI agents (ongoing)

### 8.1. Current Phase (Phase 1: T+0-12 weeks)
- Focus on unit and integration test coverage
- Establish testing patterns and infrastructure
- Document testing guidelines

### 8.2. Phase 2 (T+12-24 weeks)
- Achieve 80%+ test coverage
- Implement E2E test suite
- Begin performance benchmarking

## 9. Entry and Exit Criteria

### 9.1. Entry Criteria (Start of a Test Cycle/Phase)

-   Development phase complete with passing linter
-   Test environment stable with containers running
-   Test data fixtures available
-   Unit/integration tests passing locally
-   Build successfully deployed to test environment

### 9.2. Exit Criteria (Conclusion of a Test Cycle/Phase)

-   All planned tests executed
-   Coverage targets met (80%+ for Phase 2)
-   Critical/high-severity defects fixed and retested
-   Outstanding defects within acceptable limits
-   Test summary report prepared and reviewed
-   AI maintenance checks passed

## 10. E2E Testing Roadmap

### 10.1. Current State
- Cypress installed in UI package (v14.4.1)
- Scripts available: `cy:open` and `cy:run`
- No cypress config or test files created yet
- MSW provides foundation for API mocking

### 10.2. Implementation Plan (Phase 2)
1. **User Journey Tests**: Authentication, profile management
2. **Routine Execution**: End-to-end routine creation and execution
3. **AI Chat Integration**: Full conversation flows
4. **Cross-Browser Testing**: Chrome, Firefox, Safari

### 10.3. E2E Test Structure
```typescript
// cypress/e2e/user-journey.cy.ts
describe("User Registration Journey", () => {
    it("completes full registration flow", () => {
        cy.visit("/register");
        cy.fillRegistrationForm(userData);
        cy.submitForm();
        cy.verifyEmailSent();
        cy.confirmEmail();
        cy.verifyDashboardAccess();
    });
});
```

## 11. Performance and Security Testing

### 11.1. Performance Testing (Phase 3)

**Approach**: Integrated with swarm execution testing

-   **Baseline Metrics**: Response times, throughput, resource usage
-   **Load Scenarios**: Concurrent users, routine executions, chat sessions
-   **Tools**: k6 for load testing, Vitest bench for microbenchmarks
-   **Integration Points**: Database queries, Redis operations, AI model calls

### 11.2. Security Testing

**Approach**: Emergent through AI agent capabilities

-   **Traditional Testing**: OWASP Top 10 coverage
-   **AI-Driven Testing**: Security agents identify vulnerabilities
-   **Continuous Monitoring**: Real-time threat detection
-   **Compliance**: GDPR, data privacy requirements

## 12. CI/CD Integration

### 12.1. Test Execution Pipeline

The project uses GitHub Actions with separate workflows:

#### Test Workflow (`.github/workflows/test.yml`)
- **Triggers**: Push and PR to `main` and `dev` branches
- **Execution**: 
  - Runs tests for all packages: `pnpm run test`
  - Runs bash script tests: `pnpm run test:shell`
- **Timeout**: 15 minutes for entire workflow

#### Dev/Master Workflows
- **Test execution**: Optional via `run_test` workflow input
- **Integration**: Part of build and deployment pipeline
- **Example dispatch**:
  ```yaml
  workflow_dispatch:
    inputs:
      run_test:
        description: 'Run tests'
        type: boolean
        default: false
  ```

### 12.2. Parallelization Strategy
- Split tests by package
- Use Vitest's built-in parallel execution
- Separate unit and integration test jobs
- Cache dependencies between runs

### 12.3. Failure Handling
- Automatic retry for flaky tests (max 2 attempts)
- Detailed failure reports with logs
- Slack notifications for critical failures
- Block deployment on test failures

## 13. Common Pitfalls and Solutions

### 13.1. Import Extension Errors

**Problem**: `Cannot find module './User'`

**Solution**: Always add `.js` extension
```typescript
import { User } from "./User.js";  // ✅ Correct
```

### 13.2. Database Test Pollution

**Problem**: Tests fail when run together but pass individually

**Solution**: Use transaction wrapper
```typescript
await withDbTransaction(async (tx) => {
    // Test code here
});
```

### 13.3. Timeout Errors

**Problem**: `Test timed out after 2000ms`

**Solution**: Set extended timeouts
```typescript
test("long running test", async () => {
    // test code
}, 30000);  // 30 second timeout
```

### 13.4. Mock Service Worker Issues

**Problem**: API calls not intercepted in tests

**Solution**: Ensure MSW is started in test setup
```typescript
beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
```

### 13.5. Timeout Considerations

**Problem**: Long-running operations (tests, type checking, builds) may exceed default timeouts

**Solution**: Set conservative timeouts (e.g., 15 minutes) when running these commands to avoid premature termination

## 14. When to Mock (and When Not To)

### 14.1. Never Mock These (Use Test Containers)
- **Redis**: Use testcontainers for real Redis instance
- **PostgreSQL**: Use testcontainers with transactions
- **Core Infrastructure**: Any foundational service

### 14.2. Appropriate Mocking Scenarios

#### External API Calls
```typescript
// Mock external services that are not under our control
vi.mock('../services/stripe', () => ({
    createPaymentIntent: vi.fn().mockResolvedValue({ id: 'pi_test' })
}));
```

#### Error Scenarios
```typescript
// Mock to test specific error conditions
test('handles database connection failure', async () => {
    vi.spyOn(prisma, '$connect').mockRejectedValueOnce(
        new Error('Connection refused')
    );
    // Test error handling
});
```

#### Time-Sensitive Operations
```typescript
// Mock timers for testing time-based features
beforeEach(() => {
    vi.useFakeTimers();
});

test('session expires after timeout', async () => {
    createSession();
    vi.advanceTimersByTime(SESSION_TIMEOUT);
    expect(session.isExpired).toBe(true);
});
```

#### UI Component Testing
```typescript
// Use MSW for API mocking in UI tests
import { server } from '../__test/setup';

test('shows loading state', async () => {
    server.use(
        http.get('/api/data', () => delay('infinite'))
    );
    render(<DataComponent />);
    expect(screen.getByText('Loading...')).toBeInTheDocument();
});
```

### 14.3. Mocking Best Practices
- **Mock at boundaries**: External services, not internal modules
- **Use real implementations first**: Only mock when necessary
- **Keep mocks simple**: Return minimal realistic data
- **Document why**: Explain the reason for mocking in comments

## 15. Risk Analysis (Testing Specific)

| Risk ID | Risk Description                                      | Likelihood | Impact | Mitigation Strategy                                                                 |
| :------ | :---------------------------------------------------- | :--------- | :----- | :---------------------------------------------------------------------------------- |
| TR-001  | Insufficient test coverage leading to missed defects. | Medium     | High   | AI-driven coverage analysis; automated coverage reporting; mandatory PR checks     |
| TR-002  | Import extension errors breaking tests               | High       | Medium | Linting rules; documentation; code review checklist                               |
| TR-003  | Test environment instability                         | Low        | Medium | Containerized dependencies; health checks; automatic recovery                      |
| TR-004  | Flaky tests providing unreliable feedback            | Medium     | High   | Transaction isolation; deterministic test data; retry mechanisms                   |
| TR-005  | Slow test execution blocking development             | Medium     | Medium | Parallel execution; focused test runs; performance monitoring                      |
| TR-006  | Missing AI tier integration tests                    | Medium     | High   | Comprehensive tier testing strategy; event bus testing                             |

## 15. Test Deliverables

-   **This Test Plan**: Living document updated with project evolution
-   **Test Cases**: Documented as code with clear descriptions
-   **Coverage Reports**: Generated via `pnpm test-coverage`
-   **Test Fixtures**: Maintained in `__test` directories
-   **Defect Reports**: GitHub Issues with `bug` label
-   **Test Logs**: CI/CD artifacts and local test output
-   **AI Maintenance Reports**: Regular quality assessments

## 16. Monitoring and Metrics

### 16.1. Key Metrics
- **Test Coverage**: Target 80%+ by Phase 2
- **Test Execution Time**: Monitor for performance regression
- **Flaky Test Rate**: Target <2% flaky tests
- **Defect Discovery Rate**: Track bugs found in testing vs production

### 16.2. Reporting
- Weekly test coverage updates
- Sprint retrospective test metrics
- Monthly AI maintenance assessments

---

This Test Plan is a living document and will be updated as the project evolves. For implementation details and patterns, refer to [fixtures-and-data-flow.md](./fixtures-and-data-flow.md) and the [AI Maintenance Guide](../ai-maintenance/README.md).