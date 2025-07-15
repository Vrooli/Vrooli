# Comprehensive Form Testing Plan

## Overview

This document outlines a plan to implement comprehensive testing for the form infrastructure, including the newly migrated forms using `useStandardUpsertForm` and `useStandardBatchUpsertForm` hooks. The codebase already has excellent testing infrastructure that needs to be extended to cover the standardized form patterns.

## Current Testing Infrastructure (Strengths)

### 🎯 What We Have

#### 1. UI Form Testing (Component Level)
- **Location**: `packages/ui/src/__test/fixtures/form-testing/`
- **Purpose**: Test form logic, validation, and user interactions **without API calls**
- **Tools**: `UIFormTestFactory` + Formik helpers + React Testing Library
- **Status**: ✅ **Excellent foundation in place**

#### 2. Integration Testing (Full Stack)  
- **Location**: `packages/integration/` (mentioned but separate)
- **Purpose**: Real API calls + database persistence with testcontainers
- **Scope**: End-to-end form validation from UI → API → Database
- **Status**: ✅ **Infrastructure exists**

#### 3. Component Testing (Individual Inputs)
- **Location**: Throughout `packages/ui/src/components/inputs/`  
- **Purpose**: Individual input component behavior and Formik integration
- **Status**: ✅ **Well established**

### 🔧 Available Testing Tools

#### A. Form Test Helpers (`formTestHelpers.tsx`)
```typescript
// Comprehensive Formik integration
const { user, getFormValues, onSubmit, submitForm } = renderWithFormik(
    <MyForm />,
    {
        initialValues: { name: "", email: "" },
        formikConfig: {
            validationSchema: yup.object({
                name: testValidationSchemas.requiredString("Name"),
                email: testValidationSchemas.email(),
            }),
        },
    }
);

// Test user interactions
await formInteractions.fillField(user, "Name", "John");
await formInteractions.submitByButton(user, "Submit");

// Verify results  
formAssertions.expectFormSubmitted(onSubmit, { name: "John", email: "..." });
```

#### B. UI Form Test Factory (Advanced)
```typescript
// Comprehensive form testing without API calls
const myFormTestFactory = createUIFormTestFactory({
    objectType: "MyObject",
    validation: myObjectValidation,
    transformFunction: transformMyObjectValues,
    formFixtures: {
        minimal: { name: "Test", isPrivate: false },
        complete: { name: "Complete", description: "Full" },
        invalid: { name: "", isPrivate: false },
        edgeCase: { name: "A".repeat(200) },
    },
});

// Test validation logic
const result = await myFormTestFactory.testFormValidation("complete");
expect(result.passed).toBe(true);

// Test user interactions
const interaction = await myFormTestFactory.testUserInteraction("complete", { user });
expect(interaction.success).toBe(true);
```

#### C. MSW Integration for Component Tests
```typescript
// Mock API responses for component testing
const mswHandlers = myFormTestFactory.createMSWHandlers();
server.use(
    rest.post('/api/myobject', mswHandlers.create),
    rest.put('/api/myobject/:id', mswHandlers.update)
);
```

## Current Testing Gaps (What We Need)

### ❌ 1. Hook Testing Gap (HIGH PRIORITY)
**MISSING**: Direct unit tests for `useStandardUpsertForm` and `useStandardBatchUpsertForm`

Currently the hooks are only tested indirectly through component tests, but they need dedicated unit tests for:
- Hook state management
- Error handling scenarios  
- Edge cases (network failures, validation errors)
- Hook lifecycle (mount, unmount, cleanup)
- isMutate pattern behavior
- Translation field management
- Cache handling

### ❌ 2. Behavioral Form Testing Gap (HIGH PRIORITY)
**MISSING**: Comprehensive behavioral tests for forms using the standard hooks

Need tests that verify:
- Initial values appear correctly
- Form elements update Formik values
- Button interactions work properly
- Validation errors display correctly
- Form submission flows
- Translation switching
- AutoFill integration (where applicable)

### ❌ 3. Test Factory Coverage Gap (MEDIUM PRIORITY)
**MISSING**: Test factories for newly migrated forms

## Implementation Plan

