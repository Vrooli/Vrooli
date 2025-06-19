# Writing Effective Tests

This document provides guidelines and best practices for writing unit and integration tests for the Vrooli platform using Vitest. Adhering to these practices will help ensure our tests are reliable, maintainable, and provide good coverage.

## 1. Guiding Principles

-   **Readable:** Tests should be easy to understand. Use clear descriptions and structure your tests logically.
-   **Reliable:** Tests should produce consistent results. Avoid flaky tests that pass or fail intermittently.
-   **Maintainable:** Tests should be easy to update as the codebase evolves. Avoid overly complex or brittle tests.
-   **Fast:** Tests should run quickly to provide fast feedback to developers (though full test suites may take 15+ minutes in worst case scenarios, so always use extended timeouts).
-   **Focused:** Each test case should typically verify a single behavior or aspect of the code.
-   **Independent:** Tests should not depend on each other or the order of execution.
-   **Use Real Infrastructure:** Use testcontainers for PostgreSQL and Redis - never mock core infrastructure services.

## 2. Import Guidelines

Always follow these import conventions:

```typescript
// ✅ CORRECT: Use .js extensions in TypeScript imports
import { functionToTest } from './myModule.js';
import { userFixtures } from '../../fixtures/userFixtures.js';

// ✅ CORRECT: Import from package roots for monorepo packages
import { Shape, Validation } from '@vrooli/shared';

// ❌ WRONG: Deep imports into packages
import { snowflake } from '@vrooli/shared/id/snowflake.js';

// ❌ WRONG: Missing .js extension
import { functionToTest } from './myModule';
```

## 3. Test Structure (Arrange-Act-Assert)

A common pattern for structuring test cases is Arrange-Act-Assert (AAA):

-   **Arrange:** Set up the necessary preconditions and inputs. This might involve creating test data, setting up fixtures, or preparing database state.
-   **Act:** Execute the code under test with the arranged parameters.
-   **Assert:** Verify that the outcome of the action is as expected. Use Vitest assertions to check conditions.

```typescript
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { DbProvider } from '@vrooli/server';
import { User } from '@vrooli/shared';
import { createUser } from '../logic/user.js';

describe('User Creation', () => {
    let testUser: User;

    beforeEach(() => {
        // Arrange: Set up test data
        testUser = {
            email: 'test@example.com',
            name: 'Test User',
            isVerified: false
        };
    });

    afterEach(async () => {
        // Clean up: Remove test data to maintain test isolation
        await DbProvider.get().user.deleteMany({
            where: { email: 'test@example.com' }
        });
    });

    it('should create a new user with correct defaults', async () => {
        // Act
        const result = await createUser(testUser);

        // Assert
        expect(result).toBeDefined();
        expect(result.email).toBe(testUser.email);
        expect(result.isVerified).toBe(false);
        expect(result.id).toBeDefined();
    });
});
```

## 4. Using Vitest Hooks

Vitest provides hooks to set up preconditions or clean up after tests:

-   **`beforeAll()`**: Runs once before all tests in a `describe` block.
-   **`afterAll()`**: Runs once after all tests in a `describe` block.
-   **`beforeEach()`**: Runs before each `it` test case in a `describe` block.
-   **`afterEach()`**: Runs after each `it` test case in a `describe` block.

Use `beforeEach` and `afterEach` to ensure tests are independent by resetting state or cleaning up test data.

## 5. Using Vitest Assertions

Vitest provides a comprehensive set of assertions through its `expect` API.

**Common Assertions:**

-   `expect(value).toBe(expected);` (strict equality ===)
-   `expect(object).toEqual(expectedObject);` (deep equality for objects/arrays)
-   `expect(value).toBeTruthy();` / `expect(value).toBeFalsy();`
-   `expect(value).toBeNull();` / `expect(value).toBeUndefined();`
-   `expect(array).toContain(member);`
-   `expect(array).toHaveLength(expectedLength);`
-   `expect(fn).toThrow(ErrorType);` / `expect(fn).toThrow(/message pattern/);`

