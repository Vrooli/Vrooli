# UI Test Fixtures Architecture

This document provides a comprehensive analysis of the UI fixtures architecture, documenting the current state, critical issues, and the ideal architecture for true integration testing in the Vrooli platform.

## Current State Analysis

### Directory Structure
```
fixtures/
├── api-responses/      # Mock API response data
├── form-data/         # Form input test data
├── helpers/           # Transformation utilities (PROBLEMATIC)
├── round-trip-tests/  # End-to-end test examples (SEVERELY FLAWED)
├── sessions/          # Session and auth fixtures
├── ui-states/         # Loading, error, and success states
├── events/            # Event emitters and mock events
└── index.ts           # Central exports
```

### Critical Issues Identified

#### 1. **Rampant Use of `any` Types**
The round-trip tests extensively use `any` types, completely defeating TypeScript's purpose:
```typescript
// Current problematic pattern found throughout
async function validateSignUpFormDataReal(formData: any): Promise<string[]>
function transformFormToCreateRequestReal(formData: any)
const updateRequest: any = { id: teamId };
```

**Impact**: Loss of type safety, IntelliSense, and compile-time error detection.

#### 2. **Global State Pollution**
Helper files use global state for test storage, violating test isolation principles:
```typescript
// Found in helpers/teamTransformations.ts
const storage = (globalThis as any).__testTeamStorage || {};
(globalThis as any).__testTeamStorage = storage;
```

**Impact**: Tests can interfere with each other, leading to flaky and order-dependent test failures.

#### 3. **Fake Round-Trip Testing**
Despite being called "round-trip tests", they don't actually touch the database:
```typescript
// Current: Mock service instead of real database
const mockUserService = {
    storage: {} as Record<string, User>,
    async signUp(data: any): Promise<User> { /* fake implementation */ }
}
```

**Impact**: Tests don't validate actual database constraints, relationships, or business logic.

#### 4. **Disconnected from UI Components**
Form data fixtures don't integrate with actual React components or form libraries:
- No integration with React Hook Form
- No testing of actual form validation
- No component state management testing
- No user interaction simulation

#### 5. **Missing MSW Integration**
While MSW is mentioned in documentation, the fixtures don't provide proper MSW handlers:
- No structured MSW handler generation
- No request/response interceptor patterns
- No network error simulation capabilities

#### 6. **Inconsistent Data Shapes**
Form data structures don't consistently map to API input shapes:
- Manual transformation logic scattered across helpers
- Duplication of shape transformation logic
- Inconsistent handling of nested relationships

## Ideal UI Fixture Architecture

### Core Principles
1. **Type Safety First**: Eliminate all `any` types
2. **True Integration**: Form → API → Database → Response → UI
3. **Test Isolation**: No global state pollution
4. **Component Integration**: Work with real UI components
5. **Network Simulation**: Proper MSW integration

### Proposed Architecture

