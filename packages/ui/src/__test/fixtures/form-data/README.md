# Production-Grade Form Data Fixtures

This directory contains production-ready form data fixtures designed for comprehensive React Hook Form testing. These fixtures provide type-safe, realistic form data for testing UI components, form validation, and user interactions.

## 🎯 Key Features

- **Type Safety First**: Zero `any` types throughout, full TypeScript integration
- **React Hook Form Integration**: Direct compatibility with `useForm` and form validation
- **Real Validation**: Uses actual validation schemas from `@vrooli/shared`
- **User Interaction Simulation**: Realistic typing, form filling, and submission workflows
- **Comprehensive Scenarios**: From empty forms to complex multi-step workflows
- **Accessibility Testing**: Built-in support for form accessibility validation

## 📁 Architecture Overview

```
form-data/
├── userFormData.ts          # User registration, profile, login forms
├── teamFormData.ts          # Team creation, management, member forms  
├── projectFormData.ts       # Project creation, editing (Resources)
├── chatFormData.ts          # Chat creation, message composition
├── commentFormData.ts       # Comment creation, thread management
├── meetingFormData.ts       # Meeting scheduling, agenda management
├── bookmarkFormData.ts      # Bookmark creation (existing)
├── index.ts                 # Central exports and utilities
└── README.md                # This file
```

## 🏗️ Pattern Architecture

Each form fixture follows a consistent pattern:

```typescript
// 1. Form Data Interface
interface ObjectFormData {
    // UI-specific fields that may not exist in API
    field1: string;
    field2?: boolean;
    // Nested UI structures
    uiSpecificArray?: Array<{ ... }>;
}

// 2. Extended Form State
interface ObjectFormState {
    values: ObjectFormData;
    errors: Record<string, string>;
    touched: Record<string, boolean>;
    isDirty: boolean;
    isSubmitting: boolean;
    isValid: boolean;
    // Object-specific state
    objectState?: { ... };
}

// 3. Factory Class
export class ObjectFormDataFactory {
    // Validation schema creation
    private createSchema(): yup.ObjectSchema<ObjectFormData>
    
    // Scenario-based data generation
    createFormData(scenario: string): ObjectFormData
    
    // Form state management
    createFormState(scenario: string): ObjectFormState
    
    // React Hook Form integration
    createFormInstance(data?: Partial<ObjectFormData>): UseFormReturn<ObjectFormData>
    
    // Real validation integration
    async validateFormData(data: ObjectFormData): Promise<ValidationResult>
    
    // API transformation
    private transformToAPIInput(data: ObjectFormData): APIInput
}

// 4. Interaction Simulator
export class ObjectFormInteractionSimulator {
    // Realistic user interactions
    async simulateFormFilling(formInstance, data): Promise<void>
    async simulateTyping(formInstance, field, text): Promise<void>
    async simulateSubmission(formInstance): Promise<void>
}

// 5. Pre-configured Scenarios
export const objectFormScenarios = {
    // Static scenarios
    emptyForm: () => factory.createFormState('pristine'),
    validForm: () => factory.createFormState('valid'),
    
    // Interactive workflows
    async completeWorkflow(formInstance) { ... }
};
```

## 🚀 Quick Start

### Basic Usage

```typescript
import { 
    userFormFactory, 
    userFormSimulator, 
    userFormScenarios 
} from './form-data/index.js';

// Create form data
const registrationData = userFormFactory.createRegistrationFormData('complete');

// Create form instance
const formInstance = userFormFactory.createFormInstance('registration', registrationData);

// Validate form data
const validation = await userFormFactory.validateFormData(registrationData, 'registration');

// Simulate user interaction
await userFormSimulator.simulateRegistrationFlow(formInstance, registrationData);
```

### Testing with React Hook Form

