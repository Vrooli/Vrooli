# Production-Grade UI Fixtures Architecture

This directory contains a completely redesigned UI fixtures architecture that addresses all the critical issues identified in the existing fixtures system. This implementation provides true integration testing with type safety, real database connections, and proper test isolation.

## Core Principles

1. **Zero `any` Types**: Full TypeScript safety throughout
2. **Real Integration**: Form → Validation → API → Database → Response → UI  
3. **Test Isolation**: No global state pollution
4. **Component Integration**: Works with real UI components and React Hook Form
5. **Network Simulation**: Proper MSW integration with realistic responses

## Directory Structure

```
fixtures-updated/
├── factories/          # Type-safe fixture factories using real @vrooli/shared functions
├── integrations/       # Real database integration tests with testcontainers
├── scenarios/          # Multi-step workflow orchestrators
├── states/             # UI state management fixtures
├── api-responses/      # MSW response handlers with type safety
├── form-data/          # React Hook Form integration fixtures
├── types.ts            # Core type definitions
├── index.ts            # Central exports
└── README.md           # This file
```

## Key Features

### ✅ Type Safety First
- All fixtures use strict TypeScript interfaces
- Eliminates `any` types that plagued the old system
- Compile-time error detection for schema changes
- IntelliSense support for all fixture methods

### ✅ Real Integration Testing
- Uses actual `shapeBookmark.create()` and validation functions from `@vrooli/shared`
- Connects to real testcontainers PostgreSQL database
- Tests actual API endpoints through test server
- Verifies database constraints and relationships

### ✅ Test Isolation
- No global state pollution
- Each test gets fresh data through transactions
- Proper cleanup between tests
- Independent test execution

### ✅ Component Integration
- Works with real React Hook Form instances
- Simulates user interactions
- Tests form validation in components
- Verifies UI state transitions

### ✅ MSW Integration
- Generates MSW handlers from fixture responses
- Supports network error simulation
- Provides realistic response delays
- Enables offline testing scenarios

## Usage Examples

### Factory Pattern
```typescript
import { BookmarkFixtureFactory } from './factories/BookmarkFixtureFactory.js';

const factory = new BookmarkFixtureFactory();

// Create form data for testing
const formData = factory.createFormData('withNewList');

// Transform to API input using real shape functions
const apiInput = factory.transformToAPIInput(formData);

// Validate using real validation
const validation = await factory.validateFormData(formData);
```

### Integration Testing
```typescript
import { BookmarkIntegrationTest } from './integrations/bookmarkIntegration.js';

const integration = new BookmarkIntegrationTest();

// Execute full round-trip test
const result = await integration.testCompleteFlow({
  formData: factory.createFormData('complete'),
  shouldSucceed: true
});

expect(result.success).toBe(true);
expect(result.databaseRecord).toBeDefined();
```

### Scenario Testing
```typescript
import { UserBookmarksProjectScenario } from './scenarios/userBookmarksProject.scenario.js';

const scenario = new UserBookmarksProjectScenario();

// Test complex multi-step workflow
await scenario.execute({
  user: userFactory.createFormData('complete'),
  project: projectFactory.createFormData('public'),
  bookmarks: ['resource1', 'resource2', 'resource3']
});
```

## Architecture Benefits

### 1. **Maintainability**
- Centralized fixture management
- Single source of truth for test data
- Easy schema updates through TypeScript
- Version control friendly

### 2. **Reliability** 
- No test interference
- Predictable execution order
- Easy debugging with clear error messages
- Fast feedback loop

### 3. **Developer Experience**
- Clear, documented APIs
- Reusable test utilities
- Comprehensive error reporting
- Easy test creation

### 4. **Scalability**
- Supports complex multi-object workflows
- Handles large dataset generation
- Efficient resource management
- Parallel test execution

## Migration from Legacy Fixtures

The new architecture is designed to completely replace the problematic legacy fixtures. Key differences:

| Legacy Issue | New Solution |
|--------------|--------------|
| `any` types everywhere | Strict TypeScript interfaces |
| Global state pollution | Dependency injection pattern |
| Fake round-trip tests | Real database integration |
| No component integration | React Hook Form integration |
| Inconsistent data shapes | Real shape function usage |
| Manual MSW setup | Generated MSW handlers |

## Testing Patterns

### Pattern 1: Simple Form Test
```typescript
const formData = bookmarkFactory.createFormData('minimal');
const validation = await bookmarkFactory.validateFormData(formData);
expect(validation.isValid).toBe(true);
```

### Pattern 2: Integration Test
```typescript
const result = await bookmarkIntegration.testCreateFlow(formData);
expect(result.created).toBe(true);
expect(result.databaseRecord.id).toBeDefined();
```

### Pattern 3: UI Component Test
```typescript
const { result } = renderHook(() => useBookmarkForm());
await act(async () => {
  result.current.submitForm(formData);
});
expect(result.current.state.success).toBe(true);
```

### Pattern 4: Error Scenario Test
```typescript
const errorHandlers = bookmarkMSW.createErrorHandlers([
  { endpoint: '/api/bookmark', status: 500, message: 'Database error' }
]);
server.use(...errorHandlers);
// Test error handling in component
```

## Best Practices

### DO's ✅
- Use TypeScript interfaces for all fixtures
- Test with real validation schemas from `@vrooli/shared`
- Use actual shape functions for transformations
- Isolate tests with database transactions
- Verify database state in integration tests
- Use MSW for API mocking in component tests
- Test error scenarios comprehensively

### DON'Ts ❌
- Use `any` types in fixtures
- Create global test state
- Mock internal business logic
- Skip database verification
- Hardcode test data in components
- Ignore error scenarios
- Create fixtures without TypeScript types

## Contributing

When adding new fixtures:

1. **Follow the Factory Pattern**: Create a class that implements the standardized factory interface
2. **Use Real Functions**: Import and use actual validation and shape functions from `@vrooli/shared`
3. **Add Integration Tests**: Create corresponding integration tests that hit the database
4. **Document Scenarios**: Provide clear scenario names and documentation
5. **Include Error Cases**: Test both success and failure paths
6. **Update Exports**: Add new fixtures to the central `index.ts` file

## Technical Implementation Details

### Database Integration
- Uses testcontainers PostgreSQL for real database testing
- Each test runs in a transaction that's rolled back
- Supports complex relationship testing
- Validates database constraints

### Form Integration  
- Integrates with React Hook Form
- Supports field-level validation
- Tests form submission flows
- Simulates user interactions

### API Integration
- Uses real test server with actual endpoints
- Tests authentication and authorization
- Verifies request/response formatting
- Supports error scenario testing

### MSW Integration
- Generates handlers from fixture responses
- Supports dynamic response modification
- Enables network error simulation
- Provides realistic delays

This architecture provides a solid foundation for comprehensive, reliable, and maintainable UI testing that gives confidence in the entire application stack.