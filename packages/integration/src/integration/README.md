# Integration Configuration Framework

This directory contains the standardized integration testing infrastructure for complete form-to-database workflows. It provides a unified approach to testing that leverages existing UI form fixtures (with integrated transformation methods) and database verification capabilities.

**Note**: As of 2025-01-25, ApiInputTransformer has been eliminated. All transformation methods are now included directly in UIFormTestConfig for better cohesion and type safety.

## Architecture Overview

The integration framework follows this data flow:

```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   UI Form           │    │   Shape Data        │    │   API Input         │
│   Fixtures          │───▶│   (formToShape)     │───▶│(form transforms)    │
│   (from UI package) │    │                     │    │                     │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
                                                                 │
                                                                 ▼
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   Database          │    │   API Response      │    │   Endpoint          │
│   Verification      │◀───│   (result)          │◀───│   Logic             │
│   (Prisma direct)   │    │                     │    │   (actual calls)    │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
```

## Key Components

### 1. StandardIntegrationConfig

The main configuration interface that combines all components:

```typescript
interface StandardIntegrationConfig<TFormData, TShape, TCreateInput, TUpdateInput, TResult> {
    objectType: string;                    // e.g., "Comment", "Project"
    uiFormConfig: UIFormTestConfig;        // From UI package (includes all transformations)
    endpointCaller: EndpointCaller;        // Actual API calls
    databaseVerifier: DatabaseVerifier;    // Direct DB verification
    validation: YupModel;                  // From shared package
}
```

### 2. UIFormTestConfig (Enhanced)

The UI form test config now includes transformation methods that replace ApiInputTransformer:

```typescript
interface UIFormTestConfig<TFormData, TShape, TCreateInput, TUpdateInput, TResult> {
    // ... existing form config properties ...
    
    // NEW: Response transformation methods (replaces ApiInputTransformer)
    responseToCreateInput?: (response: TResult) => TCreateInput;
    responseToUpdateInput?: (existing: TResult, updated: Partial<TResult>) => TUpdateInput;
    responseToFormData?: (response: TResult) => TFormData;
    validateBidirectionalTransform?: (response: TResult, input: TCreateInput | TUpdateInput) => ValidationResult;
}
```

### 3. EndpointCaller

Handles actual API endpoint calls with proper error handling:

```typescript
interface EndpointCaller<TCreateInput, TUpdateInput, TResult> {
    create(input: TCreateInput, session?: Session): Promise<EndpointResult<TResult>>;
    update(id: string, input: TUpdateInput, session?: Session): Promise<EndpointResult<TResult>>;
    read(id: string, session?: Session): Promise<EndpointResult<TResult>>;
    delete(id: string, session?: Session): Promise<EndpointResult<void>>;
}
```

### 4. DatabaseVerifier

Provides direct database access for persistence verification:

```typescript
interface DatabaseVerifier<TResult> {
    findById(id: string): Promise<TResult | null>;
    findByConstraints(constraints: Record<string, any>): Promise<TResult | null>;
    verifyConsistency(apiResult: TResult, databaseResult: TResult): ConsistencyResult;
    cleanup(id: string): Promise<void>;
}
```

## Creating Integration Configurations

### Step 1: Define Form Fixtures

Use the same fixtures as your UI tests, or create new ones:

```typescript
const myObjectFormFixtures = {
    minimal: {
        name: "Test Object",
        isPrivate: false
    },
    complete: {
        name: "Complete Test Object",
        description: "Full description with all fields",
        isPrivate: false,
        tags: ["test", "integration"],
        metadata: { category: "testing" }
    },
    invalid: {
        name: "", // Empty name should fail validation
        isPrivate: false
    }
};
```

### Step 2: Create Enhanced Form Config

Add transformation methods to your UIFormTestConfig:

