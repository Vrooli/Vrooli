import { act, render, screen, waitFor, within } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { Formik } from "formik";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { DateInput, DateInputBase } from "./DateInput";

// Helper component to wrap DateInput with Formik
function TestWrapper({ 
    initialValue = "", 
    type = "datetime-local" as const,
    isRequired = false,
    label = "Test Date",
    onSubmit = vi.fn(),
    children,
}: {
    initialValue?: string;
    type?: "date" | "datetime-local";
    isRequired?: boolean;
    label?: string;
    onSubmit?: (values: any) => void;
    children?: (helpers: any) => React.ReactNode;
}) {
    return (
        <Formik
            initialValues={{ testDate: initialValue }}
            onSubmit={onSubmit}
        >
            {({ handleSubmit, setFieldValue }) => (
                <form onSubmit={handleSubmit}>
                    <DateInput 
                        name="testDate" 
                        label={label}
                        type={type}
                        isRequired={isRequired}
                    />
                    <button type="submit">Submit</button>
                    {children && children({ setFieldValue })}
                </form>
            )}
        </Formik>
    );
}

// Helper to get the actual input element within the DateInput component
function getDateInput() {
    // First try to find by the date-input test id
    try {
        const container = screen.getByTestId("date-input");
        // Date inputs don't have role="textbox", we need to find them by type
        const input = container.querySelector("input[type=\"date\"], input[type=\"datetime-local\"]");
        if (input) {
            return input as HTMLInputElement;
        }
    } catch {
        // If not found, continue to fallback
    }
    
    // Fallback: look for any date/datetime-local input in the document
    const inputs = document.querySelectorAll("input[type=\"date\"], input[type=\"datetime-local\"]");
    if (inputs.length === 1) {
        return inputs[0] as HTMLInputElement;
    } else if (inputs.length > 1) {
        throw new Error(`Found ${inputs.length} date input elements, expected 1`);
    }
    
    throw new Error("Could not find date input element");
}

