# Integration Testing Strategy

This document outlines the comprehensive testing strategy for integration tests, focusing on the complete pipeline from form interaction to database verification.

## Core Testing Philosophy

Our integration tests follow the **complete pipeline approach**:

```
Form Data → Shape Transform → Validation → Endpoint → Database → Response Validation
```

Each step is tested individually AND as part of a complete end-to-end flow.

## Architecture Overview

### 1. **Standardized Helper System**

Located in `/src/utils/standardized-helpers.ts`, this provides:

- **Consistent Interface**: All tests use the same helper methods
- **Pipeline Automation**: Complete end-to-end pipeline execution
- **Response Validation**: Structured validation of API responses
- **Database Verification**: Automated database state verification
- **Error Handling**: Comprehensive error scenarios and logging

### 2. **Test Structure Pattern**

Each integration test follows this structure:

```typescript
describe('Resource Round-Trip Tests', () => {
    // Form Data Fixtures - UI form simulation
    class FormFixtures {
        createForm(scenario: string) { /* ... */ }
    }
    
    // Shape Transformation - Form to API format
    class ShapeFixtures {
        transformToApi(formData, context) { /* ... */ }
    }
    
    // Validation - Business rules
    class ValidationFixtures {
        validateInput(apiData) { /* ... */ }
    }
    
    // Tests using standardized helpers
    describe('Complete Pipeline Tests', () => {
        it('should complete full pipeline', async () => {
            const result = await testHelpers.runPipeline(/* ... */);
            // Assertions for each pipeline stage
        });
    });
});
```

## Response Validation Strategy

### 1. **Structured Response Validation**

Response validation is implemented at multiple levels:

#### **Schema Validation**
```typescript
validateSignupResponse(response: any): ValidationResult<any> {
    const errors: string[] = [];
    
    // Required fields validation
    const requiredFields = ['id', 'sessionToken', 'languages', 'users'];
    for (const field of requiredFields) {
        if (!response[field]) {
            errors.push(`Response missing required field: ${field}`);
        }
    }
    
    // Type validation
    if (response.languages && !Array.isArray(response.languages)) {
        errors.push("Response.languages should be an array");
    }
    
    return {
        isValid: errors.length === 0,
        data: response,
        errors: errors.length > 0 ? errors : undefined
    };
}
```

#### **Data Integrity Validation**
```typescript
validateProfileUpdateResponse(response: any, expectedInput: any): ValidationResult<any> {
    const errors: string[] = [];
    
    // Validate that changes were applied
    const fieldsToCheck = ['name', 'bio', 'handle', 'theme'];
    for (const field of fieldsToCheck) {
        if (expectedInput[field] !== undefined && response[field] !== expectedInput[field]) {
            errors.push(`${field} mismatch: expected '${expectedInput[field]}', got '${response[field]}'`);
        }
    }
    
    return { isValid: errors.length === 0, data: response, errors };
}
```

### 2. **Response Validation Best Practices**

1. **Always validate response structure first**
2. **Check data integrity (input vs output)**
3. **Validate business logic constraints**
4. **Test error response formats**
5. **Include performance assertions when relevant**

#### **Example: Complete Response Validation**
```typescript
it('should validate complete signup response', async () => {
    const pipelineResult = await testHelpers.runSignupPipeline(
        formData,
        shapeTransform,
        validator
    );
    
    // 1. Endpoint success
    expect(pipelineResult.endpointResult.success).toBe(true);
    
    // 2. Response structure validation
    expect(pipelineResult.responseValidation.isValid).toBe(true);
    
    // 3. Database consistency
    expect(pipelineResult.dbVerification.isValid).toBe(true);
    
    // 4. Specific response content validation
    const response = pipelineResult.endpointResult.result;
    expect(response.users[0].handle).toMatch(/^test-user-\d+$/);
    expect(response.languages).toContain('en');
    
    // 5. Database state verification
    const dbUser = await testHelpers.verifyUserExists(response.users[0].id);
    expect(dbUser.emails[0].verifiedAt).toBeDefined();
});
```

## Standardized Test Helper Functions

### 1. **Core Helper Categories**

#### **User Management**
```typescript
// Create test users with various configurations
await testHelpers.createTestUser({
    name: 'Custom Name',
    email: 'custom@example.com',
    theme: 'dark'
});
```

#### **Endpoint Calling**
```typescript
// Call endpoints with proper authentication
const result = await testHelpers.callSignupEndpoint(inputData);
const result = await testHelpers.callProfileUpdateEndpoint(inputData, userId);
```

#### **Response Validation**
```typescript
// Validate API responses
const validation = testHelpers.validateSignupResponse(response);
const validation = testHelpers.validateProfileUpdateResponse(response, expectedInput);
```

#### **Database Verification**
```typescript
// Verify database state
const user = await testHelpers.verifyUserExists(userId);
const validation = await testHelpers.verifyUserFields(userId, expectedFields);
```

