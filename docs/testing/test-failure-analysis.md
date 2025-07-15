# Test Failure Analysis: Database Connection Issues

## Summary

After analyzing the test failures, I've identified that tests are failing with database connection errors because they are not following the established testing patterns in the codebase.

## Root Cause

The primary issue is that failing tests are creating their own `PrismaClient` instances instead of using the centralized `DbProvider` singleton that is properly initialized with the test database connection.

### Test Infrastructure Setup

1. **Global Setup** (`vitest.global-setup.ts`):
   - Starts testcontainers for PostgreSQL and Redis
   - Sets environment variables (`DB_URL`, `REDIS_URL`)
   - Runs Prisma migrations
   - Generates Prisma client

2. **Per-File Setup** (`src/__test/setup.ts`):
   - Initializes `ModelMap`
   - Initializes `DbProvider` (which uses the environment variables)
   - Sets up mocks for services

## Pattern Analysis

### ✅ Passing Tests Pattern

Tests that pass follow one of these patterns:

1. **No Database Interaction**: Pure unit tests that don't need database access
   ```typescript
   // Example: src/auth/email.test.ts
   describe("PasswordAuthService", () => {
       it("should generate URL-safe codes", () => {
           // Pure logic test, no DB needed
       });
   });
   ```

2. **Using DbProvider**: Tests that properly use the singleton
   ```typescript
   // Example: src/endpoints/logic/admin.test.ts
   const prisma = DbProvider.get();
   await prisma.user.create({ ... });
   ```

### ❌ Failing Tests Pattern

Tests that fail are creating their own PrismaClient:

```typescript
// Example: src/__test/fixtures/db/factories/CoreBusinessObjectsPart2.test.ts
beforeEach(async () => {
    prisma = new PrismaClient({  // ❌ Wrong - creates new client
        datasources: {
            db: {
                url: process.env.TEST_DATABASE_URL || "postgresql://...",
            },
        },
    });
    await prisma.$connect();
});
```

## Specific Test Categories

### Database Factory Tests
Location: `src/__test/fixtures/db/factories/*.test.ts`
- Issue: Creating new PrismaClient instances
- Fix: Use `DbProvider.get()` instead

### Endpoint Tests
Location: `src/endpoints/logic/*.test.ts`
- Some use DbProvider correctly (admin.test.ts)
- Others may need review

### Service Tests
Location: `src/services/**/*.test.ts`
- Need to verify if they're using DbProvider

## Recommended Fixes

### 1. Update Factory Tests

Replace:
```typescript
prisma = new PrismaClient({ ... });
await prisma.$connect();
```

With:
```typescript
prisma = DbProvider.get();
```

### 2. Remove Manual Cleanup

The global teardown handles container cleanup. Tests should only clean up their data:

```typescript
afterEach(async () => {
    // Clean up test data only
    await resourceFactory.cleanupAll();
    // Don't disconnect - DbProvider handles this
});
```

### 3. Use Test Fixtures Properly

Factories should accept a Prisma client as a parameter:

```typescript
export function createUserDbFactory(prisma: PrismaClient) {
    // Use the provided prisma instance
}

// In tests:
const userFactory = createUserDbFactory(DbProvider.get());
```

## Additional Observations

1. **Redis Connection Errors**: Some tests show Redis connection errors during cleanup. These are expected and handled by the error handlers in setup.ts.

2. **Test Isolation**: The infrastructure uses dynamic port mapping for containers to ensure test isolation between different test runs.

3. **Performance**: Tests run in a single fork (`singleFork: true`) to reduce overhead and share the database connection.

## Action Items

1. Update all factory test files to use `DbProvider.get()`
2. Remove manual PrismaClient creation and connection management
3. Ensure factories accept Prisma client as parameter
4. Update documentation for test patterns
5. Consider adding a linting rule to prevent `new PrismaClient()` in test files

## Example Fix

For `CoreBusinessObjectsPart2.test.ts`:

```typescript
import { DbProvider } from "../../../db/provider.js";

describe("Core Business Objects Part 2", () => {
    let prisma: PrismaClient;
    let userFactory: UserDbFactory;
    // ... other factories

    beforeEach(async () => {
        prisma = DbProvider.get(); // Use singleton
        
        // Initialize factories with the singleton prisma
        userFactory = createUserDbFactory(prisma);
        // ... other factory initializations
    });

    afterEach(async () => {
        // Clean up test data only
        await resourceFactory.cleanupAll();
        await teamFactory.cleanupAll();
        await userFactory.cleanupAll();
        // No prisma.$disconnect() needed
    });

    // ... tests
});
```