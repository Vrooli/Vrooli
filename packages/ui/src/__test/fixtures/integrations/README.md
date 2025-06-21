# Integration Tests

This directory contains true integration tests that connect the entire stack from UI forms to the database using testcontainers. These tests validate the complete data flow and eliminate the fake round-trip testing that plagued the legacy system.

## Purpose

Integration tests provide:
- **Real database testing** with testcontainers PostgreSQL
- **Complete API validation** through actual endpoints
- **End-to-end data flow verification** from form to database and back
- **Database constraint testing** with real schema validation
- **Transaction isolation** for reliable test execution

## Architecture

Each integration test follows this pattern:

```typescript
class ObjectIntegrationTest {
    constructor(
        private apiClient: TestAPIClient,
        private dbVerifier: DatabaseVerifier,
        private transactionManager: TestTransactionManager
    ) {}
    
    async testCompleteFlow(config: IntegrationTestConfig): Promise<IntegrationTestResult> {
        return this.transactionManager.executeInTransaction(async () => {
            // 1. Validate form data using real validation
            // 2. Transform to API input using real shape functions
            // 3. Make actual API call to test server
            // 4. Verify database state directly
            // 5. Fetch data back through API
            // 6. Verify UI can display the data
        });
    }
}
```

## Real Integration vs Fake Testing

### ❌ Legacy Approach (Fake Round-Trip)
```typescript
// PROBLEMATIC: Fake service with mock storage
const mockBookmarkService = {
    storage: {} as Record<string, any>,
    async create(data: any): Promise<any> {
        const id = generateId();
        this.storage[id] = { ...data, id };
        return this.storage[id];
    }
};

// This doesn't test:
// - Real database constraints
// - Actual API validation
// - Database relationships
// - Schema compliance
// - Business logic enforcement
```

### ✅ New Approach (Real Integration)
```typescript
// CORRECT: Real integration with testcontainers
export class BookmarkIntegrationTest {
    async testCreateFlow(config: IntegrationTestConfig<BookmarkFormData>): Promise<IntegrationTestResult<Bookmark>> {
        return this.transactionManager.executeInTransaction(async () => {
            // 1. Real validation using @vrooli/shared
            const validation = await bookmarkValidation.create.validate(apiInput);
            
            // 2. Real API call to test server
            const response = await this.apiClient.post('/api/bookmark', apiInput);
            
            // 3. Real database verification
            const dbRecord = await this.dbVerifier.getRecord('Bookmark', response.data.id);
            
            // 4. Real relationship verification
            const listRecord = await this.dbVerifier.getRecord('BookmarkList', dbRecord.listId);
            
            return { success: true, data: { /* complete flow data */ } };
        });
    }
}
```

## Key Integration Points

### 1. Database Integration
- Uses testcontainers PostgreSQL for real database testing
- Validates actual schema constraints and relationships
- Tests database triggers and functions
- Verifies data integrity across transactions

### 2. API Integration
- Makes real HTTP calls to test server
- Tests authentication and authorization
- Validates request/response serialization
- Tests error handling and status codes

### 3. Validation Integration
- Uses actual validation schemas from `@vrooli/shared`
- Tests complex validation rules
- Verifies error message formatting
- Tests field-level validation

### 4. Shape Function Integration
- Uses real shape transformation functions
- Tests data transformation accuracy
- Validates nested object handling
- Tests optional field processing

## Test Configuration

### Integration Test Config
```typescript
interface IntegrationTestConfig<TFormData> {
    formData: TFormData;
    shouldSucceed: boolean;
    expectedErrors?: string[];
    authContext?: AuthContext;
    databasePreConditions?: DatabasePreCondition[];
}
```

### Auth Context
```typescript
interface AuthContext {
    userId: string;
    handle: string;
    permissions: string[];
    teamMemberships?: string[];
}
```

### Database Pre-Conditions
```typescript
interface DatabasePreCondition {
    table: string;
    data: Record<string, any>;
    cleanup?: boolean;
}
```

## Test Scenarios

### Success Scenarios
- **Complete Flow**: Form → Validation → API → Database → Response
- **Relationship Creation**: Testing foreign key relationships
- **Permission Validation**: Testing access control
- **Data Transformation**: Verifying shape function accuracy

### Error Scenarios
- **Validation Errors**: Invalid form data handling
- **Database Constraints**: Foreign key violations, unique constraints
- **Permission Errors**: Unauthorized access attempts
- **Network Errors**: Connection failures, timeouts

## Usage Examples