```typescript
// ============================================
// 1. FORM FIXTURE FACTORY
// ============================================
interface FormFixtureFactory<TFormData extends Record<string, any>, TFormState = any> {
  // Create form data for different scenarios
  createFormData(scenario: ScenarioType): TFormData;
  
  // Simulate form events (for React Hook Form integration)
  simulateFormEvents(formData: TFormData): FormEvent[];
  
  // Validate using actual form validation
  validateFormData(formData: TFormData): Promise<FormValidationResult>;
  
  // Transform to API input using real shape functions
  transformToAPIInput(formData: TFormData): APICreateInput | APIUpdateInput;
  
  // Create form state for component testing
  createFormState(data: TFormData): UseFormReturn<TFormData>;
}

// Example implementation
export class UserFormFixtureFactory implements FormFixtureFactory<UserFormData> {
  createFormData(scenario: 'minimal' | 'complete' | 'invalid'): UserFormData {
    switch (scenario) {
      case 'minimal':
        return {
          email: "test@example.com",
          password: "SecurePass123!",
          handle: "testuser",
          name: "Test User",
          agreeToTerms: true
        };
      // ... other scenarios
    }
  }
  
  async validateFormData(formData: UserFormData): Promise<FormValidationResult> {
    // Use REAL validation from @vrooli/shared
    return emailSignUpFormValidation.validate(formData);
  }
  
  transformToAPIInput(formData: UserFormData): UserCreateInput {
    // Use REAL shape function from @vrooli/shared
    return shapeUser.create(formData);
  }
}

// ============================================
// 2. ROUND-TRIP ORCHESTRATOR
// ============================================
interface RoundTripOrchestrator<TFormData, TAPIResponse> {
  // Execute complete cycle with real API calls
  executeFullCycle(config: RoundTripConfig<TFormData>): Promise<RoundTripResult<TAPIResponse>>;
  
  // Test CRUD operations
  testCreateFlow(formData: TFormData): Promise<TestResult>;
  testUpdateFlow(id: string, formData: Partial<TFormData>): Promise<TestResult>;
  testDeleteFlow(id: string): Promise<TestResult>;
  
  // Verify data integrity across layers
  verifyDataIntegrity(original: TFormData, result: TAPIResponse): DataIntegrityReport;
}

export class UserRoundTripOrchestrator implements RoundTripOrchestrator<UserFormData, User> {
  constructor(
    private apiClient: TestAPIClient,
    private dbVerifier: DatabaseVerifier
  ) {}
  
  async executeFullCycle(config: RoundTripConfig<UserFormData>): Promise<RoundTripResult<User>> {
    // 1. Validate form data
    const validation = await userFormFixture.validateFormData(config.formData);
    if (!validation.isValid) {
      return { success: false, errors: validation.errors };
    }
    
    // 2. Transform to API input
    const apiInput = userFormFixture.transformToAPIInput(config.formData);
    
    // 3. Make actual API call (hits test server with testcontainers DB)
    const apiResponse = await this.apiClient.post('/api/user', apiInput);
    
    // 4. Verify database state
    const dbRecord = await this.dbVerifier.verifyUserCreated(apiResponse.data.id);
    
    // 5. Fetch via API to verify read path
    const fetchedUser = await this.apiClient.get(`/api/user/${apiResponse.data.id}`);
    
    // 6. Verify UI can display the data
    const uiData = transformAPIResponseToUIDisplay(fetchedUser.data);
    
    return {
      success: true,
      data: {
        formData: config.formData,
        apiInput,
        apiResponse: apiResponse.data,
        dbRecord,
        fetchedData: fetchedUser.data,
        uiDisplay: uiData
      }
    };
  }
}

// ============================================
// 3. UI STATE FIXTURES
// ============================================
interface UIStateFixtureFactory<TState> {
  // Create different UI states
  createLoadingState(context?: LoadingContext): TState;
  createErrorState(error: AppError): TState;
  createSuccessState(data: any, message?: string): TState;
  createEmptyState(): TState;
  
  // State transitions
  transitionToLoading(currentState: TState): TState;
  transitionToSuccess(currentState: TState, data: any): TState;
  transitionToError(currentState: TState, error: AppError): TState;
}

// ============================================
// 4. MSW HANDLER FACTORY
// ============================================
interface MSWHandlerFactory {
  // Generate handlers for different scenarios
  createSuccessHandlers(): RestHandler[];
  createErrorHandlers(errorScenarios: ErrorScenario[]): RestHandler[];
  createDelayHandlers(delays: DelayConfig): RestHandler[];
  createNetworkErrorHandlers(): RestHandler[];
  
  // Dynamic handler creation
  createCustomHandler(config: HandlerConfig): RestHandler;
}

export class UserMSWHandlerFactory implements MSWHandlerFactory {
  createSuccessHandlers(): RestHandler[] {
    return [
      rest.post('/api/user', async (req, res, ctx) => {
        const body = await req.json();
        // Use real validation
        const validation = await userValidation.create.validate(body);
        if (!validation.isValid) {
          return res(ctx.status(400), ctx.json({ errors: validation.errors }));
        }
        // Return fixture data
        return res(ctx.json(userFixtures.complete.create));
      }),
      // ... other handlers
    ];
  }
}

// ============================================
// 5. COMPONENT TEST UTILITIES
// ============================================
interface ComponentTestUtils<TProps> {
  // Render with all providers
  renderWithProviders(component: React.ComponentType<TProps>, props: TProps): RenderResult;
  
  // Simulate user interactions
  simulateFormSubmission(formData: any): Promise<void>;
  simulateFieldChange(fieldName: string, value: any): Promise<void>;
  
  // Wait for async operations
  waitForAPICall(endpoint: string): Promise<void>;
  waitForStateUpdate(predicate: () => boolean): Promise<void>;
}

// ============================================
// 6. ERROR SCENARIO TESTING
// ============================================
interface ErrorScenarioTester {
  // Test various error conditions
  testValidationErrors(invalidData: any[]): Promise<ValidationErrorReport>;
  testAPIErrors(errorResponses: APIError[]): Promise<APIErrorReport>;
  testNetworkErrors(scenarios: NetworkErrorScenario[]): Promise<NetworkErrorReport>;
  testPermissionErrors(unauthorizedActions: Action[]): Promise<PermissionErrorReport>;
}
```

### Implementation Example

```typescript
// user.round-trip.test.ts - Using the new architecture
import { UserFormFixtureFactory, UserRoundTripOrchestrator } from '../fixtures';
import { createTestAPIClient } from '../utils/testClient';
import { DatabaseVerifier } from '../utils/dbVerifier';

describe('User Registration - Full Round Trip', () => {
  let orchestrator: UserRoundTripOrchestrator;
  let formFixture: UserFormFixtureFactory;
  
  beforeAll(async () => {
    // Setup with real test database
    const apiClient = await createTestAPIClient();
    const dbVerifier = new DatabaseVerifier();
    
    orchestrator = new UserRoundTripOrchestrator(apiClient, dbVerifier);
    formFixture = new UserFormFixtureFactory();
  });
  
  it('should complete full registration cycle', async () => {
    // Create form data
    const formData = formFixture.createFormData('complete');
    
    // Execute full round trip
    const result = await orchestrator.executeFullCycle({ formData });
    
    // Verify success
    expect(result.success).toBe(true);
    expect(result.data.dbRecord).toBeDefined();
    expect(result.data.uiDisplay.name).toBe(formData.name);
    
    // Verify data integrity
    const integrity = orchestrator.verifyDataIntegrity(
      formData,
      result.data.fetchedData
    );
    expect(integrity.isValid).toBe(true);
  });
});
```

