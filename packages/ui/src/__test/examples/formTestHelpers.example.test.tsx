import React from "react";
import { describe, expect, it } from "vitest";
import * as yup from "yup";
import { TextInput } from "../../components/inputs/TextInput/TextInput.js";
import {
    createSimpleFormTest,
    formAssertions,
    formInteractions,
    formTestExamples,
    renderWithFormik,
    testValidationSchemas,
} from "../helpers/formTestHelpers.js";

/**
 * Example tests demonstrating how to use the form test helpers
 * These tests show various patterns for testing forms with multiple inputs
 */

describe("Form Test Helpers - Examples", () => {
    describe("Simple form test helper", () => {
        it("tests a contact form with multiple fields", async () => {
            const contactFormTest = formTestExamples.contactForm();

            const { user, onSubmit, getFormValues } = contactFormTest.render(
                <>
                    <TextInput name="name" label="Name" isRequired />
                    <TextInput name="email" label="Email" type="email" isRequired />
                    <TextInput name="message" label="Message" multiline isRequired />
                </>,
            );

            // Fill out the form
            await formInteractions.fillMultipleFields(user, {
                "Name": "John Doe",
                "Email": "john@example.com",
                "Message": "Hello, this is a test message",
            });

            // Verify field values
            formAssertions.expectFieldValue("Name", "John Doe");
            formAssertions.expectFieldValue("Email", "john@example.com");
            formAssertions.expectFieldValue("Message", "Hello, this is a test message");

            // Submit the form via submit button (since Message is multiline, Enter won't work)
            await formInteractions.submitByButton(user, "Submit");

            // Verify submission
            formAssertions.expectFormSubmitted(onSubmit, {
                name: "John Doe",
                email: "john@example.com",
                message: "Hello, this is a test message",
            });
        });

        it("tests form validation errors", async () => {
            const contactFormTest = formTestExamples.contactForm();

            const { user } = contactFormTest.render(
                <>
                    <TextInput name="name" label="Name" isRequired />
                    <TextInput name="email" label="Email" type="email" isRequired />
                    <TextInput name="message" label="Message" multiline isRequired />
                </>,
            );

            // Fill invalid email and trigger validation
            await formInteractions.fillField(user, "Email", "invalid-email");
            await formInteractions.triggerValidation(user, "Email");

            // Check for validation error
            formAssertions.expectFieldError("Invalid email");

            // Fill valid email
            await formInteractions.fillField(user, "Email", "john@example.com");
            await formInteractions.triggerValidation(user, "Email");

            // Error should be gone
            formAssertions.expectNoFieldError("Invalid email");
        });
    });

    describe("Custom form configuration", () => {
        it("tests a user profile form with pre-filled values", async () => {
            const initialValues = {
                username: "johndoe",
                email: "john@example.com",
                displayName: "John Doe",
                bio: "Software developer",
            };

            const validationSchema = yup.object({
                username: testValidationSchemas.username(3, 20),
                email: testValidationSchemas.email(),
                displayName: testValidationSchemas.requiredString("Display Name"),
                bio: yup.string().max(500, "Bio must be less than 500 characters"),
            });

            const { user, onSubmit, getFormValues } = renderWithFormik(
                <>
                    <TextInput name="username" label="Username" isRequired />
                    <TextInput name="email" label="Email" type="email" isRequired />
                    <TextInput name="displayName" label="Display Name" isRequired />
                    <TextInput name="bio" label="Bio" multiline />
                </>,
                {
                    initialValues,
                    formikConfig: { validationSchema },
                },
            );

            // Verify initial values are loaded
            formAssertions.expectFieldValue("Username", "johndoe");
            formAssertions.expectFieldValue("Email", "john@example.com");
            formAssertions.expectFieldValue("Display Name", "John Doe");
            formAssertions.expectFieldValue("Bio", "Software developer");

            // Update bio field
            await formInteractions.fillField(user, "Bio", "Senior Software Developer with 5+ years experience");

            // Verify form values updated
            const currentValues = getFormValues();
            expect(currentValues.bio).toBe("Senior Software Developer with 5+ years experience");

            // Submit form via Enter on username field
            await formInteractions.submitByEnter(user, "Username");

            // Verify submission with updated values
            formAssertions.expectFormSubmitted(onSubmit, {
                username: "johndoe",
                email: "john@example.com",
                displayName: "John Doe",
                bio: "Senior Software Developer with 5+ years experience",
            });
        });
    });

    describe("Complex validation scenarios", () => {
        it("tests username availability checking", async () => {
            // Simulate async validation
            const checkUsernameAvailable = async (username: string) => {
                if (username === "taken") {
                    return "Username is already taken";
                }
                return undefined;
            };

            const validationSchema = yup.object({
                username: yup.string()
                    .required("Username is required")
                    .test("availability", "Username is already taken", async (value) => {
                        if (!value) return true;
                        const error = await checkUsernameAvailable(value);
                        return !error;
                    }),
                email: testValidationSchemas.email(),
            });

            const testHelper = createSimpleFormTest(
                { username: "", email: "" },
                validationSchema,
            );
            
            const { user, assertions, interactions } = {
                ...testHelper.render(
                    <>
                        <TextInput name="username" label="Username" isRequired />
                        <TextInput name="email" label="Email" type="email" isRequired />
                    </>,
                ),
                assertions: formAssertions,
                interactions: formInteractions,
            };

            // Try a taken username
            await interactions.fillField(user, "Username", "taken");
            await interactions.triggerValidation(user, "Username");

            // Should show availability error (wait for async validation)
            await new Promise(resolve => setTimeout(resolve, 100));
            assertions.expectFieldError("Username is already taken");

            // Try an available username
            await interactions.fillField(user, "Username", "available");
            await interactions.triggerValidation(user, "Username");

            // Error should be gone
            await new Promise(resolve => setTimeout(resolve, 100));
            assertions.expectNoFieldError("Username is already taken");
        });
    });

    describe("Form state management", () => {
        it("tests form reset functionality", async () => {
            const initialValues = { name: "Original Name", email: "original@example.com" };

            const { user, resetForm, getFormValues } = renderWithFormik(
                <>
                    <TextInput name="name" label="Name" />
                    <TextInput name="email" label="Email" type="email" />
                </>,
                { initialValues },
            );

            // Modify fields
            await formInteractions.fillField(user, "Name", "Modified Name");
            await formInteractions.fillField(user, "Email", "modified@example.com");

            // Verify changes
            expect(getFormValues().name).toBe("Modified Name");
            expect(getFormValues().email).toBe("modified@example.com");

            // Reset form
            resetForm();

            // Verify reset to original values
            expect(getFormValues().name).toBe("Original Name");
            expect(getFormValues().email).toBe("original@example.com");
        });

        it("tests programmatic field updates", async () => {
            const { setFieldValue, getFormValues } = renderWithFormik(
                <>
                    <TextInput name="firstName" label="First Name" />
                    <TextInput name="lastName" label="Last Name" />
                    <TextInput name="fullName" label="Full Name" readOnly />
                </>,
                {
                    initialValues: { firstName: "", lastName: "", fullName: "" },
                },
            );

            // Set field values programmatically
            await setFieldValue("firstName", "John");
            await setFieldValue("lastName", "Doe");
            await setFieldValue("fullName", "John Doe");

            // Verify values updated
            const values = getFormValues();
            expect(values.firstName).toBe("John");
            expect(values.lastName).toBe("Doe");
            expect(values.fullName).toBe("John Doe");

            // Verify UI reflects the changes
            formAssertions.expectFieldValue("First Name", "John");
            formAssertions.expectFieldValue("Last Name", "Doe");
            formAssertions.expectFieldValue("Full Name", "John Doe");
        });
    });
});
