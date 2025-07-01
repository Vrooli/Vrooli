/* eslint-disable react-perf/jsx-no-new-object-as-prop */
import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { Form, Formik } from "formik";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import * as yup from "yup";
import { formAssertions, formInteractions, renderWithFormik, testValidationSchemas } from "../../../__test/helpers/formTestHelpers.js";
import { TextInput, TextInputBase, TranslatedTextInput } from "./TextInput.js";

// Helper function to create mock icons
function MockIcon() {
    return <span data-testid="mock-icon">Icon</span>;
}

describe("TextInputBase", () => {
    describe("Basic rendering", () => {
        it("renders as a basic input field", () => {
            render(<TextInputBase />);

            const input = screen.getByRole("textbox");
            expect(input).toBeDefined();
            expect(input.tagName).toBe("INPUT");
        });

        it("renders with provided value", () => {
            render(<TextInputBase value="test value" onChange={vi.fn()} />);

            const input = screen.getByRole("textbox") as HTMLInputElement;
            expect(input.value).toBe("test value");
        });

        it("renders with placeholder text", () => {
            render(<TextInputBase placeholder="Enter text here" />);

            const input = screen.getByRole("textbox");
            expect(input.getAttribute("placeholder")).toBe("Enter text here");
        });

        it("accepts custom className prop", () => {
            render(<TextInputBase className="custom-class" />);

            const input = screen.getByRole("textbox");
            expect(input).toBeDefined();
            expect(input.tagName).toBe("INPUT");
        });
    });

    describe("Label behavior", () => {
        it("renders with label", () => {
            render(<TextInputBase label="Test Label" />);

            const label = screen.getByText("Test Label");
            expect(label).toBeDefined();
            expect(label.tagName).toBe("LABEL");
        });

        it("shows required asterisk when isRequired is true", () => {
            render(<TextInputBase label="Required Field" isRequired />);

            const label = screen.getByText("Required Field");
            const asterisk = screen.getByText("*");
            expect(label).toBeDefined();
            expect(asterisk).toBeDefined();
        });

        it("does not show asterisk when isRequired is false", () => {
            render(<TextInputBase label="Optional Field" isRequired={false} />);

            const label = screen.getByText("Optional Field");
            expect(label).toBeDefined();
            expect(screen.queryByText("*")).toBeNull();
        });

        it("does not render label when not provided", () => {
            render(<TextInputBase />);

            expect(screen.queryByText(/label/i)).toBeNull();
        });
    });

    describe("Helper text behavior", () => {
        it("renders helper text when provided", () => {
            render(<TextInputBase helperText="This is helper text" />);

            const helperText = screen.getByText("This is helper text");
            expect(helperText).toBeDefined();
            expect(helperText.tagName).toBe("P");
        });

        it("does not render helper text when not provided", () => {
            render(<TextInputBase />);

            // Should not find any paragraph elements (which helper text uses)
            expect(screen.queryByRole("paragraph")).toBeNull();
        });

        it("renders error helper text with error styling", () => {
            render(<TextInputBase helperText="Error message" error />);

            const helperText = screen.getByText("Error message");
            expect(helperText).toBeDefined();
        });

        it("handles boolean helper text", () => {
            render(<TextInputBase helperText={true} />);

            const helperText = screen.getByText("true");
            expect(helperText).toBeDefined();
        });
    });

    describe("Input states", () => {
        it("handles disabled state", () => {
            render(<TextInputBase disabled />);

            const input = screen.getByRole("textbox") as HTMLInputElement;
            expect(input.disabled).toBe(true);
        });

        it("handles error state", () => {
            render(<TextInputBase error />);

            const input = screen.getByRole("textbox");
            expect(input).toBeDefined();
            // Error styling is handled via CSS classes, not ARIA attributes
        });

        it("handles required state", () => {
            render(<TextInputBase required />);

            const input = screen.getByRole("textbox") as HTMLInputElement;
            expect(input.required).toBe(true);
        });

        it("handles readonly state", () => {
            render(<TextInputBase readOnly />);

            const input = screen.getByRole("textbox") as HTMLInputElement;
            expect(input.readOnly).toBe(true);
        });
    });

    describe("Input variants", () => {
        it("renders outline variant", () => {
            render(<TextInputBase variant="outline" />);

            const input = screen.getByRole("textbox");
            expect(input).toBeDefined();
        });

        it("renders filled variant", () => {
            render(<TextInputBase variant="filled" />);

            const input = screen.getByRole("textbox");
            expect(input).toBeDefined();
        });

        it("renders underline variant", () => {
            render(<TextInputBase variant="underline" />);

            const input = screen.getByRole("textbox");
            expect(input).toBeDefined();
        });
    });

    describe("Input sizes", () => {
        it("renders small size", () => {
            render(<TextInputBase size="sm" />);

            const input = screen.getByRole("textbox");
            expect(input).toBeDefined();
        });

        it("renders medium size", () => {
            render(<TextInputBase size="md" />);

            const input = screen.getByRole("textbox");
            expect(input).toBeDefined();
        });

        it("renders large size", () => {
            render(<TextInputBase size="lg" />);

            const input = screen.getByRole("textbox");
            expect(input).toBeDefined();
        });
    });

    describe("Multiline behavior", () => {
        it("renders as textarea when multiline is true", () => {
            render(<TextInputBase multiline />);

            const textarea = screen.getByRole("textbox");
            expect(textarea).toBeDefined();
            expect(textarea.tagName).toBe("TEXTAREA");
        });

        it("renders with specified rows", () => {
            render(<TextInputBase multiline rows={5} />);

            const textarea = screen.getByRole("textbox") as HTMLTextAreaElement;
            expect(textarea.getAttribute("rows")).toBe("5");
        });

        it("renders with specified cols", () => {
            render(<TextInputBase multiline cols={50} />);

            const textarea = screen.getByRole("textbox") as HTMLTextAreaElement;
            expect(textarea.getAttribute("cols")).toBe("50");
        });
    });

    describe("Adornments", () => {
        it("renders with start adornment", () => {
            render(
                <TextInputBase
                    startAdornment={<MockIcon />}
                />,
            );

            const input = screen.getByRole("textbox");
            const startAdornment = screen.getByTestId("mock-icon");

            expect(input).toBeDefined();
            expect(startAdornment).toBeDefined();
        });

        it("renders with end adornment", () => {
            render(
                <TextInputBase
                    endAdornment={<MockIcon />}
                />,
            );

            const input = screen.getByRole("textbox");
            const endAdornment = screen.getByTestId("mock-icon");

            expect(input).toBeDefined();
            expect(endAdornment).toBeDefined();
        });

        it("renders with both start and end adornments", () => {
            render(
                <TextInputBase
                    startAdornment={<span data-testid="start-icon">Start</span>}
                    endAdornment={<span data-testid="end-icon">End</span>}
                />,
            );

            const input = screen.getByRole("textbox");
            const startAdornment = screen.getByTestId("start-icon");
            const endAdornment = screen.getByTestId("end-icon");

            expect(input).toBeDefined();
            expect(startAdornment).toBeDefined();
            expect(endAdornment).toBeDefined();
        });

        it("focuses input when clicking on container with adornments", async () => {
            const user = userEvent.setup();

            render(
                <TextInputBase
                    startAdornment={<MockIcon />}
                />,
            );

            const input = screen.getByRole("textbox");
            const container = input.closest("div");

            await act(async () => {
                if (container) await user.click(container);
            });

            expect(document.activeElement).toBe(input);
        });

        it("does not focus disabled input when clicking container", async () => {
            const user = userEvent.setup();

            render(
                <TextInputBase
                    disabled
                    startAdornment={<MockIcon />}
                />,
            );

            const input = screen.getByRole("textbox");
            const container = input.closest("div");

            await act(async () => {
                if (container) await user.click(container);
            });

            expect(document.activeElement).not.toBe(input);
        });
    });

    describe("User interaction", () => {
        it("handles typing input", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();

            render(<TextInputBase onChange={onChange} />);

            const input = screen.getByRole("textbox");

            await act(async () => {
                await user.type(input, "test input");
            });

            expect(onChange).toHaveBeenCalled();
            expect((input as HTMLInputElement).value).toBe("test input");
        });

        it("handles focus events", async () => {
            const onFocus = vi.fn();
            const user = userEvent.setup();

            render(<TextInputBase onFocus={onFocus} />);

            const input = screen.getByRole("textbox");

            await act(async () => {
                await user.click(input);
            });

            expect(onFocus).toHaveBeenCalledTimes(1);
        });

        it("handles blur events", async () => {
            const onBlur = vi.fn();
            const user = userEvent.setup();

            render(<TextInputBase onBlur={onBlur} />);

            const input = screen.getByRole("textbox");

            await act(async () => {
                await user.click(input);
                await user.tab();
            });

            expect(onBlur).toHaveBeenCalledTimes(1);
        });

        it("handles keydown events", async () => {
            const onKeyDown = vi.fn();
            const user = userEvent.setup();

            render(<TextInputBase onKeyDown={onKeyDown} />);

            const input = screen.getByRole("textbox");

            await act(async () => {
                await user.click(input);
                await user.keyboard("{Enter}");
            });

            expect(onKeyDown).toHaveBeenCalled();
        });
    });

    describe("Submit functionality", () => {
        it("calls onSubmit when Enter is pressed and enterWillSubmit is true", async () => {
            const onSubmit = vi.fn();
            const user = userEvent.setup();

            render(
                <TextInputBase
                    enterWillSubmit
                    onSubmit={onSubmit}
                />,
            );

            const input = screen.getByRole("textbox");

            await act(async () => {
                await user.click(input);
                await user.keyboard("{Enter}");
            });

            expect(onSubmit).toHaveBeenCalledTimes(1);
        });

        it("does not call onSubmit when Enter is pressed with Shift key", async () => {
            const onSubmit = vi.fn();
            const user = userEvent.setup();

            render(
                <TextInputBase
                    enterWillSubmit
                    onSubmit={onSubmit}
                />,
            );

            const input = screen.getByRole("textbox");

            await act(async () => {
                await user.click(input);
                await user.keyboard("{Shift>}{Enter}{/Shift}");
            });

            expect(onSubmit).not.toHaveBeenCalled();
        });

        it("does not call onSubmit when enterWillSubmit is false", async () => {
            const onSubmit = vi.fn();
            const user = userEvent.setup();

            render(
                <TextInputBase
                    enterWillSubmit={false}
                    onSubmit={onSubmit}
                />,
            );

            const input = screen.getByRole("textbox");

            await act(async () => {
                await user.click(input);
                await user.keyboard("{Enter}");
            });

            expect(onSubmit).not.toHaveBeenCalled();
        });
    });

    describe("Full width behavior", () => {
        it("renders correctly when fullWidth is true", () => {
            render(<TextInputBase fullWidth />);

            const input = screen.getByRole("textbox");
            expect(input).toBeDefined();
            // Verify functional behavior rather than CSS classes
            expect(input.tagName).toBe("INPUT");
        });

        it("renders correctly when fullWidth is false", () => {
            render(<TextInputBase fullWidth={false} />);

            const input = screen.getByRole("textbox");
            expect(input).toBeDefined();
            // Verify functional behavior rather than CSS classes
            expect(input.tagName).toBe("INPUT");
        });
    });

    describe("Input type variations", () => {
        it("renders password input type", () => {
            render(<TextInputBase type="password" />);

            const input = screen.getByDisplayValue("") || document.querySelector("input[type=password]") as HTMLInputElement;
            expect(input?.getAttribute("type")).toBe("password");
        });

        it("renders email input type", () => {
            render(<TextInputBase type="email" />);

            const input = screen.getByRole("textbox");
            expect(input.getAttribute("type")).toBe("email");
        });

        it("renders number input type", () => {
            render(<TextInputBase type="number" />);

            const input = screen.getByRole("spinbutton");
            expect(input.getAttribute("type")).toBe("number");
        });

        it("renders tel input type", () => {
            render(<TextInputBase type="tel" />);

            const input = screen.getByRole("textbox");
            expect(input.getAttribute("type")).toBe("tel");
        });
    });
});

