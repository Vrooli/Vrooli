import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Divider, DividerFactory } from "./Divider.js";

describe("Divider Component", () => {
    describe("Basic rendering", () => {
        it("renders a horizontal divider by default", () => {
            render(<Divider />);
            const divider = screen.getByRole("separator");
            expect(divider).toBeDefined();
            expect(divider.getAttribute("aria-orientation")).toBe("horizontal");
        });

        it("renders a vertical divider when specified", () => {
            render(<Divider orientation="vertical" />);
            const divider = screen.getByRole("separator");
            expect(divider.getAttribute("aria-orientation")).toBe("vertical");
        });

        it("applies custom className", () => {
            const customClass = "custom-divider-class";
            render(<Divider className={customClass} />);
            const divider = screen.getByRole("separator");
            expect(divider.classList.contains(customClass)).toBe(true);
        });
    });

    describe("With children", () => {
        it("renders text content in the middle", () => {
            render(<Divider>OR</Divider>);
            expect(screen.getByText("OR")).toBeDefined();
        });

        it("renders complex children content", () => {
            render(
                <Divider>
                    <span data-testid="complex-content">
                        <span>Multiple</span>
                        <span>Elements</span>
                    </span>
                </Divider>,
            );
            expect(screen.getByTestId("complex-content")).toBeDefined();
        });
    });


    describe("Text alignment", () => {
        it.each(["left", "center", "right"] as const)(
            "aligns text to %s when children provided",
            (textAlign) => {
                render(<Divider textAlign={textAlign}>Text</Divider>);
                expect(screen.getByText("Text")).toBeDefined();
            },
        );
    });


    describe("Accessibility", () => {
        it("has correct ARIA attributes for horizontal divider", () => {
            render(<Divider />);
            const divider = screen.getByRole("separator");
            expect(divider.getAttribute("role")).toBe("separator");
            expect(divider.getAttribute("aria-orientation")).toBe("horizontal");
        });

        it("has correct ARIA attributes for vertical divider", () => {
            render(<Divider orientation="vertical" />);
            const divider = screen.getByRole("separator");
            expect(divider.getAttribute("role")).toBe("separator");
            expect(divider.getAttribute("aria-orientation")).toBe("vertical");
        });
    });

    describe("Factory components", () => {
        it("renders DividerFactory.Horizontal", () => {
            render(<DividerFactory.Horizontal />);
            const divider = screen.getByRole("separator");
            expect(divider.getAttribute("aria-orientation")).toBe("horizontal");
        });

        it("renders DividerFactory.Vertical", () => {
            render(<DividerFactory.Vertical />);
            const divider = screen.getByRole("separator");
            expect(divider.getAttribute("aria-orientation")).toBe("vertical");
        });

    });

    describe("Edge cases", () => {
        it("renders correctly with empty string children", () => {
            render(<Divider>{""}</Divider>);
            const divider = screen.getByRole("separator");
            expect(divider).toBeDefined();
        });

        it("renders correctly with null children", () => {
            render(<Divider>{null}</Divider>);
            const divider = screen.getByRole("separator");
            expect(divider).toBeDefined();
        });

        it("renders correctly with undefined children", () => {
            render(<Divider>{undefined}</Divider>);
            const divider = screen.getByRole("separator");
            expect(divider).toBeDefined();
        });
    });

    describe("Integration scenarios", () => {
        it("works correctly in a form layout", () => {
            render(
                <form>
                    <input type="text" />
                    <Divider>OR</Divider>
                    <input type="email" />
                </form>,
            );
            expect(screen.getByText("OR")).toBeDefined();
        });

        it("works correctly in a flex container", () => {
            render(
                <div style={{ display: "flex", alignItems: "center" }}>
                    <span>Left content</span>
                    <Divider orientation="vertical" />
                    <span>Right content</span>
                </div>,
            );
            const divider = screen.getByRole("separator");
            expect(divider.getAttribute("aria-orientation")).toBe("vertical");
        });
    });
});