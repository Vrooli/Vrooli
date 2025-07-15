import { act, render, screen, type RenderResult } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { Form, Formik, type FormikConfig, type FormikProps } from "formik";
import React, { type ReactNode } from "react";
import { vi } from "vitest";
import * as yup from "yup";

/**
 * Configuration for rendering a component with Formik context
 */
export interface FormTestConfig<T = any> {
    /** Initial form values */
    initialValues: T;
    /** Formik configuration (validation, onSubmit, etc.) */
    formikConfig?: Partial<FormikConfig<T>>;
    /** Whether to wrap in a <Form> component (default: true) */
    wrapInForm?: boolean;
    /** Custom render function that receives formik props */
    customRender?: (formikProps: FormikProps<T>) => ReactNode;
}

/**
 * Result of rendering a form with test helpers
 */
export interface FormTestResult<T = any> extends RenderResult {
    /** User event utilities pre-configured */
    user: ReturnType<typeof userEvent.setup>;
    /** Mock function for form submission */
    onSubmit: ReturnType<typeof vi.fn>;
    /** Get current form values */
    getFormValues: () => T;
    /** Submit the form programmatically */
    submitForm: () => Promise<void>;
    /** Reset the form to initial values */
    resetForm: () => void;
    /** Set form field value programmatically */
    setFieldValue: (field: keyof T, value: any) => Promise<void>;
    /** Get form field error */
    getFieldError: (field: keyof T) => string | undefined;
    /** Check if field has been touched */
    isFieldTouched: (field: keyof T) => boolean;
}

/**
 * Renders a component wrapped in Formik with test utilities
 */
export function renderWithFormik<T = any>(
    component: ReactNode,
    config: FormTestConfig<T>,
): FormTestResult<T> {
    const {
        initialValues,
        formikConfig = {},
        wrapInForm = true,
        customRender,
    } = config;

    const onSubmit = vi.fn();
    const user = userEvent.setup();
    
    let formikRef: FormikProps<T> | null = null;

    const TestComponent = () => (
        <Formik
            initialValues={initialValues}
            onSubmit={onSubmit}
            {...formikConfig}
        >
            {(formikProps) => {
                formikRef = formikProps;
                
                if (customRender) {
                    return customRender(formikProps);
                }
                
                if (wrapInForm) {
                    return (
                        <Form>
                            {component}
                            <button type="submit" data-testid="form-submit-button">Submit</button>
                        </Form>
                    );
                }
                
                return <>{component}</>;
            }}
        </Formik>
    );

    const renderResult = render(<TestComponent />);

    return {
        ...renderResult,
        user,
        onSubmit,
        getFormValues: () => formikRef?.values as T,
        submitForm: async () => {
            await act(async () => {
                await formikRef?.submitForm();
            });
        },
        resetForm: () => {
            act(() => {
                formikRef?.resetForm();
            });
        },
        setFieldValue: async (field: keyof T, value: any) => {
            await act(async () => {
                await formikRef?.setFieldValue(field as string, value);
            });
        },
        getFieldError: (field: keyof T) => {
            const errors = formikRef?.errors;
            return errors?.[field] as string | undefined;
        },
        isFieldTouched: (field: keyof T) => {
            const touched = formikRef?.touched;
            return Boolean(touched?.[field]);
        },
    };
}

/**
 * Common validation schemas for testing
 */
