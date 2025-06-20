# Fixture Factories

This directory contains type-safe fixture factories that create test data using real functions from `@vrooli/shared`. These factories eliminate the use of `any` types and provide integration with actual validation and shape transformation functions.

## Purpose

Fixture factories provide:
- **Type-safe data generation** for all object types
- **Real validation** using actual validation schemas
- **Real transformations** using actual shape functions
- **Consistent API** across all object types
- **Scenario-based testing** with named scenarios

## Architecture

Each factory implements the `FixtureFactory` interface and provides:

```typescript
interface FixtureFactory<TFormData, TCreateInput, TUpdateInput, TObject> {
    readonly objectType: string;
    createFormData(scenario: string): TFormData;
    transformToAPIInput(formData: TFormData): TCreateInput;
    createUpdateInput(id: string, updates: Partial<TFormData>): TUpdateInput;
    createMockResponse(overrides?: Partial<TObject>): TObject;
    validateFormData(formData: TFormData): Promise<ValidationResult>;
    createMSWHandlers(): MSWHandlers;
}
```

## Key Principles

### 1. Real Function Usage
All factories use actual functions from `@vrooli/shared`:

```typescript
// ✅ CORRECT: Use real shape function
import { shapeBookmark } from "@vrooli/shared";

transformToAPIInput(formData: BookmarkFormData): BookmarkCreateInput {
    return shapeBookmark.create({
        __typename: "Bookmark",
        id: this.generateId(),
        to: { __typename: formData.bookmarkFor, id: formData.forConnect },
        list: formData.listId ? { id: formData.listId } : null
    });
}

// ❌ WRONG: Manual transformation (old approach)
transformToAPIInput(formData: any): any {
    return {
        id: generateId(),
        bookmarkFor: formData.bookmarkFor,
        forConnect: formData.forConnect
    };
}
```

### 2. Real Validation
Factories use actual validation schemas:

```typescript
// ✅ CORRECT: Use real validation
import { bookmarkValidation } from "@vrooli/shared";

async validateFormData(formData: BookmarkFormData): Promise<ValidationResult> {
    try {
        const apiInput = this.transformToAPIInput(formData);
        await bookmarkValidation.create.validate(apiInput);
        return { isValid: true };
    } catch (error) {
        return {
            isValid: false,
            errors: error.errors || [error.message]
        };
    }
}

// ❌ WRONG: Manual validation (old approach)
async validateFormData(formData: any): Promise<any> {
    const errors = [];
    if (!formData.forConnect) errors.push("forConnect required");
    return { isValid: errors.length === 0, errors };
}
```

### 3. Type Safety
No `any` types allowed:

```typescript
// ✅ CORRECT: Strict typing
export class BookmarkFixtureFactory implements FixtureFactory<
    BookmarkFormData,
    BookmarkCreateInput,
    BookmarkUpdateInput,
    Bookmark
> {
    createFormData(scenario: BookmarkScenario): BookmarkFormData {
        // Implementation with strict types
    }
}

// ❌ WRONG: Any types (old approach)
export class BookmarkFixtureFactory {
    createFormData(scenario: any): any {
        // Loses all type safety
    }
}
```

## Factory Implementation Pattern

Each factory follows this structure:

```typescript
export class ObjectFixtureFactory implements FixtureFactory<FormData, CreateInput, UpdateInput, Object> {
    readonly objectType = "objectName";
    
    // 1. Data generation methods
    createFormData(scenario: Scenario): FormData { /* */ }
    private generateId(): string { /* */ }
    
    // 2. Transformation methods
    transformToAPIInput(formData: FormData): CreateInput { /* */ }
    createUpdateInput(id: string, updates: Partial<FormData>): UpdateInput { /* */ }
    
    // 3. Mock response methods
    createMockResponse(overrides?: Partial<Object>): Object { /* */ }
    
    // 4. Validation methods
    async validateFormData(formData: FormData): Promise<ValidationResult> { /* */ }
    
    // 5. Testing utilities
    createMSWHandlers(): MSWHandlers { /* */ }
    createTestCases(): TestCase[] { /* */ }
    
    // 6. Convenience methods
    createForSpecificCase(params: SpecificParams): FormData { /* */ }
}
```

## Scenario-Based Testing

Factories use named scenarios for different test cases:

### Bookmark Scenarios
- `minimal`: Bare minimum required fields
- `complete`: All fields populated
- `invalid`: Invalid data for error testing
- `withNewList`: Creates new bookmark list
- `withExistingList`: Uses existing bookmark list
- `forProject`: Bookmarking a project/team
- `forRoutine`: Bookmarking a routine/comment

