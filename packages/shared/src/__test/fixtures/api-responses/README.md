# API Response Fixtures

This directory contains the unified API response fixture infrastructure for the Vrooli platform. These fixtures provide consistent API response generation across all packages (UI, server, jobs), eliminating duplication and ensuring type safety.

## Overview

The API response fixtures leverage a sophisticated base factory pattern that:
- Eliminates 70%+ of boilerplate code compared to individual implementations
- Integrates with the existing error fixture infrastructure
- Provides type-safe response generation with full TypeScript support
- Supports MSW (Mock Service Worker) for UI testing
- Enables true round-trip testing from UI to database

## Architecture

```
BaseAPIResponseFactory (Abstract)
    ├── Provides standard response wrappers (APIResponse, PaginatedAPIResponse, etc.)
    ├── Integrates with error fixtures for consistent error responses
    ├── Generates MSW handlers automatically
    └── Handles ID generation, timestamps, and metadata

Entity-Specific Factories (Concrete)
    ├── BookmarkResponseFactory
    ├── UserResponseFactory (to be migrated)
    ├── TeamResponseFactory (to be migrated)
    └── ... (35+ more to be migrated)
```

## Key Features

### 1. Minimal Boilerplate
Instead of duplicating error methods in every fixture:
```typescript
// Old way - 100+ lines of error methods per fixture
createValidationErrorResponse(fields) { /* 20 lines */ }
createNotFoundErrorResponse(id) { /* 18 lines */ }
createPermissionErrorResponse(op) { /* 19 lines */ }
// ... etc

// New way - inherited from base
export class BookmarkResponseFactory extends BaseAPIResponseFactory<...> {
    protected readonly entityName = "bookmark";
    // Only implement entity-specific logic
}
```

### 2. Leverages Existing Infrastructure
All error responses use the comprehensive error fixtures:
```typescript
// Automatically uses validationErrorFixtures, apiErrorFixtures, etc.
factory.createValidationErrorResponse(fieldErrors);
factory.createPermissionErrorResponse("update");
factory.createRateLimitErrorResponse(100, 0, resetTime);
```

### 3. Type-Safe Generic Design
Full TypeScript support with proper generics:
```typescript
export class YourEntityResponseFactory extends BaseAPIResponseFactory<
    YourEntity,        // Response data type
    YourCreateInput,   // Create input type
    YourUpdateInput    // Update input type
> {
    // Implementation is fully type-safe
}
```

### 4. MSW Integration
Automatic handler generation for Mock Service Worker:
```typescript
// Generate all CRUD handlers with one call
const handlers = factory.createMSWHandlers();

// Or with custom configuration
const handlers = factory.createMSWHandlers({
    delay: 2000,           // Simulate network delay
    errorRate: 0.1,        // 10% error rate for chaos testing
    networkFailureRate: 0.05  // 5% network failures
});
```

## Usage

### Basic Usage
```typescript
import { bookmarkResponseFactory, bookmarkResponseScenarios } from "@vrooli/shared";

// Create mock data
const bookmark = bookmarkResponseFactory.createMockData();

// Create API responses
const successResponse = bookmarkResponseFactory.createSuccessResponse(bookmark);
const errorResponse = bookmarkResponseFactory.createValidationErrorResponse({
    forConnect: "Target object is required"
});

// Use pre-configured scenarios
const response = bookmarkResponseScenarios.createSuccess();
const error = bookmarkResponseScenarios.permissionError("delete");
```

### In Tests
```typescript
// Endpoint tests
describe("bookmark endpoints", () => {
    it("should handle validation errors", async () => {
        const expectedError = bookmarkResponseScenarios.validationError({
            forConnect: "Required field"
        });
        
        // Test endpoint behavior matches expected error structure
    });
});

// UI tests with MSW
import { setupServer } from "msw/node";

const server = setupServer(
    ...bookmarkResponseScenarios.handlers.success()
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
```

### Creating New Response Fixtures

