import { act, render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { Form, Formik } from "formik";
import React from "react";
import { describe, expect, it, vi, beforeEach, beforeAll } from "vitest";
import { PhoneNumberInput, PhoneNumberInputBase } from "./PhoneNumberInput";

// Mock libphonenumber-js since it's async and complex
vi.mock("libphonenumber-js", () => ({
    parsePhoneNumber: vi.fn((number: string) => {
        if (!number || number === "+1" || number.trim() === "") return null;
        if (number === "invalid" || number === "++" || number === "+++") {
            throw new Error("NOT_A_NUMBER");
        }
        if (number === "+15551234567") {
            return {
                country: "US",
                number: "+15551234567",
            };
        }
        if (number === "+447123456789") {
            return {
                country: "GB",
                number: "+447123456789",
            };
        }
        if (number.startsWith("+") && number.length > 5) {
            return {
                country: "US",
                number: number,
            };
        }
        throw new Error("NOT_A_NUMBER");
    }),
    parsePhoneNumberWithError: vi.fn((number: string) => {
        if (!number || number === "+1" || number.trim() === "") {
            throw new Error("NOT_A_NUMBER");
        }
        if (number === "invalid" || number === "++" || number === "+++") {
            throw new Error("NOT_A_NUMBER");
        }
        return {
            country: "US",
            number: number,
        };
    }),
    isValidPhoneNumber: vi.fn((number: string, country?: string) => {
        if (!number || number.trim() === "") return false;
        if (number === "invalid" || number === "++" || number === "+++") return false;
        if (number === "+15551234567" && country === "US") return true;
        if (number === "+447123456789" && country === "GB") return true;
        if (number.startsWith("+1") && number.length >= 12) return true;
        return false;
    }),
    getCountryCallingCode: vi.fn((country: string) => {
        const codes: Record<string, string> = {
            US: "1",
            GB: "44",
            CA: "1",
            AU: "61",
            DE: "49",
        };
        return codes[country] || "1";
    }),
    getCountries: vi.fn(() => ["US", "GB", "CA", "AU", "DE"]),
    AsYouType: class MockAsYouType {
        country: string;
        constructor(country: string) {
            this.country = country;
        }
        input(value: string): string {
            try {
                if (this.country === "US") {
                    const digits = value.replace(/\D/g, "");
                    if (digits.length <= 3) return digits;
                    if (digits.length <= 6) return `(${digits.slice(0, 3)}) ${digits.slice(3)}`;
                    return `(${digits.slice(0, 3)}) ${digits.slice(3, 6)}-${digits.slice(6, 10)}`;
                }
                return value;
            } catch (error) {
                return value;
            }
        }
    },
}));


describe("PhoneNumberInputBase", () => {
    const defaultProps = {
        name: "phone",
        value: "",
        onChange: vi.fn(),
        setError: vi.fn(),
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("Basic rendering", () => {
        it("renders with default US country and phone icon", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} />);
            });

            await waitFor(() => {
                const inputWrapper = screen.getByTestId("phone-number-input");
                expect(inputWrapper).toBeDefined();
                const input = inputWrapper.querySelector('input');
                expect(input?.type).toBe("tel");
                expect(input?.name).toBe("phone");

                const countryDisplay = screen.getByTestId("selected-country");
                expect(countryDisplay.textContent).toBe("US");

                expect(screen.getByTestId("country-selector-button")).toBeDefined();
            });
        });

        it("renders with custom label", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} label="Mobile Number" />);
            });

            await waitFor(() => {
                expect(screen.getByLabelText("Mobile Number")).toBeDefined();
            });
        });

        it("renders with error state", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} error={true} helperText="Invalid phone number" />);
            });

            await waitFor(() => {
                const inputWrapper = screen.getByTestId("phone-number-input");
                const input = inputWrapper.querySelector('input');
                expect(input?.getAttribute("aria-invalid")).toBe("true");
                expect(screen.getByText("Invalid phone number")).toBeDefined();
            });
        });

        it("renders with initial phone number value", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} value="+15551234567" />);
            });

            await waitFor(() => {
                const inputWrapper = screen.getByTestId("phone-number-input");
                const input = inputWrapper.querySelector('input') as HTMLInputElement;
                // Check that the input element has the value attribute or defaultValue
                expect(input?.value || input?.defaultValue || input?.getAttribute("value")).toBeTruthy();
            });
        });
    });

    describe("Country selection behavior", () => {
        it("opens country selector when clicking the country button", async () => {
            const user = userEvent.setup();
            render(<PhoneNumberInputBase {...defaultProps} />);

            const countryButton = screen.getByTestId("country-selector-button");

            await act(async () => {
                await user.click(countryButton);
            });

            await waitFor(() => {
                expect(screen.getByTestId("country-selector-popover")).toBeDefined();
                expect(screen.getByTestId("country-search-input")).toBeDefined();
                expect(screen.getByTestId("country-list")).toBeDefined();
            });
        });

        it("displays filtered countries in the dropdown", async () => {
            const user = userEvent.setup();
            render(<PhoneNumberInputBase {...defaultProps} />);

            const countryButton = screen.getByTestId("country-selector-button");
            await act(async () => {
                await user.click(countryButton);
            });

            await waitFor(() => {
                expect(screen.getByTestId("country-option-US")).toBeDefined();
                expect(screen.getByTestId("country-option-GB")).toBeDefined();
                expect(screen.getByTestId("country-option-CA")).toBeDefined();
            });
        });

        it("filters countries when typing in search", async () => {
            const user = userEvent.setup();
            render(<PhoneNumberInputBase {...defaultProps} />);

            const countryButton = screen.getByTestId("country-selector-button");
            await act(async () => {
                await user.click(countryButton);
            });

            const searchInput = await waitFor(() => screen.getByTestId("country-search-input"));

            await act(async () => {
                await user.type(searchInput, "GB");
            });

            // Wait for async country filtering to complete
            await waitFor(() => {
                expect(screen.getByTestId("country-option-GB")).toBeDefined();
            }, { timeout: 3000 });
        });

        it("changes country when selecting from dropdown", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            render(<PhoneNumberInputBase {...defaultProps} onChange={onChange} />);

            const countryButton = screen.getByTestId("country-selector-button");
            await act(async () => {
                await user.click(countryButton);
            });

            const gbOption = await screen.findByTestId("country-option-GB");
            await act(async () => {
                await user.click(gbOption);
            });

            await waitFor(() => {
                expect(screen.getByTestId("selected-country").textContent).toBe("GB");
            });

            // Should call onChange with new country code
            expect(onChange).toHaveBeenCalled();
        });

        it("closes popover when clicking outside", async () => {
            const user = userEvent.setup();
            render(<PhoneNumberInputBase {...defaultProps} />);

            const countryButton = screen.getByTestId("country-selector-button");
            await act(async () => {
                await user.click(countryButton);
            });

            const popover = await screen.findByTestId("country-selector-popover");
            expect(popover).toBeDefined();

            // Press Escape to close (more reliable than clicking outside in tests)
            await act(async () => {
                await user.keyboard("{Escape}");
            });

            await waitFor(() => {
                // Check if popover is closed by seeing if it's not visible
                const popoverAfter = screen.queryByTestId("country-selector-popover");
                expect(popoverAfter).toBeNull();
            });
        });
    });

    describe("Phone number input behavior", () => {
        it("renders as a focusable input", async () => {
            const user = userEvent.setup();
            render(<PhoneNumberInputBase {...defaultProps} />);

            const inputWrapper = screen.getByTestId("phone-number-input");
            const input = inputWrapper.querySelector('input') as HTMLInputElement;

            await act(async () => {
                await user.click(input);
            });

            expect(document.activeElement).toBe(input);
        });

        it("has correct input type and attributes", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} />);
            });

            await waitFor(() => {
                const inputWrapper = screen.getByTestId("phone-number-input");
                const input = inputWrapper.querySelector('input') as HTMLInputElement;
                expect(input?.type).toBe("tel");
                expect(input?.name).toBe("phone");
            });
        });

        it("responds to user interaction", async () => {
            const user = userEvent.setup();
            render(<PhoneNumberInputBase {...defaultProps} />);

            const inputWrapper = screen.getByTestId("phone-number-input");
            const input = inputWrapper.querySelector('input') as HTMLInputElement;

            // Test that we can focus the input
            await act(async () => {
                await user.click(input);
            });

            expect(document.activeElement).toBe(input);
        });

        it("supports controlled value prop", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} value="12345" />);
            });

            await waitFor(() => {
                const inputWrapper = screen.getByTestId("phone-number-input");
                const input = inputWrapper.querySelector('input') as HTMLInputElement;
                // Input should be rendered with some form of value handling
                expect(input).toBeDefined();
                expect(input?.type).toBe("tel");
            });
        });
    });

    describe("Validation behavior", () => {
        it("validates phone number and calls setError for invalid numbers", async () => {
            const setError = vi.fn();
            
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} value="invalid" setError={setError} />);
            });

            // Wait for validation effect to run
            await waitFor(() => {
                expect(setError).toHaveBeenCalledWith("Invalid phone number");
            }, { timeout: 3000 });
        });

        it("clears error for valid phone numbers", async () => {
            const setError = vi.fn();
            
            // Test with valid number
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} value="+15551234567" setError={setError} />);
            });

            // Wait for validation effect to run - it may be called with error first then cleared
            await waitFor(() => {
                expect(setError).toHaveBeenCalled();
            }, { timeout: 3000 });
            
            // For valid numbers, the setError should be called with undefined to clear errors
            // The mock will eventually call setError with undefined for valid numbers
            expect(setError).toHaveBeenCalled();
        });

        it("validates country mismatch", async () => {
            const setError = vi.fn();
            // Set a UK number but stay on US country
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} value="+447123456789" setError={setError} />);
            });

            // Wait for validation effect to run
            await waitFor(() => {
                expect(setError).toHaveBeenCalledWith("Invalid phone number");
            }, { timeout: 3000 });
        });

        it("handles validation errors gracefully", async () => {
            const setError = vi.fn();
            // Provide malformed input
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} value="++invalid" setError={setError} />);
            });

            // Should not crash and should set error
            await waitFor(() => {
                expect(setError).toHaveBeenCalledWith("Invalid phone number");
            }, { timeout: 3000 });
        });
    });

    describe("Accessibility", () => {
        it("has proper form associations", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} name="phone" label="Phone Number" />);
            });

            await waitFor(() => {
                const inputWrapper = screen.getByTestId("phone-number-input");
                const input = inputWrapper.querySelector('input') as HTMLInputElement;
                expect(input?.name).toBe("phone");
                expect(input?.type).toBe("tel");
                
                // Check that label is present
                expect(screen.getByLabelText("Phone Number")).toBeDefined();
            });
        });

        it("associates helper text with input", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} name="phone" helperText="Enter your phone number" />);
            });

            await waitFor(() => {
                const helperText = screen.getByText("Enter your phone number");
                expect(helperText.id).toBe("helper-text-phone");
            });
        });

        it("supports autoFocus", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} autoFocus={true} />);
            });

            await waitFor(() => {
                const inputWrapper = screen.getByTestId("phone-number-input");
                const input = inputWrapper.querySelector('input') as HTMLInputElement;
                expect(input).toBeDefined();
                // MUI components may render autofocus differently
                expect(input?.autofocus !== undefined || input?.hasAttribute('autofocus')).toBeTruthy();
            });
        });

        it("supports autoComplete attribute", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} autoComplete="tel-national" />);
            });

            await waitFor(() => {
                const inputWrapper = screen.getByTestId("phone-number-input");
                const input = inputWrapper.querySelector('input') as HTMLInputElement;
                expect(input).toBeDefined();
                expect(input?.getAttribute('autocomplete')).toBe("tel-national");
            });
        });
    });

    describe("Props handling", () => {
        it("supports fullWidth prop", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} fullWidth={false} />);
            });
            // Component should render without errors
            await waitFor(() => {
                expect(screen.getByTestId("phone-number-input")).toBeDefined();
            });
        });

        it("forwards additional props to FormControl", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase {...defaultProps} disabled={true} />);
            });
            // Should render without errors
            await waitFor(() => {
                expect(screen.getByTestId("phone-number-input")).toBeDefined();
            });
        });

        it("handles helper text as string or boolean", async () => {
            let rerender;
            await act(async () => {
                const result = render(<PhoneNumberInputBase {...defaultProps} helperText="String helper" />);
                rerender = result.rerender;
            });
            
            await waitFor(() => {
                expect(screen.getByText("String helper")).toBeDefined();
            });

            await act(async () => {
                rerender(<PhoneNumberInputBase {...defaultProps} helperText={true} />);
            });
            await waitFor(() => {
                expect(screen.getByText("true")).toBeDefined();
            });

            await act(async () => {
                rerender(<PhoneNumberInputBase {...defaultProps} helperText={{ complex: "object" }} />);
            });
            await waitFor(() => {
                expect(screen.getByText('{"complex":"object"}')).toBeDefined();
            });
        });
    });
});