### Usage Examples

```typescript
import { BookmarkFixtureFactory } from './BookmarkFixtureFactory.js';

const factory = new BookmarkFixtureFactory();

// Create data for different scenarios
const minimal = factory.createFormData('minimal');
const complete = factory.createFormData('complete');
const invalid = factory.createFormData('invalid');

// Transform to API input
const apiInput = factory.transformToAPIInput(complete);

// Validate data
const validation = await factory.validateFormData(minimal);
expect(validation.isValid).toBe(true);

// Create mock response
const mockBookmark = factory.createMockResponse({
    to: { __typename: 'Resource', id: 'resource123' }
});
```

## Integration with Testing

### Unit Tests
```typescript
describe('BookmarkFixtureFactory', () => {
    const factory = new BookmarkFixtureFactory();
    
    it('should create valid minimal form data', () => {
        const formData = factory.createFormData('minimal');
        expect(formData.bookmarkFor).toBeDefined();
        expect(formData.forConnect).toBeDefined();
    });
    
    it('should transform to valid API input', () => {
        const formData = factory.createFormData('complete');
        const apiInput = factory.transformToAPIInput(formData);
        expect(apiInput.id).toBeDefined();
        expect(apiInput.bookmarkFor).toBe(formData.bookmarkFor);
    });
});
```

### Integration Tests
```typescript
describe('Bookmark Integration', () => {
    it('should create bookmark through API', async () => {
        const formData = factory.createFormData('complete');
        const apiInput = factory.transformToAPIInput(formData);
        
        const response = await testClient.post('/api/bookmark', apiInput);
        expect(response.status).toBe(201);
        expect(response.data.id).toBeDefined();
    });
});
```

### Component Tests
```typescript
describe('BookmarkForm Component', () => {
    it('should submit form with factory data', async () => {
        const formData = factory.createFormData('complete');
        
        render(<BookmarkForm initialData={formData} />);
        
        await user.click(screen.getByRole('button', { name: 'Save Bookmark' }));
        
        expect(mockSubmit).toHaveBeenCalledWith(
            expect.objectContaining({
                bookmarkFor: formData.bookmarkFor,
                forConnect: formData.forConnect
            })
        );
    });
});
```

## Error Handling

Factories provide comprehensive error scenario testing:

```typescript
// Test validation errors
const invalidData = factory.createFormData('invalid');
const validation = await factory.validateFormData(invalidData);
expect(validation.isValid).toBe(false);
expect(validation.errors).toContain('forConnect is required');

// Test API errors through MSW
const errorHandlers = factory.createMSWHandlers().error;
server.use(...errorHandlers);

// Test component error handling
await user.click(submitButton);
expect(screen.getByText('Error creating bookmark')).toBeInTheDocument();
```

## Best Practices

### DO's ✅
- Use real validation and shape functions from `@vrooli/shared`
- Implement strict TypeScript interfaces
- Provide multiple scenarios for different test cases
- Include error scenarios in testing
- Use consistent naming conventions
- Document scenario purposes

### DON'Ts ❌
- Use `any` types anywhere in the factory
- Create manual transformation logic
- Skip validation integration
- Hardcode IDs or dates
- Create global state
- Mix factory concerns with test logic

## Adding New Factories

When creating a new factory:

1. **Import Required Types**: Get types from `@vrooli/shared`
2. **Implement Interface**: Use the standard `FixtureFactory` interface
3. **Define Scenarios**: Create meaningful scenario names
4. **Use Real Functions**: Import validation and shape functions
5. **Add Tests**: Create unit tests for the factory
6. **Document Scenarios**: Explain what each scenario tests
7. **Export**: Add to the main index file

Example template:
```typescript
import type { ObjectFormData, ObjectCreateInput, ObjectUpdateInput, Object } from '../types.js';
import { objectValidation, shapeObject } from '@vrooli/shared';
import type { FixtureFactory, ValidationResult } from '../types.js';

export class ObjectFixtureFactory implements FixtureFactory<ObjectFormData, ObjectCreateInput, ObjectUpdateInput, Object> {
    readonly objectType = "object";
    
    createFormData(scenario: ObjectScenario): ObjectFormData {
        // Implementation
    }
    
    // ... other required methods
}
```

This approach ensures consistent, type-safe, and reliable fixture generation across the entire test suite.