# Test Fixtures Architecture

## Overview

We use a two-layer fixture system to handle the different data shapes needed for testing:

1. **Database Fixtures** (`packages/server/src/__test/fixtures/`) - Prisma-shaped data for seeding test databases
2. **Validation Fixtures** (`packages/shared/src/validation/models/__test/fixtures/`) - API input/output shapes for testing validation and endpoints

## Why Two Types of Fixtures?

### The Problem
- API inputs use one shape (e.g., `translationsCreate: [...]`)
- Database operations use Prisma's shape (e.g., `translations: { create: [...] }`)
- Mixing these causes confusion and errors

### The Solution
- **Database fixtures** for test setup/seeding
- **Validation fixtures** for API input/output testing
- Clear separation of concerns

## Usage Pattern

### 1. Database Setup (BeforeEach)
```typescript
import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/userFixtures.js";
import { ChatDbFactory, seedTestChat } from "../../__test/fixtures/chatFixtures.js";

beforeEach(async () => {
    // Seed users using database fixtures
    const users = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });
    
    // Create chat using database fixtures
    const chat = await seedTestChat(DbProvider.get(), {
        userIds: [users[0].id, users[1].id],
        isPrivate: true,
        withMessages: true,
    });
});
```

### 2. API Testing (Test Cases)
```typescript
import { chatTestDataFactory } from "@vrooli/shared/src/validation/models/__test/fixtures/chatFixtures.js";

it("creates a chat", async () => {
    // Use validation fixtures for API input
    const input: ChatCreateInput = chatTestDataFactory.createMinimal({ 
        openToAnyoneWithInvite: true 
    });
    
    const result = await chat.createOne({ input }, { req, res });
    expect(result.openToAnyoneWithInvite).toBe(true);
});
```

## Database Fixtures

Located in `packages/server/src/__test/fixtures/`

### Features
- Follow Prisma's `CreateInput` shape
- Generate unique IDs automatically
- Provide factory methods for common patterns
- Include helper functions for complex setups

### Example: UserDbFactory
```typescript
// Minimal user
const user = UserDbFactory.createMinimal();

// User with authentication
const authUser = UserDbFactory.createWithAuth({
    name: "John Doe",
    emails: {
        create: [{
            emailAddress: "john@example.com",
            verified: true,
        }],
    },
});

// Seed multiple users
const users = await seedTestUsers(prisma, 5, { 
    withAuth: true,
    withBots: true,
    teamId: "team123",
});
```

### Example: ChatDbFactory
```typescript
// Chat with participants
const chat = ChatDbFactory.createWithParticipants(
    [userId1, userId2],
    { isPrivate: true }
);

// Full chat setup
const chat = await seedTestChat(prisma, {
    userIds: [userId1, userId2],
    isPrivate: false,
    withMessages: true,
    withInvites: true,
    teamId: "team123",
});
```

## Validation Fixtures

Located in `packages/shared/src/validation/models/__test/fixtures/`

### Features
- Follow API input/output shapes
- Provide standard test scenarios (minimal, complete, invalid, edge cases)
- Type-safe with TypeScript
- Shared across server tests, validation tests, and UI tests

### Example Usage
```typescript
// Minimal valid input
const input = chatTestDataFactory.createMinimal();

// Complete input with all fields
const input = chatTestDataFactory.createComplete({
    openToAnyoneWithInvite: false,
});

// Invalid scenarios for error testing
const invalidInput = chatFixtures.invalid.missingRequired.create;
```

## Best Practices

### 1. Use the Right Fixture for the Right Purpose
- Database fixtures for `beforeEach` setup
- Validation fixtures for API input/output testing

### 2. Don't Mix Shapes
```typescript
// ❌ Bad - mixing database shape with API testing
const input: ChatCreateInput = {
    translations: { create: [...] }, // This is Prisma syntax!
};

// ✅ Good - using validation fixtures for API
const input: ChatCreateInput = chatTestDataFactory.createMinimal();
```

### 3. Leverage Factory Methods
```typescript
// ❌ Bad - manual construction
const user = {
    id: generatePK(),
    publicId: generatePublicId(),
    handle: `user_${Math.random()}`,
    // ... lots of boilerplate
};

// ✅ Good - using factory
const user = UserDbFactory.createWithAuth();
```

### 4. Use Seed Helpers for Complex Scenarios
```typescript
// ❌ Bad - manual relationship setup
const chat = await prisma.chat.create({
    data: {
        // ... manual construction
        participants: {
            create: users.map(u => ({
                id: generatePK(),
                user: { connect: { id: u.id } },
            })),
        },
    },
});

// ✅ Good - using seed helper
const chat = await seedTestChat(prisma, {
    userIds: users.map(u => u.id),
    withMessages: true,
});
```

## Migration Guide

### Converting Existing Tests

1. **Identify Database Operations**
   ```typescript
   // Old
   await DbProvider.get().user.create({
       data: {
           id: uuid(),
           name: "Test User",
           // ... manual construction
       }
   });
   
   // New
   const user = await DbProvider.get().user.create({
       data: UserDbFactory.createMinimal({ name: "Test User" })
   });
   ```

2. **Separate API Inputs**
   ```typescript
   // Old
   const input = {
       id: uuid(),
       translationsCreate: [...], // Mixed validation shape
   };
   
   // New
   const input = chatTestDataFactory.createMinimal();
   ```

3. **Use Appropriate Assertions**
   ```typescript
   // For seeded data
   expect(privateChat.id).toBeDefined();
   expect(privateChat.isPrivate).toBe(true);
   
   // For API responses
   expect(result).toMatchObject({
       id: expect.any(String),
       openToAnyoneWithInvite: true,
   });
   ```

## Adding New Fixtures

### 1. Create Database Fixture
```typescript
// packages/server/src/__test/fixtures/modelFixtures.ts
export class ModelDbFactory {
    static createMinimal(overrides?: Partial<Prisma.ModelCreateInput>) {
        return {
            id: generatePK(),
            // ... required fields
            ...overrides,
        };
    }
}
```

### 2. Create Validation Fixture
```typescript
// packages/shared/src/validation/models/__test/fixtures/modelFixtures.ts
export const modelFixtures: ModelTestFixtures = {
    minimal: {
        create: { /* minimal valid input */ },
        update: { /* minimal update input */ },
    },
    // ... other scenarios
};
```

### 3. Document Usage
- Add examples to this README
- Update test examples
- Note any special considerations

## Common Pitfalls

### 1. ID Conflicts
- Always use generated IDs in factories
- Don't hardcode IDs unless testing specific scenarios

### 2. Shape Mismatches
- Double-check you're using the right fixture type
- Let TypeScript guide you

### 3. Over-Seeding
- Only seed what you need for each test
- Use minimal fixtures when possible

### 4. Missing Cleanup
- Always clean up in `afterEach` or `afterAll`
- Clear both database and Redis