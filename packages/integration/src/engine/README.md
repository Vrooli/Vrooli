# Integration Form Testing Infrastructure

This directory contains comprehensive round-trip form testing infrastructure that complements the UI-focused form testing in the UI package. While UI tests focus on form validation and user interactions in a DOM environment, these integration tests validate the complete data flow from form submission through API calls to database persistence using real testcontainers.

## 🎯 Purpose

The Integration Form Testing Infrastructure enables:

- **Complete Round-Trip Testing**: Test form data flow through all application layers
- **Real API Validation**: Use actual endpoint logic, not mocks
- **Database Persistence Testing**: Verify data is correctly stored and retrievable
- **Performance Testing**: Measure performance of complete form submission flow
- **Data Consistency Validation**: Ensure data transformations are correct across layers
- **Error Handling Testing**: Validate graceful handling of failures at any layer

## 🏗️ Architecture

### Separation from UI Testing

| Aspect | UI Testing (packages/ui) | Integration Testing (packages/integration) |
|--------|--------------------------|---------------------------------------------|
| **Environment** | DOM (jsdom) | Node.js with testcontainers |
| **API Calls** | Mocked with MSW | Real endpoint logic |
| **Database** | Not used | Real PostgreSQL with testcontainers |
| **Focus** | Form validation, user interactions | Complete data flow, persistence |
| **Speed** | Fast (< 1s per test) | Slower (5-30s per test) |
| **Purpose** | Component behavior testing | End-to-end workflow validation |

### Data Flow Testing

```
Form Data → Shape → Transform → API Logic → Database → Verification
    ↓         ↓        ↓          ↓           ↓          ↓
  Fixture  formToShape transform  endpoint   Prisma   findInDatabase
```

## 📁 Structure

```
packages/integration/src/form-testing/
├── README.md                              # This documentation
├── index.ts                              # Main exports and quick start
├── IntegrationFormTestFactory.ts         # Core testing infrastructure
├── FormIntegration.test.ts              # Comprehensive test examples
└── examples/
    └── CommentFormIntegration.ts         # Comment form implementation example
```

## 🔧 Core Components

### `IntegrationFormTestFactory`

The main factory class that provides:

- **Round-Trip Testing**: `testRoundTripSubmission()` - Complete form to database testing
- **Performance Testing**: `testFormPerformance()` - Load testing with metrics
- **Test Generation**: `generateIntegrationTestCases()` - Dynamic test case creation
- **Consistency Validation**: Data integrity checking across all layers

### Configuration Interface

```typescript
interface IntegrationFormTestConfig<TFormData, TShape, TCreateInput, TUpdateInput, TResult> {
    objectType: string;                    // Object type being tested
    validation: { create: ..., update: ... }; // Yup validation schemas
    transformFunction: (...) => ...;      // Data transformation function
    endpoints: { create: ..., update: ... }; // API endpoint configurations
    formFixtures: Record<string, TFormData>; // Test data scenarios
    formToShape: (formData) => TShape;     // Form data conversion
    findInDatabase: (id) => Promise<TResult>; // Database verification
    prismaModel: string;                   // Prisma model name
}
```

## 🚀 Usage Examples

### Basic Round-Trip Test

```typescript
import { createIntegrationFormTestFactory } from "./form-testing/index.js";

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

// Test complete round-trip
const result = await factory.testRoundTripSubmission("minimal", {
    isCreate: true,
    validateConsistency: true,
});

expect(result.success).toBe(true);
expect(result.apiResult?.id).toBe(result.databaseData?.id);
```

### Performance Testing

```typescript
const performanceResult = await factory.testFormPerformance("complete", {
    iterations: 10,
    concurrency: 3,
    isCreate: true,
});

expect(performanceResult.successRate).toBeGreaterThan(0.9);
expect(performanceResult.averageTime).toBeLessThan(2000);
```

### Dynamic Test Generation

```typescript
const testCases = factory.generateIntegrationTestCases();
testCases.forEach(testCase => {
    it(testCase.name, async () => {
        const result = await factory.testRoundTripSubmission(testCase.scenario, {
            isCreate: testCase.isCreate,
            validateConsistency: testCase.shouldSucceed,
        });
        expect(result.success).toBe(testCase.shouldSucceed);
    });
});
```