## Migration Plan

### Phase 1: Type Safety Restoration (Week 1)
1. Replace all `any` types with proper interfaces
2. Generate TypeScript types from Prisma schema
3. Create strict type definitions for all fixtures
4. Add ESLint rules to prevent `any` usage

### Phase 2: Test Isolation (Week 2)
1. Remove all global state usage
2. Implement proper test context providers
3. Add transaction-based test isolation
4. Create test cleanup utilities

### Phase 3: Real Integration (Weeks 3-4)
1. Set up testcontainers for UI tests
2. Create test API client with auth
3. Implement database verification utilities
4. Connect round-trip tests to real API endpoints

### Phase 4: Component Integration (Week 5)
1. Create form fixture factories for all objects
2. Integrate with React Hook Form
3. Add component rendering utilities
4. Implement user interaction simulators

### Phase 5: MSW Enhancement (Week 6)
1. Create MSW handler factories
2. Add network error scenarios
3. Implement request interceptors
4. Add response delay simulation

## Benefits of New Architecture

### 1. **True Integration Testing**
- Tests validate actual database constraints
- Real API endpoint testing
- Actual form validation logic
- Complete data flow verification

### 2. **Type Safety**
- Compile-time error detection
- IntelliSense support
- Refactoring confidence
- Self-documenting code

### 3. **Test Reliability**
- No test interference
- Predictable test execution
- Easy debugging
- Fast feedback loop

### 4. **Developer Experience**
- Clear fixture APIs
- Reusable test utilities
- Comprehensive error messages
- Easy test creation

### 5. **Maintainability**
- Centralized fixture management
- Single source of truth
- Easy schema updates
- Version control friendly

## Fixture Categories

### Form Data Fixtures
- Represent data as it appears in UI forms
- Include validation rules
- Support different input scenarios
- Map to API input shapes

### API Response Fixtures
- Mock API responses for MSW
- Include all response variations
- Support pagination/filtering
- Include error responses

### UI State Fixtures
- Loading states with progress
- Error states with recovery
- Success states with messages
- Empty states with actions

### Session Fixtures
- Authentication states
- Permission variations
- Team contexts
- Multi-tenant scenarios

### Event Fixtures
- WebSocket events
- Real-time updates
- Progress notifications
- Error events

## Testing Patterns

### Pattern 1: Form Submission
```typescript
// Test complete form submission flow
const formData = userFormFixture.createFormData('complete');
const result = await orchestrator.testCreateFlow(formData);
expect(result.created).toBe(true);
```

### Pattern 2: Error Handling
```typescript
// Test validation errors
const invalidData = userFormFixture.createFormData('invalid');
const errors = await errorTester.testValidationErrors([invalidData]);
expect(errors).toContainError('email_invalid');
```

### Pattern 3: Component Testing
```typescript
// Test form component with fixtures
const { getByLabelText, getByRole } = renderWithProviders(
  UserRegistrationForm,
  { onSubmit: jest.fn() }
);

await simulateFormSubmission(formData);
expect(getByRole('alert')).toHaveTextContent('Success');
```

## Best Practices

### DO's
- ✅ Use TypeScript interfaces for all fixtures
- ✅ Test with real validation schemas
- ✅ Use actual shape functions for transformations
- ✅ Isolate tests with transactions
- ✅ Verify database state in integration tests
- ✅ Use MSW for API mocking in component tests
- ✅ Test error scenarios comprehensively

### DON'Ts
- ❌ Use `any` types in fixtures
- ❌ Create global test state
- ❌ Mock internal business logic
- ❌ Skip database verification
- ❌ Hardcode test data in components
- ❌ Ignore error scenarios
- ❌ Create fixtures without TypeScript types

## Cross-Reference with Testing Documentation

This UI fixture architecture integrates with the broader testing strategy documented in:
- [Fixtures Overview](/docs/testing/fixtures-overview.md) - Now includes UI fixtures section
- [Round-Trip Testing](/docs/testing/round-trip-testing.md) - Extended for UI layer
- [Fixture Patterns](/docs/testing/fixture-patterns.md) - UI-specific patterns added
- [Test Execution](/docs/testing/test-execution.md) - UI test execution strategies

## Conclusion

The current UI fixture architecture has significant flaws that prevent true integration testing. By implementing the proposed architecture, we can achieve:

1. **Type-safe fixtures** that catch errors at compile time
2. **True round-trip testing** from UI form to database and back
3. **Isolated tests** that don't interfere with each other
4. **Component integration** that tests real user interactions
5. **Comprehensive coverage** including error scenarios

This transformation will require effort but will result in a robust, maintainable test suite that gives confidence in the entire application stack.