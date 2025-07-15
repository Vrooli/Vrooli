# Form Component Testing Guide

⚠️ **UPDATED**: This guide covers the legacy form testing system. For new tests, please use the **Mock Consolidation System**:

📖 **[NEW: Mock Consolidation Guide](./MOCK_CONSOLIDATION_GUIDE.md)** - Recommended for all new form tests

📖 **[Quick Start: createFormTestSuite](./createFormTestSuite.ts)** - Reduces boilerplate by 60%+

---

This guide explains how to use the enhanced form component testing helpers to write comprehensive, maintainable tests for form components.

## Overview

The form testing helpers provide a simple, declarative way to test form components with minimal boilerplate. Key features include:

- **One-line input tests** - Test any form input with a single function call
- **Bidirectional binding tests** - Ensure form values sync correctly with Formik
- **Common behavior tests** - Test cancel/submit/validation with simple configurations
- **Form fixture integration** - Use existing form fixtures for comprehensive testing
- **Accessibility testing** - Built-in support for testing proper labeling

## Quick Start

### Basic Setup

```typescript
import { 
    renderFormComponent, 
    commonInputTests, 
    commonFormBehaviors,
    createMockSession 
} from "../../../__test/helpers/formComponentTestHelpers.js";
import { MyFormComponent } from "./MyFormComponent.js";

describe("MyFormComponent", () => {
    const defaultProps = {
        isCreate: true,
        isOpen: true,
        display: "Dialog",
        onClose: vi.fn(),
        onCompleted: vi.fn(),
    };

    const mockSession = createMockSession();

    // ... tests
});
```

### One-Line Input Tests

Test individual inputs with a single line:

```typescript
it("should test name input", async () => {
    const { testInputElement } = renderFormComponent(MyForm, { defaultProps, mockSession });
    
    await testInputElement(commonInputTests.name("New Name"));
});

it("should test email with validation", async () => {
    const { testInputElement } = renderFormComponent(MyForm, { defaultProps, mockSession });
    
    await testInputElement(commonInputTests.email("test@example.com"));
});
```

### Test Multiple Inputs at Once

```typescript
it("should test all form inputs", async () => {
    const { testInputElements } = renderFormComponent(MyForm, { defaultProps, mockSession });
    
    await testInputElements([
        commonInputTests.name("Test Name"),
        commonInputTests.description("Test Description"),
        commonInputTests.version("2.0.0"),
        commonInputTests.privacy(true),
        commonInputTests.language("es"),
    ]);
});
```

### Custom Input Tests

For inputs not covered by common tests:

```typescript
const customInput: InputElementTest = {
    selector: { testId: 'custom-input' }, // or { label: 'Custom Field' }
    initialValue: "initial",
    newValue: "updated",
    formikField: 'customField',
    inputType: 'text',
    triggersValidation: true,
    expectedError: 'Custom field must be unique',
};

await testInputElement(customInput);
```

## Testing Form Behaviors

### Cancel and Submit

```typescript
it("should handle form submission", async () => {
    const { testFormBehaviors } = renderFormComponent(MyForm, { defaultProps, mockSession });
    
    await testFormBehaviors({
        testCancel: {
            buttonSelector: { testId: 'cancel-button' },
            expectedCallback: 'onClose',
        },
        testSubmit: {
            buttonSelector: { testId: 'submit-button' },
            expectedCallback: 'onCompleted',
            validFormData: {
                name: "Valid Name",
                email: "valid@example.com",
            },
        },
    });
});
```

### Validation Testing

```typescript
it("should validate required fields", async () => {
    const { testFormBehaviors } = renderFormComponent(MyForm, { defaultProps, mockSession });
    
    await testFormBehaviors({
        testValidation: {
            requiredFields: ['name', 'email'],
            customValidations: [
                {
                    field: 'email',
                    invalidValue: 'not-an-email',
                    expectedError: 'Please enter a valid email',
                },
                {
                    field: 'age',
                    invalidValue: -5,
                    expectedError: 'Age must be positive',
                },
            ],
        },
    });
});
```