```typescript
import { renderHook, act } from '@testing-library/react';
import { userFormFactory } from './form-data/index.js';

describe('User Registration Form', () => {
    it('should validate registration data', async () => {
        const { result } = renderHook(() => 
            userFormFactory.createFormInstance('registration')
        );
        
        // Fill form with valid data
        const validData = userFormFactory.createRegistrationFormData('complete');
        
        act(() => {
            result.current.reset(validData);
        });
        
        // Trigger validation
        const isValid = await result.current.trigger();
        expect(isValid).toBe(true);
        expect(result.current.formState.errors).toEqual({});
    });
});
```

### Component Testing

```typescript
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { userFormScenarios } from './form-data/index.js';

describe('RegistrationForm Component', () => {
    it('should complete registration workflow', async () => {
        const user = userEvent.setup();
        const onSubmit = jest.fn();
        
        render(<RegistrationForm onSubmit={onSubmit} />);
        
        // Use pre-configured scenario
        const { result } = renderHook(() => 
            userFormFactory.createFormInstance('registration')
        );
        
        // Execute complete workflow
        await userFormScenarios.completeRegistrationWorkflow(result.current);
        
        // Verify form submission
        expect(onSubmit).toHaveBeenCalledWith(
            expect.objectContaining({
                email: expect.any(String),
                password: expect.any(String),
                agreeToTerms: true
            })
        );
    });
});
```

## 📋 Available Form Types

### Core Objects

#### User Forms (`userFormData.ts`)
- **Registration**: Email signup with validation
- **Profile**: Profile editing with privacy settings  
- **Login**: Authentication with remember me

**Scenarios**: `empty`, `minimal`, `complete`, `invalid`, `weakPassword`, `passwordMismatch`

```typescript
// Registration workflow
const registrationData = userFormFactory.createRegistrationFormData('complete');
await userFormSimulator.simulateRegistrationFlow(formInstance, registrationData);

// Login workflow  
const loginData = userFormFactory.createLoginFormData('withEmail');
await userFormSimulator.simulateLoginFlow(formInstance, loginData);
```

#### Team Forms (`teamFormData.ts`)
- **Creation**: Team setup with members and settings
- **Profile**: Team profile editing
- **Member Management**: Role and permission management

**Scenarios**: `empty`, `minimal`, `complete`, `invalid`, `privateTeam`, `withInvites`, `openTeam`

```typescript
// Team creation with member invites
const teamData = teamFormFactory.createTeamFormData('withInvites');
await teamFormSimulator.simulateTeamCreationFlow(formInstance, teamData);
```

#### Project Forms (`projectFormData.ts`)
- **Creation**: Project setup with resources and tags
- **Editing**: Project updates and completion tracking

**Scenarios**: `empty`, `minimal`, `complete`, `invalid`, `privateProject`, `teamProject`, `completedProject`, `withResources`

```typescript
// Complete project with resources
const projectData = projectFormFactory.createFormData('withResources');
await projectFormSimulator.simulateProjectCreationFlow(formInstance, projectData);
```

### Communication Objects

#### Chat Forms (`chatFormData.ts`)
- **Chat Creation**: Chat setup with participants and AI settings
- **Message Composition**: Message creation with attachments
- **Settings**: Chat configuration and permissions

**Scenarios**: `empty`, `minimal`, `complete`, `invalid`, `privateChat`, `groupChat`, `aiChat`

```typescript
// AI-enabled chat creation
const chatData = chatFormFactory.createChatFormData('aiChat');
await chatFormSimulator.simulateAddingParticipants(formInstance, chatData.participants);

// Message with typing simulation
await chatFormSimulator.simulateMessageTyping(
    messageFormInstance, 
    'Hello team!', 
    { typingSpeed: 50 }
);
```

#### Comment Forms (`commentFormData.ts`)
- **Comment Creation**: Comment composition with markdown and mentions
- **Thread Management**: Reply handling and threading

**Scenarios**: `empty`, `minimal`, `complete`, `invalid`, `reply`, `withAttachments`, `withMentions`, `markdownComment`