## 📊 Test Results

### `IntegrationFormTestResult`

Each test returns comprehensive results:

```typescript
interface IntegrationFormTestResult {
    success: boolean;              // Overall test success
    formData: AnyObject;          // Original form data
    shapedData: AnyObject;        // Converted shape data
    transformedData: AnyObject;   // API input data
    apiResult: AnyObject | null;  // API response
    databaseData: AnyObject | null; // Database record
    timing: {                     // Performance metrics
        transformation: number;
        apiCall: number;
        databaseWrite: number;
        databaseRead: number;
        total: number;
    };
    errors: string[];             // Any errors encountered
    consistency: {                // Data consistency checks
        formToApi: boolean;
        apiToDatabase: boolean;
        overallValid: boolean;
    };
}
```

## 🎯 Testing Scenarios

### Standard Test Fixtures

Each form should include these scenario types:

- **`minimal`**: Minimum required data for successful operation
- **`complete`**: Full data with all optional fields populated
- **`edgeCase`**: Boundary values (max lengths, special characters)
- **`invalid`**: Data that should fail validation
- **`performance`**: Data optimized for performance testing

### Example Fixtures

```typescript
const myFormFixtures = {
    minimal: {
        name: "Test Item",
        isPrivate: false,
    },
    complete: {
        name: "Complete Test Item",
        description: "Detailed description with all fields",
        isPrivate: false,
        tags: ["test", "integration"],
        metadata: { category: "testing" },
    },
    edgeCase: {
        name: "A".repeat(255), // Maximum length
        isPrivate: true,
    },
    invalid: {
        name: "", // Missing required field
        isPrivate: false,
    },
};
```

## ⚡ Performance Considerations

### Test Timeouts

Integration tests require longer timeouts due to:
- Database container initialization
- Real API endpoint processing
- Database write/read operations
- Network latency simulation

Recommended timeouts:
- Simple tests: 30 seconds
- Complex tests: 60 seconds
- Performance tests: 120 seconds

### Resource Management

- Tests automatically clean database between runs
- Testcontainers are shared across test suite
- Connection pooling is handled by test setup
- Memory usage is monitored and reported

## 🔍 Debugging

### Common Issues

1. **Test Timeouts**: Increase timeout values or check container status
2. **Database Connections**: Verify testcontainers are running correctly
3. **API Errors**: Check endpoint logic and input validation
4. **Data Inconsistency**: Review transformation functions and validation schemas

### Debugging Tools

```typescript
// Enable verbose logging
process.env.DEBUG = "vrooli:integration:*";

// Check setup status
import { getSetupStatus } from "./setup/test-setup.js";
console.log("Setup status:", getSetupStatus());

// Verify database connection
import { getPrisma } from "./setup/test-setup.js";
const prisma = getPrisma();
console.log("Database connected:", !!prisma);
```

## 🚀 Getting Started

1. **Study the Example**: Start with `CommentFormIntegration.ts` to understand the pattern
2. **Create Fixtures**: Define test data for your form scenarios
3. **Configure Factory**: Set up the integration test factory with your form's functions
4. **Write Tests**: Create comprehensive test suites using the provided patterns
5. **Run Tests**: Execute with appropriate timeouts and monitor performance

## 📚 Related Documentation

- [UI Form Testing](../../ui/src/__test/fixtures/form-testing/README.md) - UI-focused form testing
- [Integration Package Overview](../README.md) - Overall integration testing strategy
- [Testcontainers Setup](../setup/README.md) - Database and Redis container management
- [Standard Form Hook](../../ui/src/hooks/useStandardUpsertForm.ts) - Form logic standardization

## 🏆 Success Metrics

- **Complete Coverage**: Test all form operations (create, update, validation)
- **Performance Benchmarks**: Sub-2-second average response times
- **Data Integrity**: 100% consistency between API and database
- **Error Handling**: Graceful failure handling at all layers
- **Maintainability**: Easy to add new forms and scenarios

This infrastructure ensures that form submissions work correctly across the entire application stack, providing confidence in data integrity and system reliability.