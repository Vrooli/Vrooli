# TRUE Round-Trip Testing in Vrooli

## ⚠️ Critical Understanding: What Round-Trip Testing Actually Means

**Round-trip testing** validates the COMPLETE data flow through your entire application stack:

```
UI Form → API Request → Server Endpoint → Business Logic → Database → Response → UI State
```

## 🚨 The Problem: Fake "Integration" Tests

Many of our current "integration" tests are **NOT true round-trip tests**. They:

1. **Bypass the API** by importing Prisma directly from `@vrooli/server`
2. **Mock API responses** with MSW instead of hitting real endpoints
3. **Directly manipulate the database** without going through server logic

### Example of WRONG Approach:
```typescript
// ❌ This is NOT round-trip testing!
import { prisma } from "@vrooli/server"; // Direct database access!

it('should save bookmark', async () => {
    // This bypasses ALL server logic, validation, and middleware!
    const bookmark = await prisma.bookmark.create({
        data: { /* ... */ }
    });
});
```

This test is actually doing: `Test → Database` (skipping the entire API layer!)

## ✅ The Solution: TRUE Round-Trip Testing

### 1. Infrastructure Setup

The UI test infrastructure now:
- Starts real PostgreSQL and Redis containers
- Runs database migrations
- **Starts the actual Express API server**
- Makes real HTTP requests to API endpoints

### 2. Correct Testing Approach

```typescript
// ✅ TRUE round-trip testing
import { createTestAPIClient } from '../setup.vitest';

it('should create bookmark through complete flow', async () => {
    const apiClient = createTestAPIClient();
    
    // 1. Authenticate (real auth flow)
    const authResponse = await apiClient.post('/api/auth/login', {
        email: 'test@example.com',
        password: 'testpass'
    });
    
    // 2. Create bookmark (real API request)
    const bookmarkResponse = await apiClient.post('/api/bookmark', {
        bookmarkFor: 'Resource',
        forConnect: 'resource_123'
    }, {
        headers: {
            'Authorization': `Bearer ${authResponse.data.token}`
        }
    });
    
    // 3. Verify through API (not direct database access!)
    const getResponse = await apiClient.get(
        `/api/bookmark/${bookmarkResponse.data.id}`
    );
    
    expect(getResponse.data.id).toBe(bookmarkResponse.data.id);
});
```

## 📋 Migration Guide

### From Fake Integration to True Round-Trip

#### Before (Wrong):
```typescript
// Direct database access
const prisma = await getTestDatabaseClient();
const user = await prisma.user.create({ /* ... */ });

// Or using MSW mocks
server.use(
    rest.post('/api/bookmark', (req, res, ctx) => {
        return res(ctx.json({ /* mock */ }));
    })
);
```

#### After (Correct):
```typescript
// Real API calls
const apiClient = createTestAPIClient();

// Create test user through API
const userResponse = await apiClient.post('/api/user', {
    name: 'Test User',
    email: 'test@example.com'
});

// All operations go through API
const bookmarkResponse = await apiClient.post('/api/bookmark', {
    userId: userResponse.data.id,
    /* ... */
});
```

## 🏗️ Test Structure

### 1. Setup Test Data
```typescript
beforeEach(async () => {
    const apiClient = createTestAPIClient();
    
    // Create test data through API endpoints
    const user = await apiClient.post('/api/test/user', {
        name: 'Test User'
    });
    
    const resource = await apiClient.post('/api/test/resource', {
        name: 'Test Resource',
        ownerId: user.data.id
    });
});
```

### 2. Perform Operations
```typescript
it('should handle complete bookmark lifecycle', async () => {
    // All operations through API
    const created = await apiClient.post('/api/bookmark', data);
    const fetched = await apiClient.get(`/api/bookmark/${created.data.id}`);
    const updated = await apiClient.put(`/api/bookmark/${created.data.id}`, updates);
    const deleted = await apiClient.delete(`/api/bookmark/${created.data.id}`);
});
```

### 3. Clean Up
```typescript
afterEach(async () => {
    // Clean up through API endpoints
    await apiClient.delete(`/api/test/cleanup/${testId}`);
});
```

## 🎯 What TRUE Round-Trip Tests Validate

1. **Authentication & Authorization**: Real JWT tokens, permission checks
2. **Request Validation**: Actual validation middleware
3. **Business Logic**: All server-side rules and transformations
4. **Database Constraints**: Foreign keys, unique constraints, etc.
5. **Response Formatting**: Actual API response structure
6. **Error Handling**: Real error responses and status codes
7. **Performance**: Actual request/response times

## 🚀 Benefits

1. **Confidence**: Tests validate the actual user experience
2. **Bug Detection**: Catches integration issues between layers
3. **Regression Prevention**: Changes in any layer are detected
4. **Documentation**: Tests show real API usage
5. **Performance Testing**: Measure actual response times

## ⚡ Performance Considerations

True round-trip tests are slower because they:
- Start real services
- Make network requests
- Execute real database queries

To optimize:
1. Use test categories (unit vs integration)
2. Run in parallel where possible
3. Share test data setup
4. Use database transactions for isolation

## 📚 Examples

See `/packages/ui/src/__test/fixtures/examples/true-round-trip.test.ts` for complete examples.

## ❌ Common Mistakes to Avoid

1. **Importing server modules in UI tests**
   ```typescript
   // ❌ WRONG
   import { prisma } from '@vrooli/server';
   ```

2. **Mocking API responses**
   ```typescript
   // ❌ WRONG (unless specifically testing UI error handling)
   server.use(rest.post('/api/*', mockHandler));
   ```

3. **Direct database manipulation**
   ```typescript
   // ❌ WRONG
   await prisma.user.deleteMany();
   ```

4. **Skipping authentication**
   ```typescript
   // ❌ WRONG
   const bookmark = await apiClient.post('/api/bookmark', data);
   // Should include auth token!
   ```

## ✅ Checklist for TRUE Round-Trip Tests

- [ ] Real API server is running
- [ ] Using actual HTTP requests (fetch/axios)
- [ ] Including authentication tokens
- [ ] NO direct database imports
- [ ] NO MSW mocking (except for external services)
- [ ] Testing actual response structure
- [ ] Measuring real performance
- [ ] Cleaning up through API

Remember: If your test doesn't make actual HTTP requests to a running server, it's NOT a true round-trip test!