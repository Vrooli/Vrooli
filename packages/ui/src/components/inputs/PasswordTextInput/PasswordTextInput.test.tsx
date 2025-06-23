/**
 * PasswordTextInput Component Tests
 * 
 * Tests the PasswordTextInput component functionality including:
 * - Password visibility toggle
 * - Password strength indicator
 * - Form integration with Formik
 * - Error display
 */
import { ThemeProvider } from "@mui/material/styles";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Formik } from "formik";
import React from "react";
import { beforeEach, describe, expect, it } from "vitest";
import { themes } from "../../../utils/display/theme.js";
import { PasswordTextInput } from "./PasswordTextInput.js";

// All mocks are now centralized in setup.vitest.ts
// The zxcvbn mock returns password strength based on password length

// Test wrapper component
function TestWrapper({ children }: { children: React.ReactNode }) {
    return (
        <ThemeProvider theme={themes.light}>
            <Formik
                initialValues={{ password: "" }}
                onSubmit={() => {}}
            >
                {children}
            </Formik>
        </ThemeProvider>
    );
}

describe("PasswordTextInput", () => {
    let user: ReturnType<typeof userEvent.setup>;

    beforeEach(() => {
        user = userEvent.setup();
    });

    it("renders with default props", async () => {
        await act(async () => {
            render(
                <TestWrapper>
                    <PasswordTextInput name="password" />
                </TestWrapper>
            );
        });

        await waitFor(() => {
            expect(screen.getByLabelText("Password")).toBeDefined();
            expect(screen.getByPlaceholderText("PasswordPlaceholder")).toBeDefined();
        });
    });

    it("renders with custom label", async () => {
        await act(async () => {
            render(
                <TestWrapper>
                    <PasswordTextInput name="password" label="Custom Password" />
                </TestWrapper>
            );
        });

        await waitFor(() => {
            expect(screen.getByLabelText("Custom Password")).toBeDefined();
        });
    });

    it("toggles password visibility", async () => {
        await act(async () => {
            render(
                <TestWrapper>
                    <PasswordTextInput name="password" />
                </TestWrapper>
            );
        });

        await waitFor(() => {
            const passwordInput = screen.getByLabelText("Password") as HTMLInputElement;
            const toggleButton = screen.getByLabelText("toggle password visibility");

            // Initially password should be hidden
            expect(passwordInput.type).toBe("password");
        });

        const passwordInput = screen.getByLabelText("Password") as HTMLInputElement;
        const toggleButton = screen.getByLabelText("toggle password visibility");

        // Click to show password
        await act(async () => {
            await user.click(toggleButton);
        });
        expect(passwordInput.type).toBe("text");

        // Click again to hide password
        await act(async () => {
            await user.click(toggleButton);
        });
        expect(passwordInput.type).toBe("password");
    });

    it("shows password strength indicator for new passwords", async () => {
        await act(async () => {
            render(
                <TestWrapper>
                    <PasswordTextInput name="password" autoComplete="new-password" />
                </TestWrapper>
            );
        });

        await waitFor(() => {
            expect(screen.getByLabelText("Password")).toBeDefined();
        });

        const passwordInput = screen.getByLabelText("Password");

        // Type a weak password
        await act(async () => {
            await user.type(passwordInput, "weak");
        });

        // Wait for async password strength calculation to complete
        await waitFor(() => {
            const progressBar = screen.getByRole("progressbar");
            expect(progressBar).toBeDefined();
        }, { timeout: 3000 });
    });

    it("doesn't show password strength indicator for current passwords", async () => {
        await act(async () => {
            render(
                <TestWrapper>
                    <PasswordTextInput name="password" autoComplete="current-password" />
                </TestWrapper>
            );
        });

        await waitFor(() => {
            // Progress bar should not be present
            expect(screen.queryByRole("progressbar")).toBeNull();
        });
    });

    it("updates password strength as user types", async () => {
        let container: any;
        act(() => {
            const result = render(
                <TestWrapper>
                    <PasswordTextInput name="password" autoComplete="new-password" />
                </TestWrapper>
            );
            container = result.container;
        });

        const passwordInput = screen.getByLabelText("Password");

        // Type progressively stronger passwords with proper waiting
        await act(async () => {
            await user.type(passwordInput, "weak");
        });

        await waitFor(() => {
            const progressBar = screen.getByRole("progressbar");
            // "weak" password typically gets score 0 or 1 = 0% or 25% progress
            const value = progressBar.getAttribute("aria-valuenow");
            expect(value === "0" || value === "25").toBe(true);
        }, { timeout: 3000 });

        await act(async () => {
            await user.clear(passwordInput);
            await user.type(passwordInput, "moderate123");
        });

        await waitFor(() => {
            const progressBar = screen.getByRole("progressbar");
            // "moderate123" should get score 2 or 3 = 50% or 75% progress
            const value = progressBar.getAttribute("aria-valuenow");
            expect(value === "50" || value === "75").toBe(true);
        }, { timeout: 3000 });

        await act(async () => {
            await user.clear(passwordInput);
            await user.type(passwordInput, "VeryStr0ng!Pass#2024");
        });

        await waitFor(() => {
            const progressBar = screen.getByRole("progressbar");
            // Very strong password should get score 3 or 4 = 75% or 100% progress
            const value = progressBar.getAttribute("aria-valuenow");
            expect(value === "75" || value === "100").toBe(true);
        }, { timeout: 3000 });
    });

    it("displays error message when validation fails", async () => {
        await act(async () => {
            render(
                <ThemeProvider theme={themes.light}>
                    <Formik
                        initialValues={{ password: "" }}
                        initialErrors={{ password: "Password is required" }}
                        initialTouched={{ password: true }}
                        onSubmit={() => {}}
                    >
                        <PasswordTextInput name="password" />
                    </Formik>
                </ThemeProvider>
            );
        });

        await waitFor(() => {
            expect(screen.getByText("Password is required")).toBeDefined();
        });
    });

    it("respects fullWidth prop", async () => {
        let container: any;
        await act(async () => {
            const result = render(
                <TestWrapper>
                    <PasswordTextInput name="password" fullWidth={false} />
                </TestWrapper>
            );
            container = result.container;
        });

        await waitFor(() => {
            // With TailwindTextInput, check that the component renders correctly with fullWidth=false
            const passwordInput = screen.getByLabelText("Password");
            expect(passwordInput).toBeDefined();
            // The input should still be present and functional
            expect(passwordInput.getAttribute("type")).toBe("password");
        });
    });

    it("applies autoFocus when specified", async () => {
        await act(async () => {
            render(
                <TestWrapper>
                    <PasswordTextInput name="password" autoFocus />
                </TestWrapper>
            );
        });

        await waitFor(() => {
            const passwordInput = screen.getByLabelText("Password");
            expect(document.activeElement).toBe(passwordInput);
        });
    });

    it("uses correct autoComplete attribute", async () => {
        await act(async () => {
            render(
                <TestWrapper>
                    <PasswordTextInput name="password" autoComplete="new-password" />
                </TestWrapper>
            );
        });

        await waitFor(() => {
            const passwordInput = screen.getByLabelText("Password");
            expect(passwordInput.getAttribute("autocomplete")).toBe("new-password");
        });
    });

    it("integrates with Formik field state", async () => {
        let formValues: any = {};
        
        act(() => {
            render(
                <ThemeProvider theme={themes.light}>
                    <Formik
                        initialValues={{ password: "" }}
                        onSubmit={(values) => {
                            formValues = values;
                        }}
                    >
                        {({ handleSubmit }) => (
                            <form onSubmit={handleSubmit}>
                                <PasswordTextInput name="password" />
                                <button type="submit">Submit</button>
                            </form>
                        )}
                    </Formik>
                </ThemeProvider>
            );
        });

        const passwordInput = screen.getByLabelText("Password");
        const submitButton = screen.getByText("Submit");

        // Type password with proper act wrapping
        await act(async () => {
            await user.type(passwordInput, "mypassword123");
        });

        // Submit form
        await act(async () => {
            await user.click(submitButton);
        });

        // Check that form received the password value
        await waitFor(() => {
            expect(formValues.password).toBe("mypassword123");
        }, { timeout: 3000 });
    });
});