## 6. Using Vitest for Mocks, Spies, and Stubs

Vitest provides built-in mocking capabilities through its `vi` utility. However, **DO NOT mock core infrastructure services** like DbProvider, CacheService, or any services that use testcontainers - these are automatically set up with real PostgreSQL and Redis instances.

### What to Mock vs What Not to Mock

**✅ DO Mock:**
- External API calls (Stripe, email services, etc.)
- File system operations
- Time-dependent functions
- Browser APIs in UI tests

**❌ DO NOT Mock:**
- DbProvider (uses real PostgreSQL via testcontainers)
- CacheService (uses real Redis via testcontainers)
- Any infrastructure services set up in global test configuration

### Mocking Examples

-   **Spies (`vi.spyOn()`):** Record information about function calls without affecting their behavior.
    ```typescript
    const emailService = { send: () => Promise.resolve({ success: true }) };
    const sendSpy = vi.spyOn(emailService, 'send');
    
    await emailService.send();
    
    expect(sendSpy).toHaveBeenCalledOnce();
    sendSpy.mockRestore(); // Important to restore original method
    ```

-   **External API Mocks:**
    ```typescript
    import { stripe } from '../services/stripe.js';
    
    // Mock external service calls
    vi.mock('../services/stripe.js', () => ({
        stripe: {
            customers: {
                create: vi.fn().mockResolvedValue({ id: 'cus_123' })
            }
        }
    }));
    
    // Test code that uses stripe
    const customer = await stripe.customers.create({ email: 'test@example.com' });
    expect(customer.id).toBe('cus_123');
    ```

-   **Time Mocking:**
    ```typescript
    beforeEach(() => {
        vi.useFakeTimers();
    });
    
    afterEach(() => {
        vi.useRealTimers();
    });
    
    it('should handle scheduled tasks', () => {
        const callback = vi.fn();
        setTimeout(callback, 1000);
        
        vi.advanceTimersByTime(1000);
        
        expect(callback).toHaveBeenCalledOnce();
    });
    ```

-   **Common Mock Assertions:**
    -   `expect(spy).toHaveBeenCalled();`
    -   `expect(spy).toHaveBeenCalledWith(arg1, arg2);`
    -   `expect(spy).toHaveBeenCalledTimes(n);`
    -   `expect(spy).toHaveReturnedWith(value);`

## 7. Test Data Management

### Using Centralized Fixtures

The project provides comprehensive fixtures in `packages/shared/src/__test/fixtures/` for all 41 object types:

```typescript
import { userFixtures, projectFixtures } from '@vrooli/shared';

describe('Project Access', () => {
    it('should allow owner to update project', async () => {
        // Use pre-built fixtures
        const owner = userFixtures.createOwner();
        const project = projectFixtures.createProject({ owner });
        
        // Test your logic
        const canUpdate = await checkProjectAccess(owner, project, 'update');
        expect(canUpdate).toBe(true);
    });
});
```

### Database Cleanup Patterns

Since we use real databases via testcontainers, proper cleanup is essential:

```typescript
import { withDbTransaction } from '../utils/testHelpers.js';

describe('User Service', () => {
    // Option 1: Use transaction wrapper for automatic rollback
    it('should create user with profile', async () => {
        await withDbTransaction(async () => {
            const user = await createUserWithProfile(userData);
            expect(user.profile).toBeDefined();
            // Transaction automatically rolls back after test
        });
    });
    
    // Option 2: Manual cleanup when transactions aren't suitable
    afterEach(async () => {
        // Clean up in reverse order of foreign key dependencies
        await DbProvider.get().profile.deleteMany({
            where: { user: { email: 'test@example.com' } }
        });
        await DbProvider.get().user.deleteMany({
            where: { email: 'test@example.com' }
        });
    });
});
```

### Test Data Best Practices

-   Use Shape types from `@vrooli/shared` instead of creating custom interfaces
-   Leverage existing fixture functions for consistent test data
-   Create unique identifiers for test data to avoid conflicts
-   Clean up test data immediately after each test