```typescript
// Detailed code review comment
const reviewData = commentFormFactory.createFormData('markdownComment');
await commentFormSimulator.simulateCommentTyping(
    formInstance, 
    reviewData.content,
    { withPauses: true }
);

// Reply workflow
await commentFormSimulator.simulateReplyWorkflow(
    formInstance,
    'parent_comment_123',
    'Thanks for the feedback!'
);
```

### Organizational Objects

#### Meeting Forms (`meetingFormData.ts`)
- **Meeting Creation**: Scheduling with participants and agenda
- **Recurrence**: Recurring meeting setup
- **Conflict Detection**: Time slot validation

**Scenarios**: `empty`, `minimal`, `complete`, `invalid`, `recurring`, `withAgenda`, `privateTeamMeeting`, `largeMeeting`, `quickStandup`, `interview`

```typescript
// Complete meeting with agenda
const meetingData = meetingFormFactory.createFormData('withAgenda');
await meetingFormSimulator.simulateMeetingScheduling(formInstance, meetingData);

// Check availability
const availability = await meetingFormFactory.checkTimeSlotAvailability(
    startTime, 
    endTime, 
    participants
);
```

## 🎨 Advanced Usage Patterns

### Multi-Step Form Workflows

```typescript
// Complex registration with multiple steps
async function simulateCompleteOnboarding() {
    // Step 1: Basic registration
    const registrationForm = userFormFactory.createFormInstance('registration');
    const registrationData = userFormFactory.createRegistrationFormData('complete');
    await userFormSimulator.simulateRegistrationFlow(registrationForm, registrationData);
    
    // Step 2: Profile setup
    const profileForm = userFormFactory.createFormInstance('profile');
    const profileData = userFormFactory.createProfileFormData('complete');
    await userFormSimulator.simulateProfileSetup(profileForm, profileData);
    
    // Step 3: Team creation
    const teamForm = teamFormFactory.createFormInstance('create');
    const teamData = teamFormFactory.createTeamFormData('complete');
    await teamFormSimulator.simulateTeamCreationFlow(teamForm, teamData);
    
    return { registrationData, profileData, teamData };
}
```

### Error Recovery Workflows

```typescript
// Test error recovery patterns
describe('Form Error Recovery', () => {
    it('should recover from validation errors', async () => {
        const formInstance = userFormFactory.createFormInstance('registration');
        
        // Start with invalid data
        const invalidData = userFormFactory.createRegistrationFormData('invalid');
        act(() => {
            formInstance.reset(invalidData);
        });
        
        // Trigger validation to show errors
        await act(async () => {
            await formInstance.trigger();
        });
        
        expect(Object.keys(formInstance.formState.errors)).toHaveLength(4);
        
        // Simulate user fixing errors
        await userFormSimulator.simulateErrorRecovery(
            formInstance,
            'email',
            'valid@example.com'
        );
        
        expect(formInstance.formState.errors.email).toBeUndefined();
    });
});
```

### Real-Time Validation Testing

```typescript
// Test real-time field validation
describe('Real-Time Validation', () => {
    it('should validate fields as user types', async () => {
        const formInstance = userFormFactory.createFormInstance('registration');
        const validationResults = [];
        
        // Track validation state changes
        formInstance.watch((data, { name }) => {
            if (name) {
                validationResults.push({
                    field: name,
                    isValid: !formInstance.formState.errors[name],
                    value: data[name]
                });
            }
        });
        
        // Simulate typing email
        await userFormSimulator.simulateTyping(
            formInstance,
            'email',
            'user@example.com'
        );
        
        // Verify validation occurred during typing
        expect(validationResults.some(r => r.field === 'email' && r.isValid)).toBe(true);
    });
});
```

### Form Performance Testing