### 📋 Phase 1: Hook Unit Tests (HIGH PRIORITY)

#### **Task 1.1: useStandardUpsertForm.test.ts**

**Create**: `packages/ui/src/hooks/useStandardUpsertForm.test.ts`

**Test Coverage:**
```typescript
import { renderHook, act } from '@testing-library/react';
import { useStandardUpsertForm } from './useStandardUpsertForm.js';

describe('useStandardUpsertForm', () => {
    // Basic functionality
    it('should initialize with correct default state', () => {
        const { result } = renderHook(() => useStandardUpsertForm(config, props));
        
        expect(result.current.isLoading).toBe(false);
        expect(result.current.session).toBeDefined();
        expect(typeof result.current.onSubmit).toBe('function');
    });

    // isMutate pattern testing
    it('should handle isMutate=false correctly', async () => {
        const mockOnCompleted = vi.fn();
        const { result } = renderHook(() => useStandardUpsertForm(config, {
            ...props,
            isMutate: false,
            onCompleted: mockOnCompleted,
        }));

        await act(async () => {
            result.current.onSubmit();
        });

        expect(mockOnCompleted).toHaveBeenCalledWith(expect.objectContaining({
            createdAt: expect.any(String),
            updatedAt: expect.any(String),
        }));
    });

    // Error handling
    it('should handle validation errors properly', async () => {
        // Test validation error scenarios
    });

    it('should handle network errors gracefully', async () => {
        // Test network failure scenarios
    });

    // Resource cleanup
    it('should clean up resources on unmount', () => {
        // Test cleanup logic
    });

    // Translation support
    it('should manage translation fields correctly', () => {
        // Test translation field management
    });

    // Cache management
    it('should handle form caching properly', () => {
        // Test cache save/restore logic
    });
});
```

#### **Task 1.2: useStandardBatchUpsertForm.test.ts**

**Create**: `packages/ui/src/hooks/useStandardBatchUpsertForm.test.ts`

**Test Coverage:**
- Array handling logic
- Batch validation
- Array transformation
- Batch submission
- Error handling for partial failures

### 📋 Phase 2: Behavioral Form Tests (HIGH PRIORITY)

#### **Task 2.1: Test Each Migrated Form**

**Forms to Test (12 total):**
1. `TeamUpsert.test.tsx` ✅ (has standard hook)
2. `ReportUpsert.test.tsx` ✅ (has standard hook)
3. `ApiUpsert.test.tsx` ✅ (has standard hook)
4. `CommentUpsert.test.tsx` ✅ (has standard hook)
5. `DataStructureUpsert.test.tsx` ✅ (has standard hook)
6. `ResourceUpsert.test.tsx` ✅ (has isMutate)
7. `DataConverterUpsert.test.tsx` ✅ (has isMutate)
8. `SmartContractUpsert.test.tsx` ✅ (has isMutate)
9. `PromptUpsert.test.tsx` ✅ (has isMutate)
10. `MemberInvitesUpsert.test.tsx` ✅ (has batch hook)
11. `ChatInvitesUpsert.test.tsx` ✅ (has batch hook)

