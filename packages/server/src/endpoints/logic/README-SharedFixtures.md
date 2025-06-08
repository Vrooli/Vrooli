# Using Shared Fixtures in Endpoint Tests

## Overview

We've created shared fixtures in `packages/shared/src/validation/models/__test/fixtures/` that can be used across API endpoint tests, validation tests, and UI form tests. This ensures consistency and reduces duplication.

## Benefits of Using Shared Fixtures

### 1. **Consistency Across Tests**
- All tests use the same test data structure
- Reduces discrepancies between API tests, validation tests, and UI tests
- Ensures test data matches actual schema requirements

### 2. **Reduced Maintenance**
- Update test data in one place when schemas change
- No need to update hardcoded IDs and data across multiple test files
- Automatic generation of valid test data

### 3. **Type Safety**
- Fixtures are typed according to the actual input/output types
- TypeScript catches mismatches at compile time
- TestDataFactory provides typed helpers

### 4. **Comprehensive Test Scenarios**
- Standard scenarios: minimal, complete, invalid, edge cases
- Consistent patterns for testing validation failures
- Easy to add new test scenarios

## Example: Converting chat.test.ts

### Before (Hardcoded Data)
```typescript
// Generate unique IDs for test users and chats
const user1Id = uuid();
const user2Id = uuid();
const chat1Id = uuid();

// Create test data manually
await DbProvider.get().chat.create({
    data: {
        id: chat1Id,
        publicId: generatePublicId(),
        isPrivate: true,
        openToAnyoneWithInvite: false,
        // ... manually constructed data
    }
});
```

### After (Using Shared Fixtures)
```typescript
// Import shared fixtures
import { chatFixtures, chatTestDataFactory } from "@vrooli/shared/src/validation/models/__test/fixtures/chatFixtures.js";
import { userFixtures, userTestDataFactory } from "@vrooli/shared/src/validation/models/__test/fixtures/userFixtures.js";

// Use consistent IDs from fixtures
const user1Id = userFixtures.minimal.create.id;
const chat1Id = chatFixtures.minimal.create.id;

// Create test data using factory
const privateChatData = chatTestDataFactory.createMinimal({
    isPrivate: true,
    openToAnyoneWithInvite: false,
});
```

## How to Use Shared Fixtures

### 1. Import the Fixtures
```typescript
import { modelFixtures, modelTestDataFactory } from "@vrooli/shared/src/validation/models/__test/fixtures/modelFixtures.js";
```

### 2. Use Fixture IDs
```typescript
const entityId = modelFixtures.minimal.create.id;
```

### 3. Create Test Data with Factory
```typescript
// Minimal valid data
const minimalData = modelTestDataFactory.createMinimal();

// Complete data with all fields
const completeData = modelTestDataFactory.createComplete();

// Override specific fields
const customData = modelTestDataFactory.createMinimal({
    name: "Custom Name",
    isPrivate: true
});
```

### 4. Use Invalid Scenarios for Negative Tests
```typescript
// Test validation failures
const invalidData = modelFixtures.invalid.missingRequired.create;
await expect(async () => {
    await endpoint.createOne({ input: invalidData }, { req, res });
}).rejects.toThrow();
```

## Available Fixtures

Current fixtures in `packages/shared/src/validation/models/__test/fixtures/`:
- `apiKeyExternalFixtures.ts` - External API key management
- `bookmarkFixtures.ts` - Bookmark creation and updates
- `chatFixtures.ts` - Chat creation with translations, invites, messages
- `commentFixtures.ts` - Comment creation and threading
- `meetingFixtures.ts` - Meeting scheduling and invites
- `memberFixtures.ts` - Team member management
- `runFixtures.ts` - Routine execution runs
- `userFixtures.ts` - User/bot creation and profile updates

## Migration Guide

1. **Identify Hardcoded Test Data**
   - Look for `uuid()` calls
   - Hardcoded object literals
   - Manual ID generation

2. **Replace with Fixture Imports**
   ```typescript
   // Old
   const userId = uuid();
   
   // New
   const userId = userFixtures.minimal.create.id;
   ```

3. **Use TestDataFactory for Dynamic Data**
   ```typescript
   // Old
   const input = {
       id: uuid(),
       name: "Test Chat",
       openToAnyoneWithInvite: true
   };
   
   // New
   const input = chatTestDataFactory.createMinimal({
       openToAnyoneWithInvite: true
   });
   ```

4. **Convert Assertions to Vitest**
   ```typescript
   // Old (Chai)
   expect(result).to.not.be.null;
   expect(result.id).to.equal(chatId);
   
   // New (Vitest)
   expect(result).not.toBeNull();
   expect(result.id).toEqual(chatId);
   ```

## Best Practices

1. **Use Appropriate Fixture Level**
   - `minimal` for basic functionality tests
   - `complete` for comprehensive tests
   - `invalid` for error handling tests
   - `edgeCases` for boundary conditions

2. **Override Only What's Needed**
   ```typescript
   // Good - only override specific fields
   const data = factory.createMinimal({ name: "Custom" });
   
   // Avoid - recreating entire object
   const data = { id: uuid(), name: "Custom", ... };
   ```

3. **Leverage Type Safety**
   - Let TypeScript guide you to valid data structures
   - Use factory methods for compile-time checking

4. **Keep Fixtures Updated**
   - When schemas change, update fixtures first
   - Run all tests to catch breaking changes

## Limitations

1. **Database-Specific Logic**
   - Some tests may need database-specific setup
   - Fixtures provide input data, not database state

2. **Complex Relationships**
   - May need additional setup for complex data relationships
   - Use fixtures as building blocks

3. **Dynamic Data**
   - Some tests need truly unique data (e.g., handles)
   - Use `testValues` utilities for generation

## Future Improvements

1. **More Fixture Coverage**
   - Add fixtures for remaining models
   - Create composite fixtures for complex scenarios

2. **Test Utilities**
   - Helper functions for common setup patterns
   - Database seeding utilities using fixtures

3. **Documentation**
   - Auto-generate fixture documentation from schemas
   - Visual fixture browser for developers