#### **Complete Pipeline**
```typescript
// Run entire pipeline with logging
const result = await testHelpers.runSignupPipeline(
    formData,
    shapeTransform,
    validator
);
testHelpers.logPipelineResult('Test Name', result);
```

### 2. **Helper Standardization Guidelines**

#### **Naming Conventions**
- **Actions**: `createTestX`, `callXEndpoint`, `verifyXExists`
- **Validation**: `validateXResponse`, `verifyXFields`
- **Pipeline**: `runXPipeline`

#### **Return Value Standards**
```typescript
interface EndpointResult<T> {
    success: boolean;
    result?: T;
    error?: any;
    statusCode?: number;
}

interface ValidationResult<T> {
    isValid: boolean;
    data?: T;
    errors?: any[];
}
```

#### **Error Handling Standards**
- Always return structured error information
- Include HTTP status codes when relevant
- Provide descriptive error messages
- Log errors for debugging without throwing

### 3. **Authentication Patterns**

#### **Session Types**
```typescript
// Unauthenticated (for signup, public endpoints)
const { req, res } = await mockLoggedOutSession();

// Authenticated user session
const testUser = { ...loggedInUserNoPremiumData(), id: userId };
const { req, res } = await mockAuthenticatedSession(testUser);

// API key authentication
const { req, res } = await mockApiSession(apiToken, permissions, userData);
```

## Testing Best Practices

### 1. **Test Organization**

#### **File Structure**
```
/src/round-trip/
├── user.test.ts                 # Legacy approach
├── user-standardized.test.ts    # New standardized approach
├── bookmark.test.ts
├── project.test.ts
└── comment.test.ts

/src/scenarios/
├── user-onboarding.test.ts      # Multi-step workflows
└── collaboration.test.ts

/src/utils/
├── standardized-helpers.ts      # Main helper system
├── simple-helpers.ts           # Backward compatibility
└── test-helpers.ts             # Legacy helpers
```

#### **Test Categories**

1. **Complete Pipeline Tests**: End-to-end form → database validation
2. **Validation Failure Tests**: Error handling and edge cases  
3. **Individual Component Tests**: Isolated testing of each stage
4. **Edge Cases and Error Handling**: Boundary conditions

### 2. **Performance Considerations**

#### **Test Timeouts**
```typescript
it('should complete full pipeline', async () => {
    // Pipeline tests need longer timeouts
}, 30000); // 30 seconds for complete pipeline
```

#### **Database Cleanup**
```typescript
afterEach(async () => {
    // Clean up test data to avoid interference
    await testHelpers.cleanupTestUser(userId);
});
```

#### **Container Management**
- Tests share PostgreSQL and Redis containers
- Containers are started once per test suite
- Database is reset between test files

### 3. **Debugging and Logging**

#### **Pipeline Logging**
```typescript
testHelpers.logPipelineResult('Test Name', pipelineResult);
```

This outputs:
```
🔍 Pipeline Result for: Test Name
   Form Data: { name: "Test User", email: "test@example.com" }
   Shaped Data: { name: "Test User", email: "test@example.com", password: "..." }
   Validation: ✅
   Endpoint: ✅
   Response: ✅
   Database: ✅
```

#### **Error Investigation**
```typescript
if (!pipelineResult.endpointResult.success) {
    console.error('Endpoint failed:', pipelineResult.endpointResult.error);
    console.error('Status code:', pipelineResult.endpointResult.statusCode);
}
```

## Implementation Recommendations

### 1. **Migrating Existing Tests**

1. **Keep existing tests running** while implementing new standardized approach
2. **Create parallel standardized versions** (e.g., `user-standardized.test.ts`)
3. **Gradually migrate** to standardized helpers
4. **Remove legacy tests** once standardized versions are proven

### 2. **Adding New Endpoint Tests**

1. **Start with standardized helpers** from the beginning
2. **Follow the established patterns** for form fixtures, shape transforms, validation
3. **Implement complete pipeline tests first**, then add edge cases
4. **Include response validation** for all success and error scenarios

### 3. **Response Validation Implementation**

For each new endpoint, implement:

1. **Response structure validation** - Check required fields and types
2. **Data integrity validation** - Verify input changes are reflected in output  
3. **Business logic validation** - Check domain-specific constraints
4. **Error response validation** - Validate error formats and messages

### 4. **Test Helper Standardization**

When adding new helpers:

1. **Follow naming conventions** established in `standardized-helpers.ts`
2. **Return structured results** using `EndpointResult` and `ValidationResult` types
3. **Include comprehensive error handling** with descriptive messages
4. **Add logging capabilities** for debugging
5. **Document helper functions** with clear examples

## Conclusion

This testing strategy provides:

- **Comprehensive coverage** of the entire request/response cycle
- **Consistent patterns** across all integration tests  
- **Robust error handling** and debugging capabilities
- **Scalable architecture** for adding new endpoint tests
- **Clear separation of concerns** between form simulation, validation, and endpoint testing

The standardized helper system ensures that all integration tests follow the same patterns and provide the same level of validation, making the test suite maintainable and reliable.