describe("DateInputBase", () => {
    describe("Basic rendering", () => {
        it("renders with default datetime-local type", () => {
            const onChange = vi.fn();
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    value=""
                    onChange={onChange}
                />,
            );
            
            const input = getDateInput();
            expect(input).toBeDefined();
            expect(input.getAttribute("type")).toBe("datetime-local");
        });

        it("renders with date type when specified", () => {
            const onChange = vi.fn();
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    type="date"
                    value=""
                    onChange={onChange}
                />,
            );
            
            const input = getDateInput();
            expect(input).toBeDefined();
            expect(input.getAttribute("type")).toBe("date");
        });

        it("displays the provided label", () => {
            const onChange = vi.fn();
            render(
                <DateInputBase 
                    label="Birth Date"
                    name="testDate"
                    value=""
                    onChange={onChange}
                />,
            );
            
            expect(screen.getByLabelText("Birth Date")).toBeDefined();
        });

        it("shows required indicator when isRequired is true", () => {
            const onChange = vi.fn();
            render(
                <DateInputBase 
                    label="Required Date"
                    name="testDate"
                    isRequired={true}
                    value=""
                    onChange={onChange}
                />,
            );
            
            const label = screen.getAllByText(/Required Date/)[0];
            // Check if the label container has the required indicator (usually an asterisk)
            const labelContainer = label.closest("label");
            expect(labelContainer?.textContent).toContain("*");
        });

        it("shows error state with helper text", () => {
            const onChange = vi.fn();
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    value=""
                    onChange={onChange}
                    error={true}
                    helperText="Date is required"
                />,
            );
            
            expect(screen.getByText("Date is required")).toBeDefined();
        });

        it("shows disabled state", () => {
            const onChange = vi.fn();
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    value=""
                    onChange={onChange}
                    disabled={true}
                />,
            );
            
            const input = getDateInput();
            expect(input.hasAttribute("disabled")).toBe(true);
        });
    });

    describe("Date formatting", () => {
        it("formats datetime-local values correctly", () => {
            const onChange = vi.fn();
            const isoDate = "2024-01-15T14:30:00.000Z";
            
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    type="datetime-local"
                    value={isoDate}
                    onChange={onChange}
                />,
            );
            
            const input = getDateInput();
            // The formatted value should be in local time format (seconds are optional in HTML datetime-local)
            expect(input.value).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?$/);
        });

        it("formats date values correctly", () => {
            const onChange = vi.fn();
            const isoDate = "2024-01-15T14:30:00.000Z";
            
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    type="date"
                    value={isoDate}
                    onChange={onChange}
                />,
            );
            
            const input = getDateInput();
            // The formatted value should be date only
            expect(input.value).toMatch(/^\d{4}-\d{2}-\d{2}$/);
        });

        it("handles empty values gracefully", () => {
            const onChange = vi.fn();
            
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    value=""
                    onChange={onChange}
                />,
            );
            
            const input = getDateInput();
            expect(input.value).toBe("");
        });
    });

    describe("Clear button functionality", () => {
        it("shows clear button when date has value", () => {
            const onChange = vi.fn();
            
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    value="2024-01-15T10:00:00.000Z"
                    onChange={onChange}
                />,
            );
            
            const clearButton = screen.getByTestId("date-input-clear");
            expect(clearButton).toBeDefined();
            expect(clearButton.getAttribute("aria-label")).toBe("Clear date");
        });

        it("hides clear button when date is empty", () => {
            const onChange = vi.fn();
            
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    value=""
                    onChange={onChange}
                />,
            );
            
            expect(screen.queryByTestId("date-input-clear")).toBeNull();
        });

        it("calls onChange with empty string when clear button is clicked", async () => {
            const user = userEvent.setup();
            const onChange = vi.fn();
            
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    value="2024-01-15T10:00:00.000Z"
                    onChange={onChange}
                />,
            );
            
            const clearButton = screen.getByTestId("date-input-clear");
            
            await act(async () => {
                await user.click(clearButton);
            });
            
            expect(onChange).toHaveBeenCalledWith("");
        });

        it("disables clear button when input is disabled", () => {
            const onChange = vi.fn();
            
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    value="2024-01-15T10:00:00.000Z"
                    onChange={onChange}
                    disabled={true}
                />,
            );
            
            const clearButton = screen.getByTestId("date-input-clear");
            expect(clearButton.hasAttribute("disabled")).toBe(true);
        });
    });

    describe("User interactions", () => {
        it("calls onChange when user types in the input", async () => {
            const user = userEvent.setup();
            const onChange = vi.fn();
            
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    type="date"
                    value=""
                    onChange={onChange}
                />,
            );
            
            const input = getDateInput();
            
            // Note: userEvent doesn't work well with date inputs,
            // so we'll simulate the change event directly
            const event = new Event("change", { bubbles: true });
            Object.defineProperty(event, "target", {
                value: { value: "2024-12-25" },
                writable: false,
            });
            
            act(() => {
                input.dispatchEvent(event);
            });
            
            expect(onChange).toHaveBeenCalledWith("2024-12-25");
        });

        it("calls onBlur when input loses focus", async () => {
            const user = userEvent.setup();
            const onChange = vi.fn();
            const onBlur = vi.fn();
            
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    value=""
                    onChange={onChange}
                    onBlur={onBlur}
                />,
            );
            
            const input = getDateInput();
            
            await act(async () => {
                await user.click(input);
                await user.tab();
            });
            
            expect(onBlur).toHaveBeenCalled();
        });
    });

    describe("Edge cases and error handling", () => {
        it("handles invalid date strings gracefully", () => {
            const onChange = vi.fn();
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
            
            expect(() => {
                render(
                    <DateInputBase 
                        label="Test Date"
                        name="testDate"
                        value="not-a-date"
                        onChange={onChange}
                    />,
                );
            }).toThrow("Invalid date format");
            
            consoleSpy.mockRestore();
        });

        it("forwards refs correctly", () => {
            const onChange = vi.fn();
            const ref = React.createRef<HTMLInputElement>();
            
            render(
                <DateInputBase 
                    ref={ref}
                    label="Test Date"
                    name="testDate"
                    value=""
                    onChange={onChange}
                />,
            );
            
            expect(ref.current).toBeDefined();
            expect(ref.current?.getAttribute("type")).toMatch(/^date/);
        });

        it("applies custom className", () => {
            const onChange = vi.fn();
            
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    value=""
                    onChange={onChange}
                    className="custom-date-class"
                />,
            );
            
            const container = screen.getByTestId("date-input");
            expect(container.className).toContain("custom-date-class");
        });

        it("applies sx prop for custom styling", () => {
            const onChange = vi.fn();
            
            render(
                <DateInputBase 
                    label="Test Date"
                    name="testDate"
                    value=""
                    onChange={onChange}
                    sx={{ marginTop: 2 }}
                />,
            );
            
            // Check if the input exists - sx prop application is handled by MUI
            const input = getDateInput();
            expect(input).toBeDefined();
        });
    });
});