describe("TextInput (Formik-enabled wrapper)", () => {
    describe("Formik integration using test helpers", () => {
        it("integrates with Formik form state", async () => {
            const { user, getFormValues } = renderWithFormik(
                <TextInput
                    name="username"
                    label="Username"
                    placeholder="Enter username"
                />,
                { initialValues: { username: "" } },
            );

            const label = screen.getByText("Username");
            const input = screen.getByRole("textbox");

            expect(label).toBeDefined();
            expect(input.getAttribute("placeholder")).toBe("Enter username");
            expect((input as HTMLInputElement).value).toBe("");

            await formInteractions.fillField(user, "Username", "testuser");

            expect(getFormValues().username).toBe("testuser");
            formAssertions.expectFieldValue("Username", "testuser");
        });

        it("displays validation errors from Formik", async () => {
            const { user } = renderWithFormik(
                <TextInput name="email" label="Email" type="email" />,
                {
                    initialValues: { email: "" },
                    formikConfig: {
                        validationSchema: yup.object({
                            email: testValidationSchemas.email(),
                        }),
                    },
                },
            );

            await formInteractions.fillField(user, "Email", "invalid-email");
            await formInteractions.triggerValidation(user, "Email");

            formAssertions.expectFieldError("Invalid email");
        });

        it("shows required field validation", async () => {
            const { user } = renderWithFormik(
                <TextInput name="name" label="Name" isRequired />,
                {
                    initialValues: { name: "" },
                    formikConfig: {
                        validationSchema: yup.object({
                            name: testValidationSchemas.requiredString("Name"),
                        }),
                    },
                },
            );

            await formInteractions.triggerValidation(user, "Name");

            formAssertions.expectFieldError("Name is required");
        });

        it("handles form submission with Formik", async () => {
            const { user, onSubmit, submitForm } = renderWithFormik(
                <TextInput
                    name="message"
                    label="Message"
                    enterWillSubmit
                />,
                { initialValues: { message: "" } },
            );

            await formInteractions.fillField(user, "Message", "Hello world");

            // Manually submit the form since we can't pass submitForm to enterWillSubmit
            await submitForm();

            formAssertions.expectFormSubmitted(onSubmit, { message: "Hello world" });
        });

        it("displays validation errors correctly", async () => {
            const { user } = renderWithFormik(
                <TextInput name="username" label="Username" />,
                {
                    initialValues: { username: "" },
                    formikConfig: {
                        validationSchema: yup.object({
                            username: testValidationSchemas.username(3, 16),
                        }),
                    },
                },
            );

            await formInteractions.fillField(user, "Username", "ab");
            await formInteractions.triggerValidation(user, "Username");

            formAssertions.expectFieldError("Username must be at least 3 characters");
        });

        it("works with custom validation function", async () => {
            function customValidate(value: string) {
                if (value && value.includes("bad")) {
                    return "Value cannot contain 'bad'";
                }
                return undefined;
            }

            const { user } = renderWithFormik(
                <TextInput
                    name="content"
                    label="Content"
                    validate={customValidate}
                />,
                { initialValues: { content: "" } },
            );

            await formInteractions.fillField(user, "Content", "This is bad content");
            await formInteractions.triggerValidation(user, "Content");

            formAssertions.expectFieldError("Value cannot contain 'bad'");
        });
    });

    describe("Legacy Formik integration tests", () => {
        // Keep one example of the old approach for comparison
        it("integrates with Formik form state (legacy approach)", async () => {
            const onSubmit = vi.fn();
            const user = userEvent.setup();

            render(
                <Formik
                    initialValues={{ username: "" }}
                    onSubmit={onSubmit}
                >
                    <Form>
                        <TextInput
                            name="username"
                            label="Username"
                            placeholder="Enter username"
                        />
                    </Form>
                </Formik>,
            );

            const input = screen.getByRole("textbox");
            const label = screen.getByText("Username");

            expect(label).toBeDefined();
            expect(input.getAttribute("placeholder")).toBe("Enter username");
            expect((input as HTMLInputElement).value).toBe("");

            await act(async () => {
                await user.type(input, "testuser");
            });

            expect((input as HTMLInputElement).value).toBe("testuser");
        });
    });
});

