# Integration Tests Package

This package contains round-trip and scenario tests for Vrooli that test complete user workflows through multiple layers of the application.

## Overview

Unlike unit tests that test individual components in isolation, integration tests verify that different parts of the system work together correctly. This package runs in a pure Node.js environment with access to all other packages.

## Test Types

### Round-Trip Tests (`/src/round-trip/`)
- Test a **single operation** through all layers
- Example: Creating a bookmark from UI form data → validation → API → database
- Use real implementations (no mocks for core infrastructure)
- Verify data transformations at each layer

### Scenario Tests (`/src/scenarios/`)
- Test **multi-step workflows** that span multiple operations
- Example: User onboarding flow (signup → profile → team → project)
- Verify complex business logic and state transitions
- Test background jobs and side effects

## Running Tests

```bash
# Run all integration tests
pnpm test

# Run only round-trip tests
pnpm test:round-trip

# Run only scenario tests
pnpm test:scenarios

# Run tests in watch mode
pnpm test:watch

# Run with coverage
pnpm test:coverage
```

## Test Structure

```
/src/
  /setup/
    global-setup.ts      # Starts test containers (PostgreSQL, Redis)
    test-setup.ts        # Per-test setup (migrations, cleanup)
  /utils/
    test-helpers.ts      # Common test utilities
  /round-trip/
    *.test.ts           # Single operation tests
  /scenarios/
    *.test.ts           # Multi-step workflow tests
  /fixtures/
    *.ts                # Shared test data
```

## Key Features

1. **Real Database**: Uses testcontainers to spin up PostgreSQL
2. **Real Redis**: Uses testcontainers for Redis instance
3. **Full Stack Access**: Can import from UI, Server, and Shared packages
4. **Automatic Cleanup**: Database is truncated between tests
5. **Long Timeouts**: Configured for complex operations (2min test, 5min setup)

## Writing Tests

### Round-Trip Test Example
```typescript
it('should create a bookmark through the full stack', async () => {
    // 1. Create form data (as if from UI)
    const formData = { /* ... */ };
    
    // 2. Shape the data (UI layer)
    const shapedData = shape(formData, ...);
    
    // 3. Validate and transform (shared layer)
    const validatedInput = transformInput(shapedData, ...);
    
    // 4. Call endpoint logic (server layer)
    const result = await Bookmark.Create.performLogic({ /* ... */ });
    
    // 5. Verify result and database state
    expect(result).toBeDefined();
});
```

### Scenario Test Example
```typescript
it('should complete user onboarding flow', async () => {
    // Step 1: User signs up
    const { user, sessionData } = await createTestUser();
    
    // Step 2: Complete profile
    await User.Update.performLogic({ /* ... */ });
    
    // Step 3: Create team
    await Team.Create.performLogic({ /* ... */ });
    
    // Step 4: Create project
    await Project.Create.performLogic({ /* ... */ });
    
    // Verify final state
    const finalUser = await prisma.user.findUnique({ /* ... */ });
    expect(finalUser.memberships).toHaveLength(1);
});
```

## Best Practices

1. **Test User Workflows**: Focus on real user scenarios, not implementation details
2. **Use Real Data**: Create data through the same paths users would
3. **Verify Side Effects**: Check that emails would be sent, jobs queued, etc.
4. **Clean State**: Each test should start with a clean database
5. **Descriptive Names**: Use clear test names that describe the scenario

## Troubleshooting

### Tests timing out
- Increase timeout in test: `it('...', async () => { /* ... */ }, 300000)`
- Check if containers started properly in logs

### Cannot import from other packages
- Ensure all packages are built: `pnpm run build`
- Check tsconfig paths are correct

### Database connection errors
- Verify Docker is running
- Check no other containers using same ports
- Look at global-setup.ts output for connection strings