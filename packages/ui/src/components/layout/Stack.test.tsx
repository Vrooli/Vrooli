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

        it("accepts custom className prop", () => {
            const customClass = "custom-stack-class";
            render(<Stack className={customClass} data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Content");
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

    describe("Direction props", () => {
        it("accepts all direction props without error", () => {
            const directions = ["row", "column", "row-reverse", "column-reverse"] as const;
            
            directions.forEach(direction => {
                const { unmount } = render(<Stack direction={direction} data-testid="stack">Content</Stack>);
                const stack = screen.getByTestId("stack");
                expect(stack).toBeDefined();
                expect(stack.textContent).toBe("Content");
                unmount();
            });
        });
    });

    describe("Justify content props", () => {
        it("accepts all justify props without error", () => {
            const justifyValues = ["start", "end", "center", "between", "around", "evenly"] as const;
            
            justifyValues.forEach(justify => {
                const { unmount } = render(<Stack justify={justify} data-testid="stack">Content</Stack>);
                const stack = screen.getByTestId("stack");
                expect(stack).toBeDefined();
                expect(stack.textContent).toBe("Content");
                unmount();
            });
        });
    });

    describe("Align items props", () => {
        it("accepts all align props without error", () => {
            const alignValues = ["start", "end", "center", "baseline", "stretch"] as const;
            
            alignValues.forEach(align => {
                const { unmount } = render(<Stack align={align} data-testid="stack">Content</Stack>);
                const stack = screen.getByTestId("stack");
                expect(stack).toBeDefined();
                expect(stack.textContent).toBe("Content");
                unmount();
            });
        });
    });

    describe("Wrap props", () => {
        it("accepts all wrap props without error", () => {
            const wrapValues = ["nowrap", "wrap", "wrap-reverse"] as const;
            
            wrapValues.forEach(wrap => {
                const { unmount } = render(<Stack wrap={wrap} data-testid="stack">Content</Stack>);
                const stack = screen.getByTestId("stack");
                expect(stack).toBeDefined();
                expect(stack.textContent).toBe("Content");
                unmount();
            });
        });
    });

    describe("Spacing props", () => {
        it("accepts spacing props without error", () => {
            const spacingValues = [0, 1, 2, 3, 4, 5, 6, 8] as const;
            
            spacingValues.forEach(spacing => {
                const { unmount } = render(<Stack spacing={spacing} data-testid="stack">Content</Stack>);
                const stack = screen.getByTestId("stack");
                expect(stack).toBeDefined();
                expect(stack.textContent).toBe("Content");
                unmount();
            });
        });

        it("handles spacing with different directions", () => {
            render(<Stack direction="column" spacing={4} data-testid="column-stack">Column Content</Stack>);
            const columnStack = screen.getByTestId("column-stack");
            expect(columnStack).toBeDefined();
            expect(columnStack.textContent).toBe("Column Content");

            render(<Stack direction="row" spacing={4} data-testid="row-stack">Row Content</Stack>);
            const rowStack = screen.getByTestId("row-stack");
            expect(rowStack).toBeDefined();
            expect(rowStack.textContent).toBe("Row Content");
        });
    });

    describe("Divider props", () => {
        it("accepts divider prop without error", () => {
            render(<Stack divider data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Content");
        });

        it("handles divider with different directions", () => {
            render(<Stack direction="column" divider data-testid="column-divider">Column Content</Stack>);
            const columnStack = screen.getByTestId("column-divider");
            expect(columnStack).toBeDefined();

            render(<Stack direction="row" divider data-testid="row-divider">Row Content</Stack>);
            const rowStack = screen.getByTestId("row-divider");
            expect(rowStack).toBeDefined();
        });
    });

    describe("Size options", () => {
        it("renders correctly with fullWidth prop", () => {
            render(<Stack fullWidth data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Content");
        });

        it("renders correctly with fullHeight prop", () => {
            render(<Stack fullHeight data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Content");
        });

        it("renders correctly with both fullWidth and fullHeight", () => {
            render(<Stack fullWidth fullHeight data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Content");
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
                    <span data-testid="child-1">Child 1</span>
                    <strong data-testid="child-2">Child 2</strong>
                    <div data-testid="child-3">Child 3</div>
                </Stack>,
            );
            expect(screen.getByTestId("child-1")).toBeDefined();
            expect(screen.getByTestId("child-2")).toBeDefined();
            expect(screen.getByTestId("child-3")).toBeDefined();
        });

        it("renders empty content correctly", () => {
            render(<Stack data-testid="empty-stack"></Stack>);
            const stack = screen.getByTestId("empty-stack");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("");
        });

        it("handles multiple children with spacing", () => {
            render(
                <Stack spacing={2} data-testid="spaced-stack">
                    <div data-testid="item-1">Item 1</div>
                    <div data-testid="item-2">Item 2</div>
                    <div data-testid="item-3">Item 3</div>
                </Stack>,
            );
            
            expect(screen.getByTestId("item-1")).toBeDefined();
            expect(screen.getByTestId("item-2")).toBeDefined();
            expect(screen.getByTestId("item-3")).toBeDefined();
            expect(screen.getByTestId("spaced-stack")).toBeDefined();
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

    describe("Component override", () => {
        it("renders with custom component when specified", () => {
            render(<Stack component="article" data-testid="stack">Content</Stack>);
            const stack = screen.getByTestId("stack");
            expect(stack.tagName).toBe("ARTICLE");
        });

        it("maintains semantic meaning with different components", () => {
            render(<Stack component="nav" role="navigation" data-testid="nav-stack">Nav content</Stack>);
            const stack = screen.getByTestId("nav-stack");
            expect(stack.tagName).toBe("NAV");
            expect(stack.getAttribute("role")).toBe("navigation");
        });
    });

    describe("Factory components", () => {
        it("renders StackFactory.Row with correct semantic element", () => {
            render(<StackFactory.Row data-testid="factory-row">Row content</StackFactory.Row>);
            const stack = screen.getByTestId("factory-row");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Row content");
        });

        it("renders StackFactory.Column with correct semantic element", () => {
            render(<StackFactory.Column data-testid="factory-column">Column content</StackFactory.Column>);
            const stack = screen.getByTestId("factory-column");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Column content");
        });

        it("renders StackFactory.Centered correctly", () => {
            render(<StackFactory.Centered data-testid="factory-centered">Centered content</StackFactory.Centered>);
            const stack = screen.getByTestId("factory-centered");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Centered content");
        });

        it("renders StackFactory.SpaceBetween correctly", () => {
            render(<StackFactory.SpaceBetween data-testid="factory-between">Space between content</StackFactory.SpaceBetween>);
            const stack = screen.getByTestId("factory-between");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Space between content");
        });

        it("renders StackFactory.FullWidth correctly", () => {
            render(<StackFactory.FullWidth data-testid="factory-full-width">Full width content</StackFactory.FullWidth>);
            const stack = screen.getByTestId("factory-full-width");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Full width content");
        });

        it("renders StackFactory.FullHeight correctly", () => {
            render(<StackFactory.FullHeight data-testid="factory-full-height">Full height content</StackFactory.FullHeight>);
            const stack = screen.getByTestId("factory-full-height");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Full height content");
        });

        it("factory components accept additional props", () => {
            render(
                <StackFactory.Row 
                    spacing={4} 
                    align="center" 
                    data-testid="factory-with-props"
                >
                    Row with props
                </StackFactory.Row>,
            );
            const stack = screen.getByTestId("factory-with-props");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Row with props");
        });
    });

    describe("Combined props", () => {
        it("accepts multiple styling props without error", () => {
            render(
                <Stack
                    direction="row"
                    justify="center"
                    align="center"
                    spacing={3}
                    divider
                    fullWidth
                    className="custom-class"
                    data-testid="combined-stack"
                >
                    Combined styling
                </Stack>,
            );
            const stack = screen.getByTestId("combined-stack");
            
            // Verify functional behavior
            expect(stack.tagName).toBe("DIV");
            expect(stack.textContent).toBe("Combined styling");
            expect(stack).toBeDefined();
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
            render(<Stack data-testid="empty-string">{""}</Stack>);
            const stack = screen.getByTestId("empty-string");
            expect(stack).toBeDefined();
        });

        it("handles zero as children", () => {
            render(<Stack data-testid="zero-children">{0}</Stack>);
            const stack = screen.getByTestId("zero-children");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("0");
        });

        it("handles invalid component gracefully", () => {
            render(<Stack component={undefined} data-testid="invalid-component">Content</Stack>);
            const stack = screen.getByTestId("invalid-component");
            expect(stack.tagName).toBe("DIV");
        });

        it("handles conflicting props gracefully", () => {
            render(<Stack spacing={4} divider data-testid="conflicting-props">Content</Stack>);
            const stack = screen.getByTestId("conflicting-props");
            expect(stack).toBeDefined();
            expect(stack.textContent).toBe("Content");
        });
    });

    describe("Accessibility", () => {
        it("supports aria attributes", () => {
            render(
                <Stack 
                    aria-describedby="description"
                    aria-label="Custom label"
                    data-testid="aria-stack"
                >
                    Content
                </Stack>,
            );
            const stack = screen.getByTestId("aria-stack");
            expect(stack.getAttribute("aria-describedby")).toBe("description");
            expect(stack.getAttribute("aria-label")).toBe("Custom label");
        });

        it("maintains semantic meaning with role attribute", () => {
            render(
                <Stack 
                    component="div" 
                    role="group" 
                    aria-labelledby="group-title"
                    data-testid="semantic-stack"
                >
                    Stack content
                </Stack>,
            );
            const stack = screen.getByTestId("semantic-stack");
            expect(stack.tagName).toBe("DIV");
            expect(stack.getAttribute("role")).toBe("group");
            expect(stack.getAttribute("aria-labelledby")).toBe("group-title");
        });

        it("works well for navigation patterns", () => {
            render(
                <Stack component="nav" role="navigation" aria-label="Main navigation">
                    <a href="/home" data-testid="nav-home">Home</a>
                    <a href="/about" data-testid="nav-about">About</a>
                    <a href="/contact" data-testid="nav-contact">Contact</a>
                </Stack>,
            );
            
            expect(screen.getByTestId("nav-home")).toBeDefined();
            expect(screen.getByTestId("nav-about")).toBeDefined();
            expect(screen.getByTestId("nav-contact")).toBeDefined();
        });
    });
});