## 8. Asynchronous Tests

Vitest handles asynchronous tests seamlessly. The preferred approach is using async/await:

```typescript
// ✅ PREFERRED: Using async/await
it('should complete an async operation', async () => {
    const result = await asyncOperation();
    expect(result).toBe('expected');
});

// Also supported: Returning promises
it('should complete an async operation (promise)', () => {
    return asyncOperation().then(result => {
        expect(result).toBe('expected');
    });
});

// Testing async errors
it('should handle async errors', async () => {
    await expect(asyncOperationThatFails()).rejects.toThrow('Expected error');
});
```

## 9. What to Test (and What Not To)

-   **Do Test:**
    -   Public API of your modules/classes
    -   Business logic and algorithms
    -   Boundary conditions and edge cases
    -   Error handling paths
    -   Permission and access control logic
    -   Data validation and transformation
    -   AI tier interactions and state transitions
-   **Don't Test (Generally):**
    -   Private methods directly (test them via public methods)
    -   Third-party libraries (test your integration, not their internals)
    -   Database/ORM internals (Prisma is already tested)
    -   Simple getters/setters without logic

## 10. Project-Specific Testing Patterns

### Testing AI Tiers

```typescript
describe('Tier 1 Coordination', () => {
    it('should coordinate task distribution', async () => {
        const swarm = await createTestSwarm();
        const task = createTestTask();
        
        const result = await tier1Coordinator.assignTask(swarm, task);
        
        expect(result.assignment).toBeDefined();
        expect(result.assignment.agentId).toBeDefined();
    });
});
```

### Testing API Endpoints

```typescript
import { endpointsUser } from '../endpoints/logic/user.js';

describe('User Endpoints', () => {
    it('should create user with valid data', async () => {
        const userData = userFixtures.createUserInput();
        
        const result = await endpointsUser.create({
            input: userData,
            context: createTestContext()
        });
        
        expect(result.success).toBe(true);
        expect(result.data.email).toBe(userData.email);
    });
});
```

### Testing with Authentication

```typescript
import { createTestUser, createAuthenticatedContext } from '../testUtils.js';

describe('Protected Operations', () => {
    let authContext;
    
    beforeEach(async () => {
        const user = await createTestUser();
        authContext = createAuthenticatedContext(user);
    });
    
    it('should allow authenticated user to access resource', async () => {
        const result = await protectedOperation(authContext);
        expect(result.success).toBe(true);
    });
});
```

## 11. Test File Organization

-   Test files use the `*.test.ts` naming convention
-   Place tests in the same directory as the code being tested
-   Use `__test` directory (NOT `__tests`) for:
    -   Fixtures (`__test/fixtures/`)
    -   Mocks (`__test/mocks/`)
    -   Test utilities (`__test/helpers/`)

```
src/
├── services/
│   ├── userService.ts
│   ├── userService.test.ts      # Unit tests
│   └── __test/
│       ├── fixtures/
│       │   └── userFixtures.ts
│       └── helpers/
│           └── userTestHelpers.ts
```

## 12. Common Pitfalls and Solutions

| Issue | Solution |
|-------|----------|
| Import errors | Always use `.js` extensions in TypeScript imports |
| Test pollution | Clean up database after each test |
| Timeout errors | Increase timeout for integration tests (15+ minutes) |
| Mock leakage | Use `vi.clearAllMocks()` in `afterEach` |
| Transaction conflicts | Use manual cleanup instead of `withDbTransaction` when code uses transactions |

## 13. Running Tests

```bash
# Run all tests (needs 5+ min timeout)
pnpm test

# Run tests for a specific package
cd packages/server && pnpm test

# Run tests in watch mode
cd packages/server && pnpm test-watch

# Run tests with coverage
cd packages/server && pnpm test-coverage

# Run a specific test file
cd packages/server && pnpm test src/services/userService.test.ts
```

Remember: Tests can take 15+ minutes to run. Always use extended timeouts for test commands.

By following these guidelines, we can build a robust and effective test suite that supports the development and maintenance of Vrooli. 