```typescript
// Test form performance with large datasets
describe('Form Performance', () => {
    it('should handle large participant lists efficiently', async () => {
        const participants = Array.from({ length: 100 }, (_, i) => ({
            handle: `user${i}`,
            role: 'participant' as const,
            isRequired: false
        }));
        
        const formInstance = meetingFormFactory.createFormInstance({
            inviteParticipants: participants
        });
        
        const startTime = performance.now();
        
        // Simulate form operations
        act(() => {
            formInstance.setValue('name', 'Large Meeting');
            formInstance.trigger(); // Trigger validation
        });
        
        const endTime = performance.now();
        const duration = endTime - startTime;
        
        // Performance assertion
        expect(duration).toBeLessThan(1000); // Should complete within 1 second
        expect(formInstance.formState.isValid).toBe(true);
    });
});
```

## 🔧 Validation Integration

### Real Validation Usage

All fixtures use actual validation schemas from `@vrooli/shared`:

```typescript
// Example validation integration
export class UserFormDataFactory {
    async validateFormData(formData: UserRegistrationFormData): Promise<ValidationResult> {
        try {
            // 1. Client-side validation with yup schema
            await this.createRegistrationSchema().validate(formData, { abortEarly: false });
            
            // 2. Transform to API input
            const apiInput = this.transformRegistrationToAPIInput(formData);
            
            // 3. Server-side validation with actual schemas
            await userValidation.create.validate(apiInput);
            
            return { isValid: true, apiInput };
        } catch (error) {
            return { isValid: false, errors: this.parseValidationErrors(error) };
        }
    }
}
```

### Custom Validation Rules

```typescript
// Adding custom validation for business rules
const customUserSchema = userFormFactory.createRegistrationSchema().concat(
    yup.object({
        handle: yup.string().test(
            'handle-availability',
            'Username is already taken',
            async (value) => {
                if (!value) return true;
                return await checkHandleAvailability(value);
            }
        )
    })
);
```

## 🎭 Accessibility Testing Integration

### Form Accessibility Validation

```typescript
import { axe, toHaveNoViolations } from 'jest-axe';

expect.extend(toHaveNoViolations);

describe('Form Accessibility', () => {
    it('should have no accessibility violations', async () => {
        const formData = userFormFactory.createRegistrationFormData('complete');
        
        render(<RegistrationForm initialData={formData} />);
        
        const results = await axe(document.body);
        expect(results).toHaveNoViolations();
    });
    
    it('should properly announce form errors', async () => {
        const invalidData = userFormFactory.createRegistrationFormData('invalid');
        
        render(<RegistrationForm initialData={invalidData} />);
        
        // Submit to trigger validation
        const submitButton = screen.getByRole('button', { name: /submit/i });
        await user.click(submitButton);
        
        // Check for error announcements
        const errorMessages = screen.getAllByRole('alert');
        expect(errorMessages.length).toBeGreaterThan(0);
        
        // Verify ARIA attributes
        const invalidFields = screen.getAllByAttribute('aria-invalid', 'true');
        expect(invalidFields.length).toBeGreaterThan(0);
    });
});
```

### Keyboard Navigation Testing

```typescript
describe('Keyboard Navigation', () => {
    it('should support full keyboard navigation', async () => {
        const user = userEvent.setup();
        
        render(<RegistrationForm />);
        
        // Tab through all form fields
        await user.tab(); // Email field
        expect(screen.getByLabelText(/email/i)).toHaveFocus();
        
        await user.tab(); // Password field
        expect(screen.getByLabelText(/password/i)).toHaveFocus();
        
        await user.tab(); // Confirm password field
        expect(screen.getByLabelText(/confirm password/i)).toHaveFocus();
        
        // Test form submission with Enter key
        await user.keyboard('{Enter}');
        // Verify submission behavior
    });
});
```

## 🔍 Testing Strategies

### Unit Testing

