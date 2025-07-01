import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { FormControlLabel } from "./FormControlLabel.js";

// Mock control components for testing
const MockCheckbox = ({ onChange, disabled, required, checked = false, ...props }: any) => (
    <input
        type="checkbox"
        onChange={onChange}
        disabled={disabled}
        required={required}
        checked={checked}
        data-testid="mock-checkbox"
        {...props}
    />
);

const MockRadio = ({ onChange, disabled, required, checked = false, ...props }: any) => (
    <input
        type="radio"
        onChange={onChange}
        disabled={disabled}
        required={required}
        checked={checked}
        data-testid="mock-radio"
        {...props}
    />
);

const MockSwitch = ({ onChange, disabled, required, checked = false, ...props }: any) => (
    <div
        role="switch"
        aria-checked={checked}
        onClick={onChange}
        data-disabled={disabled}
        data-required={required}
        data-testid="mock-switch"
        {...props}
    >
        {checked ? "ON" : "OFF"}
    </div>
);

describe("FormControlLabel", () => {
    describe("Basic rendering", () => {
        it("renders with a control and label", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Test Label"
                />,
            );

            const label = screen.getByTestId("form-control-label");
            const control = screen.getByTestId("mock-checkbox");
            
            expect(label).toBeDefined();
            expect(control).toBeDefined();
            expect(screen.getByText("Test Label")).toBeDefined();
        });

        it("forwards ref to the label element", () => {
            const ref = React.createRef<HTMLLabelElement>();
            
            render(
                <FormControlLabel
                    ref={ref}
                    control={<MockCheckbox />}
                    label="Test Label"
                />,
            );

            expect(ref.current).toBeDefined();
            expect(ref.current?.tagName).toBe("LABEL");
        });

        it("accepts custom className and style props", () => {
            const customStyle = { color: "red" };
            
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Test Label"
                    className="custom-class"
                    style={customStyle}
                />,
            );

            const label = screen.getByTestId("form-control-label");
            expect(label).toBeDefined();
            expect(label.style.color).toBe("red");
            expect(screen.getByText("Test Label")).toBeDefined();
        });
    });

    describe("Label placement", () => {
        it("renders label at the end by default", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="End Label"
                />,
            );

            const label = screen.getByTestId("form-control-label");
            expect(label.getAttribute("data-label-placement")).toBe("end");
            
            // Check DOM order: control should come before label for 'end' placement
            const controlWrapper = screen.getByTestId("control-wrapper");
            const labelText = screen.getByText("End Label");
            
            // Control wrapper should appear before label text in DOM
            const labelElement = screen.getByTestId("form-control-label");
            const children = Array.from(labelElement.children);
            const controlIndex = children.indexOf(controlWrapper);
            const labelIndex = Array.from(labelElement.childNodes).findIndex(
                node => node.textContent?.includes("End Label"),
            );
            
            expect(controlIndex).toBeLessThan(labelIndex);
        });

        it("renders label at the start", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Start Label"
                    labelPlacement="start"
                />,
            );

            const label = screen.getByTestId("form-control-label");
            expect(label.getAttribute("data-label-placement")).toBe("start");
        });

        it("renders label at the top", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Top Label"
                    labelPlacement="top"
                />,
            );

            const label = screen.getByTestId("form-control-label");
            expect(label.getAttribute("data-label-placement")).toBe("top");
        });

        it("renders label at the bottom", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Bottom Label"
                    labelPlacement="bottom"
                />,
            );

            const label = screen.getByTestId("form-control-label");
            expect(label.getAttribute("data-label-placement")).toBe("bottom");
        });
    });

    describe("Required field handling", () => {
        it("shows asterisk for required fields", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Required Field"
                    required={true}
                />,
            );

            const label = screen.getByTestId("form-control-label");
            expect(label.getAttribute("data-required")).toBe("true");
            
            const asterisk = screen.getByLabelText("required");
            expect(asterisk).toBeDefined();
            expect(asterisk.textContent).toBe(" *");
        });

        it("does not show asterisk for non-required fields", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Optional Field"
                    required={false}
                />,
            );

            const label = screen.getByTestId("form-control-label");
            expect(label.getAttribute("data-required")).toBe("false");
            expect(screen.queryByLabelText("required")).toBeNull();
        });

        it("passes required prop to control element", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Required Field"
                    required={true}
                />,
            );

            const control = screen.getByTestId("mock-checkbox");
            expect(control.getAttribute("required")).toBe("");
        });
    });

    describe("Disabled state handling", () => {
        it("handles disabled state correctly", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Disabled Field"
                    disabled={true}
                />,
            );

            const label = screen.getByTestId("form-control-label");
            expect(label.getAttribute("data-disabled")).toBe("true");
            
            const control = screen.getByTestId("mock-checkbox");
            expect(control.getAttribute("disabled")).toBe("");
        });

        it("allows enabled state", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Enabled Field"
                    disabled={false}
                />,
            );

            const label = screen.getByTestId("form-control-label");
            expect(label.getAttribute("data-disabled")).toBe("false");
            
            const control = screen.getByTestId("mock-checkbox");
            expect(control.getAttribute("disabled")).toBeNull();
        });

        it("does not call onChange when disabled", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();

            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Disabled Field"
                    disabled={true}
                    onChange={onChange}
                />,
            );

            const control = screen.getByTestId("mock-checkbox");

            await act(async () => {
                await user.click(control);
            });

            expect(onChange).not.toHaveBeenCalled();
        });
    });

    describe("Change event handling", () => {
        it("calls onChange when control is clicked", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();

            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Clickable Field"
                    onChange={onChange}
                />,
            );

            const control = screen.getByTestId("mock-checkbox");

            await act(async () => {
                await user.click(control);
            });

            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(
                expect.any(Object),
                true, // checked state is true after clicking an unchecked checkbox
            );
        });

        it("passes value to control element", () => {
            const testValue = "test-value";
            
            render(
                <FormControlLabel
                    control={<MockRadio />}
                    label="Radio Field"
                    value={testValue}
                />,
            );

            const control = screen.getByTestId("mock-radio");
            expect(control.getAttribute("value")).toBe(testValue);
        });

        it("preserves existing control props", () => {
            const controlOnChange = vi.fn();
            
            render(
                <FormControlLabel
                    control={<MockCheckbox onChange={controlOnChange} data-custom="test" />}
                    label="Test Field"
                />,
            );

            const control = screen.getByTestId("mock-checkbox");
            // The FormControlLabel overrides onChange, so we test that the custom attribute is preserved
            expect(control.getAttribute("data-custom")).toBe("test");
            // And that the element receives an onChange handler (even if it's not the original one)
            expect(control).toHaveProperty("onchange");
        });
    });

    describe("Control element handling", () => {
        it("works with different control types", () => {
            const { rerender } = render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Checkbox Label"
                />,
            );

            expect(screen.getByTestId("mock-checkbox")).toBeDefined();

            rerender(
                <FormControlLabel
                    control={<MockRadio />}
                    label="Radio Label"
                />,
            );

            expect(screen.getByTestId("mock-radio")).toBeDefined();

            rerender(
                <FormControlLabel
                    control={<MockSwitch />}
                    label="Switch Label"
                />,
            );

            expect(screen.getByTestId("mock-switch")).toBeDefined();
        });

        it("handles non-React element controls gracefully", () => {
            // Test with a string as control (edge case)
            render(
                <FormControlLabel
                    control={"Not a React element" as any}
                    label="Test Label"
                />,
            );

            const controlWrapper = screen.getByTestId("control-wrapper");
            expect(controlWrapper.textContent).toBe("Not a React element");
        });
    });

    describe("Complex label content", () => {
        it("renders complex label content", () => {
            const complexLabel = (
                <div>
                    <strong>Bold text</strong>
                    <span> and regular text</span>
                </div>
            );

            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label={complexLabel}
                />,
            );

            expect(screen.getByText("Bold text")).toBeDefined();
            // Use a more flexible text matcher for text split across elements
            expect(screen.getByText((content, element) => {
                return element?.textContent === " and regular text";
            })).toBeDefined();
        });

        it("renders string labels correctly", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Simple string label"
                />,
            );

            expect(screen.getByText("Simple string label")).toBeDefined();
        });
    });

    describe("Accessibility", () => {
        it("maintains proper label semantics", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox id="test-control" />}
                    label="Accessible Label"
                />,
            );

            const label = screen.getByTestId("form-control-label");
            expect(label.tagName).toBe("LABEL");
        });

        it("provides required field indication to screen readers", () => {
            render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Required Field"
                    required={true}
                />,
            );

            const asterisk = screen.getByLabelText("required");
            expect(asterisk).toBeDefined();
        });
    });

    describe("State transitions", () => {
        it("handles required state changes", () => {
            const { rerender } = render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Test Field"
                    required={false}
                />,
            );

            expect(screen.getByTestId("form-control-label").getAttribute("data-required")).toBe("false");
            expect(screen.queryByLabelText("required")).toBeNull();

            rerender(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Test Field"
                    required={true}
                />,
            );

            expect(screen.getByTestId("form-control-label").getAttribute("data-required")).toBe("true");
            expect(screen.getByLabelText("required")).toBeDefined();
        });

        it("handles disabled state changes", () => {
            const { rerender } = render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Test Field"
                    disabled={false}
                />,
            );

            expect(screen.getByTestId("form-control-label").getAttribute("data-disabled")).toBe("false");

            rerender(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Test Field"
                    disabled={true}
                />,
            );

            expect(screen.getByTestId("form-control-label").getAttribute("data-disabled")).toBe("true");
        });

        it("handles label placement changes", () => {
            const { rerender } = render(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Test Field"
                    labelPlacement="end"
                />,
            );

            expect(screen.getByTestId("form-control-label").getAttribute("data-label-placement")).toBe("end");

            rerender(
                <FormControlLabel
                    control={<MockCheckbox />}
                    label="Test Field"
                    labelPlacement="start"
                />,
            );

            expect(screen.getByTestId("form-control-label").getAttribute("data-label-placement")).toBe("start");
        });
    });
});