**Test Pattern for Each Form:**
```typescript
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TeamUpsert } from './TeamUpsert.js';

describe('TeamUpsert Form Behavior', () => {
    it('should display initial values correctly', () => {
        render(<TeamUpsert isCreate={true} isOpen={true} />);
        
        // Check that form elements appear
        expect(screen.getByLabelText(/name/i)).toBeInTheDocument();
        expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /submit/i })).toBeInTheDocument();
    });

    it('should update formik values when user types', async () => {
        const user = userEvent.setup();
        render(<TeamUpsert isCreate={true} isOpen={true} />);
        
        const nameInput = screen.getByLabelText(/name/i);
        await user.type(nameInput, 'Test Team');
        
        expect(nameInput).toHaveValue('Test Team');
    });

    it('should validate required fields', async () => {
        const user = userEvent.setup();
        render(<TeamUpsert isCreate={true} isOpen={true} />);
        
        // Try to submit without filling required fields
        await user.click(screen.getByRole('button', { name: /submit/i }));
        
        expect(screen.getByText(/name is required/i)).toBeInTheDocument();
    });

    it('should handle form submission correctly', async () => {
        const mockOnCompleted = vi.fn();
        const user = userEvent.setup();
        
        render(
            <TeamUpsert 
                isCreate={true} 
                isOpen={true} 
                onCompleted={mockOnCompleted}
            />
        );
        
        // Fill form
        await user.type(screen.getByLabelText(/name/i), 'Test Team');
        await user.type(screen.getByLabelText(/description/i), 'Test Description');
        
        // Submit
        await user.click(screen.getByRole('button', { name: /submit/i }));
        
        expect(mockOnCompleted).toHaveBeenCalled();
    });

    // For isMutate forms
    it('should handle non-mutating submission (isMutate=false)', async () => {
        const mockOnCompleted = vi.fn();
        render(
            <ResourceUpsert 
                isCreate={true} 
                isMutate={false}
                isOpen={true} 
                onCompleted={mockOnCompleted}
            />
        );
        
        // Fill and submit form
        // Verify onCompleted called with form data, not API response
    });

    // For translation forms
    it('should handle language switching correctly', async () => {
        // Test translation field behavior
    });

    // For AutoFill forms
    it('should handle AutoFill integration', async () => {
        // Test AutoFill button and functionality
    });
});
```

### 📋 Phase 3: Complete Test Factory Coverage (MEDIUM PRIORITY)

#### **Task 3.1: Create Missing Test Factories**

**Current Status:**
- ✅ **DataStructureFormTest.ts** (exists)
- ✅ **ApiFormTest.ts** (exists) 
- ✅ **TeamFormTest.ts** (exists)
- ✅ **ReportFormTest.ts** (exists)
- ✅ **CommentFormTest.ts** (exists)

**Need to Create:**
- ❌ **ResourceFormTest.ts** 
- ❌ **DataConverterFormTest.ts** 
- ❌ **SmartContractFormTest.ts** 
- ❌ **PromptFormTest.ts** 
- ❌ **MemberInvitesFormTest.ts** (batch pattern)
- ❌ **ChatInvitesFormTest.ts** (batch pattern)

**Pattern for Each Test Factory:**
```typescript
// packages/ui/src/__test/fixtures/form-testing/examples/ResourceFormTest.ts
import { 
    resourceVersionValidation, 
    resourceVersionTranslationValidation,
    endpointsResource,
    type ResourceVersion,
    type ResourceVersionCreateInput,
    type ResourceVersionUpdateInput,
    type ResourceVersionShape,
} from "@vrooli/shared";
import { resourceInitialValues, transformResourceValues } from "../../../../views/objects/resource/ResourceUpsert.js";
import { createUIFormTestFactory, type UIFormTestConfig } from "../UIFormTestFactory.js";

interface ResourceFormData {
    name: string;
    description: string;
    link: string;
    usedFor: string;
    isPrivate: boolean;
    language: string;
}

const resourceFormFixtures = {
    minimal: {
        name: "Test Resource",
        description: "A test resource",
        link: "https://example.com",
        usedFor: "Context",
        isPrivate: false,
        language: "en",
    },
    complete: {
        // Complete fixture
    },
    invalid: {
        // Invalid fixture (empty required fields)
    },
    edgeCase: {
        // Edge case fixture (long values, special characters)
    }
};

function formToShape(formData: ResourceFormData): ResourceVersionShape {
    // Convert form data to shape format
}

const resourceTestConfig: UIFormTestConfig<
    ResourceFormData,
    ResourceVersionShape,
    ResourceVersionCreateInput,
    ResourceVersionUpdateInput,
    ResourceVersion
> = {
    objectType: "Resource",
    validation: resourceVersionValidation,
    translationValidation: resourceVersionTranslationValidation,
    transformFunction: transformResourceValues,
    endpoints: {
        create: endpointsResource.createOne,
        update: endpointsResource.updateOne,
    },
    initialValuesFunction: resourceInitialValues,
    formFixtures: resourceFormFixtures,
    formToShape,
};

export const resourceFormTestFactory = createUIFormTestFactory(resourceTestConfig);
export const resourceTestCases = resourceFormTestFactory.generateUITestCases();
```