```typescript
describe('Form Data Factory', () => {
    it('should create valid form data', () => {
        const formData = userFormFactory.createRegistrationFormData('complete');
        
        expect(formData).toMatchObject({
            email: expect.stringMatching(/^[^\s@]+@[^\s@]+\.[^\s@]+$/),
            password: expect.stringMatching(/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])/),
            agreeToTerms: true,
            agreeToPrivacyPolicy: true
        });
    });
    
    it('should validate form data correctly', async () => {
        const validData = userFormFactory.createRegistrationFormData('complete');
        const validation = await userFormFactory.validateFormData(validData, 'registration');
        
        expect(validation.isValid).toBe(true);
        expect(validation.errors).toBeUndefined();
        expect(validation.apiInput).toBeDefined();
    });
});
```

### Integration Testing

```typescript
describe('Form Integration', () => {
    it('should complete end-to-end form workflow', async () => {
        const mockAPI = setupMockAPI();
        
        // Create form with realistic data
        const formData = userFormFactory.createRegistrationFormData('complete');
        
        // Render component
        render(<RegistrationForm />);
        
        // Simulate user interaction
        const formInstance = userFormFactory.createFormInstance('registration', formData);
        await userFormScenarios.completeRegistrationWorkflow(formInstance);
        
        // Verify API call
        expect(mockAPI.post).toHaveBeenCalledWith('/api/user', {
            email: formData.email,
            password: formData.password,
            handle: formData.handle,
            name: formData.name
        });
    });
});
```

### Performance Testing

```typescript
describe('Form Performance', () => {
    it('should handle rapid form updates efficiently', async () => {
        const formInstance = teamFormFactory.createFormInstance('create');
        
        // Measure performance of rapid updates
        const updates = Array.from({ length: 100 }, (_, i) => ({
            name: `Team ${i}`,
            handle: `team${i}`
        }));
        
        const startTime = performance.now();
        
        for (const update of updates) {
            act(() => {
                formInstance.setValue('name', update.name);
                formInstance.setValue('handle', update.handle);
            });
        }
        
        const endTime = performance.now();
        expect(endTime - startTime).toBeLessThan(500); // Should complete quickly
    });
});
```

## 🔧 Configuration and Customization

### Custom Form Factory

```typescript
// Extend existing factory for custom needs
export class CustomUserFormFactory extends UserFormDataFactory {
    createRegistrationFormData(scenario: string): UserRegistrationFormData & { customField?: string } {
        const baseData = super.createRegistrationFormData(scenario);
        
        if (scenario === 'withCustomData') {
            return {
                ...baseData,
                customField: 'Custom value'
            };
        }
        
        return baseData;
    }
    
    async validateCustomFormData(formData: UserRegistrationFormData & { customField?: string }): Promise<ValidationResult> {
        const baseValidation = await super.validateFormData(formData, 'registration');
        
        if (!baseValidation.isValid) {
            return baseValidation;
        }
        
        // Add custom validation
        if (formData.customField && formData.customField.length < 3) {
            return {
                isValid: false,
                errors: { customField: 'Custom field must be at least 3 characters' }
            };
        }
        
        return baseValidation;
    }
}
```

### Environment-Specific Configuration

```typescript
// Configure fixtures based on environment
const formConfig = {
    development: {
        typingSpeed: 10, // Faster typing in dev
        includeDebugData: true,
        enablePerformanceTesting: false
    },
    test: {
        typingSpeed: 0, // Instant typing in tests
        includeDebugData: false,
        enablePerformanceTesting: true
    },
    production: {
        typingSpeed: 50, // Realistic typing
        includeDebugData: false,
        enablePerformanceTesting: false
    }
};

export function createConfiguredSimulator(environment: keyof typeof formConfig) {
    const config = formConfig[environment];
    return new UserFormInteractionSimulator(config.typingSpeed);
}
```

## 🛠️ Troubleshooting

### Common Issues

#### Type Errors
**Problem**: TypeScript errors about form field types
**Solution**: Ensure you're using the correct form data interface and that all required fields are provided

```typescript
// ❌ Wrong - missing required fields
const userData: UserRegistrationFormData = {
    email: 'test@example.com'
    // Missing password, agreeToTerms, etc.
};

// ✅ Correct - use factory or provide all required fields
const userData = userFormFactory.createRegistrationFormData('minimal');
```

