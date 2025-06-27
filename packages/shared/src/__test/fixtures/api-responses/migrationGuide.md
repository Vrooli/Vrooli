# API Response Fixture Migration Guide

This guide demonstrates how to migrate API response fixtures from the UI package to the shared package, leveraging the new base factory infrastructure.

## Before vs After Comparison

### Before (UI Package) - ~666 lines
```typescript
// packages/ui/src/__test/fixtures/api-responses/bookmarkResponses.ts
export interface APIResponse<T> { /* duplicated in every file */ }
export interface APIErrorResponse { /* duplicated in every file */ }
export interface PaginatedAPIResponse<T> { /* duplicated in every file */ }

export class BookmarkResponseFactory {
    private generateRequestId(): string { /* duplicated */ }
    private generateId(): string { /* duplicated */ }
    
    createValidationErrorResponse(fieldErrors) { /* 20 lines */ }
    createNotFoundErrorResponse(bookmarkId) { /* 18 lines */ }
    createPermissionErrorResponse(operation) { /* 19 lines */ }
    createNetworkErrorResponse() { /* 17 lines */ }
    createServerErrorResponse() { /* 17 lines */ }
    // ... plus MSW handlers with duplicate logic
}
```

### After (Shared Package) - ~200 lines (70% reduction!)
```typescript
// packages/shared/src/__test/fixtures/api-responses/bookmarkResponses.ts
import { BaseAPIResponseFactory } from "./base.js";

export class BookmarkResponseFactory extends BaseAPIResponseFactory<
    Bookmark, BookmarkCreateInput, BookmarkUpdateInput
> {
    protected readonly entityName = "bookmark";
    
    // Only implement entity-specific logic
    createMockData(options?: MockDataOptions): Bookmark { /* ... */ }
    createFromInput(input: BookmarkCreateInput): Bookmark { /* ... */ }
    updateFromInput(existing: Bookmark, input: BookmarkUpdateInput): Bookmark { /* ... */ }
    validateCreateInput(input): Promise<ValidationResult> { /* ... */ }
    validateUpdateInput(input): Promise<ValidationResult> { /* ... */ }
}

// Error scenarios now leverage base factory
export const bookmarkResponseScenarios = {
    validationError: (fields?) => factory.createValidationErrorResponse(fields),
    notFoundError: (id?) => factory.createNotFoundErrorResponse(id),
    // etc...
};
```

## Migration Steps

### 1. Create New Response Factory

```typescript
import { BaseAPIResponseFactory } from "./base.js";
import type { YourEntity, YourCreateInput, YourUpdateInput } from "../../../api/types.js";
import { yourValidation } from "../../../validation/models/yourEntity.js";

export class YourEntityResponseFactory extends BaseAPIResponseFactory<
    YourEntity,
    YourCreateInput,
    YourUpdateInput
> {
    protected readonly entityName = "your-entity";
    
    // Implement required methods...
}
```

### 2. Implement Required Methods

```typescript
createMockData(options?: MockDataOptions): YourEntity {
    const id = options?.overrides?.id || this.generateId();
    const now = new Date().toISOString();
    
    return {
        __typename: "YourEntity",
        id,
        createdAt: now,
        updatedAt: now,
        // ... entity-specific fields
        ...options?.overrides,
    };
}

async validateCreateInput(input: YourCreateInput) {
    try {
        await yourValidation.create.parseAsync(input);
        return { valid: true };
    } catch (error: any) {
        // Convert zod errors to field errors
        return { valid: false, errors: extractFieldErrors(error) };
    }
}
```

### 3. Create Response Scenarios

```typescript
export const yourEntityResponseScenarios = {
    // Success scenarios
    createSuccess: (entity?) => {
        const factory = new YourEntityResponseFactory();
        return factory.createSuccessResponse(entity || factory.createMockData());
    },
    
    // Error scenarios - all inherited from base!
    validationError: (fields?) => new YourEntityResponseFactory().createValidationErrorResponse(fields),
    notFoundError: (id?) => new YourEntityResponseFactory().createNotFoundErrorResponse(id),
    permissionError: (op?) => new YourEntityResponseFactory().createPermissionErrorResponse(op),
    rateLimitError: () => new YourEntityResponseFactory().createRateLimitErrorResponse(100, 0, new Date()),
    serverError: () => new YourEntityResponseFactory().createServerErrorResponse(),
    
    // MSW handlers
    handlers: {
        success: () => new YourEntityResponseFactory().createMSWHandlers(),
        withErrors: (rate) => new YourEntityResponseFactory().createMSWHandlers({ errorRate: rate }),
    },
};
```

### 4. Update Imports

```typescript
// Old import from UI
import { bookmarkResponseScenarios } from "@ui/test/fixtures/api-responses/bookmarkResponses";

// New import from shared
import { bookmarkResponseScenarios } from "@vrooli/shared";
```

## Benefits Achieved

1. **70% Code Reduction**: From ~666 lines to ~200 lines per fixture
2. **Zero Duplication**: No more copy-paste of interfaces and error methods
3. **Consistent Error Handling**: Leverages shared error infrastructure
4. **Type Safety**: Full TypeScript support with proper generics
5. **MSW Integration**: Automatic handler generation with configurable behavior
6. **Cross-Package Usage**: Available to all packages (ui, server, jobs, etc.)
7. **Future-Proof**: New error types automatically available to all fixtures

## Quick Migration Checklist

- [ ] Create new factory extending `BaseAPIResponseFactory`
- [ ] Set `entityName` property
- [ ] Implement `createMockData()` method
- [ ] Implement `createFromInput()` and `updateFromInput()` methods
- [ ] Implement validation methods using existing validation schemas
- [ ] Create response scenarios object
- [ ] Export factory and scenarios
- [ ] Update imports in consuming code
- [ ] Delete old UI fixture file
- [ ] Run tests to verify functionality