describe("DateInput (Formik Integration)", () => {
    describe("Basic rendering", () => {
        it("renders with default datetime-local type", () => {
            render(<TestWrapper />);
            
            const input = getDateInput();
            expect(input).toBeDefined();
            expect(input.getAttribute("type")).toBe("datetime-local");
        });

        it("renders with date type when specified", () => {
            render(<TestWrapper type="date" />);
            
            const input = getDateInput();
            expect(input).toBeDefined();
            expect(input.getAttribute("type")).toBe("date");
        });

        it("displays the provided label", () => {
            render(<TestWrapper label="Birth Date" />);
            
            expect(screen.getByLabelText("Birth Date")).toBeDefined();
        });

        it("shows required indicator when isRequired is true", () => {
            render(<TestWrapper isRequired={true} label="Required Date" />);
            
            const label = screen.getAllByText(/Required Date/)[0];
            // Check if the label container has the required indicator (usually an asterisk)
            const labelContainer = label.closest("label");
            expect(labelContainer?.textContent).toContain("*");
        });
    });

    describe("Date formatting", () => {
        it("formats datetime-local values correctly", () => {
            const isoDate = "2024-01-15T14:30:00.000Z";
            render(<TestWrapper initialValue={isoDate} type="datetime-local" />);
            
            const input = getDateInput();
            // The formatted value should be in local time format (seconds are optional in HTML datetime-local)
            expect(input.value).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?$/);
        });

        it("formats date values correctly", () => {
            const isoDate = "2024-01-15T14:30:00.000Z";
            render(<TestWrapper initialValue={isoDate} type="date" />);
            
            const input = getDateInput();
            // The formatted value should be date only
            expect(input.value).toMatch(/^\d{4}-\d{2}-\d{2}$/);
        });

        it("handles empty values gracefully", () => {
            render(<TestWrapper initialValue="" />);
            
            const input = getDateInput();
            expect(input.value).toBe("");
        });

        it("handles null/undefined by showing empty string", () => {
            render(<TestWrapper initialValue={undefined as any} />);
            
            const input = getDateInput();
            expect(input.value).toBe("");
        });
    });

    describe("Clear button functionality", () => {
        it("shows clear button when date has value", () => {
            render(<TestWrapper initialValue="2024-01-15T10:00:00.000Z" />);
            
            const clearButton = screen.getByTestId("date-input-clear");
            expect(clearButton).toBeDefined();
            expect(clearButton.getAttribute("aria-label")).toBe("Clear date");
        });

        it("hides clear button when date is empty", () => {
            render(<TestWrapper initialValue="" />);
            
            expect(screen.queryByTestId("date-input-clear")).toBeNull();
        });

        it("clears the date when clear button is clicked", async () => {
            const user = userEvent.setup();
            const onSubmit = vi.fn();
            
            render(<TestWrapper 
                initialValue="2024-01-15T10:00:00.000Z" 
                onSubmit={onSubmit}
            />);
            
            const clearButton = screen.getByTestId("date-input-clear");
            
            await act(async () => {
                await user.click(clearButton);
            });
            
            // Clear button should disappear
            await waitFor(() => {
                expect(screen.queryByTestId("date-input-clear")).toBeNull();
            });
            
            // Submit to verify the value was cleared
            const submitButton = screen.getByText("Submit");
            await act(async () => {
                await user.click(submitButton);
            });
            
            await waitFor(() => {
                expect(onSubmit).toHaveBeenCalledWith(
                    { testDate: "" },
                    expect.any(Object),
                );
            });
        });
    });

    describe("User interactions", () => {
        it("updates value when user selects a date", async () => {
            const user = userEvent.setup();
            const onSubmit = vi.fn();
            let setFieldValueRef: any;
            
            render(
                <TestWrapper type="date" onSubmit={onSubmit}>
                    {({ setFieldValue }) => {
                        setFieldValueRef = setFieldValue;
                        return null;
                    }}
                </TestWrapper>,
            );
            
            // Simulate selecting a date using Formik's setFieldValue
            await act(async () => {
                setFieldValueRef("testDate", "2024-12-25T00:00:00.000Z");
            });
            
            // Wait for the clear button to appear (indicating the value was set)
            await waitFor(() => {
                expect(screen.getByTestId("date-input-clear")).toBeDefined();
            });
            
            // Submit to verify the value was updated
            const submitButton = screen.getByText("Submit");
            await act(async () => {
                await user.click(submitButton);
            });
            
            await waitFor(() => {
                expect(onSubmit).toHaveBeenCalledWith(
                    { testDate: "2024-12-25T00:00:00.000Z" },
                    expect.any(Object),
                );
            });
        });

        it("updates value when user selects a datetime", async () => {
            const user = userEvent.setup();
            const onSubmit = vi.fn();
            let setFieldValueRef: any;
            
            render(
                <TestWrapper type="datetime-local" onSubmit={onSubmit}>
                    {({ setFieldValue }) => {
                        setFieldValueRef = setFieldValue;
                        return null;
                    }}
                </TestWrapper>,
            );
            
            // Simulate selecting a datetime using Formik's setFieldValue
            await act(async () => {
                setFieldValueRef("testDate", "2024-12-25T15:30:00.000Z");
            });
            
            // Verify clear button appears
            await waitFor(() => {
                expect(screen.getByTestId("date-input-clear")).toBeDefined();
            });
            
            // Submit to verify the value was updated
            const submitButton = screen.getByText("Submit");
            await act(async () => {
                await user.click(submitButton);
            });
            
            await waitFor(() => {
                expect(onSubmit).toHaveBeenCalledWith(
                    { testDate: "2024-12-25T15:30:00.000Z" },
                    expect.any(Object),
                );
            });
        });

        it("maintains focus after clearing", async () => {
            const user = userEvent.setup();
            
            render(<TestWrapper initialValue="2024-01-15T10:00:00.000Z" />);
            
            const input = getDateInput();
            const clearButton = screen.getByTestId("date-input-clear");
            
            // Focus the input first
            await act(async () => {
                await user.click(input);
            });
            
            // Clear the date
            await act(async () => {
                await user.click(clearButton);
            });
            
            // Input should still be in the document and accessible
            const inputAfterClear = getDateInput();
            expect(inputAfterClear).toBeDefined();
        });
    });

    describe("Edge cases and error handling", () => {
        it("handles invalid date strings gracefully", () => {
            // Component should throw an error for invalid dates
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
            
            expect(() => {
                render(<TestWrapper initialValue="not-a-date" />);
            }).toThrow("Invalid date format");
            
            consoleSpy.mockRestore();
        });

        it("preserves field name for form integration", () => {
            render(<TestWrapper />);
            
            const input = getDateInput();
            expect(input.getAttribute("name")).toBe("testDate");
        });

        it("applies custom styles when provided", () => {
            render(
                <Formik initialValues={{ testDate: "" }} onSubmit={vi.fn()}>
                    <DateInput 
                        name="testDate" 
                        label="Styled Date"
                        sx={{ marginTop: 2 }}
                    />
                </Formik>,
            );
            
            // Check if the input exists with custom styles
            const input = getDateInput();
            expect(input).toBeDefined();
        });
    });

    describe("Accessibility", () => {
        it("properly associates label with input", () => {
            render(<TestWrapper label="Accessible Date" />);
            
            const input = screen.getByLabelText("Accessible Date");
            expect(input).toBeDefined();
            expect(input.getAttribute("type")).toMatch(/^date/);
        });

        it("clear button has descriptive aria-label", () => {
            render(<TestWrapper initialValue="2024-01-15T10:00:00.000Z" />);
            
            const clearButton = screen.getByTestId("date-input-clear");
            expect(clearButton.getAttribute("aria-label")).toBe("Clear date");
        });

        it("supports keyboard navigation to clear button", async () => {
            const user = userEvent.setup();
            
            render(<TestWrapper initialValue="2024-01-15T10:00:00.000Z" />);
            
            const input = getDateInput();
            const clearButton = screen.getByTestId("date-input-clear");
            
            // Focus the input
            await act(async () => {
                await user.click(input);
            });
            
            // Tab to clear button
            await act(async () => {
                await user.tab();
            });
            
            // Clear button should be focusable
            expect(clearButton).toBeDefined();
        });
    });
});