```typescript
const myObjectFormTestConfig: UIFormTestConfig<MyObjectFormData, MyObjectShape, MyObjectCreateInput, MyObjectUpdateInput, MyObject> = {
    // ... existing form config properties ...
    
    // NEW: Response transformation methods
    responseToCreateInput: (response: MyObject): MyObjectCreateInput => ({
        name: response.name,
        description: response.description,
        isPrivate: response.isPrivate,
        // Map other fields from response to create input format
    }),
    
    responseToUpdateInput: (existing: MyObject, updated: Partial<MyObject>): MyObjectUpdateInput => ({
        id: existing.id,
        name: updated.name ?? existing.name,
        description: updated.description ?? existing.description,
        isPrivate: updated.isPrivate ?? existing.isPrivate,
        // Map other updatable fields
    }),
    
    responseToFormData: (response: MyObject): MyObjectFormData => ({
        name: response.name,
        description: response.description,
        isPrivate: response.isPrivate,
        // Map response back to form data for display
    }),
    
    validateBidirectionalTransform: (response: MyObject, input: MyObjectCreateInput | MyObjectUpdateInput) => {
        const errors: string[] = [];
        
        // Validate that essential fields match
        if ('name' in input && response.name !== input.name) {
            errors.push(`Name mismatch: response=${response.name}, input=${input.name}`);
        }
        
        return {
            isValid: errors.length === 0,
            errors,
            warnings: []
        };
    }
};
```

### Step 3: Create Endpoint Caller

Handle actual API calls:

```typescript
const myObjectEndpointCaller: EndpointCaller<MyObjectCreateInput, MyObjectUpdateInput, MyObject> = {
    create: async (input: MyObjectCreateInput, session?: Session) => {
        const startTime = Date.now();
        
        try {
            // Import actual endpoint logic
            const { default: myObjectLogic } = await import('@vrooli/server/src/endpoints/logic/myObject.js');
            
            const result = await myObjectLogic.Create.performLogic({
                input,
                userData: session,
            });
            
            return {
                success: true,
                data: result,
                timing: Date.now() - startTime
            };
            
        } catch (error) {
            return {
                success: false,
                error: {
                    code: 'CREATE_FAILED',
                    message: error instanceof Error ? error.message : 'Unknown error',
                    statusCode: 500
                },
                timing: Date.now() - startTime
            };
        }
    },
    
    // Implement update, read, delete similarly...
};
```

### Step 3: Create Database Verifier

Provide direct database access:

```typescript
const myObjectDatabaseVerifier: DatabaseVerifier<MyObject> = {
    findById: async (id: string): Promise<MyObject | null> => {
        try {
            const prisma = getPrisma();
            const result = await prisma.myObject.findUnique({
                where: { id },
                include: {
                    // Include related data as needed
                }
            });
            
            return result ? {
                ...result,
                __typename: "MyObject" as const,
                // Map other fields as needed
            } as MyObject : null;
            
        } catch (error) {
            console.error('Database find error:', error);
            return null;
        }
    },
    
    verifyConsistency: (apiResult: MyObject, databaseResult: MyObject) => {
        const differences: Array<{ field: string; apiValue: any; dbValue: any }> = [];
        
        // Check essential fields
        if (apiResult.id !== databaseResult.id) {
            differences.push({ field: 'id', apiValue: apiResult.id, dbValue: databaseResult.id });
        }
        
        if (apiResult.name !== databaseResult.name) {
            differences.push({ field: 'name', apiValue: apiResult.name, dbValue: databaseResult.name });
        }
        
        return {
            consistent: differences.length === 0,
            differences
        };
    },
    
    cleanup: async (id: string): Promise<void> => {
        try {
            const prisma = getPrisma();
            await prisma.myObject.delete({ where: { id } });
        } catch (error) {
            console.error('Cleanup error:', error);
        }
    }
};
```

### Step 4: Combine into Configuration

Create the complete integration configuration:

```typescript
export const myObjectIntegrationConfig: StandardIntegrationConfig<
    MyObjectFormData,
    MyObjectShape,
    MyObjectCreateInput,
    MyObjectUpdateInput,
    MyObject
> = {
    objectType: "MyObject",
    uiFormConfig: myObjectFormTestConfig, // Contains all transformation methods
    endpointCaller: myObjectEndpointCaller,
    databaseVerifier: myObjectDatabaseVerifier,
    validation: myObjectValidation // Import from shared package
};
```

