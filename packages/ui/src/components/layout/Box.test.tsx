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

        it("accepts custom className prop", () => {
            const customClass = "custom-box-class";
            render(<Box className={customClass} data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("Content");
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
        it("accepts all variant props without error", () => {
            const variants = ["default", "paper", "outlined", "elevated", "subtle"] as const;
            
            variants.forEach(variant => {
                const { unmount } = render(<Box variant={variant} data-testid="box">Content</Box>);
                const box = screen.getByTestId("box");
                expect(box).toBeDefined();
                expect(box.textContent).toBe("Content");
                unmount();
            });
        });
    });

    describe("Padding", () => {
        it("accepts all padding props without error", () => {
            const paddingValues = ["none", "xs", "sm", "md", "lg", "xl"] as const;
            
            paddingValues.forEach(padding => {
                const { unmount } = render(<Box padding={padding} data-testid="box">Content</Box>);
                const box = screen.getByTestId("box");
                expect(box).toBeDefined();
                expect(box.textContent).toBe("Content");
                unmount();
            });
        });
    });

    describe("Border radius", () => {
        it("accepts all border radius props without error", () => {
            const borderRadiusValues = ["none", "sm", "md", "lg", "xl", "full"] as const;
            
            borderRadiusValues.forEach(borderRadius => {
                const { unmount } = render(<Box borderRadius={borderRadius} data-testid="box">Content</Box>);
                const box = screen.getByTestId("box");
                expect(box).toBeDefined();
                expect(box.textContent).toBe("Content");
                unmount();
            });
        });
    });

    describe("Size options", () => {
        it("renders correctly with fullWidth prop", () => {
            render(<Box fullWidth data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("Content");
        });

        it("renders correctly with fullHeight prop", () => {
            render(<Box fullHeight data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("Content");
        });

        it("renders correctly with both fullWidth and fullHeight", () => {
            render(<Box fullWidth fullHeight data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("Content");
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
                    <span data-testid="child-1">Child 1</span>
                    <strong data-testid="child-2">Child 2</strong>
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

    describe("Component override", () => {
        it("renders with custom component when specified", () => {
            render(<Box component="article" data-testid="box">Content</Box>);
            const box = screen.getByTestId("box");
            expect(box.tagName).toBe("ARTICLE");
        });

        it("maintains semantic meaning with different components", () => {
            render(<Box component="main" role="main" data-testid="main-box">Main content</Box>);
            const box = screen.getByTestId("main-box");
            expect(box.tagName).toBe("MAIN");
            expect(box.getAttribute("role")).toBe("main");
        });
    });

    describe("Factory components", () => {
        it("renders BoxFactory.Paper with correct semantic element", () => {
            render(<BoxFactory.Paper data-testid="factory-paper">Paper content</BoxFactory.Paper>);
            const box = screen.getByTestId("factory-paper");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("Paper content");
        });

        it("renders BoxFactory.Outlined with correct semantic element", () => {
            render(<BoxFactory.Outlined data-testid="factory-outlined">Outlined content</BoxFactory.Outlined>);
            const box = screen.getByTestId("factory-outlined");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("Outlined content");
        });

        it("renders BoxFactory.Elevated with correct semantic element", () => {
            render(<BoxFactory.Elevated data-testid="factory-elevated">Elevated content</BoxFactory.Elevated>);
            const box = screen.getByTestId("factory-elevated");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("Elevated content");
        });

        it("renders BoxFactory.Subtle with correct semantic element", () => {
            render(<BoxFactory.Subtle data-testid="factory-subtle">Subtle content</BoxFactory.Subtle>);
            const box = screen.getByTestId("factory-subtle");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("Subtle content");
        });

        it("factory components accept additional props", () => {
            render(
                <BoxFactory.Paper 
                    padding="lg" 
                    borderRadius="md" 
                    data-testid="factory-with-props"
                >
                    Paper with props
                </BoxFactory.Paper>,
            );
            const box = screen.getByTestId("factory-with-props");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("Paper with props");
        });
    });

    describe("Combined props", () => {
        it("accepts multiple styling props without error", () => {
            render(
                <Box
                    variant="paper"
                    padding="lg"
                    borderRadius="md"
                    fullWidth
                    className="custom-class"
                    data-testid="combined-box"
                >
                    Combined styling
                </Box>,
            );
            const box = screen.getByTestId("combined-box");
            
            // Verify functional behavior
            expect(box.tagName).toBe("DIV");
            expect(box.textContent).toBe("Combined styling");
            expect(box).toBeDefined();
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
            render(<Box data-testid="empty-string">{""}</Box>);
            const box = screen.getByTestId("empty-string");
            expect(box).toBeDefined();
        });

        it("handles zero as children", () => {
            render(<Box data-testid="zero-children">{0}</Box>);
            const box = screen.getByTestId("zero-children");
            expect(box).toBeDefined();
            expect(box.textContent).toBe("0");
        });

        it("handles invalid component gracefully", () => {
            render(<Box component={undefined} data-testid="invalid-component">Content</Box>);
            const box = screen.getByTestId("invalid-component");
            expect(box.tagName).toBe("DIV");
        });
    });

    describe("Accessibility", () => {
        it("supports aria attributes", () => {
            render(
                <Box 
                    aria-describedby="description"
                    aria-label="Custom label"
                    data-testid="aria-box"
                >
                    Content
                </Box>,
            );
            const box = screen.getByTestId("aria-box");
            expect(box.getAttribute("aria-describedby")).toBe("description");
            expect(box.getAttribute("aria-label")).toBe("Custom label");
        });

        it("maintains semantic meaning with role attribute", () => {
            render(
                <Box 
                    component="div" 
                    role="region" 
                    aria-labelledby="region-title"
                    data-testid="semantic-box"
                >
                    Box content
                </Box>,
            );
            const box = screen.getByTestId("semantic-box");
            expect(box.tagName).toBe("DIV");
            expect(box.getAttribute("role")).toBe("region");
            expect(box.getAttribute("aria-labelledby")).toBe("region-title");
        });
    });
});