### Display Modes and States

```typescript
it("should test all form states", async () => {
    const { testFormBehaviors } = renderFormComponent(MyForm, { defaultProps, mockSession });
    
    await testFormBehaviors({
        testDisplayModes: {
            dialog: true,
            page: true,
        },
        testFormStates: {
            create: true,
            update: true,
            loading: true,
            disabled: true,
        },
    });
});
```

## Bidirectional Data Binding

Test that form inputs properly sync with Formik:

```typescript
it("should test field binding", async () => {
    const { testFieldBinding } = renderFormComponent(MyForm, { defaultProps, mockSession });
    
    // Test that updating input updates formik values
    await testFieldBinding('name', 'Updated Name', 'text');
    await testFieldBinding('isActive', true, 'checkbox');
});

it("should extract form values", async () => {
    const { fillForm, getFormValues } = renderFormComponent(MyForm, { defaultProps, mockSession });
    
    const testData = {
        name: "Test",
        email: "test@example.com",
        isActive: true,
    };
    
    await fillForm(testData);
    const values = getFormValues();
    expect(values).toMatchObject(testData);
});
```

## Using Form Fixtures

Integrate with existing form fixtures from `packages/ui/src/__test/fixtures/form-testing/`:

```typescript
import { createFormTestWithFixtures } from "../../../__test/helpers/formComponentTestHelpers.js";
import { myFormTestConfig } from "../../../__test/fixtures/form-testing/MyFormTest.js";

describe("Form Fixture Integration", () => {
    const fixtureHelper = createFormTestWithFixtures(
        MyFormComponent,
        myFormTestConfig,
        defaultProps
    );

    it("should test with minimal fixture", async () => {
        await fixtureHelper.testWithFixture('minimal');
    });

    it("should test all fixtures", async () => {
        await fixtureHelper.testAllFixtures();
    });

    it("should test specific scenario", async () => {
        await fixtureHelper.testScenario('validationScenario');
    });
});
```

## Complete Example

Here's a complete test suite for a form component:

```typescript
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
    renderFormComponent,
    commonInputTests,
    commonFormBehaviors,
    createMockSession,
    createFormTestWithFixtures,
} from "../../../__test/helpers/formComponentTestHelpers.js";
import { userFormTestConfig } from "../../../__test/fixtures/form-testing/UserFormTest.js";
import { UserFormComponent } from "./UserFormComponent.js";

describe("UserFormComponent", () => {
    const defaultProps = {
        isCreate: true,
        isOpen: true,
        display: "Dialog",
        onClose: vi.fn(),
        onCompleted: vi.fn(),
        onDeleted: vi.fn(),
    };

    const mockSession = createMockSession();

    beforeEach(() => {
        vi.clearAllMocks();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Input Elements", () => {
        it("should test all inputs in one go", async () => {
            const { testInputElements } = renderFormComponent(
                UserFormComponent,
                { defaultProps, mockSession }
            );

            await testInputElements([
                commonInputTests.name("John Doe"),
                commonInputTests.email("john@example.com"),
                commonInputTests.description("A test user"),
                commonInputTests.privacy(true),
            ]);
        });
    });

    describe("Form Behaviors", () => {
        it("should handle all standard behaviors", async () => {
            const { testFormBehaviors } = renderFormComponent(
                UserFormComponent,
                { defaultProps, mockSession }
            );

            await testFormBehaviors(commonFormBehaviors.complete(['name', 'email']));
        });
    });

    describe("Complete Form Flow", () => {
        it("should fill and submit form", async () => {
            const { fillForm, findElement, user } = renderFormComponent(
                UserFormComponent,
                { defaultProps, mockSession }
            );

            await fillForm({
                name: "John Doe",
                email: "john@example.com",
                bio: "Software developer",
                isActive: true,
            });

            const submitButton = findElement({ testId: 'submit-button' });
            await user.click(submitButton);

            expect(defaultProps.onCompleted).toHaveBeenCalled();
        });
    });

    describe("Fixture Integration", () => {
        const fixtureHelper = createFormTestWithFixtures(
            UserFormComponent,
            userFormTestConfig,
            defaultProps
        );

        it("should test all fixtures", async () => {
            await fixtureHelper.testAllFixtures();
        });
    });
});
```

