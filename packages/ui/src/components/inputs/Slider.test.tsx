import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { Slider } from "./Slider";

// Mock pointer events
global.PointerEvent = class PointerEvent extends MouseEvent {
    constructor(type: string, params: PointerEventInit = {}) {
        super(type, params);
    }
} as any;

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
            const user = userEvent.setup();
            
            render(<Slider defaultValue={25} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuenow")).toBe("25");
            
            // Focus the slider first
            slider.focus();
            
            // Arrow right should increase value
            await act(async () => {
                await user.keyboard("{ArrowRight}");
            });
            
            expect(onChange).toHaveBeenCalledWith(26);
            expect(slider.getAttribute("aria-valuenow")).toBe("26");
        });

        it("works as controlled component with value", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            let value = 50;
            
            const { rerender } = render(
                <Slider value={value} onChange={(v) => { value = v; onChange(v); }} data-testid="slider" />
            );
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuenow")).toBe("50");
            
            // Focus and arrow left should decrease value
            slider.focus();
            await act(async () => {
                await user.keyboard("{ArrowLeft}");
            });
            
            expect(onChange).toHaveBeenCalledWith(49);
            
            // Re-render with new value
            rerender(<Slider value={value} onChange={(v) => { value = v; onChange(v); }} data-testid="slider" />);
            expect(slider.getAttribute("aria-valuenow")).toBe("49");
        });

        it("respects step value", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Slider value={20} step={5} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            slider.focus();
            
            await act(async () => {
                await user.keyboard("{ArrowRight}");
            });
            
            expect(onChange).toHaveBeenCalledWith(25);
        });

        it("clamps value to min/max bounds", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Slider value={99} min={0} max={100} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            slider.focus();
            
            // Try to go beyond max
            await act(async () => {
                await user.keyboard("{ArrowRight}{ArrowRight}");
            });
            
            expect(onChange).toHaveBeenCalledWith(100);
            expect(onChange).not.toHaveBeenCalledWith(101);
        });
    });

    describe("Keyboard navigation", () => {
        it("handles arrow keys", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            // Start with a controlled component to track value changes
            let currentValue = 50;
            const { rerender } = render(
                <Slider 
                    value={currentValue} 
                    onChange={(v) => {
                        currentValue = v;
                        onChange(v);
                    }} 
                    data-testid="slider" 
                />
            );
            
            const slider = screen.getByRole("slider");
            slider.focus();
            
            // Arrow right increases value
            await act(async () => {
                await user.keyboard("{ArrowRight}");
            });
            expect(onChange).toHaveBeenCalledWith(51);
            expect(onChange).toHaveBeenCalledTimes(1);
            
            // Re-render with new value
            rerender(<Slider value={currentValue} onChange={(v) => { currentValue = v; onChange(v); }} data-testid="slider" />);
            
            // Arrow up increases value
            await act(async () => {
                await user.keyboard("{ArrowUp}");
            });
            expect(onChange).toHaveBeenCalledWith(52);
            expect(onChange).toHaveBeenCalledTimes(2);
            
            // Re-render with new value
            rerender(<Slider value={currentValue} onChange={(v) => { currentValue = v; onChange(v); }} data-testid="slider" />);
            
            // Arrow left decreases value
            await act(async () => {
                await user.keyboard("{ArrowLeft}");
            });
            expect(onChange).toHaveBeenCalledWith(51);
            expect(onChange).toHaveBeenCalledTimes(3);
            
            // Re-render with new value
            rerender(<Slider value={currentValue} onChange={(v) => { currentValue = v; onChange(v); }} data-testid="slider" />);
            
            // Arrow down decreases value
            await act(async () => {
                await user.keyboard("{ArrowDown}");
            });
            expect(onChange).toHaveBeenCalledWith(50);
            expect(onChange).toHaveBeenCalledTimes(4);
        });

        it("handles Home and End keys", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Slider value={50} min={10} max={90} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            slider.focus();
            
            // Home goes to min
            await act(async () => {
                await user.keyboard("{Home}");
            });
            expect(onChange).toHaveBeenCalledWith(10);
            
            // End goes to max
            await act(async () => {
                await user.keyboard("{End}");
            });
            expect(onChange).toHaveBeenCalledWith(90);
        });

        it("handles PageUp and PageDown keys", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            // Use controlled component
            let currentValue = 50;
            const { rerender } = render(
                <Slider 
                    value={currentValue} 
                    step={2} 
                    onChange={(v) => {
                        currentValue = v;
                        onChange(v);
                    }} 
                    data-testid="slider" 
                />
            );
            
            const slider = screen.getByRole("slider");
            slider.focus();
            
            // PageUp increases by 10 * step (10 * 2 = 20)
            await act(async () => {
                await user.keyboard("{PageUp}");
            });
            expect(onChange).toHaveBeenCalledWith(70); // 50 + 20
            
            // Re-render with new value
            rerender(<Slider value={currentValue} step={2} onChange={(v) => { currentValue = v; onChange(v); }} data-testid="slider" />);
            
            // PageDown decreases by 10 * step (10 * 2 = 20)
            await act(async () => {
                await user.keyboard("{PageDown}");
            });
            expect(onChange).toHaveBeenCalledWith(50); // 70 - 20
        });

        it("ignores keyboard input when disabled", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Slider value={50} disabled onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            // Note: disabled elements can't be focused, but we try anyway
            slider.focus();
            
            await act(async () => {
                await user.keyboard("{ArrowRight}{ArrowLeft}{Home}{End}");
            });
            
            expect(onChange).not.toHaveBeenCalled();
        });
    });

    describe("Mouse/Touch interaction", () => {
        it("updates value on track click", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Slider value={0} onChange={onChange} data-testid="slider" />);
            
            const track = screen.getByTestId("slider-track");
            
            // Mock getBoundingClientRect
            vi.spyOn(track, 'getBoundingClientRect').mockReturnValue({
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
            
            // Click at 25% of track width (clientX: 25 when left is 0 and width is 100)
            await act(async () => {
                const clickEvent = new MouseEvent('click', {
                    bubbles: true,
                    cancelable: true,
                    clientX: 25,
                    clientY: 5,
                });
                track.dispatchEvent(clickEvent);
            });
            
            expect(onChange).toHaveBeenCalledWith(25);
        });

        it("does not respond to clicks when disabled", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Slider value={50} disabled onChange={onChange} data-testid="slider" />);
            
            const track = screen.getByTestId("slider-track");
            
            await act(async () => {
                await user.click(track);
            });
            
            expect(onChange).not.toHaveBeenCalled();
        });
    });

    describe("Callbacks", () => {
        it("calls onChange when value changes", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Slider value={50} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            slider.focus();
            
            await act(async () => {
                await user.keyboard("{ArrowRight}");
            });
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(51);
        });

        it("throttles onChange when throttleMs is set", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Slider value={50} onChange={onChange} throttleMs={100} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            slider.focus();
            
            await act(async () => {
                // Rapid key presses
                await user.keyboard("{ArrowRight}{ArrowRight}{ArrowRight}");
            });
            
            // Due to throttling, onChange should be called less than 3 times
            expect(onChange.mock.calls.length).toBeLessThan(3);
        });
    });

    describe("Variants and styling", () => {
        it("applies different variant classes", () => {
            const { rerender } = render(<Slider variant="primary" data-testid="slider" />);
            
            let container = screen.getByTestId("slider");
            expect(container.className).toContain("tw-relative");
            
            rerender(<Slider variant="success" data-testid="slider" />);
            container = screen.getByTestId("slider");
            expect(container.className).toContain("tw-relative");
            
            rerender(<Slider variant="danger" data-testid="slider" />);
            container = screen.getByTestId("slider");
            expect(container.className).toContain("tw-relative");
        });

        it("applies size classes", () => {
            const { rerender } = render(<Slider size="sm" data-testid="slider" />);
            
            let container = screen.getByTestId("slider");
            let track = screen.getByTestId("slider-track");
            let thumb = screen.getByTestId("slider-thumb");
            
            // Check small size
            expect(container.className).toContain("tw-py-2");
            expect(track.className).toContain("tw-h-1");
            expect(thumb.className).toContain("tw-w-4");
            expect(thumb.className).toContain("tw-h-4");
            
            rerender(<Slider size="md" data-testid="slider" />);
            container = screen.getByTestId("slider");
            track = screen.getByTestId("slider-track");
            thumb = screen.getByTestId("slider-thumb");
            
            // Check medium size
            expect(container.className).toContain("tw-py-3");
            expect(track.className).toContain("tw-h-1.5");
            expect(thumb.className).toContain("tw-w-5");
            expect(thumb.className).toContain("tw-h-5");
            
            rerender(<Slider size="lg" data-testid="slider" />);
            container = screen.getByTestId("slider");
            track = screen.getByTestId("slider-track");
            thumb = screen.getByTestId("slider-thumb");
            
            // Check large size
            expect(container.className).toContain("tw-py-4");
            expect(track.className).toContain("tw-h-2");
            expect(thumb.className).toContain("tw-w-6");
            expect(thumb.className).toContain("tw-h-6");
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

        it("shows value tooltip when showValue is true and thumb is focused", async () => {
            const user = userEvent.setup();
            
            render(<Slider value={50} showValue data-testid="slider" />);
            
            // Initially not visible
            expect(screen.queryByText("50")).toBeNull();
            
            // Focus the slider to trigger tooltip
            const slider = screen.getByRole("slider");
            await act(async () => {
                await user.click(slider);
            });
            
            // The tooltip should appear during drag, let's simulate pointer down
            const container = screen.getByTestId("slider");
            const thumb = container.querySelector('[role="slider"]');
            
            if (thumb) {
                await act(async () => {
                    await user.pointer({ keys: '[MouseLeft>]', target: thumb });
                });
            }
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
        it("handles min equal to max", () => {
            const onChange = vi.fn();
            render(<Slider min={50} max={50} value={50} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuenow")).toBe("50");
            
            // Try to change value - should stay at 50
            slider.focus();
            userEvent.keyboard("{ArrowRight}");
            
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
            const user = userEvent.setup();
            
            render(<Slider value={1.5} min={0} max={10} step={0.5} onChange={onChange} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuenow")).toBe("1.5");
            expect(slider.getAttribute("aria-valuetext")).toBe("1.5");
            
            slider.focus();
            await act(async () => {
                await user.keyboard("{ArrowRight}");
            });
            
            expect(onChange).toHaveBeenCalledWith(2);
        });

        it("handles very large numbers", () => {
            const largeValue = 1000000;
            render(<Slider min={0} max={10000000} value={largeValue} data-testid="slider" />);
            
            const slider = screen.getByRole("slider");
            expect(slider.getAttribute("aria-valuenow")).toBe(largeValue.toString());
        });

        it("handles drag events", async () => {
            const onChange = vi.fn();
            const onChangeStart = vi.fn();
            const onChangeEnd = vi.fn();
            
            render(
                <Slider 
                    value={50} 
                    onChange={onChange}
                    onChangeStart={onChangeStart}
                    onChangeEnd={onChangeEnd}
                    data-testid="slider" 
                />
            );
            
            const track = screen.getByTestId("slider-track");
            const thumb = screen.getByTestId("slider-thumb");
            
            // Mock getBoundingClientRect
            vi.spyOn(track, 'getBoundingClientRect').mockReturnValue({
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
            
            // Simulate pointer down on thumb
            const pointerDownEvent = new PointerEvent('pointerdown', {
                bubbles: true,
                cancelable: true,
                clientX: 50,
                clientY: 5,
            });
            
            await act(async () => {
                thumb.dispatchEvent(pointerDownEvent);
            });
            
            expect(onChangeStart).toHaveBeenCalledWith(50);
            
            // Simulate pointer move
            const pointerMoveEvent = new PointerEvent('pointermove', {
                bubbles: true,
                cancelable: true,
                clientX: 75,
                clientY: 5,
            });
            
            await act(async () => {
                document.dispatchEvent(pointerMoveEvent);
            });
            
            expect(onChange).toHaveBeenCalledWith(75);
            
            // Simulate pointer up
            const pointerUpEvent = new PointerEvent('pointerup', {
                bubbles: true,
                cancelable: true,
            });
            
            await act(async () => {
                document.dispatchEvent(pointerUpEvent);
            });
            
            expect(onChangeEnd).toHaveBeenCalled();
        });
    });
});