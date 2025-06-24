# Integration Tests Package

This package contains round-trip and scenario tests for Vrooli that test complete user workflows through multiple layers of the application.

## Overview

Unlike unit tests that test individual components in isolation, integration tests verify that different parts of the system work together correctly. This package runs in a pure Node.js environment with access to all other packages.

## Test Types

### 🚀 Integration Testing Engine (`/src/engine/`) - **CORE INFRASTRUCTURE**
- **Production-grade testing framework** for complete form submission workflows
- **True round-trip testing**: Real API calls through actual endpoints
- **Database persistence testing**: Uses testcontainers with real PostgreSQL
- **Performance testing**: Built-in metrics for complete data flow
- **Type-safe**: Full TypeScript support with comprehensive validation

### Form Integration Tests (`/src/form/`)
- Test **single form workflows** through all layers
- Complete data flow from form submission to database persistence
- Uses the testing engine for comprehensive validation

### Integration Examples (`/src/examples/`)
- **Reference implementations** showing how to configure form tests
- Examples for Comment, Bookmark, Project, and User forms
- Demonstrates best practices for testing setup

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
  /engine/               # 🚀 CORE: Integration testing infrastructure
    IntegrationFormTestFactory.ts  # Core testing factory
    FormIntegration.test.ts        # Engine validation tests
    index.ts                       # Public API exports
    README.md                      # Engine documentation
  /examples/             # 📚 REFERENCE: How to configure tests
    BookmarkFormIntegration.ts     # Bookmark form test configuration
    CommentFormIntegration.ts      # Comment form test configuration
    ProjectFormIntegration.ts      # Project form test configuration
    UserFormIntegration.ts         # User form test configuration
  /form/                 # ✅ TESTS: Single form integration tests
    bookmark.test.ts               # Complete bookmark workflow testing
    comment.test.ts                # Complete comment workflow testing  
    project.test.ts                # Complete project workflow testing
    user.test.ts                   # Complete user workflow testing
    bookmark-alternative.test.ts   # Alternative bookmark test patterns
    user-alternative.test.ts       # Alternative user test patterns
  /scenarios/            # 🔗 WORKFLOWS: Multi-step integration tests
    user-onboarding.test.ts        # Complete user onboarding flow
    user-onboarding-workflow.test.ts  # Extended onboarding scenarios
```

## Key Features

1. **Real Database**: Uses testcontainers to spin up PostgreSQL
2. **Real Redis**: Uses testcontainers for Redis instance
3. **Full Stack Access**: Can import from UI, Server, and Shared packages
4. **Automatic Cleanup**: Database is truncated between tests
5. **Long Timeouts**: Configured for complex operations (2min test, 5min setup)

## Writing Tests

### 🚀 Integration Testing Example (RECOMMENDED)
```typescript
import { createIntegrationFormTestFactory } from "./engine/index.js";

const factory = createIntegrationFormTestFactory({
    objectType: "Comment",
    validation: commentValidation,
    transformFunction: transformCommentValues,
    endpoints: { create: endpointsComment.createOne, update: endpointsComment.updateOne },
    formFixtures: commentFormFixtures,
    formToShape: commentFormToShape,
    findInDatabase: findCommentInDatabase,
    prismaModel: "comment",
});

// Test complete form workflow with real API and database
it('should complete full comment submission flow', async () => {
    const result = await factory.testRoundTripSubmission("minimal", {
        isCreate: true,
        validateConsistency: true,
    });
    
    expect(result.success).toBe(true);
    expect(result.apiResult?.id).toBe(result.databaseData?.id);
    expect(result.consistency.overallValid).toBe(true);
});
```

### Example Setup Pattern
```typescript
// Reference: see /src/examples/CommentFormIntegration.ts
import { commentFormIntegrationFactory } from "../examples/CommentFormIntegration.js";