## Common Input Test Reference

### Built-in Input Tests

- `commonInputTests.name(value, initialValue?)` - Text input for name fields
- `commonInputTests.description(value, initialValue?)` - Textarea for descriptions
- `commonInputTests.version(value, initialValue?)` - Version input fields
- `commonInputTests.privacy(value, initialValue?)` - Privacy toggle switches
- `commonInputTests.language(value, initialValue?)` - Language selectors
- `commonInputTests.code(fieldName, value, initialValue?)` - Code editor inputs
- `commonInputTests.email(value, initialValue?)` - Email inputs with validation
- `commonInputTests.url(value, initialValue?)` - URL inputs with validation

### Custom Input Configuration

```typescript
interface InputElementTest {
    selector: ElementSelector;        // How to find the element
    initialValue?: any;              // Expected initial value
    newValue: any;                   // Value to change to
    formikField: string;             // Formik field name
    inputType?: 'text' | 'checkbox' | 'select' | 'textarea' | 'number' | 'radio';
    triggersValidation?: boolean;     // Whether to test validation
    expectedError?: string;          // Expected validation error
    customInteraction?: Function;    // Custom interaction logic
}
```

## Common Behavior Test Reference

- `commonFormBehaviors.standard()` - Basic cancel/submit tests
- `commonFormBehaviors.dialog()` - Dialog-specific behaviors
- `commonFormBehaviors.page()` - Page-specific behaviors
- `commonFormBehaviors.complete(requiredFields)` - All behaviors with validation

## Advanced Features

### Custom Element Selection

```typescript
const element = findElement({
    testId: 'my-element',        // By test ID
    role: 'textbox',             // By ARIA role
    label: 'Email Address',      // By label text
    placeholder: 'Enter email',  // By placeholder
    name: 'email',              // By name attribute
    selector: () => {...}       // Custom selector function
});
```

### Error Handling

```typescript
// Get specific field error
const emailError = getFormError('email');

// Assert multiple errors
await assertFormErrors({
    name: 'Name is required',
    email: 'Invalid email format',
});
```

### Accessibility Testing

```typescript
it("should have accessible form inputs", async () => {
    const { findElement } = renderFormComponent(MyForm, { defaultProps, mockSession });
    
    // Inputs should be findable by label
    const nameByLabel = findElement({ label: "Full Name" });
    expect(nameByLabel).toBeInTheDocument();
    expect(nameByLabel).toHaveAttribute('id');
});
```

## Best Practices

1. **Use one-line tests for simple cases** - Keep tests concise and readable
2. **Group related tests** - Use describe blocks to organize tests logically
3. **Test the happy path first** - Ensure basic functionality works before edge cases
4. **Use fixtures for complex data** - Leverage existing fixtures for comprehensive testing
5. **Test accessibility** - Ensure all inputs have proper labels and ARIA attributes
6. **Mock heavy dependencies** - Keep tests fast by mocking complex components
7. **Clear mocks between tests** - Use beforeEach/afterEach for clean test isolation

## Troubleshooting

### Element Not Found

If `findElement` fails, check:
- Component renders the expected test IDs
- Mocked components include proper test IDs
- Element is visible (not conditionally hidden)
- Using correct selector strategy

### Formik Values Not Updating

Ensure:
- Input has proper `name` attribute
- Formik field name matches exactly
- Component properly connects to Formik context
- Using correct input type in tests

### Validation Not Triggering

Check:
- Form has proper validation schema
- Submit button triggers form submission
- Validation errors render with expected text
- Using `testFormBehaviors` with validation config

## Migration Guide

If updating existing tests:

1. Replace manual render/userEvent setup with `renderFormComponent`
2. Convert individual input tests to use `testInputElement`
3. Replace form submission logic with `testFormBehaviors`
4. Use `createFormTestWithFixtures` for fixture-based tests
5. Remove redundant test utilities and helpers