1. **Create the factory class**:
```typescript
import { BaseAPIResponseFactory } from "./base.js";

export class YourEntityResponseFactory extends BaseAPIResponseFactory<
    YourEntity,
    YourCreateInput,
    YourUpdateInput
> {
    protected readonly entityName = "your-entity";
    
    createMockData(options?: MockDataOptions): YourEntity {
        // Implement mock data generation
    }
    
    createFromInput(input: YourCreateInput): YourEntity {
        // Convert input to entity
    }
    
    updateFromInput(existing: YourEntity, input: YourUpdateInput): YourEntity {
        // Apply updates to existing entity
    }
    
    async validateCreateInput(input: YourCreateInput) {
        // Validate using existing validation schemas
    }
    
    async validateUpdateInput(input: YourUpdateInput) {
        // Validate update input
    }
}
```

2. **Create response scenarios**:
```typescript
export const yourEntityResponseScenarios = {
    // Success scenarios
    createSuccess: (entity?) => /* ... */,
    listSuccess: (entities?) => /* ... */,
    
    // Error scenarios (all inherited!)
    validationError: (fields?) => new YourEntityResponseFactory().createValidationErrorResponse(fields),
    notFoundError: (id?) => new YourEntityResponseFactory().createNotFoundErrorResponse(id),
    permissionError: (op?) => new YourEntityResponseFactory().createPermissionErrorResponse(op),
    
    // MSW handlers
    handlers: {
        success: () => new YourEntityResponseFactory().createMSWHandlers(),
        withErrors: (rate) => new YourEntityResponseFactory().createMSWHandlers({ errorRate: rate }),
    },
};
```

3. **Export from index**:
```typescript
// In index.ts
export * from "./yourEntityResponses.js";
```

## Migration Status

### Completed ✅
- Base infrastructure (`base.ts`, `types.ts`)
- Bookmark response fixtures (proof of concept)
- Integration with error fixtures
- MSW handler generation
- Documentation and migration guide

### To Be Migrated 📋
The following fixtures from `packages/ui/src/__test/fixtures/api-responses/` need migration:
- High Priority: User, Team, Auth, Chat (core functionality)
- Medium Priority: Resource, Project, Routine, Meeting, Schedule
- Lower Priority: All remaining fixtures (25+ files)

## Benefits

1. **Code Reduction**: 70%+ reduction in fixture code (from ~666 to ~200 lines per fixture)
2. **Consistency**: All packages use the same response structures
3. **Type Safety**: Full TypeScript support with zero `any` types
4. **Error Handling**: Leverages comprehensive error infrastructure
5. **Maintainability**: Changes to base affect all fixtures automatically
6. **Testing Power**: Enables true round-trip testing across all layers
7. **Developer Experience**: Simple, intuitive API with powerful features

## Best Practices

1. **Always extend BaseAPIResponseFactory** - Don't recreate functionality
2. **Use existing validation schemas** - Leverage `validation/models/*.js`
3. **Keep mock data realistic** - Use proper IDs, timestamps, and relationships
4. **Leverage scenarios** - Pre-configured scenarios reduce test complexity
5. **Document entity-specific behavior** - Add comments for special cases

## Future Enhancements

1. **GraphQL Support**: Add GraphQL-specific response wrappers
2. **WebSocket Events**: Integrate with event fixtures for real-time testing
3. **Batch Operations**: Enhanced support for bulk operations
4. **Performance Testing**: Built-in performance measurement
5. **Automated Migration**: Script to migrate remaining UI fixtures

## Related Documentation

- [Testing Ideology](/docs/testing/README.md) - Overall testing philosophy
- [Error Fixtures](/packages/shared/src/__test/fixtures/errors/README.md) - Error infrastructure
- [API Fixtures](/packages/shared/src/__test/fixtures/api-inputs/README.md) - API validation fixtures
- [Migration Guide](./migrationGuide.md) - Step-by-step migration instructions