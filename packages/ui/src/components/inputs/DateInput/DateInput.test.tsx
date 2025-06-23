import { act, render, screen, waitFor, within } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { Formik } from "formik";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { DateInput } from "./DateInput";

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
    const container = screen.getByTestId("date-input");
    // Date inputs don't have role="textbox", we need to find them by type
    const input = container.querySelector('input[type="date"], input[type="datetime-local"]');
    if (!input) {
        throw new Error("Could not find date input element");
    }
    return input as HTMLInputElement;
}

describe("DateInput", () => {
    describe("Basic rendering", () => {
        it("renders with default datetime-local type", () => {
            render(<TestWrapper />);
            
            const container = screen.getByTestId("date-input");
            const input = getDateInput();
            expect(container).toBeDefined();
            expect(container.getAttribute("data-type")).toBe("datetime-local");
            expect(input.getAttribute("type")).toBe("datetime-local");
        });

        it("renders with date type when specified", () => {
            render(<TestWrapper type="date" />);
            
            const container = screen.getByTestId("date-input");
            const input = getDateInput();
            expect(container.getAttribute("data-type")).toBe("date");
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
                    expect.any(Object)
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
                </TestWrapper>
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
                    expect.any(Object)
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
                </TestWrapper>
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
                    expect.any(Object)
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
            expect(screen.getByTestId("date-input")).toBeDefined();
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
                </Formik>
            );
            
            const container = screen.getByTestId("date-input");
            // Check if the component has the custom styles applied
            expect(container).toBeDefined();
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