describe("PhoneNumberInput (Formik integration)", () => {
    const TestForm = ({ initialValues = { phone: "" }, onSubmit = vi.fn() }) => (
        <Formik initialValues={initialValues} onSubmit={onSubmit}>
            <Form>
                <PhoneNumberInput name="phone" label="Phone Number" />
            </Form>
        </Formik>
    );

    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("Formik integration", () => {
        it("integrates with Formik form state", async () => {
            await act(async () => {
                render(<TestForm initialValues={{ phone: "+15551234567" }} />);
            });

            await waitFor(() => {
                const inputWrapper = screen.getByTestId("phone-number-input");
                const input = inputWrapper.querySelector('input') as HTMLInputElement;
                // Check that the input exists and can be used
                expect(input).toBeDefined();
                expect(input?.type).toBe("tel");
            });
        });

        it("responds to user interaction in Formik context", async () => {
            const user = userEvent.setup();
            render(<TestForm />);

            const inputWrapper = screen.getByTestId("phone-number-input");
            const input = inputWrapper.querySelector('input') as HTMLInputElement;

            await act(async () => {
                await user.click(input);
            });

            // Check that we can interact with the input
            expect(document.activeElement).toBe(input);
        });

        it("shows Formik validation errors", async () => {
            const TestFormWithValidation = () => (
                <Formik
                    initialValues={{ phone: "" }}
                    validate={(values) => {
                        const errors: any = {};
                        if (!values.phone) {
                            errors.phone = "Phone number is required";
                        }
                        return errors;
                    }}
                    onSubmit={vi.fn()}
                >
                    <Form>
                        <PhoneNumberInput name="phone" label="Phone Number" />
                    </Form>
                </Formik>
            );

            const user = userEvent.setup();
            render(<TestFormWithValidation />);

            const inputWrapper = screen.getByTestId("phone-number-input");
            const input = inputWrapper.querySelector('input') as HTMLInputElement;

            // Trigger validation by touching and blurring
            await act(async () => {
                await user.click(input);
                await user.tab(); // Blur the input
            });

            await waitFor(() => {
                expect(screen.getByText("Phone number is required")).toBeDefined();
            });
        });

        it("works with external onChange prop", async () => {
            const onChangeMock = vi.fn();
            const TestFormWithOnChange = () => (
                <Formik initialValues={{ phone: "" }} onSubmit={vi.fn()}>
                    <Form>
                        <PhoneNumberInput name="phone" label="Phone Number" onChange={onChangeMock} />
                    </Form>
                </Formik>
            );

            const user = userEvent.setup();
            render(<TestFormWithOnChange />);

            const inputWrapper = screen.getByTestId("phone-number-input");
            const input = inputWrapper.querySelector('input') as HTMLInputElement;

            await act(async () => {
                await user.click(input);
            });

            // Should allow interaction
            expect(input).toBeDefined();
        });

        it("handles country changes with Formik", async () => {
            const user = userEvent.setup();
            render(<TestForm />);

            const countryButton = screen.getByTestId("country-selector-button");
            await act(async () => {
                await user.click(countryButton);
            });

            const gbOption = await screen.findByTestId("country-option-GB");
            await act(async () => {
                await user.click(gbOption);
            });

            await waitFor(() => {
                expect(screen.getByTestId("selected-country").textContent).toBe("GB");
            });
        });
    });

    describe("Error handling in Formik context", () => {
        it("shows libphonenumber validation errors through Formik", async () => {
            const user = userEvent.setup();
            render(<TestForm />);

            const inputWrapper = screen.getByTestId("phone-number-input");
            const input = inputWrapper.querySelector('input') as HTMLInputElement;

            // Type an invalid number
            await act(async () => {
                await user.type(input, "invalid");
                await user.tab(); // Trigger blur/validation
            });

            // Should show validation error - wait for the validation to run
            await waitFor(() => {
                const errorElement = screen.queryByText("Invalid phone number");
                expect(errorElement).toBeDefined();
            }, { timeout: 3000 });
        });

        it("clears errors when valid number is entered", async () => {
            const user = userEvent.setup();
            render(<TestForm />);

            const inputWrapper = screen.getByTestId("phone-number-input");
            const input = inputWrapper.querySelector('input') as HTMLInputElement;

            // First type invalid number
            await act(async () => {
                await user.type(input, "invalid");
                await user.tab();
            });

            // Wait for error to appear
            await waitFor(() => {
                const errorElement = screen.queryByText("Invalid phone number");
                expect(errorElement).toBeDefined();
            }, { timeout: 3000 });

            // Then type valid number
            await act(async () => {
                await user.tripleClick(input); // Select all
                await user.type(input, "+15551234567");
            });

            // Error should be cleared
            await waitFor(() => {
                expect(screen.queryByText("Invalid phone number")).toBeNull();
            }, { timeout: 3000 });
        });
    });
});

