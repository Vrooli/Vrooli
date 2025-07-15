import { act, render, screen, type RenderResult } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { Form, Formik, type FormikConfig, type FormikProps } from "formik";
import React, { type ReactNode } from "react";
import { vi } from "vitest";
import * as yup from "yup";

/**
 * Optimized Form Test Helpers
 * 
 * This file contains performance-optimized versions of form test helpers that:
 * - Use faster user event configuration with reduced delays
 * - Cache DOM queries to avoid repeated lookups
 * - Batch operations where possible
 * - Skip animations and transitions
 */

// Configure userEvent for faster testing
const createFastUser = () => userEvent.setup({
    delay: null, // Remove artificial typing delay
    pointerEventsCheck: 0, // Skip pointer events check
    skipHover: true, // Skip hover simulation
    skipClick: false, // Still simulate clicks
});

// Cache for input elements to avoid repeated DOM queries
const inputCache = new Map<string, HTMLElement>();

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
    /** Use fast user event setup (default: true) */
    useFastUser?: boolean;
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
 * Optimized version with performance improvements
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
        useFastUser = true,
    } = config;

    // Clear input cache when rendering new component
    inputCache.clear();

    const onSubmit = vi.fn();
    const user = useFastUser ? createFastUser() : userEvent.setup();
    
    let formikRef: FormikProps<T> | null = null;

    const TestComponent = () => (
        <Formik
            initialValues={initialValues}
            onSubmit={onSubmit}
            validateOnChange={false} // Reduce validation calls during testing
            validateOnBlur={false}    // Only validate on submit for tests
            {...formikConfig}
        >
            {(formikProps) => {
                formikRef = formikProps;
                
                if (customRender) {
                    return customRender(formikProps);
                }
                
                if (wrapInForm) {
                    return <Form>{component}</Form>;
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
            // Don't wrap in act() - let the test handle it
            await formikRef?.submitForm();
        },
        resetForm: () => {
            // Don't wrap in act() - let the test handle it
            formikRef?.resetForm();
        },
        setFieldValue: async (field: keyof T, value: any) => {
            // Don't wrap in act() - let the test handle it
            await formikRef?.setFieldValue(field as string, value);
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
 * Optimized helper functions for common form interactions
 */
export const formInteractions = {
    /** Fill out a text input field with optimizations */
    fillField: async (
        user: ReturnType<typeof userEvent.setup>,
        fieldName: string,
        value: string,
    ) => {
        const input = formInteractions.findInputByLabel(fieldName);
        // Click to focus, select all, then type (replaces existing content)
        await user.click(input);
        // Use triple-click to select all text, then type to replace
        await user.tripleClick(input);
        await user.type(input, value);
    },

    /** Find input field by label text with caching */
    findInputByLabel: (fieldName: string): HTMLElement => {
        // Check cache first
        const cacheKey = fieldName.toLowerCase();
        if (inputCache.has(cacheKey)) {
            return inputCache.get(cacheKey)!;
        }

        let input: HTMLElement | null = null;

        // Strategy 1: Try to find by accessible name (label association)
        try {
            input = screen.getByRole("textbox", { name: new RegExp(fieldName, "i") });
            if (input) {
                inputCache.set(cacheKey, input);
                return input;
            }
        } catch {}

        // Strategy 1.5: Try to find password inputs by label text
        if (fieldName.toLowerCase().includes("password")) {
            try {
                // Find all password inputs and check their labels
                const passwordInputs = document.querySelectorAll("input[type=\"password\"], [data-testid=\"password-input\"]") as NodeListOf<HTMLInputElement>;
                for (const passwordInput of passwordInputs) {
                    // Check if the label associated with this input matches
                    const labels = document.querySelectorAll("label");
                    for (const label of labels) {
                        if (label.textContent && new RegExp(fieldName, "i").test(label.textContent)) {
                            // Check if this label is associated with the password input
                            const labelFor = label.getAttribute("for");
                            const inputId = passwordInput.id || passwordInput.name;
                            if (labelFor === inputId || label.contains(passwordInput)) {
                                inputCache.set(cacheKey, passwordInput);
                                return passwordInput;
                            }
                        }
                    }
                    
                    // Also check if the input has a name attribute that matches
                    if (passwordInput.name && new RegExp(fieldName.replace(/\s+/g, ""), "i").test(passwordInput.name)) {
                        inputCache.set(cacheKey, passwordInput);
                        return passwordInput;
                    }
                }
            } catch {}
        }

        // Strategy 2: Find by test-id if available (fastest)
        try {
            const testId = fieldName.toLowerCase().replace(/\s+/g, "-") + "-input";
            input = screen.getByTestId(testId);
            if (input) {
                inputCache.set(cacheKey, input);
                return input;
            }
        } catch {}

        // Strategy 2.5: For password fields, try password-input test-id
        if (fieldName.toLowerCase().includes("password")) {
            try {
                const passwordInputs = screen.getAllByTestId("password-input");
                if (passwordInputs.length === 1) {
                    // If there's only one password input, use it
                    inputCache.set(cacheKey, passwordInputs[0]);
                    return passwordInputs[0];
                } else if (passwordInputs.length > 1) {
                    // Multiple password inputs - try to distinguish by name
                    for (const passwordInput of passwordInputs) {
                        const name = passwordInput.getAttribute("name") || "";
                        if (fieldName.toLowerCase().includes("confirm") && name.toLowerCase().includes("confirm")) {
                            inputCache.set(cacheKey, passwordInput);
                            return passwordInput;
                        } else if (!fieldName.toLowerCase().includes("confirm") && !name.toLowerCase().includes("confirm")) {
                            inputCache.set(cacheKey, passwordInput);
                            return passwordInput;
                        }
                    }
                }
            } catch {}
        }

        // Strategy 3: Find by name attribute
        try {
            const normalizedFieldName = fieldName.toLowerCase().replace(/\s+/g, "");
            input = document.querySelector(`input[name="${normalizedFieldName}"], textarea[name="${normalizedFieldName}"]`) as HTMLElement;
            if (input) {
                inputCache.set(cacheKey, input);
                return input;
            }
        } catch {}

        // Strategy 4: Find by placeholder text
        try {
            input = screen.getByPlaceholderText(new RegExp(fieldName, "i"));
            if (input) {
                inputCache.set(cacheKey, input);
                return input;
            }
        } catch {}

        throw new Error(`Could not find input field with label "${fieldName}"`);
    },

    /** Submit form by clicking submit button */
    submitByButton: async (
        user: ReturnType<typeof userEvent.setup>,
        buttonText = "Submit",
    ) => {
        // Try to find by test-id first (faster)
        let submitButton: HTMLElement;
        try {
            submitButton = screen.getByTestId("submit-button");
        } catch {
            submitButton = screen.getByRole("button", { name: new RegExp(buttonText, "i") });
        }
        
        // Don't wrap in act() to avoid overlapping calls
        await user.click(submitButton);
    },

    /** Fill multiple fields in parallel when possible */
    fillMultipleFields: async (
        user: ReturnType<typeof userEvent.setup>,
        fields: Record<string, string>,
    ) => {
        // Pre-fetch all inputs to cache them
        const inputs: Array<[string, HTMLElement]> = [];
        for (const fieldName of Object.keys(fields)) {
            inputs.push([fieldName, formInteractions.findInputByLabel(fieldName)]);
        }

        // Fill fields sequentially (can't truly parallelize user interactions)
        for (const [fieldName, value] of Object.entries(fields)) {
            await formInteractions.fillField(user, fieldName, value);
        }
    },

    /** Trigger validation without blur (faster) */
    triggerValidation: async (
        user: ReturnType<typeof userEvent.setup>,
        fieldName: string,
    ) => {
        const input = formInteractions.findInputByLabel(fieldName);
        await act(async () => {
            await user.click(input);
            // Trigger validation directly instead of tab
            input.dispatchEvent(new Event("blur", { bubbles: true }));
        });
    },
};

/**
 * Assertion helpers remain the same as they don't impact performance
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
};

// Export the same validation schemas for compatibility
export { testValidationSchemas } from "./formTestHelpers.js";
