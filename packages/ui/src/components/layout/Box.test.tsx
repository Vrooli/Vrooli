import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Box, BoxFactory } from "./Box.js";

describe("Box Component", () => {
    describe("Basic rendering", () => {
        it("renders with default props", () => {
            render(<Box data-testid="box">Test content</Box>);
            const box = screen.getByTestId("box");
            expect(box).toBeDefined();
            expect(box.tagName).toBe("DIV");
            expect(box.textContent).toBe("Test content");
        });

        it("renders with custom HTML element", () => {
            render(<Box component="section" data-testid="box">Section content</Box>);
            const box = screen.getByTestId("box");
            expect(box.tagName).toBe("SECTION");
            expect(box.textContent).toBe("Section content");
        });

        it("applies custom className", () => {
            const customClass = "custom-box-class";
            render(<Box className={customClass} data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            expect(box.classList.contains(customClass)).toBe(true);
        });

        it("forwards ref correctly", () => {
            let refElement: HTMLElement | null = null;
            const TestComponent = () => (
                <Box ref={(el) => { refElement = el; }} data-testid="box">Content</Box>
            );
            render(<TestComponent />);
            expect(refElement).toBeTruthy();
            expect(refElement?.tagName).toBe("DIV");
        });
    });

    describe("Variants", () => {
        it.each([
            ["default", "tw-bg-background-default"],
            ["paper", "tw-bg-background-paper"],
            ["outlined", "tw-border"],
            ["elevated", "tw-shadow-md"],
            ["subtle", "tw-bg-gray-50"],
        ] as const)("applies %s variant classes", (variant, expectedClass) => {
            render(<Box variant={variant} data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            expect(box.classList.contains(expectedClass)).toBe(true);
        });
    });

    describe("Padding", () => {
        it.each([
            ["none", ""],
            ["xs", "tw-p-1"],
            ["sm", "tw-p-2"],
            ["md", "tw-p-4"],
            ["lg", "tw-p-6"],
            ["xl", "tw-p-8"],
        ] as const)("applies %s padding", (padding, expectedClass) => {
            render(<Box padding={padding} data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            if (expectedClass) {
                expect(box.classList.contains(expectedClass)).toBe(true);
            }
        });
    });

    describe("Border radius", () => {
        it.each([
            ["none", "tw-rounded-none"],
            ["sm", "tw-rounded-sm"],
            ["md", "tw-rounded-md"],
            ["lg", "tw-rounded-lg"],
            ["xl", "tw-rounded-xl"],
            ["full", "tw-rounded-full"],
        ] as const)("applies %s border radius", (borderRadius, expectedClass) => {
            render(<Box borderRadius={borderRadius} data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            expect(box.classList.contains(expectedClass)).toBe(true);
        });
    });

    describe("Size options", () => {
        it("applies full width when specified", () => {
            render(<Box fullWidth data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            expect(box.classList.contains("tw-w-full")).toBe(true);
        });

        it("applies full height when specified", () => {
            render(<Box fullHeight data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            expect(box.classList.contains("tw-h-full")).toBe(true);
        });

        it("applies both full width and height", () => {
            render(<Box fullWidth fullHeight data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            expect(box.classList.contains("tw-w-full")).toBe(true);
            expect(box.classList.contains("tw-h-full")).toBe(true);
        });
    });

    describe("Content", () => {
        it("renders text content", () => {
            render(<Box>Simple text content</Box>);
            expect(screen.getByText("Simple text content")).toBeDefined();
        });

        it("renders complex children content", () => {
            render(
                <Box>
                    <div data-testid="child-1">Child 1</div>
                    <span data-testid="child-2">Child 2</span>
                </Box>,
            );
            expect(screen.getByTestId("child-1")).toBeDefined();
            expect(screen.getByTestId("child-2")).toBeDefined();
        });

        it("renders empty content correctly", () => {
            render(<Box data-testid="empty-box"></Box>);
            const box = screen.getByTestId("empty-box");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("");
        });
    });

    describe("HTML attributes", () => {
        it("passes through HTML attributes", () => {
            render(
                <Box
                    id="test-id"
                    role="region"
                    aria-label="Test box"
                    data-testid="box"
                >
                    Content
                </Box>,
            );
            const box = screen.getByTestId("box");
            expect(box.getAttribute("id")).toBe("test-id");
            expect(box.getAttribute("role")).toBe("region");
            expect(box.getAttribute("aria-label")).toBe("Test box");
        });

        it("handles event handlers", () => {
            let clicked = false;
            const handleClick = () => { clicked = true; };
            
            render(<Box onClick={handleClick} data-testid="clickable-box">Click me</Box>);
            const box = screen.getByTestId("clickable-box");
            box.click();
            expect(clicked).toBe(true);
        });
    });

    describe("Factory components", () => {
        it("renders BoxFactory.Paper", () => {
            render(<BoxFactory.Paper data-testid="factory-paper">Paper content</BoxFactory.Paper>);
            const box = screen.getByTestId("factory-paper");
            expect(box.classList.contains("tw-bg-background-paper")).toBe(true);
        });

        it("renders BoxFactory.Card", () => {
            render(<BoxFactory.Card data-testid="factory-card">Card content</BoxFactory.Card>);
            const box = screen.getByTestId("factory-card");
            expect(box.classList.contains("tw-bg-background-paper")).toBe(true);
            expect(box.classList.contains("tw-shadow-md")).toBe(true);
            expect(box.classList.contains("tw-p-6")).toBe(true);
            expect(box.classList.contains("tw-rounded-lg")).toBe(true);
        });

        it("renders BoxFactory.Outlined", () => {
            render(<BoxFactory.Outlined data-testid="factory-outlined">Outlined content</BoxFactory.Outlined>);
            const box = screen.getByTestId("factory-outlined");
            expect(box.classList.contains("tw-border")).toBe(true);
        });

        it("renders BoxFactory.Subtle", () => {
            render(<BoxFactory.Subtle data-testid="factory-subtle">Subtle content</BoxFactory.Subtle>);
            const box = screen.getByTestId("factory-subtle");
            expect(box.classList.contains("tw-bg-gray-50")).toBe(true);
        });

        it("renders BoxFactory.FullWidth", () => {
            render(<BoxFactory.FullWidth data-testid="factory-full-width">Full width content</BoxFactory.FullWidth>);
            const box = screen.getByTestId("factory-full-width");
            expect(box.classList.contains("tw-w-full")).toBe(true);
        });

        it("renders BoxFactory.FlexCenter", () => {
            render(<BoxFactory.FlexCenter data-testid="factory-flex-center">Centered content</BoxFactory.FlexCenter>);
            const box = screen.getByTestId("factory-flex-center");
            expect(box.classList.contains("tw-flex")).toBe(true);
            expect(box.classList.contains("tw-items-center")).toBe(true);
            expect(box.classList.contains("tw-justify-center")).toBe(true);
        });

        it("renders BoxFactory.FlexBetween", () => {
            render(<BoxFactory.FlexBetween data-testid="factory-flex-between">Between content</BoxFactory.FlexBetween>);
            const box = screen.getByTestId("factory-flex-between");
            expect(box.classList.contains("tw-flex")).toBe(true);
            expect(box.classList.contains("tw-items-center")).toBe(true);
            expect(box.classList.contains("tw-justify-between")).toBe(true);
        });
    });

    describe("Class name combinations", () => {
        it("combines multiple styling props correctly", () => {
            render(
                <Box
                    variant="elevated"
                    padding="lg"
                    borderRadius="xl"
                    fullWidth
                    className="custom-class"
                    data-testid="combined-box"
                >
                    Combined styling
                </Box>,
            );
            const box = screen.getByTestId("combined-box");
            
            // Variant
            expect(box.classList.contains("tw-bg-background-paper")).toBe(true);
            expect(box.classList.contains("tw-shadow-md")).toBe(true);
            
            // Padding
            expect(box.classList.contains("tw-p-6")).toBe(true);
            
            // Border radius
            expect(box.classList.contains("tw-rounded-xl")).toBe(true);
            
            // Full width
            expect(box.classList.contains("tw-w-full")).toBe(true);
            
            // Custom class
            expect(box.classList.contains("custom-class")).toBe(true);
        });
    });

    describe("Edge cases", () => {
        it("handles null children", () => {
            render(<Box data-testid="null-children">{null}</Box>);
            const box = screen.getByTestId("null-children");
            expect(box).toBeDefined();
        });

        it("handles undefined children", () => {
            render(<Box data-testid="undefined-children">{undefined}</Box>);
            const box = screen.getByTestId("undefined-children");
            expect(box).toBeDefined();
        });

        it("handles empty string children", () => {
            render(<Box data-testid="empty-string">{""}
</Box>);
            const box = screen.getByTestId("empty-string");
            expect(box).toBeDefined();
        });

        it("handles zero as children", () => {
            render(<Box data-testid="zero-children">{0}</Box>);
            const box = screen.getByTestId("zero-children");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("0");
        });
    });

    describe("Integration scenarios", () => {
        it("works as a form container", () => {
            render(
                <Box component="form" variant="outlined" padding="lg" borderRadius="md" data-testid="form-box">
                    <input type="text" placeholder="Name" />
                    <input type="email" placeholder="Email" />
                    <button type="submit">Submit</button>
                </Box>,
            );
            const formBox = screen.getByTestId("form-box");
            expect(formBox.tagName).toBe("FORM");
            expect(screen.getByPlaceholderText("Name")).toBeDefined();
            expect(screen.getByPlaceholderText("Email")).toBeDefined();
            expect(screen.getByRole("button", { name: "Submit" })).toBeDefined();
        });

        it("works in a nested layout", () => {
            render(
                <Box variant="paper" padding="lg" data-testid="outer-box">
                    <Box variant="subtle" padding="md" data-testid="inner-box-1">
                        Inner content 1
                    </Box>
                    <Box variant="outlined" padding="sm" data-testid="inner-box-2">
                        Inner content 2
                    </Box>
                </Box>,
            );
            
            const outerBox = screen.getByTestId("outer-box");
            const innerBox1 = screen.getByTestId("inner-box-1");
            const innerBox2 = screen.getByTestId("inner-box-2");
            
            expect(outerBox.contains(innerBox1)).toBe(true);
            expect(outerBox.contains(innerBox2)).toBe(true);
            expect(innerBox1.textContent).toBe("Inner content 1");
            expect(innerBox2.textContent).toBe("Inner content 2");
        });

        it("works with different semantic elements", () => {
            render(
                <div data-testid="layout-container">
                    <Box component="header" variant="paper" padding="md" data-testid="header-box">
                        Header content
                    </Box>
                    <Box component="main" variant="default" padding="lg" data-testid="main-box">
                        Main content
                    </Box>
                    <Box component="aside" variant="outlined" padding="sm" data-testid="aside-box">
                        Sidebar content
                    </Box>
                    <Box component="footer" variant="subtle" padding="xs" data-testid="footer-box">
                        Footer content
                    </Box>
                </div>,
            );
            
            expect(screen.getByTestId("header-box").tagName).toBe("HEADER");
            expect(screen.getByTestId("main-box").tagName).toBe("MAIN");
            expect(screen.getByTestId("aside-box").tagName).toBe("ASIDE");
            expect(screen.getByTestId("footer-box").tagName).toBe("FOOTER");
        });
    });
});