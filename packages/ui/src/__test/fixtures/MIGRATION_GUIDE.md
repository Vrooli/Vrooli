# UI Fixtures Migration Guide

This guide helps you migrate from the legacy fixture system to the new type-safe factory system.

## Quick Comparison

### Old Way (Legacy)
```typescript
// Using any types and manual transformations
const formData: any = {
    email: "test@example.com",
    password: "password123"
};

const apiInput = transformFormToCreateRequestReal(formData);
const result = await mockUserService.signUp(apiInput);
```

### New Way (Type-Safe)
```typescript
// Full type safety and integration
import { UserFixtureFactory } from "@/test/fixtures";

const factory = new UserFixtureFactory(apiClient, dbVerifier);
const formData = factory.createFormData("minimal");
const result = await factory.testCreateFlow(formData);
```

## Key Improvements

1. **Zero `any` Types**: Every fixture is fully typed
2. **Real Integration**: Tests hit actual test database via testcontainers
3. **Shared Package Integration**: Uses real validation and shape functions
4. **Consistent Patterns**: Same interface for all object types
5. **Better IntelliSense**: Full autocomplete support

## Migration Steps

### Step 1: Identify Your Test Type

#### Unit Test (Component Only)
```typescript
// Old
import { minimalUserResponse } from "../fixtures/api-responses/userResponses";

// New
import { getFixture } from "@/test/fixtures";
const userFixture = getFixture("user");
const mockUser = userFixture.createMockResponse();
```

#### Integration Test (API + DB)
```typescript
// Old
const result = await testUserRoundTrip(formData);

// New
const userFixture = getFixture("user");
const result = await userFixture.testRoundTrip(formData);
```

### Step 2: Replace Form Data Creation

```typescript
// Old
const formData = {
    email: "test@example.com",
    password: "password",
    confirmPassword: "password",
    // ... manual data
};

// New
const userFixture = getFixture("user");
const formData = userFixture.createFormData("minimal");
// or with overrides
const customData = userFixture.form.mergeFormData(
    userFixture.createFormData("minimal"),
    { email: "custom@example.com" }
);
```

### Step 3: Replace MSW Handlers

```typescript
// Old
server.use(
    rest.post("/api/user", (req, res, ctx) => {
        return res(ctx.json({ /* manual response */ }));
    })
);

// New
const userFixture = getFixture("user");
server.use(...userFixture.handlers.createSuccessHandlers());
// or for error scenarios
server.use(...userFixture.handlers.createErrorHandlers([
    { type: "validation", config: { errors: ["Invalid email"] } }
]));
```

### Step 4: Replace Validation

```typescript
// Old
const errors = validateSignUpFormDataReal(formData);

// New
const userFixture = getFixture("user");
const result = await userFixture.form.validateFormData(formData);
if (!result.isValid) {
    console.error(result.errors);
}
```

### Step 5: Test Round-Trip Flows

```typescript
// Old (Fake round-trip with mock storage)
const mockService = { storage: {} };
const result = await mockService.create(data);

// New (Real round-trip with test database)
const userFixture = getFixture("user");
const result = await userFixture.testRoundTrip();
// Data actually persisted to test database!
```

## Common Patterns

### Creating Test Data
```typescript
const factory = getFixture("project");

// Single item
const project = await factory.testCreateFlow();

// Multiple items
const projects = await Promise.all([
    factory.testCreateFlow(factory.createFormData("minimal")),
    factory.testCreateFlow(factory.createFormData("complete")),
    factory.testCreateFlow(factory.createFormData("withTeam"))
]);
```

### Testing Relationships
```typescript
// Create user with team
const userFactory = getFixture("user");
const teamFactory = getFixture("team");

const team = await teamFactory.testCreateFlow();
const userData = userFactory.form.mergeFormData(
    userFactory.createFormData("complete"),
    { teamId: team.id }
);
const user = await userFactory.testCreateFlow(userData);
```

### Testing Error Scenarios
```typescript
const factory = getFixture("comment");

// Test validation error
const invalidData = factory.createFormData("invalid");
await expect(factory.testCreateFlow(invalidData))
    .rejects.toThrow("Validation failed");

// Test permission error
factory.setupMSW("error");
await expect(factory.testCreateFlow())
    .rejects.toThrow("Permission denied");
```

### Component Testing
```typescript
import { render } from "@testing-library/react";
import { UserProfile } from "./UserProfile";

const factory = getFixture("user");

// Setup MSW
factory.setupMSW("success");

// Render component
const { getByText } = render(
    <UserProfile userId="123" />
);

// Component will load data via MSW
await waitFor(() => {
    expect(getByText("Test User")).toBeInTheDocument();
});
```

## Fixture Scenarios

Each factory provides standard scenarios:

- **minimal**: Only required fields
- **complete**: All fields populated
- **invalid**: Various validation failures
- **[custom]**: Object-specific scenarios

Example:
```typescript
factory.createFormData("minimal");   // Required fields only
factory.createFormData("complete");  // All fields
factory.createFormData("invalid");   // Will fail validation
factory.createFormData("bot");       // User-specific: bot user
```

## Best Practices

1. **Always Use Factories**: Don't create raw objects manually
2. **Test Real Flows**: Use round-trip tests for integration testing
3. **Leverage IntelliSense**: Let TypeScript guide you
4. **Reuse Fixtures**: Use the registry to share fixtures across tests
5. **Test Errors**: Always test error scenarios, not just success

## Troubleshooting

### "No fixture registered for object type"
```typescript
// Make sure fixtures are initialized
import { initializeFixtures } from "@/test/setup";
await initializeFixtures();
```

### Type Errors
```typescript
// Use proper types from @vrooli/shared
import type { User, UserCreateInput } from "@vrooli/shared";
```

### Test Database Not Available
```typescript
// Ensure testcontainers is running
beforeAll(async () => {
    const { apiClient, dbVerifier } = await setupTestEnvironment();
    // Initialize fixtures with real clients
});
```

## Need Help?

1. Check the [Implementation Strategy](./IMPLEMENTATION_STRATEGY.md)
2. Look at examples: `userFixtureFactory.ts`, `tagFixtureFactory.ts`
3. Use TypeScript IntelliSense for guidance
4. Follow the patterns consistently