export const testValidationSchemas = {
    /** Basic required string field */
    requiredString: (fieldName: string, message?: string) =>
        yup.string().required(message || `${fieldName} is required`),
    
    /** Email validation */
    email: (message?: string) =>
        yup.string().email(message || "Invalid email").required("Email is required"),
    
    /** Password with minimum length */
    password: (minLength = 8, message?: string) =>
        yup.string().min(minLength, message || `Password must be at least ${minLength} characters`).required("Password is required"),
    
    /** Username with alphanumeric validation */
    username: (minLength = 3, maxLength = 16) =>
        yup.string()
            .min(minLength, `Username must be at least ${minLength} characters`)
            .max(maxLength, `Username must be no more than ${maxLength} characters`)
            .matches(/^[a-zA-Z0-9_]+$/, "Username can only contain letters, numbers, and underscores")
            .required("Username is required"),
    
    /** URL validation */
    url: (message?: string) =>
        yup.string().url(message || "Invalid URL").required("URL is required"),
    
    /** Number with min/max */
    number: (min?: number, max?: number) => {
        let schema = yup.number().required("This field is required");
        if (min !== undefined) schema = schema.min(min, `Must be at least ${min}`);
        if (max !== undefined) schema = schema.max(max, `Must be no more than ${max}`);
        return schema;
    },
};

/**
 * Helper functions for common form interactions
 */
export const formInteractions = {
    /** Fill out a text input field */
    fillField: async (
        user: ReturnType<typeof userEvent.setup>,
        fieldName: string,
        value: string,
    ) => {
        const input = formInteractions.findInputByLabel(fieldName);
        await act(async () => {
            await user.clear(input);
            await user.type(input, value);
        });
    },

    /** Find input field by label text */
    findInputByLabel: (fieldName: string): HTMLElement => {
        // Strategy 1: Try to find by accessible name (label association)
        try {
            return screen.getByRole("textbox", { name: new RegExp(fieldName, "i") });
        } catch {}

        // Strategy 2: Find by proximity to label text
        try {
            const labelElement = screen.getByText(new RegExp(fieldName, "i"));
            const container = labelElement.closest("div");
            if (container) {
                const input = container.querySelector("input, textarea") as HTMLElement;
                if (input) return input;
            }
        } catch {}

        // Strategy 3: Find by name attribute matching the field name
        try {
            const normalizedFieldName = fieldName.toLowerCase().replace(/\s+/g, "");
            const input = document.querySelector(`input[name="${normalizedFieldName}"], textarea[name="${normalizedFieldName}"]`) as HTMLElement;
            if (input) return input;
        } catch {}

        // Strategy 4: Find by placeholder text
        try {
            return screen.getByPlaceholderText(new RegExp(fieldName, "i"));
        } catch {}

        throw new Error(`Could not find input field with label "${fieldName}". Available textboxes: ${screen.getAllByRole("textbox").map(el => el.getAttribute("name") || el.getAttribute("placeholder") || "unnamed").join(", ")}`);
    },

    /** Fill out a field by test id */
    fillFieldByTestId: async (
        user: ReturnType<typeof userEvent.setup>,
        testId: string,
        value: string,
    ) => {
        const input = screen.getByTestId(testId);
        await act(async () => {
            await user.clear(input);
            await user.type(input, value);
        });
    },

    /** Submit form by clicking submit button */
    submitByButton: async (
        user: ReturnType<typeof userEvent.setup>,
        buttonText = "Submit",
    ) => {
        const submitButton = screen.getByRole("button", { name: new RegExp(buttonText, "i") });
        await act(async () => {
            await user.click(submitButton);
        });
    },

    /** Submit form by pressing Enter in a field */
    submitByEnter: async (
        user: ReturnType<typeof userEvent.setup>,
        fieldName?: string,
    ) => {
        // Try to submit via enter key first
        if (fieldName) {
            let input: HTMLElement;
            try {
                input = formInteractions.findInputByLabel(fieldName);
                await act(async () => {
                    await user.click(input);
                    await user.keyboard("{Enter}");
                });
                // Give a moment for the submission to process
                await new Promise(resolve => setTimeout(resolve, 10));
                return;
            } catch {
                // Fall back to submit button if field not found
            }
        }
        
        // Fallback: click the submit button
        try {
            const submitButton = screen.getByTestId("form-submit-button");
            await act(async () => {
                await user.click(submitButton);
            });
        } catch {
            // Last resort: try to find any submit button
            const submitButton = screen.getByRole("button", { name: /submit/i });
            await act(async () => {
                await user.click(submitButton);
            });
        }
    },

    /** Trigger field validation by focusing and blurring */
    triggerValidation: async (
        user: ReturnType<typeof userEvent.setup>,
        fieldName: string,
    ) => {
        const input = formInteractions.findInputByLabel(fieldName);
        await act(async () => {
            await user.click(input);
            await user.tab(); // Blur to trigger validation
        });
    },

    /** Fill multiple fields in sequence */
    fillMultipleFields: async (
        user: ReturnType<typeof userEvent.setup>,
        fields: Record<string, string>,
    ) => {
        for (const [fieldName, value] of Object.entries(fields)) {
            await formInteractions.fillField(user, fieldName, value);
        }
    },
};

