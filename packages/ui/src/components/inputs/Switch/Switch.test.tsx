import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { Switch, SwitchFactory } from "./Switch";

describe("Switch", () => {
    describe("Basic rendering", () => {
        it("renders unchecked switch by default", () => {
            render(<Switch />);
            
            const checkbox = screen.getByRole("switch") as HTMLInputElement;
            expect(checkbox).toBeDefined();
            expect(checkbox.getAttribute("aria-checked")).toBe("false");
            expect(checkbox.checked).toBe(false);
        });

        it("renders checked switch when checked prop is true", () => {
            render(<Switch checked={true} onChange={vi.fn()} />);
            
            const checkbox = screen.getByRole("switch") as HTMLInputElement;
            expect(checkbox.getAttribute("aria-checked")).toBe("true");
            expect(checkbox.checked).toBe(true);
        });

        it("renders with label when provided", () => {
            render(<Switch label="Enable notifications" />);
            
            const label = screen.getByText("Enable notifications");
            expect(label).toBeDefined();
            expect(label.tagName).toBe("LABEL");
        });

        it("renders without label when not provided", () => {
            render(<Switch />);
            
            expect(screen.queryByText("Toggle switch")).toBeNull();
        });

        it("renders with custom id when provided", () => {
            render(<Switch id="custom-switch-id" />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.id).toBe("custom-switch-id");
        });

        it("generates unique id when not provided", () => {
            render(
                <>
                    <Switch />
                    <Switch />
                </>
            );
            
            const switches = screen.getAllByRole("switch");
            expect(switches[0].id).toBeDefined();
            expect(switches[1].id).toBeDefined();
            expect(switches[0].id).not.toBe(switches[1].id);
        });
    });

    describe("User interactions", () => {
        it("toggles checked state on click", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Switch checked={false} onChange={onChange} />);
            
            // Find the visual switch container that has the onClick handler
            const visualSwitch = screen.getByTestId("switch-visual-container");
            
            await act(async () => {
                await user.click(visualSwitch);
            });
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(true, expect.any(Object));
        });

        it("toggles checked state on checkbox click", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Switch checked={false} onChange={onChange} />);
            
            const checkbox = screen.getByRole("switch");
            
            await act(async () => {
                await user.click(checkbox);
            });
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(true, expect.any(Object));
        });

        it("toggles on Space key press", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Switch checked={false} onChange={onChange} />);
            
            const checkbox = screen.getByRole("switch");
            checkbox.focus();
            
            await act(async () => {
                await user.keyboard(" ");
            });
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(true, expect.any(Object));
        });

        it("toggles on Enter key press", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Switch checked={false} onChange={onChange} />);
            
            const checkbox = screen.getByRole("switch");
            checkbox.focus();
            
            await act(async () => {
                await user.keyboard("{Enter}");
            });
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(true, expect.any(Object));
        });

        it("does not toggle when disabled", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Switch checked={false} onChange={onChange} disabled />);
            
            const visualSwitch = screen.getByTestId("switch-visual-container");
            
            await act(async () => {
                await user.click(visualSwitch);
            });
            
            expect(onChange).not.toHaveBeenCalled();
        });

        it("does not toggle on keyboard when disabled", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Switch checked={false} onChange={onChange} disabled />);
            
            const checkbox = screen.getByRole("switch");
            checkbox.focus();
            
            await act(async () => {
                await user.keyboard(" ");
                await user.keyboard("{Enter}");
            });
            
            expect(onChange).not.toHaveBeenCalled();
        });
    });

    describe("Accessibility", () => {
        it("has proper ARIA attributes", () => {
            render(<Switch checked={true} onChange={vi.fn()} />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("role")).toBe("switch");
            expect(checkbox.getAttribute("aria-checked")).toBe("true");
            expect(checkbox.getAttribute("type")).toBe("checkbox");
        });

        it("uses label as aria-label when label is string", () => {
            render(<Switch label="Dark mode" />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("aria-label")).toBe("Dark mode");
        });

        it("uses provided aria-label over label", () => {
            render(<Switch label="Dark mode" aria-label="Toggle dark theme" />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("aria-label")).toBe("Toggle dark theme");
        });

        it("uses default aria-label when no label provided", () => {
            render(<Switch />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("aria-label")).toBe("Toggle switch");
        });

        it("supports aria-labelledby", () => {
            render(
                <>
                    <span id="switch-label">Custom Label</span>
                    <Switch aria-labelledby="switch-label" />
                </>
            );
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("aria-labelledby")).toBe("switch-label");
        });

        it("supports aria-describedby", () => {
            render(
                <>
                    <span id="switch-description">This enables dark mode</span>
                    <Switch aria-describedby="switch-description" />
                </>
            );
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("aria-describedby")).toBe("switch-description");
        });

        it("provides screen reader feedback for state", () => {
            render(<Switch checked={true} onChange={vi.fn()} />);
            
            const status = screen.getByRole("status");
            expect(status).toBeDefined();
            expect(status.textContent).toBe("On");
        });

        it("updates screen reader feedback when toggled", async () => {
            const user = userEvent.setup();
            const { rerender } = render(<Switch checked={false} onChange={vi.fn()} />);
            
            expect(screen.getByRole("status").textContent).toBe("Off");
            
            rerender(<Switch checked={true} onChange={vi.fn()} />);
            
            expect(screen.getByRole("status").textContent).toBe("On");
        });

        it("label is clickable and toggles switch", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<Switch label="Click me" checked={false} onChange={onChange} />);
            
            const label = screen.getByText("Click me");
            
            await act(async () => {
                await user.click(label);
            });
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(true, expect.any(Object));
        });
    });

    describe("Visual states", () => {
        it("indicates disabled state visually", () => {
            render(<Switch label="Disabled switch" disabled />);
            
            const label = screen.getByText("Disabled switch");
            expect(label.className).toContain("tw-opacity-50");
            expect(label.className).toContain("tw-cursor-not-allowed");
        });

        it("track has data-checked attribute", () => {
            const { rerender } = render(<Switch checked={false} onChange={vi.fn()} />);
            
            let track = screen.getByRole("switch").parentElement?.querySelector("[data-checked]");
            expect(track?.getAttribute("data-checked")).toBe("false");
            
            rerender(<Switch checked={true} onChange={vi.fn()} />);
            
            track = screen.getByRole("switch").parentElement?.querySelector("[data-checked]");
            expect(track?.getAttribute("data-checked")).toBe("true");
        });
    });

    describe("Label positioning", () => {
        it("positions label on the right by default", () => {
            render(<Switch label="Right label" />);
            
            const container = screen.getByRole("switch").closest("div");
            expect(container?.className).toContain("tw-flex-row-reverse");
        });

        it("positions label on the left when specified", () => {
            render(<Switch label="Left label" labelPosition="left" />);
            
            const container = screen.getByRole("switch").closest("div");
            expect(container?.className).not.toContain("tw-flex-row-reverse");
        });

        it("renders no label when labelPosition is none", () => {
            render(<Switch label="Hidden label" labelPosition="none" />);
            
            expect(screen.queryByText("Hidden label")).toBeNull();
        });
    });

    describe("Variants", () => {
        const variants = ["default", "success", "warning", "danger", "space", "neon", "theme", "custom"] as const;
        
        variants.forEach(variant => {
            it(`renders ${variant} variant`, () => {
                render(<Switch variant={variant} />);
                
                const checkbox = screen.getByRole("switch");
                expect(checkbox).toBeDefined();
            });
        });

        it("applies custom color when variant is custom", () => {
            render(<Switch variant="custom" color="#FF0000" checked={true} onChange={vi.fn()} />);
            
            const track = screen.getByRole("switch").parentElement?.querySelector("[data-checked='true']");
            const style = track?.getAttribute("style") || "";
            // Check for either format of color
            expect(style.includes("background-color: rgb(255, 0, 0)") || style.includes("background-color: #FF0000")).toBe(true);
        });

        it("renders theme variant with sun/moon icons", () => {
            render(<Switch variant="theme" checked={false} onChange={vi.fn()} />);
            
            const svgs = screen.getByRole("switch").parentElement?.querySelectorAll("svg");
            expect(svgs?.length).toBeGreaterThan(0);
        });
    });

    describe("Sizes", () => {
        const sizes = ["sm", "md", "lg"] as const;
        
        sizes.forEach(size => {
            it(`renders ${size} size`, () => {
                render(<Switch size={size} />);
                
                const checkbox = screen.getByRole("switch");
                expect(checkbox).toBeDefined();
            });
        });
    });

    describe("SwitchFactory", () => {
        it("renders Default factory switch", () => {
            render(<SwitchFactory.Default />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Success factory switch", () => {
            render(<SwitchFactory.Success />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Warning factory switch", () => {
            render(<SwitchFactory.Warning />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Danger factory switch", () => {
            render(<SwitchFactory.Danger />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Space factory switch", () => {
            render(<SwitchFactory.Space />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Neon factory switch", () => {
            render(<SwitchFactory.Neon />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Theme factory switch", () => {
            render(<SwitchFactory.Theme />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("factory switches accept props except variant", () => {
            const onChange = vi.fn();
            render(<SwitchFactory.Default checked={true} onChange={onChange} label="Test" />);
            
            const checkbox = screen.getByRole("switch") as HTMLInputElement;
            expect(checkbox.checked).toBe(true);
            expect(screen.getByText("Test")).toBeDefined();
        });
    });

    describe("Controlled vs Uncontrolled", () => {
        it("works as controlled component", async () => {
            const user = userEvent.setup();
            const ControlledSwitch = () => {
                const [checked, setChecked] = React.useState(false);
                return <Switch checked={checked} onChange={setChecked} />;
            };
            
            render(<ControlledSwitch />);
            
            const checkbox = screen.getByRole("switch") as HTMLInputElement;
            expect(checkbox.checked).toBe(false);
            
            await act(async () => {
                await user.click(checkbox);
            });
            
            expect(checkbox.checked).toBe(true);
        });

        it("maintains state when onChange is not provided", () => {
            render(<Switch checked={true} />);
            
            const checkbox = screen.getByRole("switch") as HTMLInputElement;
            expect(checkbox.checked).toBe(true);
        });
    });

    describe("Forwarded ref", () => {
        it("forwards ref to input element", () => {
            const ref = React.createRef<HTMLInputElement>();
            render(<Switch ref={ref} />);
            
            expect(ref.current).toBeInstanceOf(HTMLInputElement);
            expect(ref.current?.type).toBe("checkbox");
        });
    });

    describe("Additional props", () => {
        it("passes additional props to input", () => {
            render(<Switch data-testid="custom-switch" name="theme-toggle" />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("data-testid")).toBe("custom-switch");
            expect(checkbox.getAttribute("name")).toBe("theme-toggle");
        });

        it("applies custom className", () => {
            render(<Switch className="custom-class" />);
            
            const switchContainer = screen.getByRole("switch").closest("div")?.parentElement;
            expect(switchContainer?.querySelector(".custom-class")).toBeDefined();
        });
    });
});