// Use pre-configured factory from examples
it('should create a comment through complete workflow', async () => {
    const result = await commentFormIntegrationFactory.testRoundTripSubmission("minimal", {
        isCreate: true,
        validateConsistency: true,
    });
    
    expect(result.success).toBe(true);
    expect(result.databaseData?.text).toBe("This is a test comment");
    expect(result.timing.total).toBeLessThan(5000);
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

## Migration Guide

### Creating New Integration Tests

1. **Review Examples**: Check `/src/examples/` for patterns matching your object type
2. **Set Up Form Fixtures**: Create realistic form data scenarios in your example file
3. **Configure Factory**: Use `createIntegrationFormTestFactory` with your object's configuration
4. **Write Tests**: Create test file in `/src/form/` using `testRoundTripSubmission`

### Implementation Pattern:

**Step 1 - Example Configuration**:
```typescript
// /src/examples/MyObjectFormIntegration.ts
export const myObjectFormIntegrationFactory = createIntegrationFormTestFactory({
    objectType: "MyObject",
    validation: myObjectValidation,
    // ... other configuration
});
```

**Step 2 - Form Integration Test**:
```typescript
// /src/form/my-object.test.ts
import { myObjectFormIntegrationFactory } from "../examples/MyObjectFormIntegration.js";

it('should complete my-object workflow', async () => {
    const result = await myObjectFormIntegrationFactory.testRoundTripSubmission("minimal", {
        isCreate: true,
        validateConsistency: true,
    });
    
    expect(result.success).toBe(true);
});
```

## Best Practices

1. **Use Integration Testing Engine**: For all new tests requiring database persistence
2. **Test User Workflows**: Focus on real user scenarios, not implementation details
3. **True Round-Trip**: Always test through actual API endpoints, never bypass layers
4. **Verify Side Effects**: Check that emails would be sent, jobs queued, etc.
5. **Clean State**: Each test should start with a clean database
6. **Descriptive Names**: Use clear test names that describe the scenario

## Package Organization

### ✅ Current Structure

All integration tests have been reorganized into the new clear structure:

| Object Type | Form Test | Example Config | Status |
|-------------|-----------|----------------|--------|
| Comment | `form/comment.test.ts` | `examples/CommentFormIntegration.ts` | ✅ Complete |
| Project | `form/project.test.ts` | `examples/ProjectFormIntegration.ts` | ✅ Complete |
| User | `form/user.test.ts` | `examples/UserFormIntegration.ts` | ✅ Complete |
| Bookmark | `form/bookmark.test.ts` | `examples/BookmarkFormIntegration.ts` | ✅ Complete |
| Alternative patterns | `form/*-alternative.test.ts` | (Use primary examples) | ✅ Complete |

### 🧹 Structure Reorganization

The integration package has been reorganized for clarity:
- **Engine** (`/src/engine/`): Core testing infrastructure 
- **Examples** (`/src/examples/`): Reference implementations for test configuration
- **Form Tests** (`/src/form/`): Single form workflow integration tests
- **Scenarios** (`/src/scenarios/`): Multi-step workflow tests

Legacy tests that bypassed the API layer have been removed.

### 🎯 Integration Testing Benefits

- **True End-to-End Testing**: Real HTTP API calls through complete application stack
- **Performance Metrics**: Built-in timing and load testing capabilities
- **Data Consistency**: Validation across all application layers
- **Error Scenarios**: Comprehensive validation failure testing
- **Type Safety**: Full TypeScript support with proper shared types
- **Clear Organization**: Separated engine, examples, tests, and scenarios
- **Maintainability**: Standardized patterns with clear reference implementations

### 📊 Testing Coverage

- **Signup & Authentication**: Complete user registration and session management
- **Profile Management**: User profile updates and privacy settings
- **Content Creation**: Comments, projects, and bookmarks
- **Data Relationships**: Lists, teams, translations, and tags
- **Error Handling**: Validation failures and edge cases
- **Performance**: Concurrent operations and load testing

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