#### **Task 3.2: Create Batch Form Test Pattern**

**Special handling for batch forms** (MemberInvites, ChatInvites):
```typescript
// Different pattern for array-based forms
interface MemberInvitesFormData {
    invites: Array<{
        userEmail: string;
        willBeAdmin: boolean;
        message: string;
    }>;
}

const memberInvitesFormFixtures = {
    minimal: {
        invites: [
            { userEmail: "test@example.com", willBeAdmin: false, message: "" }
        ]
    },
    complete: {
        invites: [
            { userEmail: "user1@example.com", willBeAdmin: true, message: "Welcome!" },
            { userEmail: "user2@example.com", willBeAdmin: false, message: "Join us!" }
        ]
    },
    // ...
};
```

### 📋 Phase 4: Integration Tests (LOWER PRIORITY)

#### **Task 4.1: Create Integration Test Suite**

**Create**: `packages/integration/src/forms/` (if doesn't exist)

**Tests for:**
- Form data flows correctly through entire stack
- Database persistence works correctly
- Cache invalidation happens properly
- API errors are handled gracefully
- Performance under load

### 🎯 Implementation Priorities

#### **Phase 1: Hook Unit Tests** (Week 1)
**Priority**: 🔥 **HIGH** - Critical foundation
1. `useStandardUpsertForm.test.ts`
2. `useStandardBatchUpsertForm.test.ts`
3. Cover all hook methods, error cases, edge scenarios

#### **Phase 2: Behavioral Form Tests** (Week 2-3)
**Priority**: 🔥 **HIGH** - Ensure form reliability
1. Create behavioral tests for all 11 migrated forms
2. Test initial values, user interactions, validation, submission
3. Special testing for isMutate and batch patterns

#### **Phase 3: Test Factory Coverage** (Week 4)
**Priority**: 🟡 **MEDIUM** - Standardize testing approach
1. Create 6 missing test factories
2. Ensure consistent testing patterns across all forms

#### **Phase 4: Integration Tests** (Week 5+)
**Priority**: 🟢 **LOWER** - Nice to have
1. End-to-end form testing with real APIs
2. Performance and load testing

## Success Metrics

### 📊 Testing Coverage Goals

1. **Hook Coverage**: 100% line coverage for both standard hooks
2. **Form Coverage**: Behavioral tests for all 11 migrated forms
3. **Test Factory Coverage**: Factories for all migrated forms
4. **Integration Coverage**: Basic end-to-end tests for standard patterns

### 📈 Quality Metrics

1. **Fast Feedback**: UI tests complete in <5 seconds
2. **Reliable Tests**: <1% flaky test rate
3. **Maintainable**: Centralized testing patterns
4. **Realistic**: Uses real validation and transform functions

## Benefits

### 🏆 For Developers
- **Confidence**: Comprehensive test coverage for form changes
- **Fast Development**: Standard testing patterns for new forms
- **Early Detection**: Catch form issues before they reach users
- **Documentation**: Tests serve as usage examples

### 🏆 For Users
- **Reliability**: Forms work consistently across the application
- **Performance**: Well-tested forms perform optimally
- **Accessibility**: Tests ensure forms work for all users

### 🏆 For Maintenance
- **Centralized Testing**: Standard patterns make tests easy to maintain
- **Regression Prevention**: Comprehensive coverage prevents bugs
- **Refactoring Safety**: Tests enable confident code changes

## Files to Create/Modify

### New Test Files (Priority Order)
1. `packages/ui/src/hooks/useStandardUpsertForm.test.ts`
2. `packages/ui/src/hooks/useStandardBatchUpsertForm.test.ts`
3. `packages/ui/src/views/objects/team/TeamUpsert.test.tsx`
4. `packages/ui/src/views/objects/resource/ResourceUpsert.test.tsx`
5. `packages/ui/src/views/objects/dataConverter/DataConverterUpsert.test.tsx`
6. `packages/ui/src/views/objects/smartContract/SmartContractUpsert.test.tsx`
7. `packages/ui/src/views/objects/prompt/PromptUpsert.test.tsx`
8. `packages/ui/src/views/objects/memberInvite/MemberInvitesUpsert.test.tsx`
9. `packages/ui/src/views/objects/chatInvite/ChatInvitesUpsert.test.tsx`
10. (Continue for all migrated forms)

### New Test Factory Files
1. `packages/ui/src/__test/fixtures/form-testing/examples/ResourceFormTest.ts`
2. `packages/ui/src/__test/fixtures/form-testing/examples/DataConverterFormTest.ts`
3. `packages/ui/src/__test/fixtures/form-testing/examples/SmartContractFormTest.ts`
4. `packages/ui/src/__test/fixtures/form-testing/examples/PromptFormTest.ts`
5. `packages/ui/src/__test/fixtures/form-testing/examples/MemberInvitesFormTest.ts`
6. `packages/ui/src/__test/fixtures/form-testing/examples/ChatInvitesFormTest.ts`

## Related Documentation

- [Form Testing Infrastructure README](/packages/ui/src/__test/fixtures/form-testing/README.md)
- [Fixtures Overview](/docs/testing/fixtures-overview.md)
- [TRUE Round-Trip Testing](/docs/testing/TRUE-ROUND-TRIP-TESTING.md)
- [Form Migration Guide](/packages/ui/src/__test/fixtures/form-testing/MIGRATION_GUIDE.md)
- [Special Forms Documentation](/packages/ui/src/__test/fixtures/form-testing/SPECIAL_FORMS.md)

---

**Status**: 🚧 **In Progress** - Phases 1-3 partially completed

### Current Implementation Status (Updated 2025-06-24)

#### ✅ Phase 1: Hook Unit Tests - **COMPLETED**
- ✅ `useStandardUpsertForm.test.ts` 
- ✅ `useStandardBatchUpsertForm.test.ts`

#### ✅ Phase 2: Behavioral Form Tests - **10/10 COMPLETED** 
- ✅ `TeamUpsert.test.tsx` 
- ✅ `ResourceUpsert.test.tsx`
- ✅ `ReportUpsert.test.tsx` **CREATED 2025-06-24**
- ⏭️ `ApiUpsert.test.tsx` **SKIPPED** (not migrated to useStandardUpsertForm yet)
- ✅ `CommentUpsert.test.tsx` **CREATED 2025-06-24**
- ✅ `DataStructureUpsert.test.tsx` **CREATED 2025-06-24**
- ✅ `DataConverterUpsert.test.tsx` **CREATED 2025-06-24**
- ✅ `SmartContractUpsert.test.tsx` **CREATED 2025-06-24**
- ✅ `PromptUpsert.test.tsx` **CREATED 2025-06-24**
- ✅ `MemberInvitesUpsert.test.tsx` **CREATED 2025-06-24** (batch form)
- ✅ `ChatInvitesUpsert.test.tsx` **CREATED 2025-06-24** (batch form)

#### ✅ Phase 3: Test Factory Coverage - **11/11 COMPLETED**
- ✅ `DataStructureFormTest.ts`
- ✅ `TeamFormTest.ts`
- ✅ `ReportFormTest.ts` 
- ✅ `ApiFormTest.ts`
- ✅ `CommentFormTest.ts`
- ✅ `ResourceFormTest.ts` **CREATED 2025-06-24**
- ✅ `DataConverterFormTest.ts` **CREATED 2025-06-24**
- ✅ `SmartContractFormTest.ts` **CREATED 2025-06-24**
- ✅ `PromptFormTest.ts` **CREATED 2025-06-24**
- ✅ `MemberInvitesFormTest.ts` **CREATED 2025-06-24** (batch form)
- ✅ `ChatInvitesFormTest.ts` **CREATED 2025-06-24** (batch form)

#### ❌ Phase 4: Integration Tests - **NOT STARTED**

### **CURRENT STATUS**: Phases 1-3 Complete! 🎉🎉

**✅ MAJOR MILESTONE**: All essential form testing infrastructure is now in place!

**NEXT PRIORITY**: Phase 4 (Integration Tests) - Optional enhancement for full-stack testing

**Estimated Effort**: Phase 4 is lower priority and can be done as needed
**Team Impact**: High value for development velocity and code quality