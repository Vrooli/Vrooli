# Test.each Pattern Analysis for Auth Tests

## Overview
This document analyzes the viability of using `test.each` or `it.each` pattern to address the nested test function issue in authentication tests.

## Current Situation

### Problem Statement
- Nested test functions are not allowed in Vitest (and most test runners)
- Authentication tests have repetitive patterns with slight variations
- Need to maintain clarity while avoiding duplication

### Current Test Structure
```typescript
describe("emailSignUp", () => {
    it("should create a new user with valid input", async () => {
        // Test implementation
    });
    
    it("should reject signup with existing email", async () => {
        // Test implementation
    });
    
    it("should reject signup with profane name", async () => {
        // Test implementation
    });
    
    it("should reject invalid email format", async () => {
        // Test implementation
    });
    
    it("should reject weak password", async () => {
        // Test implementation
    });
});
```

## Test.each Pattern Examples in Codebase

### Example 1: URL Test (packages/shared/src/utils/url.test.ts)
```typescript
const testCases = [
    {
        description: "handles plain strings without percent signs",
        input: "simple string",
        encoded: "simple string",
        decoded: "simple string",
    },
    // ... more test cases
];

test.each(testCases)("$description", ({ input, encoded, decoded }) => {
    const encodedValue = encodeValue(input);
    expect(encodedValue).toEqual(encoded);
    
    const decodedValue = decodeValue(encodedValue);
    expect(decodedValue).toEqual(decoded);
});
```

### Example 2: Translation Tools Test (packages/ui/src/utils/display/translationTools.test.ts)
```typescript
const localeTestCases = [
    { input: "de", expected: "de", description: "load the specified valid locale" },
    { input: "unknown-locale", expected: "en-US", description: "fallback to en-US when an unknown locale is requested" },
    // ... more test cases
];

it.each(localeTestCases)("should $description", async ({ input, expected }) => {
    const locale = await loadLocale(input);
    expect(locale.code).toEqual(expected);
});
```

### Example 3: Embeddings Test (packages/jobs/src/schedules/embeddings.test.ts)
```typescript
it.each([
    { objectType: "Chat", hasTranslations: true },
    { objectType: "Issue", hasTranslations: true },
    { objectType: "Meeting", hasTranslations: true },
    // ... more test cases
])("should process $objectType embeddings correctly", async ({ objectType, hasTranslations: _hasTranslations }) => {
    // Test implementation
});
```

## Proposed Solution for Auth Tests

### Option 1: Group Similar Tests with test.each

```typescript
describe("emailSignUp", () => {
    // Success cases
    const successCases = [
        {
            description: "create a new user with valid input",
            input: {
                name: "Test User",
                email: "test@example.com",
                password: "SecurePassword123!",
                marketingEmails: false,
                theme: "dark",
            },
            expected: {
                isLoggedIn: true,
                userCount: 1,
                userName: "Test User",
            },
        },
    ];
    
    it.each(successCases)("should $description", async ({ input, expected }) => {
        const { req, res } = await mockLoggedOutSession();
        const result = await auth.emailSignUp({ input }, { req, res }, auth_emailSignUp);
        
        expect(result.__typename).toBe("Session");
        expect(result.isLoggedIn).toBe(expected.isLoggedIn);
        expect(result.users).toHaveLength(expected.userCount);
        if (expected.userName) {
            expect(result.users[0].name).toBe(expected.userName);
        }
    });
    
    // Error cases
    const errorCases = [
        {
            description: "reject signup with existing email",
            setup: async () => {
                await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        emails: {
                            create: [{
                                id: generatePK(),
                                emailAddress: "existing@example.com",
                            }],
                        },
                    }),
                });
            },
            input: {
                name: "New User",
                email: "existing@example.com",
                password: "SecurePassword123!",
                marketingEmails: false,
                theme: "light",
            },
            errorType: CustomError,
        },
        {
            description: "reject signup with profane name",
            setup: null,
            input: {
                name: "fuck",
                email: "test@example.com",
                password: "SecurePassword123!",
                marketingEmails: false,
                theme: "light",
            },
            errorType: CustomError,
        },
        {
            description: "reject invalid email format",
            setup: null,
            input: {
                name: "Test User",
                email: "invalid-email",
                password: "SecurePassword123!",
                marketingEmails: false,
                theme: "light",
            },
            errorType: Error,
        },
        {
            description: "reject weak password",
            setup: null,
            input: {
                name: "Test User",
                email: "test@example.com",
                password: "weak",
                marketingEmails: false,
                theme: "light",
            },
            errorType: Error,
        },
    ];
    
    it.each(errorCases)("should $description", async ({ setup, input, errorType }) => {
        const { req, res } = await mockLoggedOutSession();
        
        if (setup) {
            await setup();
        }
        
        await expect(auth.emailSignUp({ input }, { req, res }, auth_emailSignUp))
            .rejects.toThrow(errorType);
    });
});
```