describe("Component state changes", () => {
    it("toggles between normal and error states", () => {
        const { rerender } = render(<TextInputBase />);

        const input = screen.getByRole("textbox");
        expect(input).toBeDefined();

        // Switch to error state
        rerender(<TextInputBase error helperText="Error message" />);

        const errorText = screen.getByText("Error message");
        expect(errorText).toBeDefined();

        // Switch back to normal state
        rerender(<TextInputBase />);

        expect(screen.queryByText("Error message")).toBeNull();
    });

    it("toggles between enabled and disabled states", () => {
        const { rerender } = render(<TextInputBase />);

        let input = screen.getByRole("textbox") as HTMLInputElement;
        expect(input.disabled).toBe(false);

        // Switch to disabled state
        rerender(<TextInputBase disabled />);

        input = screen.getByRole("textbox") as HTMLInputElement;
        expect(input.disabled).toBe(true);

        // Switch back to enabled state
        rerender(<TextInputBase disabled={false} />);

        input = screen.getByRole("textbox") as HTMLInputElement;
        expect(input.disabled).toBe(false);
    });

    it("toggles between single-line and multiline", () => {
        const { rerender } = render(<TextInputBase />);

        let input = screen.getByRole("textbox");
        expect(input.tagName).toBe("INPUT");

        // Switch to multiline
        rerender(<TextInputBase multiline />);

        input = screen.getByRole("textbox");
        expect(input.tagName).toBe("TEXTAREA");

        // Switch back to single-line
        rerender(<TextInputBase multiline={false} />);

        input = screen.getByRole("textbox");
        expect(input.tagName).toBe("INPUT");
    });
});

describe("TranslatedTextInput", () => {
    describe("Translation integration", () => {
        it("renders with basic translation structure", () => {
            render(
                <Formik
                    initialValues={{
                        translations: [
                            { id: "1", language: "en", title: "Test Title" },
                        ],
                    }}
                    onSubmit={vi.fn()}
                >
                    <Form>
                        <TranslatedTextInput
                            language="en"
                            name="title"
                            label="Title"
                        />
                    </Form>
                </Formik>,
            );

            const input = screen.getByRole("textbox");
            expect((input as HTMLInputElement).value).toBe("Test Title");
        });
    });
});

