import { act, render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import * as yup from "yup";
import { renderWithFormik, formAssertions } from "../../__test/helpers/formTestHelpers";
import { Slider } from "./Slider";

// Mock pointer events
global.PointerEvent = class PointerEvent extends MouseEvent {
    constructor(type: string, params: PointerEventInit = {}) {
        super(type, params);
    }
} as any;

// Disable animations and transitions for tests
beforeEach(() => {
    // Add style to disable animations
    const style = document.createElement("style");
    style.innerHTML = `
        *, *::before, *::after {
            animation-duration: 0s !important;
            animation-delay: 0s !important;
            transition-duration: 0s !important;
            transition-delay: 0s !important;
        }
    `;
    style.setAttribute("data-test", "disable-animations");
    document.head.appendChild(style);
});

afterEach(() => {
    // Clean up animation disabling style
    const style = document.querySelector("[data-test=\"disable-animations\"]");
    if (style) {
        style.remove();
    }
    // Clear all mocks after each test
    vi.clearAllMocks();
});

describe("Slider", () => {
    describe("Basic rendering", () => {
        it("renders with default props", () => {
            render(<Slider data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider).toBeDefined();
            expect(slider.getAttribute("aria-valuemin")).toBe("0");
            expect(slider.getAttribute("aria-valuemax")).toBe("100");
            expect(slider.getAttribute("aria-valuenow")).toBe("0");
            expect(slider.getAttribute("aria-valuetext")).toBe("0");
            expect(slider.getAttribute("aria-disabled")).toBe("false");
            expect(slider.getAttribute("tabIndex")).toBe("0");
        });

        it("renders with custom min, max, and value", () => {
            render(<Slider min={10} max={50} value={30} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuemin")).toBe("10");
            expect(slider.getAttribute("aria-valuemax")).toBe("50");
            expect(slider.getAttribute("aria-valuenow")).toBe("30");
        });

        it("renders with label", () => {
            render(<Slider label="Volume Control" data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-label")).toBe("Volume Control");
            expect(screen.getByText("Volume Control")).toBeDefined();
        });

        it("renders disabled state", () => {
            render(<Slider disabled data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-disabled")).toBe("true");
            expect(slider.getAttribute("tabIndex")).toBe("-1");
        });
    });

    describe("Value control", () => {
        it("works as uncontrolled component with defaultValue", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            
            render(<Slider defaultValue={25} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuenow")).toBe("25");
            
            // Focus the slider first
            await user.click(slider);
            
            // Arrow right should increase value
            await user.keyboard("{ArrowRight}");
            
            await waitFor(() => {
                expect(onChange).toHaveBeenCalledWith(26);
            });
            
            expect(slider.getAttribute("aria-valuenow")).toBe("26");
        });

        it("works as controlled component with value", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            let value = 50;
            
            const { rerender } = render(
                <Slider value={value} onChange={(v) => { value = v; onChange(v); }} data-testid="slider" />,
            );
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuenow")).toBe("50");
            
            // Focus and arrow left should decrease value
            await user.click(slider);
            await user.keyboard("{ArrowLeft}");
            
            await waitFor(() => {
                expect(onChange).toHaveBeenCalledWith(49);
            });
            
            // Re-render with new value
            rerender(<Slider value={value} onChange={(v) => { value = v; onChange(v); }} data-testid="slider" />);
            expect(slider.getAttribute("aria-valuenow")).toBe("49");
        });

        it("respects step value", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            
            render(<Slider value={20} step={5} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            await user.click(slider);
            await user.keyboard("{ArrowRight}");
            
            await waitFor(() => {
                expect(onChange).toHaveBeenCalledWith(25);
            });
        });

        it("clamps value to min/max bounds", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            
            render(<Slider value={99} min={0} max={100} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            await user.click(slider);
            
            // Try to go beyond max
            await user.keyboard("{ArrowRight}{ArrowRight}");
            
            await waitFor(() => {
                expect(onChange).toHaveBeenCalledWith(100);
                expect(onChange).not.toHaveBeenCalledWith(101);
            });
        });
    });

    describe("Keyboard navigation", () => {
        it("handles arrow keys", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            
            render(<Slider value={50} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            await user.click(slider);
            
            // Test arrow right
            await user.keyboard("{ArrowRight}");
            expect(onChange).toHaveBeenCalledWith(51);
            
            // Test arrow left
            onChange.mockClear();
            await user.keyboard("{ArrowLeft}");
            expect(onChange).toHaveBeenCalledWith(50);
            
            // Test arrow up
            onChange.mockClear();
            await user.keyboard("{ArrowUp}");
            expect(onChange).toHaveBeenCalledWith(51);
            
            // Test arrow down
            onChange.mockClear();
            await user.keyboard("{ArrowDown}");
            expect(onChange).toHaveBeenCalledWith(50);
        });

        it("handles Home and End keys", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            
            render(<Slider value={50} min={10} max={90} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            await user.click(slider);
            
            // Home goes to min
            await user.keyboard("{Home}");
            expect(onChange).toHaveBeenCalledWith(10);
            
            // End goes to max
            onChange.mockClear();
            await user.keyboard("{End}");
            expect(onChange).toHaveBeenCalledWith(90);
        });

        it("handles PageUp and PageDown keys", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            
            render(<Slider value={50} step={2} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            await user.click(slider);
            
            // PageUp increases by 10 * step (10 * 2 = 20)
            await user.keyboard("{PageUp}");
            expect(onChange).toHaveBeenCalledWith(70);
            
            // PageDown decreases by 10 * step (10 * 2 = 20)
            onChange.mockClear();
            await user.keyboard("{PageDown}");
            expect(onChange).toHaveBeenCalledWith(50);
        });

        it("ignores keyboard input when disabled", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            
            render(<Slider value={50} disabled onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            
            // Try keyboard navigation (won't work because disabled elements can't be focused)
            await user.keyboard("{ArrowRight}{ArrowLeft}{Home}{End}");
            
            expect(onChange).not.toHaveBeenCalled();
        });
    });

    describe("Mouse/Touch interaction", () => {
        it("updates value on track click", async () => {
            const onChange = vi.fn();
            
            render(<Slider value={0} onChange={onChange} data-testid="slider" />);
            
            const track = screen.getByTestId("slider-track");
            
            // Mock getBoundingClientRect
            vi.spyOn(track, "getBoundingClientRect").mockReturnValue({
                left: 0,
                width: 100,
                right: 100,
                top: 0,
                bottom: 10,
                height: 10,
                x: 0,
                y: 0,
                toJSON: () => ({}),
            });
            
            // Click at 25% of track width
            const clickEvent = new MouseEvent("click", {
                bubbles: true,
                cancelable: true,
                clientX: 25,
                clientY: 5,
            });
            
            track.dispatchEvent(clickEvent);
            
            expect(onChange).toHaveBeenCalledWith(25);
        });

        it("does not respond to clicks when disabled", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            
            render(<Slider value={50} disabled onChange={onChange} data-testid="slider" />);
            
            const track = screen.getByTestId("slider-track");
            
            await user.click(track);
            
            expect(onChange).not.toHaveBeenCalled();
        });
    });

    describe("Callbacks", () => {
        it("calls onChange when value changes", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            
            render(<Slider value={50} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            await user.click(slider);
            await user.keyboard("{ArrowRight}");
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(51);
        });

        it("throttles onChange when throttleMs is set", async () => {
            vi.useFakeTimers();
            const onChange = vi.fn();
            
            render(<Slider value={50} onChange={onChange} throttleMs={100} data-testid="slider" />);
            
            const track = screen.getByTestId("slider-track");
            
            // Mock getBoundingClientRect
            vi.spyOn(track, "getBoundingClientRect").mockReturnValue({
                left: 0,
                width: 100,
                right: 100,
                top: 0,
                bottom: 10,
                height: 10,
                x: 0,
                y: 0,
                toJSON: () => ({}),
            });
            
            // Simulate rapid clicks
            for (let i = 0; i < 5; i++) {
                const clickEvent = new MouseEvent("click", {
                    bubbles: true,
                    cancelable: true,
                    clientX: 20 + i * 10,
                    clientY: 5,
                });
                track.dispatchEvent(clickEvent);
                vi.advanceTimersByTime(50); // Less than throttle time
            }
            
            // Due to throttling, onChange should be called less than 5 times
            expect(onChange.mock.calls.length).toBeLessThan(5);
            
            vi.useRealTimers();
        });
    });

    describe("Variants and styling", () => {
        it("accepts different variant props", () => {
            const variants = ["primary", "secondary", "success", "warning", "danger", "space", "neon"] as const;
            
            variants.forEach(variant => {
                const { container } = render(<Slider variant={variant} data-testid={`slider-${variant}`} />);
                expect(container.querySelector("[role=\"slider\"]")).toBeTruthy();
            });
        });

        it("accepts different size props", () => {
            const sizes = ["sm", "md", "lg"] as const;
            
            sizes.forEach(size => {
                const { container } = render(<Slider size={size} data-testid={`slider-${size}`} />);
                expect(container.querySelector("[role=\"slider\"]")).toBeTruthy();
            });
        });

        it("applies custom color when variant is custom", () => {
            render(<Slider variant="custom" color="#ff0000" data-testid="slider" />);
            
            const container = screen.getByTestId("slider");
            expect(container.style.getPropertyValue("--slider-color")).toBe("#ff0000");
        });
    });

    describe("Marks", () => {
        it("renders marks with labels", () => {
            const marks = [
                { value: 0, label: "Min" },
                { value: 50, label: "Mid" },
                { value: 100, label: "Max" },
            ];
            
            render(<Slider marks={marks} data-testid="slider" />);
            
            expect(screen.getByText("Min")).toBeDefined();
            expect(screen.getByText("Mid")).toBeDefined();
            expect(screen.getByText("Max")).toBeDefined();
        });

        it("renders marks without labels using value", () => {
            const marks = [
                { value: 25 },
                { value: 75 },
            ];
            
            render(<Slider marks={marks} data-testid="slider" />);
            
            expect(screen.getByText("25")).toBeDefined();
            expect(screen.getByText("75")).toBeDefined();
        });
    });

    describe("Value display", () => {
        it("does not show value tooltip by default", () => {
            render(<Slider value={50} data-testid="slider" />);
            
            const valueDisplay = screen.queryByText("50");
            expect(valueDisplay).toBeNull();
        });

        it("formats decimal values correctly", () => {
            render(<Slider value={33.333} step={0.1} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuetext")).toBe("33.3");
        });
    });

    describe("Accessibility", () => {
        it("has proper ARIA attributes", () => {
            render(<Slider value={75} min={0} max={100} label="Volume" data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-label")).toBe("Volume");
            expect(slider.getAttribute("aria-valuemin")).toBe("0");
            expect(slider.getAttribute("aria-valuemax")).toBe("100");
            expect(slider.getAttribute("aria-valuenow")).toBe("75");
            expect(slider.getAttribute("aria-valuetext")).toBe("75");
            expect(slider.getAttribute("aria-disabled")).toBe("false");
        });

        it("is keyboard navigable when enabled", () => {
            render(<Slider data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("tabIndex")).toBe("0");
        });

        it("is not keyboard navigable when disabled", () => {
            render(<Slider disabled data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("tabIndex")).toBe("-1");
        });
    });

    describe("Factory components", () => {
        it("renders primary variant through factory", () => {
            render(<Slider.Primary value={50} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider).toBeDefined();
            expect(slider.getAttribute("aria-valuenow")).toBe("50");
        });

        it("renders custom variant through factory with color", () => {
            render(<Slider.Custom value={50} color="#00ff00" data-testid="slider" />);
            
            const container = screen.getByTestId("slider");
            expect(container.style.getPropertyValue("--slider-color")).toBe("#00ff00");
        });
    });

    describe("Edge cases", () => {
        it("handles min equal to max", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            
            render(<Slider min={50} max={50} value={50} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuenow")).toBe("50");
            
            // Try to change value - should stay at 50
            await user.click(slider);
            await user.keyboard("{ArrowRight}");
            
            // onChange might be called but value should remain 50
            if (onChange.mock.calls.length > 0) {
                expect(onChange).toHaveBeenCalledWith(50);
            }
        });

        it("handles negative values", () => {
            render(<Slider min={-100} max={100} value={-50} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuemin")).toBe("-100");
            expect(slider.getAttribute("aria-valuemax")).toBe("100");
            expect(slider.getAttribute("aria-valuenow")).toBe("-50");
        });

        it("handles decimal step values", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup({ delay: null });
            
            render(<Slider value={1.5} min={0} max={10} step={0.5} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuenow")).toBe("1.5");
            expect(slider.getAttribute("aria-valuetext")).toBe("1.5");
            
            await user.click(slider);
            await user.keyboard("{ArrowRight}");
            
            expect(onChange).toHaveBeenCalledWith(2);
        });

        it("handles very large numbers", () => {
            const largeValue = 1000000;
            render(<Slider min={0} max={10000000} value={largeValue} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuenow")).toBe(largeValue.toString());
        });
    });

    describe("Formik integration", () => {
        it("works as a controlled component within Formik", async () => {
            const { user, getFormValues } = renderWithFormik(
                <Slider name="volume" label="Volume" data-testid="slider" />,
                {
                    initialValues: { volume: 50 },
                },
            );

            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuenow")).toBe("50");
            expect(getFormValues().volume).toBe(50);

            await user.click(slider);
            await user.keyboard("{ArrowRight}");

            await waitFor(() => {
                expect(getFormValues().volume).toBe(51);
            });
        });

        it("submits form values correctly", async () => {
            const { user, onSubmit, submitForm } = renderWithFormik(
                <>
                    <Slider name="brightness" label="Brightness" data-testid="slider" />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { brightness: 30 },
                },
            );

            const slider = screen.getByRole("slider");
            await user.click(slider);
            
            await user.keyboard("{ArrowRight}{ArrowRight}{ArrowRight}{ArrowRight}{ArrowRight}");

            await submitForm();

            await waitFor(() => {
                formAssertions.expectFormSubmitted(onSubmit, { brightness: 35 });
            });
        });

        it("validates range with min/max", async () => {
            const validationSchema = yup.object({
                age: yup.number()
                    .min(18, "Must be at least 18")
                    .max(100, "Must be no more than 100")
                    .required("Age is required"),
            });

            const { user, onSubmit, setFieldValue } = renderWithFormik(
                <>
                    <Slider name="age" label="Age" min={0} max={120} data-testid="slider" />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { age: 15 },
                    formikConfig: { validationSchema },
                },
            );

            const submitButton = screen.getByRole("button", { name: /submit/i });
            
            await user.click(submitButton);

            await waitFor(() => {
                expect(onSubmit).not.toHaveBeenCalled();
                formAssertions.expectFieldError("Must be at least 18");
            });

            // Set valid age
            await setFieldValue("age", 25);
            
            await user.click(submitButton);

            await waitFor(() => {
                formAssertions.expectFormSubmitted(onSubmit, { age: 25 });
            });
        });
    });
});