### Basic Integration Test
```typescript
describe('Bookmark Integration', () => {
    let integration: BookmarkIntegrationTest;
    
    beforeEach(async () => {
        integration = new BookmarkIntegrationTest(
            await createTestAPIClient(),
            new DatabaseVerifier(),
            new TestTransactionManager()
        );
    });
    
    it('should create bookmark through complete flow', async () => {
        const formData = bookmarkFixtures.createFormData('complete');
        
        const result = await integration.testCompleteFlow({
            formData,
            shouldSucceed: true
        });
        
        expect(result.success).toBe(true);
        expect(result.data.databaseRecord).toBeDefined();
        expect(result.data.fetchedData.id).toBe(result.data.apiResponse.id);
    });
});
```

### Error Handling Test
```typescript
it('should handle validation errors correctly', async () => {
    const invalidFormData = bookmarkFixtures.createFormData('invalid');
    
    const result = await integration.testCompleteFlow({
        formData: invalidFormData,
        shouldSucceed: false,
        expectedErrors: ['bookmarkFor must be a valid type']
    });
    
    expect(result.success).toBe(false);
    expect(result.errors).toContain('bookmarkFor must be a valid type');
});
```

### Complex Scenario Test
```typescript
it('should create bookmark with new list', async () => {
    const formData = bookmarkFixtures.createFormData('withNewList');
    
    const result = await integration.testCompleteFlow({
        formData,
        shouldSucceed: true
    });
    
    // Verify bookmark was created
    expect(result.data.databaseRecord).toBeDefined();
    
    // Verify new list was created
    const listRecord = await integration.dbVerifier.getRecord(
        'BookmarkList', 
        result.data.databaseRecord.listId
    );
    expect(listRecord.label).toBe(formData.newListLabel);
});
```

## Database Transaction Management

### Transaction Isolation
```typescript
export class TestTransactionManager {
    async executeInTransaction<T>(fn: () => Promise<T>): Promise<T> {
        const transaction = await this.db.transaction();
        try {
            const result = await fn();
            await transaction.rollback(); // Always rollback for test isolation
            return result;
        } catch (error) {
            await transaction.rollback();
            throw error;
        }
    }
}
```

### Database Verification
```typescript
export class DatabaseVerifier {
    async verifyRecordExists(table: string, id: string): Promise<boolean> {
        const record = await this.db[table].findUnique({ where: { id } });
        return record !== null;
    }
    
    async verifyRecordData(table: string, id: string, expected: Record<string, any>): Promise<boolean> {
        const record = await this.db[table].findUnique({ where: { id } });
        if (!record) return false;
        
        return Object.entries(expected).every(([key, value]) => 
            record[key] === value
        );
    }
}
```

## Test Client Setup

### API Client
```typescript
export class TestAPIClient {
    constructor(private baseUrl: string) {}
    
    async post<T>(endpoint: string, data: any): Promise<APIResponse<T>> {
        const response = await fetch(`${this.baseUrl}${endpoint}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        
        return {
            data: await response.json(),
            status: response.status,
            headers: Object.fromEntries(response.headers.entries()),
            duration: performance.now() - startTime
        };
    }
}
```

## Performance Considerations

### Test Optimization
- **Parallel Execution**: Tests run in parallel with isolated transactions
- **Container Reuse**: Testcontainers are shared across test files
- **Connection Pooling**: Database connections are pooled for efficiency
- **Cleanup Strategy**: Automatic cleanup prevents data accumulation

### Monitoring
- **Duration Tracking**: All tests track execution time
- **Memory Usage**: Monitor container resource usage
- **Query Performance**: Track database query execution times
- **Error Rates**: Monitor test failure patterns

## Best Practices

### DO's ✅
- Use real testcontainers for database testing
- Execute tests in isolated transactions
- Verify database state after API calls
- Test complete data flow end-to-end
- Include both success and error scenarios
- Use real validation and shape functions

### DON'Ts ❌
- Mock database operations in integration tests
- Share data between tests
- Skip database verification steps
- Test only happy path scenarios
- Use fake services or storage
- Hardcode test data or IDs

## Troubleshooting

### Common Issues
1. **Container Startup Failures**: Check Docker daemon and port availability
2. **Transaction Deadlocks**: Ensure proper transaction ordering
3. **Schema Mismatches**: Verify migrations are applied correctly
4. **Connection Timeouts**: Increase timeout values for slow environments
5. **Memory Issues**: Monitor container resource limits

### Debug Strategies
- Enable database query logging
- Use transaction debugging tools
- Monitor container logs
- Track API request/response cycles
- Verify test data isolation

This integration testing approach provides confidence that the entire application stack works correctly together, from UI forms all the way down to the database and back.