/**
 * Assertion helpers for form testing
 */
export const formAssertions = {
    /** Assert field has specific value */
    expectFieldValue: (fieldName: string, expectedValue: string) => {
        const input = formInteractions.findInputByLabel(fieldName) as HTMLInputElement;
        expect(input.value).toBe(expectedValue);
    },

    /** Assert field has error message */
    expectFieldError: (errorMessage: string) => {
        expect(screen.getByText(errorMessage)).toBeDefined();
    },

    /** Assert field has no error */
    expectNoFieldError: (errorMessage: string) => {
        expect(screen.queryByText(errorMessage)).toBeNull();
    },

    /** Assert form submitted with specific values */
    expectFormSubmitted: (
        onSubmit: ReturnType<typeof vi.fn>,
        expectedValues: any,
        submissionCount = 1,
    ) => {
        expect(onSubmit).toHaveBeenCalledTimes(submissionCount);
        expect(onSubmit).toHaveBeenCalledWith(expectedValues, expect.any(Object));
    },

    /** Assert field is required (has asterisk) */
    expectFieldRequired: (fieldName: string) => {
        // Look for the label with the field name, then check for asterisk
        const label = screen.getByText(new RegExp(fieldName, "i"));
        const container = label.closest("div") || label.parentElement;
        expect(container?.textContent).toContain("*");
    },

    /** Assert field is disabled */
    expectFieldDisabled: (fieldName: string) => {
        const input = formInteractions.findInputByLabel(fieldName) as HTMLInputElement;
        expect(input.disabled).toBe(true);
    },
};

/**
 * Create a simple form test with basic setup
 * Useful for quick tests that don't need complex configuration
 */
export function createSimpleFormTest<T = any>(
    initialValues: T,
    validationSchema?: yup.ObjectSchema<any>,
) {
    return {
        render: (component: ReactNode) => renderWithFormik(component, {
            initialValues,
            formikConfig: validationSchema ? { validationSchema } : {},
        }),
        schemas: testValidationSchemas,
        interactions: formInteractions,
        assertions: formAssertions,
    };
}

/**
 * Example usage patterns for common form scenarios
 */
export const formTestExamples = {
    /** Test a simple contact form */
    contactForm: () => createSimpleFormTest(
        { name: "", email: "", message: "" },
        yup.object({
            name: testValidationSchemas.requiredString("Name"),
            email: testValidationSchemas.email(),
            message: testValidationSchemas.requiredString("Message"),
        }),
    ),

    /** Test a user registration form */
    registrationForm: () => createSimpleFormTest(
        { username: "", email: "", password: "", confirmPassword: "" },
        yup.object({
            username: testValidationSchemas.username(),
            email: testValidationSchemas.email(),
            password: testValidationSchemas.password(),
            confirmPassword: yup.string()
                .oneOf([yup.ref("password")], "Passwords must match")
                .required("Please confirm your password"),
        }),
    ),

    /** Test a profile editing form */
    profileForm: () => createSimpleFormTest(
        { 
            name: "John Doe", 
            email: "john@example.com", 
            bio: "Software developer",
            website: "https://johndoe.com",
        },
        yup.object({
            name: testValidationSchemas.requiredString("Name"),
            email: testValidationSchemas.email(),
            bio: yup.string().max(500, "Bio must be less than 500 characters"),
            website: yup.string().url("Invalid URL").nullable(),
        }),
    ),
};
