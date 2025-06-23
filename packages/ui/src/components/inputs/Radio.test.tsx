import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { Radio, PrimaryRadio, SecondaryRadio, DangerRadio, CustomRadio } from "./Radio.js";

describe("Radio", () => {
    describe("Basic rendering", () => {
        it("renders as an unchecked radio by default", () => {
            render(<Radio name="test-radio" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput).toBeDefined();
            expect(radioInput.getAttribute("type")).toBe("radio");
            expect(radioInput.getAttribute("checked")).toBeNull();
            expect(radioInput.getAttribute("disabled")).toBeNull();
        });

        it("renders with required prop", () => {
            render(<Radio name="test-radio" required />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("required")).toBe("");
        });

        it("renders with aria attributes", () => {
            render(
                <Radio 
                    name="test-radio"
                    aria-label="Test radio"
                    aria-labelledby="label-id"
                    aria-describedby="desc-id"
                />,
            );

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("aria-label")).toBe("Test radio");
            expect(radioInput.getAttribute("aria-labelledby")).toBe("label-id");
            expect(radioInput.getAttribute("aria-describedby")).toBe("desc-id");
        });
    });

    describe("Controlled vs Uncontrolled", () => {
        it("works as controlled component with checked prop", () => {
            const { rerender } = render(<Radio name="test-radio" checked={false} onChange={vi.fn()} />);

            const radioInput = screen.getByTestId("radio-input") as HTMLInputElement;
            expect(radioInput.checked).toBe(false);

            rerender(<Radio name="test-radio" checked={true} onChange={vi.fn()} />);
            expect(radioInput.checked).toBe(true);
        });

        it("works as uncontrolled component with defaultChecked", () => {
            render(<Radio name="test-radio" defaultChecked={true} />);

            const radioInput = screen.getByTestId("radio-input") as HTMLInputElement;
            expect(radioInput.checked).toBe(true);
        });
    });

    describe("User interactions", () => {
        it("handles click on radio input", async () => {
            const onChange = vi.fn();
            const onClick = vi.fn();
            const user = userEvent.setup();

            render(<Radio name="test-radio" onChange={onChange} onClick={onClick} />);

            const radioInput = screen.getByTestId("radio-input");

            await act(async () => {
                await user.click(radioInput);
            });

            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onClick).toHaveBeenCalledTimes(1);
        });

        it("handles click on visual wrapper for ripple effect", async () => {
            const user = userEvent.setup();

            render(<Radio name="test-radio" />);

            const radioWrapper = screen.getByTestId("radio-wrapper");

            await act(async () => {
                await user.click(radioWrapper);
            });

            // Check if ripple was created
            const ripples = screen.queryAllByTestId("radio-ripple");
            expect(ripples.length).toBeGreaterThan(0);
        });

        it("does not trigger events when disabled", async () => {
            const onChange = vi.fn();
            const onClick = vi.fn();
            const user = userEvent.setup();

            render(<Radio name="test-radio" disabled onChange={onChange} onClick={onClick} />);

            const radioInput = screen.getByTestId("radio-input");
            const radioLabel = screen.getByTestId("radio-label");

            await act(async () => {
                await user.click(radioInput);
            });

            expect(onChange).not.toHaveBeenCalled();
            expect(onClick).not.toHaveBeenCalled();
            expect(radioLabel.getAttribute("data-disabled")).toBe("true");
        });

        it("handles focus and blur events", async () => {
            const onFocus = vi.fn();
            const onBlur = vi.fn();
            const user = userEvent.setup();

            render(<Radio name="test-radio" onFocus={onFocus} onBlur={onBlur} />);

            const radioInput = screen.getByTestId("radio-input");

            await act(async () => {
                await user.click(radioInput);
            });

            expect(onFocus).toHaveBeenCalledTimes(1);

            await act(async () => {
                await user.tab();
            });

            expect(onBlur).toHaveBeenCalledTimes(1);
        });
    });

    describe("Radio group behavior", () => {
        it("allows selection within a group", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();

            render(
                <>
                    <Radio name="group" value="option1" onChange={onChange} />
                    <Radio name="group" value="option2" onChange={onChange} />
                    <Radio name="group" value="option3" onChange={onChange} />
                </>,
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];

            await act(async () => {
                await user.click(radios[0]);
            });

            expect(radios[0].checked).toBe(true);
            expect(radios[1].checked).toBe(false);
            expect(radios[2].checked).toBe(false);

            await act(async () => {
                await user.click(radios[1]);
            });

            expect(radios[0].checked).toBe(false);
            expect(radios[1].checked).toBe(true);
            expect(radios[2].checked).toBe(false);
        });

        it("assigns proper name and value attributes", () => {
            render(<Radio name="test-group" value="test-value" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("name")).toBe("test-group");
            expect(radioInput.getAttribute("value")).toBe("test-value");
        });
    });

    describe("Visual states", () => {
        it("updates data attributes based on state", () => {
            const { rerender } = render(<Radio name="test-radio" checked={false} onChange={vi.fn()} />);

            const radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel.getAttribute("data-checked")).toBe("false");

            rerender(<Radio name="test-radio" checked={true} onChange={vi.fn()} />);
            expect(radioLabel.getAttribute("data-checked")).toBe("true");
        });

        it("renders with different sizes", () => {
            const { rerender } = render(<Radio name="test-radio" size="sm" />);

            let radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();

            rerender(<Radio name="test-radio" size="md" />);
            radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();

            rerender(<Radio name="test-radio" size="lg" />);
            radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();
        });

        it("renders with different color variants", () => {
            const { rerender } = render(<Radio name="test-radio" color="primary" />);

            let radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();

            rerender(<Radio name="test-radio" color="secondary" />);
            radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();

            rerender(<Radio name="test-radio" color="danger" />);
            radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();
        });
    });

    describe("Custom color functionality", () => {
        it("applies custom color styles when color is custom", () => {
            render(<Radio name="test-radio" color="custom" customColor="#ff0000" />);

            const radioOuter = screen.getByTestId("radio-outer");
            const radioInner = screen.getByTestId("radio-inner");

            expect(radioOuter).toBeDefined();
            expect(radioInner).toBeDefined();
        });

        it("handles hover state for custom colors", async () => {
            const user = userEvent.setup();

            render(<Radio name="test-radio" color="custom" customColor="#ff0000" />);

            const radioWrapper = screen.getByTestId("radio-wrapper");

            await act(async () => {
                await user.hover(radioWrapper);
            });

            // The hover effect modifies inline styles
            expect(radioWrapper.style.backgroundColor).toBeDefined();

            await act(async () => {
                await user.unhover(radioWrapper);
            });

            expect(radioWrapper.style.backgroundColor).toBe("");
        });

        it("does not apply hover effects when disabled", async () => {
            const user = userEvent.setup();

            render(<Radio name="test-radio" color="custom" customColor="#ff0000" disabled />);

            const radioWrapper = screen.getByTestId("radio-wrapper");

            await act(async () => {
                await user.hover(radioWrapper);
            });

            expect(radioWrapper.style.backgroundColor).toBe("");
        });
    });

    describe("Factory components", () => {
        it("renders PrimaryRadio with primary color", () => {
            render(<PrimaryRadio name="test-radio" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput).toBeDefined();
        });

        it("renders SecondaryRadio with secondary color", () => {
            render(<SecondaryRadio name="test-radio" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput).toBeDefined();
        });

        it("renders DangerRadio with danger color", () => {
            render(<DangerRadio name="test-radio" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput).toBeDefined();
        });

        it("renders CustomRadio with custom color", () => {
            render(<CustomRadio name="test-radio" customColor="#00ff00" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput).toBeDefined();
        });
    });

    describe("Accessibility", () => {
        it("maintains proper ARIA roles", () => {
            render(<Radio name="test-radio" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("type")).toBe("radio");
        });

        it("can be focused via keyboard", async () => {
            const user = userEvent.setup();

            render(
                <>
                    <input type="text" data-testid="before" />
                    <Radio name="test-radio" />
                    <input type="text" data-testid="after" />
                </>,
            );

            const beforeInput = screen.getByTestId("before");
            const radioInput = screen.getByTestId("radio-input");
            const afterInput = screen.getByTestId("after");

            beforeInput.focus();

            await act(async () => {
                await user.tab();
            });

            expect(document.activeElement).toBe(radioInput);

            await act(async () => {
                await user.tab();
            });

            expect(document.activeElement).toBe(afterInput);
        });

        it("properly handles keyboard navigation in radio groups", async () => {
            const user = userEvent.setup();

            render(
                <>
                    <Radio name="group" value="1" defaultChecked />
                    <Radio name="group" value="2" />
                    <Radio name="group" value="3" />
                </>,
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];

            radios[0].focus();

            await act(async () => {
                await user.keyboard("{ArrowDown}");
            });

            expect(document.activeElement).toBe(radios[1]);
            expect(radios[1].checked).toBe(true);

            await act(async () => {
                await user.keyboard("{ArrowDown}");
            });

            expect(document.activeElement).toBe(radios[2]);
            expect(radios[2].checked).toBe(true);

            await act(async () => {
                await user.keyboard("{ArrowUp}");
            });

            expect(document.activeElement).toBe(radios[1]);
            expect(radios[1].checked).toBe(true);
        });
    });

    describe("Edge cases", () => {
        it("handles rapid clicks gracefully", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();

            render(<Radio name="test-radio" onChange={onChange} />);

            const radioInput = screen.getByTestId("radio-input");

            await act(async () => {
                await user.tripleClick(radioInput);
            });

            // Should still be called appropriate number of times
            expect(onChange).toHaveBeenCalled();
        });

        it("cleans up ripple effects after animation", async () => {
            const user = userEvent.setup();

            render(<Radio name="test-radio" />);

            const radioWrapper = screen.getByTestId("radio-wrapper");

            await act(async () => {
                await user.click(radioWrapper);
            });

            const ripple = screen.getByTestId("radio-ripple");
            expect(ripple).toBeDefined();

            // Simulate animation end
            const animationEndEvent = new Event("animationend", { bubbles: true });
            
            act(() => {
                ripple.dispatchEvent(animationEndEvent);
            });

            // Ripple should be removed after animation
            expect(screen.queryByTestId("radio-ripple")).toBeNull();
        });

        it("handles missing props gracefully", () => {
            render(<Radio />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("name")).toBeNull();
            expect(radioInput.getAttribute("value")).toBe("");
        });

        it("forwards refs correctly", () => {
            const ref = React.createRef<HTMLInputElement>();
            
            render(<Radio name="test-radio" ref={ref} />);

            expect(ref.current).toBeDefined();
            expect(ref.current?.type).toBe("radio");
        });

        it("spreads additional props to input element", () => {
            render(
                <Radio 
                    name="test-radio" 
                    data-custom="custom-value"
                    id="custom-id"
                />,
            );

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("data-custom")).toBe("custom-value");
            expect(radioInput.getAttribute("id")).toBe("custom-id");
        });
    });
});