### Option 2: Keep Individual Tests (Current Approach)

**Pros:**
- Each test is self-contained and easy to understand
- Easy to debug specific test failures
- Clear test descriptions in test output
- No complex test data structures

**Cons:**
- More repetitive code
- Harder to add new test cases
- More maintenance when changing test structure

### Option 3: Hybrid Approach

Group only truly similar tests that have identical structure:

```typescript
describe("emailSignUp validation errors", () => {
    const validationErrorCases = [
        {
            description: "invalid email format",
            input: { email: "invalid-email", /* ... */ },
        },
        {
            description: "weak password",
            input: { email: "test@example.com", password: "weak", /* ... */ },
        },
        {
            description: "missing required fields",
            input: { email: "", /* ... */ },
        },
    ];
    
    it.each(validationErrorCases)("should reject $description", async ({ input }) => {
        const { req, res } = await mockLoggedOutSession();
        await expect(auth.emailSignUp({ input }, { req, res }, auth_emailSignUp))
            .rejects.toThrow();
    });
});
```

## Pros and Cons Analysis

### Pros of test.each Pattern
1. **DRY (Don't Repeat Yourself)**: Reduces code duplication
2. **Easy to add new test cases**: Just add to the array
3. **Consistent test structure**: Enforces uniformity
4. **Better for data-driven tests**: When testing many variations of inputs
5. **Easier maintenance**: Change test logic in one place

### Cons of test.each Pattern
1. **Complexity**: Test data structure can become complex
2. **Readability**: May be harder to understand at a glance
3. **Debugging**: Stack traces might be less clear
4. **Setup variations**: Different tests may need different setup/teardown
5. **Loss of flexibility**: Harder to have unique assertions per test

## Recommendation

For the auth.test.ts file, I recommend a **hybrid approach**:

1. **Keep individual tests for complex scenarios** that have unique setup, teardown, or assertions
2. **Use test.each for validation errors** where the structure is identical
3. **Group similar authentication methods** (emailLogIn, walletInit, etc.) that follow the same pattern

### Example Implementation Strategy:

```typescript
// Group validation errors
describe("input validation", () => {
    const validationCases = [
        { method: "emailSignUp", field: "email", value: "invalid", error: "Invalid email" },
        { method: "emailSignUp", field: "password", value: "weak", error: "Password too weak" },
        { method: "emailLogIn", field: "email", value: "", error: "Email required" },
    ];
    
    it.each(validationCases)(
        "$method should reject when $field is $value",
        async ({ method, field, value, error }) => {
            // Common validation test logic
        }
    );
});

// Keep complex tests separate
describe("emailSignUp business logic", () => {
    it("should create user with all features", async () => {
        // Complex test with multiple assertions
    });
    
    it("should handle existing email with proper error", async () => {
        // Test with specific setup and error checking
    });
});
```

## Conclusion

The test.each pattern can improve the auth tests, but should be applied judiciously:
- Use it for truly repetitive validation tests
- Keep complex business logic tests as individual tests
- Consider maintainability and readability over pure DRY principles
- Document the test data structure well when using test.each

This approach balances code reuse with test clarity and maintainability.