#### Validation Failures
**Problem**: Form validation fails unexpectedly
**Solution**: Check that validation schemas match between form and API

```typescript
// Debug validation issues
const validation = await userFormFactory.validateFormData(formData, 'registration');
if (!validation.isValid) {
    console.log('Validation errors:', validation.errors);
    // Fix specific validation issues
}
```

#### Timing Issues in Tests
**Problem**: Tests fail due to async validation timing
**Solution**: Use proper async/await patterns and waitFor utilities

```typescript
// ❌ Wrong - not waiting for validation
formInstance.setValue('email', 'test@example.com');
expect(formInstance.formState.errors.email).toBeUndefined();

// ✅ Correct - wait for validation
await act(async () => {
    formInstance.setValue('email', 'test@example.com');
});
await waitFor(() => {
    expect(formInstance.formState.errors.email).toBeUndefined();
});
```

#### Performance Issues
**Problem**: Slow form tests
**Solution**: Use faster typing speeds or skip interaction simulation for unit tests

```typescript
// Fast simulation for unit tests
const fastSimulator = new UserFormInteractionSimulator(0); // Instant typing

// Or skip simulation entirely for pure validation tests
const validation = await userFormFactory.validateFormData(formData, 'registration');
expect(validation.isValid).toBe(true);
```

### Debug Strategies

1. **Enable Debug Logging**: Add console logs to understand form state changes
2. **Use React Hook Form DevTools**: Install and use the DevTools for debugging
3. **Test Isolation**: Test form components in isolation from the rest of the app
4. **Validation Debugging**: Test validation functions separately from form components

## 📈 Best Practices

### DO's ✅

- **Use Type-Safe Fixtures**: Always use the provided TypeScript interfaces
- **Test Real User Workflows**: Use interaction simulators for realistic testing
- **Validate with Real Schemas**: Use actual validation from `@vrooli/shared`
- **Test Error Scenarios**: Include invalid data and error recovery testing
- **Check Accessibility**: Use axe-core and other a11y testing tools
- **Test Performance**: Ensure forms perform well with large datasets
- **Use Realistic Data**: Generate data that matches real user input patterns

### DON'Ts ❌

- **Don't Mock Form Validation**: Use real validation schemas
- **Don't Skip Error Testing**: Always test invalid data scenarios
- **Don't Ignore Accessibility**: Include a11y testing in your test suite
- **Don't Use Hardcoded Data**: Use factory-generated data for flexibility
- **Don't Skip Interaction Testing**: Test how users actually use forms
- **Don't Forget Edge Cases**: Test boundary conditions and unusual input

## 🚀 Future Enhancements

### Planned Additions

1. **Additional Form Types**: Resource, Routine, API, Schedule, Tag fixtures
2. **Advanced Validation**: Cross-field validation and async validation
3. **Internationalization**: Multi-language form testing support
4. **Workflow Orchestration**: Complex multi-form user journeys
5. **Visual Testing**: Screenshot comparison for form layouts
6. **Performance Benchmarks**: Automated performance testing

### Contributing

When adding new form fixtures:

1. Follow the established pattern architecture
2. Include comprehensive scenarios (empty, minimal, complete, invalid)
3. Add interaction simulators for realistic user behavior
4. Include both success and error path testing
5. Document all scenarios and usage patterns
6. Add TypeScript types for all interfaces
7. Include accessibility testing considerations

## 📚 Related Documentation

- [Fixtures Overview](../../README.md) - Overall fixture architecture
- [Testing Strategy](../../../../docs/testing/test-strategy.md) - Broader testing approach
- [React Hook Form Documentation](https://react-hook-form.com/) - Form library docs
- [Yup Validation](https://github.com/jquense/yup) - Validation schema library

---

This comprehensive form fixtures system provides everything needed for thorough, realistic, and maintainable form testing in the Vrooli application.