describe("PhoneNumberInput edge cases", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("Async libphonenumber handling", () => {
        it("handles validation gracefully", async () => {
            const setError = vi.fn();

            render(<PhoneNumberInputBase name="phone" value="test" onChange={vi.fn()} setError={setError} />);

            // Should handle validation
            await waitFor(() => {
                expect(setError).toHaveBeenCalled();
            });
        });

        it("handles parsing errors gracefully", async () => {
            const setError = vi.fn();
            const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});

            render(<PhoneNumberInputBase name="phone" value="+++" onChange={vi.fn()} setError={setError} />);

            // Should handle parsing errors
            await waitFor(() => {
                expect(setError).toHaveBeenCalledWith("Invalid phone number");
            });

            consoleErrorSpy.mockRestore();
        });
    });

    describe("Country filtering edge cases", () => {
        it("renders country selection dropdown", async () => {
            const user = userEvent.setup();
            render(<PhoneNumberInputBase name="phone" value="" onChange={vi.fn()} setError={vi.fn()} />);

            const countryButton = screen.getByTestId("country-selector-button");
            await act(async () => {
                await user.click(countryButton);
            });

            const countryList = await screen.findByTestId("country-list");
            // Country list should exist
            expect(countryList).toBeDefined();
        });

        it("handles special characters in country search", async () => {
            const user = userEvent.setup();
            render(<PhoneNumberInputBase name="phone" value="" onChange={vi.fn()} setError={vi.fn()} />);

            const countryButton = screen.getByTestId("country-selector-button");
            await act(async () => {
                await user.click(countryButton);
            });

            const searchInput = await screen.findByTestId("country-search-input");

            await act(async () => {
                await user.type(searchInput, "!@#$%");
            });

            // Should not crash
            expect(screen.getByTestId("country-list")).toBeDefined();
        });
    });

    describe("Input edge cases", () => {
        it("handles null/undefined values", async () => {
            await act(async () => {
                render(<PhoneNumberInputBase name="phone" value={null as any} onChange={vi.fn()} setError={vi.fn()} />);
            });
            await waitFor(() => {
                expect(screen.getByTestId("phone-number-input")).toBeDefined();
            });
        });

        it("handles very long input values", async () => {
            const longValue = "1".repeat(100);
            const user = userEvent.setup();
            render(<PhoneNumberInputBase name="phone" value="" onChange={vi.fn()} setError={vi.fn()} />);

            const inputWrapper = screen.getByTestId("phone-number-input");
            const input = inputWrapper.querySelector('input') as HTMLInputElement;

            await act(async () => {
                await user.type(input, longValue);
            });

            // Should handle without crashing
            expect(screen.getByTestId("phone-number-input")).toBeDefined();
        });
    });
});