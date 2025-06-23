import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Stack, StackFactory } from "./Stack.js";

describe("Stack Component", () => {
    describe("Basic rendering", () => {
        it("renders with default props", () => {
            render(<Stack data-testid="stack">Test content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack).toBeDefined();
            expect(stack.tagName).toBe("DIV");
            expect(stack.textContent).toBe("Test content");
        });

        it("renders with custom HTML element", () => {
            render(<Stack component="section" data-testid="stack">Section content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.tagName).toBe("SECTION");
            expect(stack.textContent).toBe("Section content");
        });

        it("applies custom className", () => {
            const customClass = "custom-stack-class";
            render(<Stack className={customClass} data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains(customClass)).toBe(true);
        });

        it("forwards ref correctly", () => {
            let refElement: HTMLElement | null = null;
            const TestComponent = () => (
                <Stack ref={(el) => { refElement = el; }} data-testid="stack">Content</Stack>
            );
            render(<TestComponent />);
            expect(refElement).toBeTruthy();
            expect(refElement?.tagName).toBe("DIV");
        });
    });

    describe("Direction", () => {
        it.each([
            ["row", "tw-flex-row"],
            ["column", "tw-flex-col"],
            ["row-reverse", "tw-flex-row-reverse"],
            ["column-reverse", "tw-flex-col-reverse"],
        ] as const)("applies %s direction classes", (direction, expectedClass) => {
            render(<Stack direction={direction} data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains(expectedClass)).toBe(true);
        });

        it("defaults to column direction", () => {
            render(<Stack data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains("tw-flex-col")).toBe(true);
        });
    });

    describe("Justify content", () => {
        it.each([
            ["start", "tw-justify-start"],
            ["end", "tw-justify-end"],
            ["center", "tw-justify-center"],
            ["between", "tw-justify-between"],
            ["around", "tw-justify-around"],
            ["evenly", "tw-justify-evenly"],
        ] as const)("applies %s justify classes", (justify, expectedClass) => {
            render(<Stack justify={justify} data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains(expectedClass)).toBe(true);
        });
    });

    describe("Align items", () => {
        it.each([
            ["start", "tw-items-start"],
            ["end", "tw-items-end"],
            ["center", "tw-items-center"],
            ["baseline", "tw-items-baseline"],
            ["stretch", "tw-items-stretch"],
        ] as const)("applies %s align classes", (align, expectedClass) => {
            render(<Stack align={align} data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains(expectedClass)).toBe(true);
        });
    });

    describe("Wrap", () => {
        it.each([
            ["nowrap", "tw-flex-nowrap"],
            ["wrap", "tw-flex-wrap"],
            ["wrap-reverse", "tw-flex-wrap-reverse"],
        ] as const)("applies %s wrap classes", (wrap, expectedClass) => {
            render(<Stack wrap={wrap} data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains(expectedClass)).toBe(true);
        });
    });

    describe("Spacing", () => {
        describe("Column direction spacing", () => {
            it.each([
                ["none", ""],
                ["xs", "tw-space-y-1"],
                ["sm", "tw-space-y-2"],
                ["md", "tw-space-y-4"],
                ["lg", "tw-space-y-6"],
                ["xl", "tw-space-y-8"],
                ["2xl", "tw-space-y-12"],
            ] as const)("applies %s column spacing", (spacing, expectedClass) => {
                render(<Stack direction="column" spacing={spacing} data-testid="stack">Content</Stack>);
                const stack = screen.getByTestId("stack");
                if (expectedClass) {
                    expect(stack.classList.contains(expectedClass)).toBe(true);
                }
            });
        });

        describe("Row direction spacing", () => {
            it.each([
                ["none", ""],
                ["xs", "tw-space-x-1"],
                ["sm", "tw-space-x-2"],
                ["md", "tw-space-x-4"],
                ["lg", "tw-space-x-6"],
                ["xl", "tw-space-x-8"],
                ["2xl", "tw-space-x-12"],
            ] as const)("applies %s row spacing", (spacing, expectedClass) => {
                render(<Stack direction="row" spacing={spacing} data-testid="stack">Content</Stack>);
                const stack = screen.getByTestId("stack");
                if (expectedClass) {
                    expect(stack.classList.contains(expectedClass)).toBe(true);
                }
            });
        });
    });

    describe("Dividers", () => {
        describe("Column direction dividers", () => {
            it.each([
                ["xs", "tw-divide-y-1"],
                ["sm", "tw-divide-y-2"],
                ["md", "tw-divide-y-4"],
                ["lg", "tw-divide-y-6"],
                ["xl", "tw-divide-y-8"],
                ["2xl", "tw-divide-y-12"],
            ] as const)("applies %s column divider", (spacing, expectedClass) => {
                render(<Stack direction="column" spacing={spacing} divider data-testid="stack">Content</Stack>);
                const stack = screen.getByTestId("stack");
                expect(stack.classList.contains("tw-divide-y")).toBe(true);
                expect(stack.classList.contains(expectedClass)).toBe(true);
            });
        });

        describe("Row direction dividers", () => {
            it.each([
                ["xs", "tw-divide-x-1"],
                ["sm", "tw-divide-x-2"],
                ["md", "tw-divide-x-4"],
                ["lg", "tw-divide-x-6"],
                ["xl", "tw-divide-x-8"],
                ["2xl", "tw-divide-x-12"],
            ] as const)("applies %s row divider", (spacing, expectedClass) => {
                render(<Stack direction="row" spacing={spacing} divider data-testid="stack">Content</Stack>);
                const stack = screen.getByTestId("stack");
                expect(stack.classList.contains("tw-divide-x")).toBe(true);
                expect(stack.classList.contains(expectedClass)).toBe(true);
            });
        });

        it("uses spacing classes when divider is false", () => {
            render(<Stack direction="row" spacing="md" divider={false} data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains("tw-space-x-4")).toBe(true);
            expect(stack.classList.contains("tw-divide-x")).toBe(false);
        });

        it("uses divider classes when divider is true", () => {
            render(<Stack direction="row" spacing="md" divider data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains("tw-divide-x")).toBe(true);
            expect(stack.classList.contains("tw-space-x-4")).toBe(false);
        });
    });

    describe("Size options", () => {
        it("applies full width when specified", () => {
            render(<Stack fullWidth data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains("tw-w-full")).toBe(true);
        });

        it("applies full height when specified", () => {
            render(<Stack fullHeight data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains("tw-h-full")).toBe(true);
        });

        it("applies both full width and height", () => {
            render(<Stack fullWidth fullHeight data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains("tw-w-full")).toBe(true);
            expect(stack.classList.contains("tw-h-full")).toBe(true);
        });
    });

    describe("Base flex class", () => {
        it("always applies base flex class", () => {
            render(<Stack data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.classList.contains("tw-flex")).toBe(true);
        });
    });

    describe("Content", () => {
        it("renders text content", () => {
            render(<Stack>Simple text content</Stack>);
            expect(screen.getByText("Simple text content")).toBeDefined();
        });

        it("renders complex children content", () => {
            render(
                <Stack>
                    <div data-testid="child-1">Child 1</div>
                    <span data-testid="child-2">Child 2</span>
                </Stack>,
            );
            expect(screen.getByTestId("child-1")).toBeDefined();
            expect(screen.getByTestId("child-2")).toBeDefined();
        });

        it("renders empty content correctly", () => {
            render(<Stack data-testid="empty-stack"></Stack>);
            const stack = screen.getByTestId("empty-stack");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("");
        });
    });

    describe("HTML attributes", () => {
        it("passes through HTML attributes", () => {
            render(
                <Stack
                    id="test-id"
                    role="region"
                    aria-label="Test stack"
                    data-testid="stack"
                >
                    Content
                </Stack>,
            );
            const stack = screen.getByTestId("stack");
            expect(stack.getAttribute("id")).toBe("test-id");
            expect(stack.getAttribute("role")).toBe("region");
            expect(stack.getAttribute("aria-label")).toBe("Test stack");
        });

        it("handles event handlers", () => {
            let clicked = false;
            const handleClick = () => { clicked = true; };
            
            render(<Stack onClick={handleClick} data-testid="clickable-stack">Click me</Stack>);
            const stack = screen.getByTestId("clickable-stack");
            stack.click();
            expect(clicked).toBe(true);
        });
    });

    describe("Factory components", () => {
        it("renders StackFactory.Vertical", () => {
            render(<StackFactory.Vertical data-testid="factory-vertical">Vertical content</StackFactory.Vertical>);
            const stack = screen.getByTestId("factory-vertical");
            expect(stack.classList.contains("tw-flex-col")).toBe(true);
            expect(stack.classList.contains("tw-space-y-4")).toBe(true);
        });

        it("renders StackFactory.Horizontal", () => {
            render(<StackFactory.Horizontal data-testid="factory-horizontal">Horizontal content</StackFactory.Horizontal>);
            const stack = screen.getByTestId("factory-horizontal");
            expect(stack.classList.contains("tw-flex-row")).toBe(true);
            expect(stack.classList.contains("tw-space-x-4")).toBe(true);
        });

        it("renders StackFactory.Center", () => {
            render(<StackFactory.Center data-testid="factory-center">Centered content</StackFactory.Center>);
            const stack = screen.getByTestId("factory-center");
            expect(stack.classList.contains("tw-justify-center")).toBe(true);
            expect(stack.classList.contains("tw-items-center")).toBe(true);
        });

        it("renders StackFactory.Between", () => {
            render(<StackFactory.Between data-testid="factory-between">Between content</StackFactory.Between>);
            const stack = screen.getByTestId("factory-between");
            expect(stack.classList.contains("tw-justify-between")).toBe(true);
        });

        it("renders StackFactory.HorizontalFull", () => {
            render(<StackFactory.HorizontalFull data-testid="factory-horizontal-full">Full width content</StackFactory.HorizontalFull>);
            const stack = screen.getByTestId("factory-horizontal-full");
            expect(stack.classList.contains("tw-flex-row")).toBe(true);
            expect(stack.classList.contains("tw-w-full")).toBe(true);
        });

        it("renders StackFactory.VerticalFull", () => {
            render(<StackFactory.VerticalFull data-testid="factory-vertical-full">Full height content</StackFactory.VerticalFull>);
            const stack = screen.getByTestId("factory-vertical-full");
            expect(stack.classList.contains("tw-flex-col")).toBe(true);
            expect(stack.classList.contains("tw-h-full")).toBe(true);
        });

        it("renders StackFactory.Navbar", () => {
            render(<StackFactory.Navbar data-testid="factory-navbar">Navbar content</StackFactory.Navbar>);
            const stack = screen.getByTestId("factory-navbar");
            expect(stack.classList.contains("tw-flex-row")).toBe(true);
            expect(stack.classList.contains("tw-justify-between")).toBe(true);
            expect(stack.classList.contains("tw-items-center")).toBe(true);
            expect(stack.classList.contains("tw-w-full")).toBe(true);
        });

        it("renders StackFactory.Sidebar", () => {
            render(<StackFactory.Sidebar data-testid="factory-sidebar">Sidebar content</StackFactory.Sidebar>);
            const stack = screen.getByTestId("factory-sidebar");
            expect(stack.classList.contains("tw-flex-col")).toBe(true);
            expect(stack.classList.contains("tw-h-full")).toBe(true);
            expect(stack.classList.contains("tw-space-y-2")).toBe(true);
        });

        it("renders StackFactory.ButtonGroup", () => {
            render(<StackFactory.ButtonGroup data-testid="factory-button-group">Button group content</StackFactory.ButtonGroup>);
            const stack = screen.getByTestId("factory-button-group");
            expect(stack.classList.contains("tw-flex-row")).toBe(true);
            expect(stack.classList.contains("tw-divide-x")).toBe(true);
        });

        it("renders StackFactory.Form", () => {
            render(<StackFactory.Form data-testid="factory-form">Form content</StackFactory.Form>);
            const stack = screen.getByTestId("factory-form");
            expect(stack.classList.contains("tw-flex-col")).toBe(true);
            expect(stack.classList.contains("tw-space-y-6")).toBe(true);
            expect(stack.classList.contains("tw-w-full")).toBe(true);
        });
    });

    describe("Class name combinations", () => {
        it("combines multiple styling props correctly", () => {
            render(
                <Stack
                    direction="row"
                    justify="center"
                    align="center"
                    wrap="wrap"
                    spacing="lg"
                    fullWidth
                    className="custom-class"
                    data-testid="combined-stack"
                >
                    Combined styling
                </Stack>,
            );
            const stack = screen.getByTestId("combined-stack");
            
            // Direction
            expect(stack.classList.contains("tw-flex-row")).toBe(true);
            
            // Justify
            expect(stack.classList.contains("tw-justify-center")).toBe(true);
            
            // Align
            expect(stack.classList.contains("tw-items-center")).toBe(true);
            
            // Wrap
            expect(stack.classList.contains("tw-flex-wrap")).toBe(true);
            
            // Spacing
            expect(stack.classList.contains("tw-space-x-6")).toBe(true);
            
            // Full width
            expect(stack.classList.contains("tw-w-full")).toBe(true);
            
            // Custom class
            expect(stack.classList.contains("custom-class")).toBe(true);
        });

        it("applies divider instead of spacing when both are specified", () => {
            render(
                <Stack
                    direction="row"
                    spacing="md"
                    divider
                    data-testid="divider-stack"
                >
                    Content with divider
                </Stack>,
            );
            const stack = screen.getByTestId("divider-stack");
            
            // Should have divider classes
            expect(stack.classList.contains("tw-divide-x")).toBe(true);
            expect(stack.classList.contains("tw-divide-x-4")).toBe(true);
            
            // Should NOT have spacing classes
            expect(stack.classList.contains("tw-space-x-4")).toBe(false);
        });
    });

    describe("Edge cases", () => {
        it("handles null children", () => {
            render(<Stack data-testid="null-children">{null}</Stack>);
            const stack = screen.getByTestId("null-children");
            expect(stack).toBeDefined();
        });

        it("handles undefined children", () => {
            render(<Stack data-testid="undefined-children">{undefined}</Stack>);
            const stack = screen.getByTestId("undefined-children");
            expect(stack).toBeDefined();
        });

        it("handles empty string children", () => {
            render(<Stack data-testid="empty-string">{""}
</Stack>);
            const stack = screen.getByTestId("empty-string");
            expect(stack).toBeDefined();
        });

        it("handles zero as children", () => {
            render(<Stack data-testid="zero-children">{0}</Stack>);
            const stack = screen.getByTestId("zero-children");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("0");
        });
    });

    describe("Integration scenarios", () => {
        it("works as a navigation container", () => {
            render(
                <Stack component="nav" direction="row" spacing="lg" data-testid="nav-stack">
                    <a href="#home">Home</a>
                    <a href="#about">About</a>
                    <a href="#contact">Contact</a>
                </Stack>,
            );
            const navStack = screen.getByTestId("nav-stack");
            expect(navStack.tagName).toBe("NAV");
            expect(screen.getByRole("link", { name: "Home" })).toBeDefined();
            expect(screen.getByRole("link", { name: "About" })).toBeDefined();
            expect(screen.getByRole("link", { name: "Contact" })).toBeDefined();
        });

        it("works in a nested layout", () => {
            render(
                <Stack direction="column" spacing="lg" data-testid="outer-stack">
                    <Stack direction="row" spacing="md" data-testid="inner-stack-1">
                        <span>Left</span>
                        <span>Right</span>
                    </Stack>
                    <Stack direction="column" spacing="sm" data-testid="inner-stack-2">
                        <span>Top</span>
                        <span>Bottom</span>
                    </Stack>
                </Stack>,
            );
            
            const outerStack = screen.getByTestId("outer-stack");
            const innerStack1 = screen.getByTestId("inner-stack-1");
            const innerStack2 = screen.getByTestId("inner-stack-2");
            
            expect(outerStack.contains(innerStack1)).toBe(true);
            expect(outerStack.contains(innerStack2)).toBe(true);
            expect(screen.getByText("Left")).toBeDefined();
            expect(screen.getByText("Right")).toBeDefined();
            expect(screen.getByText("Top")).toBeDefined();
            expect(screen.getByText("Bottom")).toBeDefined();
        });

        it("works with different semantic elements", () => {
            render(
                <div data-testid="layout-container">
                    <Stack component="header" direction="row" justify="between" data-testid="header-stack">
                        <span>Logo</span>
                        <span>Menu</span>
                    </Stack>
                    <Stack component="main" direction="column" spacing="lg" data-testid="main-stack">
                        <h1>Main Content</h1>
                        <p>Content paragraph</p>
                    </Stack>
                    <Stack component="aside" direction="column" spacing="sm" data-testid="aside-stack">
                        <h2>Sidebar</h2>
                        <nav>Navigation</nav>
                    </Stack>
                    <Stack component="footer" direction="row" justify="center" data-testid="footer-stack">
                        <span>Footer content</span>
                    </Stack>
                </div>,
            );
            
            expect(screen.getByTestId("header-stack").tagName).toBe("HEADER");
            expect(screen.getByTestId("main-stack").tagName).toBe("MAIN");
            expect(screen.getByTestId("aside-stack").tagName).toBe("ASIDE");
            expect(screen.getByTestId("footer-stack").tagName).toBe("FOOTER");
        });

        it("works as a form layout", () => {
            render(
                <Stack component="form" direction="column" spacing="lg" data-testid="form-stack">
                    <Stack direction="column" spacing="sm">
                        <label htmlFor="name">Name</label>
                        <input id="name" type="text" />
                    </Stack>
                    <Stack direction="column" spacing="sm">
                        <label htmlFor="email">Email</label>
                        <input id="email" type="email" />
                    </Stack>
                    <Stack direction="row" justify="end" spacing="md">
                        <button type="button">Cancel</button>
                        <button type="submit">Submit</button>
                    </Stack>
                </Stack>,
            );
            
            const formStack = screen.getByTestId("form-stack");
            expect(formStack.tagName).toBe("FORM");
            expect(screen.getByLabelText("Name")).toBeDefined();
            expect(screen.getByLabelText("Email")).toBeDefined();
            expect(screen.getByRole("button", { name: "Cancel" })).toBeDefined();
            expect(screen.getByRole("button", { name: "Submit" })).toBeDefined();
        });

        it("works as a card layout", () => {
            render(
                <Stack direction="row" wrap="wrap" spacing="lg" data-testid="card-grid">
                    {Array.from({ length: 3 }, (_, i) => (
                        <div key={i} className="card">
                            <Stack direction="column" spacing="md">
                                <h3>Card {i + 1}</h3>
                                <p>Card content goes here</p>
                                <Stack direction="row" justify="end">
                                    <button>Action</button>
                                </Stack>
                            </Stack>
                        </div>
                    ))}
                </Stack>,
            );
            
            const cardGrid = screen.getByTestId("card-grid");
            expect(cardGrid.classList.contains("tw-flex-wrap")).toBe(true);
            expect(screen.getByText("Card 1")).toBeDefined();
            expect(screen.getByText("Card 2")).toBeDefined();
            expect(screen.getByText("Card 3")).toBeDefined();
        });
    });
});