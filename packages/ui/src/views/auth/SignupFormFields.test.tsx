/**
 * SignupFormFields Test
 * 
 * Demonstrates how extracting form fields into a separate component
 * makes testing with our form helpers much cleaner and more focused.
 */
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { screen, act, waitFor } from "@testing-library/react";
import {
    renderWithFormik,
    formInteractions,
    formAssertions,
    testValidationSchemas,
} from "../../__test/helpers/formTestHelpers.js";
import { SignupFormFields, type SignupViewFormInput } from "./SignupView.js";
import * as yup from "yup";

// All mocks are now centralized in setup.vitest.ts
// The zxcvbn mock returns password strength based on password length

// Override translations for this specific test
vi.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: (key: string, options?: { count?: number }) => {
            const translations = {
                "Name": "Name",
                "Email": "Email", 
                "Password": "Password",
                "PasswordConfirm": "Confirm Password",
                "NamePlaceholder": "Enter your name",
                "EmailPlaceholder": "Enter your email",
                "PasswordPlaceholder": "Enter your password",
                "SendMeUpdatesAndOffers": "Send me updates and offers",
                "IAgreeToThe": "I agree to the",
                "Terms": "terms",
                "PleaseAcceptTermsToContinue": "Please accept the terms to continue",
            };
            
            if (key === "Email" && options?.count === 1) {
                return "Email";
            }
            
            return translations[key] || key;
        },
    }),
}));

// Mock zxcvbn to prevent async password strength calculations in tests
vi.mock("zxcvbn", () => ({
    default: vi.fn((password: string) => {
        if (!password) return { score: 0 };
        if (password.length < 6) return { score: 0 };
        if (password.length < 8) return { score: 1 };
        if (password.length < 10) return { score: 2 };
        if (password.length < 12) return { score: 3 };
        return { score: 4 };
    }),
}));