## Using Integration Configurations

### With IntegrationEngine

```typescript
import { createIntegrationEngine } from '../engine/IntegrationEngine.js';
import { myObjectIntegrationConfig } from './MyObjectIntegrationConfig.js';

const engine = createIntegrationEngine(myObjectIntegrationConfig);

// Single test
const result = await engine.executeTest('minimal', {
    isCreate: true,
    validateConsistency: true
});

// Batch tests
const scenarios = engine.generateTestScenarios();
const batchResult = await engine.executeBatch(scenarios);
```

### In Test Files

```typescript
describe('MyObject Integration Tests', () => {
    const engine = createIntegrationEngine(myObjectIntegrationConfig);
    
    it('should complete object creation workflow', async () => {
        const result = await engine.executeTest('complete', {
            isCreate: true,
            validateConsistency: true,
            timeout: 30000
        });
        
        expect(result.success).toBe(true);
        expect(result.dataFlow.apiResponse?.id).toBe(result.dataFlow.databaseData?.id);
        expect(result.consistency.overallValid).toBe(true);
        expect(result.timing.total).toBeLessThan(5000);
    });
    
    it('should handle validation errors gracefully', async () => {
        const result = await engine.executeTest('invalid', {
            isCreate: true,
            validateConsistency: false
        });
        
        expect(result.success).toBe(false);
        expect(result.errors.length).toBeGreaterThan(0);
    });
    
    it('should test all scenarios in batch', async () => {
        const scenarios = engine.generateTestScenarios();
        const batchResult = await engine.executeBatch(scenarios, {}, true, 3);
        
        expect(batchResult.performance.successRate).toBeGreaterThan(0.8);
        expect(batchResult.summary.passed).toBeGreaterThan(0);
    });
});
```

## File Organization

```
/src/integration/
├── types.ts                           # Core interfaces and types
├── index.ts                          # Main exports
├── CommentIntegrationConfig.ts        # Example implementation
├── MyObjectIntegrationConfig.ts       # Your implementation
└── README.md                         # This documentation
```

## Best Practices

### Configuration Design

1. **Reuse UI Fixtures**: Always start with existing UI form fixtures
2. **Real API Calls**: Use actual endpoint logic, not mocks
3. **Direct DB Access**: Use Prisma directly for database verification
4. **Comprehensive Error Handling**: Handle all failure scenarios
5. **Performance Awareness**: Monitor timing and resource usage

### Data Transformation

1. **Bidirectional Consistency**: Ensure response ↔ input transformations are reliable
2. **Field Mapping**: Map all essential fields between formats
3. **Validation**: Validate transformations thoroughly
4. **Edge Cases**: Handle null/undefined values appropriately

### Testing Strategy

1. **Comprehensive Coverage**: Test all fixture scenarios
2. **Error Scenarios**: Include invalid data testing
3. **Performance Testing**: Use batch execution for load testing
4. **Cleanup**: Always clean up test data
5. **Isolation**: Ensure tests don't interfere with each other

## Troubleshooting

### Common Issues

1. **Type Mismatches**: Ensure type mappings are correct between form data, shapes, inputs, and results
2. **Missing Fields**: Check that all required fields are mapped in transformers
3. **Database Constraints**: Ensure test data respects database constraints
4. **Async Issues**: Use proper async/await patterns throughout
5. **Cleanup Failures**: Handle cleanup errors gracefully

### Debugging Tips

1. **Enable Detailed Logging**: Set `captureTimings: true` for performance insights
2. **Check Data Flow**: Examine the `dataFlow` object in test results
3. **Validate Transformations**: Use the transformer's `validateTransformation` method
4. **Database State**: Check database state manually if consistency fails
5. **Error Details**: Examine the `errors` array for specific failure information

This framework provides a solid foundation for comprehensive integration testing that validates the complete data flow from UI forms through API endpoints to database persistence.