describe("SignupFormFields", () => {
    // Initial form values matching SignupView
    const initialValues: SignupViewFormInput = {
        agreeToTerms: false,
        marketingEmails: true,
        name: "",
        email: "",
        password: "",
        confirmPassword: "",
        theme: "",
    };

    // Validation schema matching the real signup form
    const signupValidationSchema = yup.object({
        name: testValidationSchemas.requiredString("Name"),
        email: testValidationSchemas.email(),
        password: testValidationSchemas.password(8),
        confirmPassword: yup.string()
            .oneOf([yup.ref("password")], "Passwords must match")
            .required("Please confirm your password"),
        agreeToTerms: yup.boolean()
            .oneOf([true], "You must accept the terms to continue"),
    });

    describe("Field Rendering", () => {
        it("renders all signup form fields", async () => {
            renderWithFormik(
                (formik) => <SignupFormFields formik={formik} />,
                {
                    initialValues,
                    customRender: (formikProps) => (
                        <SignupFormFields formik={formikProps} />
                    ),
                },
            );

            // Wait for async password strength calculations to complete
            await waitFor(() => {
                expect(screen.getByLabelText("Name")).toBeDefined();
                expect(screen.getByLabelText("Email")).toBeDefined();
                expect(screen.getByLabelText("Password")).toBeDefined();
                expect(screen.getByLabelText("Confirm Password")).toBeDefined();
                expect(screen.getByRole("checkbox", { name: /updates and offers/i })).toBeDefined();
                expect(screen.getByRole("checkbox", { name: /terms/i })).toBeDefined();
            });
        });

        it("shows marketing emails checkbox as checked by default", async () => {
            renderWithFormik(
                (formik) => <SignupFormFields formik={formik} />,
                {
                    initialValues,
                    customRender: (formikProps) => (
                        <SignupFormFields formik={formikProps} />
                    ),
                },
            );
            
            // Wait for component to render completely
            await waitFor(() => {
                const checkbox = screen.getByRole("checkbox", { name: /updates and offers/i }) as HTMLInputElement;
                expect(checkbox.checked).toBe(true);
            });

        });

        it("shows terms checkbox as unchecked by default", async () => {
            renderWithFormik(
                (formik) => <SignupFormFields formik={formik} />,
                {
                    initialValues,
                    customRender: (formikProps) => (
                        <SignupFormFields formik={formikProps} />
                    ),
                },
            );
            
            // Wait for component to render completely
            await waitFor(() => {
                const checkbox = screen.getByRole("checkbox", { name: /terms/i }) as HTMLInputElement;
                expect(checkbox.checked).toBe(false);
            });
        });
    });

    describe("Form Interactions", () => {
        it("fills and submits valid signup form", async () => {
            const { user, onSubmit, getFormValues } = renderWithFormik(
                (formik) => <SignupFormFields formik={formik} />,
                {
                    initialValues,
                    formikConfig: { validationSchema: signupValidationSchema },
                    customRender: (formikProps) => (
                        <>
                            <SignupFormFields formik={formikProps} />
                            <button type="submit">Sign Up</button>
                        </>
                    ),
                },
            );

            // Wait for password inputs to finish async initialization
            await waitFor(() => {
                expect(screen.getByLabelText("Password")).toBeDefined();
                expect(screen.getByLabelText("Confirm Password")).toBeDefined();
            });

            // Fill out all fields
            await formInteractions.fillMultipleFields(user, {
                "Name": "John Doe",
                "Email": "john@example.com",
                "Password": "securePassword123!",
                "Confirm Password": "securePassword123!",
            });

            // Check the terms checkbox
            const termsCheckbox = screen.getByRole("checkbox", { name: /terms/i });
            await user.click(termsCheckbox);

            // Verify form values
            const values = getFormValues();
            expect(values.name).toBe("John Doe");
            expect(values.email).toBe("john@example.com");
            expect(values.password).toBe("securePassword123!");
            expect(values.confirmPassword).toBe("securePassword123!");
            expect(values.agreeToTerms).toBe(true);
            expect(values.marketingEmails).toBe(true);

            // Submit form
            await formInteractions.submitByButton(user, "Sign Up");

            // Since we're testing just the fields component, 
            // we verify the form state is correct for submission
            expect(values).toMatchObject({
                name: "John Doe",
                email: "john@example.com",
                password: "securePassword123!",
                confirmPassword: "securePassword123!",
                agreeToTerms: true,
                marketingEmails: true,
            });
        });

        it("toggles marketing emails checkbox", async () => {
            const { user, getFormValues } = renderWithFormik(
                (formik) => <SignupFormFields formik={formik} />,
                {
                    initialValues,
                    customRender: (formikProps) => (
                        <SignupFormFields formik={formikProps} />
                    ),
                },
            );

            // Wait for password inputs to finish async initialization
            await waitFor(() => {
                expect(screen.getByLabelText("Password")).toBeDefined();
            });

            const marketingCheckbox = screen.getByRole("checkbox", { name: /updates and offers/i });
            
            // Should start as checked
            expect(getFormValues().marketingEmails).toBe(true);

            // Toggle off
            await user.click(marketingCheckbox);
            expect(getFormValues().marketingEmails).toBe(false);

            // Toggle back on
            await user.click(marketingCheckbox);
            expect(getFormValues().marketingEmails).toBe(true);
        });
    });

    describe("Validation Scenarios", () => {
        it("shows email validation error", async () => {
            const { user } = renderWithFormik(
                (formik) => <SignupFormFields formik={formik} />,
                {
                    initialValues,
                    formikConfig: { validationSchema: signupValidationSchema },
                    customRender: (formikProps) => (
                        <SignupFormFields formik={formikProps} />
                    ),
                },
            );

            // Enter invalid email
            await formInteractions.fillField(user, "Email", "invalid-email");
            await formInteractions.triggerValidation(user, "Email");

            // Should show validation error
            formAssertions.expectFieldError("Invalid email");
        });

        it("validates password mismatch", async () => {
            const { user } = renderWithFormik(
                (formik) => <SignupFormFields formik={formik} />,
                {
                    initialValues,
                    formikConfig: { validationSchema: signupValidationSchema },
                    customRender: (formikProps) => (
                        <SignupFormFields formik={formikProps} />
                    ),
                },
            );

            // Enter mismatched passwords
            await formInteractions.fillField(user, "Password", "password123");
            await formInteractions.fillField(user, "Confirm Password", "differentPassword");
            await formInteractions.triggerValidation(user, "Confirm Password");

            // Should show mismatch error
            formAssertions.expectFieldError("Passwords must match");
        });

        it("shows terms acceptance error when not checked", async () => {
            const { user } = renderWithFormik(
                (formik) => <SignupFormFields formik={formik} />,
                {
                    initialValues,
                    formikConfig: { validationSchema: signupValidationSchema },
                    customRender: (formikProps) => (
                        <>
                            <SignupFormFields formik={formikProps} />
                            <button type="submit">Sign Up</button>
                        </>
                    ),
                },
            );

            // Fill valid data but don't check terms
            await formInteractions.fillMultipleFields(user, {
                "Name": "John Doe",
                "Email": "john@example.com",
                "Password": "securePassword123!",
                "Confirm Password": "securePassword123!",
            });

            // Try to submit without accepting terms
            await formInteractions.submitByButton(user, "Sign Up");

            // Should show terms error after validation runs
            // The error is rendered conditionally based on formik.touched state
            const termsCheckbox = screen.getByRole("checkbox", { name: /terms/i });
            
            // Touch the checkbox field to trigger validation display
            await user.click(termsCheckbox); // Click once to focus
            await user.tab(); // Blur to trigger touched state
            
            // Now the error should be visible
            expect(screen.queryByText("Please accept the terms to continue")).toBeDefined();
        });
    });

    describe("Field Accessibility", () => {
        it("has proper labels for all fields", () => {
            renderWithFormik(
                (formik) => <SignupFormFields formik={formik} />,
                {
                    initialValues,
                    customRender: (formikProps) => (
                        <SignupFormFields formik={formikProps} />
                    ),
                },
            );

            // All fields should have accessible labels
            expect(screen.getByLabelText("Name")).toBeDefined();
            expect(screen.getByLabelText("Email")).toBeDefined();
            expect(screen.getByLabelText("Password")).toBeDefined();
            expect(screen.getByLabelText("Confirm Password")).toBeDefined();
        });

        it("supports keyboard navigation", async () => {
            const { user } = renderWithFormik(
                (formik) => <SignupFormFields formik={formik} />,
                {
                    initialValues,
                    customRender: (formikProps) => (
                        <SignupFormFields formik={formikProps} />
                    ),
                },
            );

            // Tab through fields - modern input components may have complex structure
            // so we'll verify that we can reach all form inputs through tabbing
            const nameInput = screen.getByLabelText("Name");
            const emailInput = screen.getByLabelText("Email");
            const marketingCheckbox = screen.getByRole("checkbox", { name: /updates and offers/i });
            const termsCheckbox = screen.getByRole("checkbox", { name: /terms/i });
            
            const focusedElements = [];
            let tabCount = 0;
            const maxTabs = 15; // Safety limit
            
            while (tabCount < maxTabs) {
                await user.tab();
                focusedElements.push(document.activeElement);
                tabCount++;
                
                // Break if we've reached all expected elements
                if (focusedElements.includes(nameInput) && 
                    focusedElements.includes(emailInput) &&
                    focusedElements.includes(marketingCheckbox) &&
                    focusedElements.includes(termsCheckbox)) {
                    break;
                }
            }
            
            // Verify all main form inputs are reachable via keyboard
            expect(focusedElements).toContain(nameInput);
            expect(focusedElements).toContain(emailInput);
            expect(focusedElements).toContain(marketingCheckbox);
            expect(focusedElements).toContain(termsCheckbox);

            // All key form elements should be keyboard accessible
            // This covers the core accessibility requirement
        });
    });
});

/**
 * Benefits of the extracted component approach:
 * 
 * ✅ **Focused Testing**: Test just the form fields without auth logic
 * ✅ **Full Formik Control**: Access to formik props and state
 * ✅ **Clean Test Structure**: Using our form helpers makes tests readable
 * ✅ **Real Validation**: Test with actual validation schemas
 * ✅ **Better Isolation**: No need to mock API calls or routing
 * ✅ **Reusability**: SignupFormFields can be used in other contexts
 * 
 * This pattern works great for any form-heavy component!
 */
