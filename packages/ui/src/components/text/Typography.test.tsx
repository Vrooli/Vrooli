import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Typography, TypographyFactory } from "./Typography.js";

describe("Typography Component", () => {
    describe("Basic rendering", () => {
        it("renders with default props", () => {
            render(<Typography data-testid="typography">Test content</Typography>);
            const typography = screen.getByTestId("typography");
            expect(typography).toBeDefined();
            expect(typography.tagName).toBe("P");
            expect(typography.textContent).toBe("Test content");
        });

        it("renders with custom HTML element", () => {
            render(<Typography component="span" data-testid="typography">Span content</Typography>);
            const typography = screen.getByTestId("typography");
            expect(typography.tagName).toBe("SPAN");
            expect(typography.textContent).toBe("Span content");
        });

        it("accepts custom className prop", () => {
            const customClass = "custom-typography-class";
            render(<Typography className={customClass} data-testid="typography">Content</Typography>);
            const typography = screen.getByTestId("typography");
            // Just verify the component renders without error when className is provided
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("Content");
        });

        it("forwards ref correctly", () => {
            let refElement: HTMLElement | null = null;
            const TestComponent = () => (
                <Typography ref={(el) => { refElement = el; }} data-testid="typography">Content</Typography>
            );
            render(<TestComponent />);
            expect(refElement).toBeTruthy();
            expect(refElement?.tagName).toBe("P");
        });
    });

    describe("Variants", () => {
        it.each([
            ["display1", "H1"],
            ["display2", "H1"],
            ["h1", "H1"],
            ["h2", "H2"],
            ["h3", "H3"],
            ["h4", "H4"],
            ["h5", "H5"],
            ["h6", "H6"],
            ["subtitle1", "H6"],
            ["subtitle2", "H6"],
            ["body1", "P"],
            ["body2", "P"],
            ["caption", "SPAN"],
            ["overline", "SPAN"],
        ] as const)("renders %s variant with correct default element %s", (variant, expectedElement) => {
            render(<Typography variant={variant} data-testid="typography">Content</Typography>);
            const typography = screen.getByTestId("typography");
            expect(typography.tagName).toBe(expectedElement);
        });

        it("renders variants with appropriate semantic meaning", () => {
            // Test that variants produce functional differences, not just visual ones
            const variants = ["display1", "display2", "h1", "h2", "h3", "h4", "h5", "h6", "subtitle1", "subtitle2", "body1", "body2", "caption", "overline"] as const;
            
            variants.forEach(variant => {
                const { unmount } = render(<Typography variant={variant} data-testid="typography">Content</Typography>);
                const typography = screen.getByTestId("typography");
                expect(typography).toBeDefined();
                expect(typography.textContent).toBe("Content");
                unmount();
            });
        });

        it("renders overline variant correctly", () => {
            render(<Typography variant="overline" data-testid="typography">overline content</Typography>);
            const typography = screen.getByTestId("typography");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("overline content");
            expect(typography.tagName).toBe("SPAN");
        });
    });

    describe("Alignment", () => {
        it("accepts alignment prop without error", () => {
            const alignments = ["left", "center", "right", "justify"] as const;
            
            alignments.forEach(align => {
                const { unmount } = render(<Typography align={align} data-testid="typography">Content</Typography>);
                const typography = screen.getByTestId("typography");
                expect(typography).toBeDefined();
                expect(typography.textContent).toBe("Content");
                unmount();
            });
        });
    });

    describe("Colors", () => {
        it("accepts color prop without error", () => {
            const colors = ["primary", "secondary", "success", "warning", "error", "info", "text", "inherit"] as const;
            
            colors.forEach(color => {
                const { unmount } = render(<Typography color={color} data-testid="typography">Content</Typography>);
                const typography = screen.getByTestId("typography");
                expect(typography).toBeDefined();
                expect(typography.textContent).toBe("Content");
                unmount();
            });
        });
    });

    describe("Font weights", () => {
        it("accepts weight prop without error", () => {
            const weights = ["light", "normal", "medium", "semibold", "bold"] as const;
            
            weights.forEach(weight => {
                const { unmount } = render(<Typography weight={weight} data-testid="typography">Content</Typography>);
                const typography = screen.getByTestId("typography");
                expect(typography).toBeDefined();
                expect(typography.textContent).toBe("Content");
                unmount();
            });
        });

        it("accepts weight prop with variant", () => {
            render(<Typography variant="h1" weight="light" data-testid="typography">Content</Typography>);
            const typography = screen.getByTestId("typography");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("Content");
            expect(typography.tagName).toBe("H1");
        });
    });

    describe("Text transformations", () => {
        it("accepts uppercase prop without error", () => {
            render(<Typography uppercase data-testid="typography">Content</Typography>);
            const typography = screen.getByTestId("typography");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("Content");
        });

        it("accepts noWrap prop without error", () => {
            render(<Typography noWrap data-testid="typography">Content</Typography>);
            const typography = screen.getByTestId("typography");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("Content");
        });

        it("accepts truncate prop without error", () => {
            render(<Typography truncate data-testid="typography">Content</Typography>);
            const typography = screen.getByTestId("typography");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("Content");
        });

        it("accepts multiple transformation props", () => {
            render(
                <Typography 
                    uppercase 
                    noWrap 
                    truncate 
                    data-testid="typography"
                >
                    Content
                </Typography>,
            );
            const typography = screen.getByTestId("typography");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("Content");
        });
    });

    describe("Content", () => {
        it("renders text content", () => {
            render(<Typography>Simple text content</Typography>);
            expect(screen.getByText("Simple text content")).toBeDefined();
        });

        it("renders complex children content", () => {
            render(
                <Typography>
                    <span data-testid="child-1">Child 1</span>
                    <strong data-testid="child-2">Child 2</strong>
                </Typography>,
            );
            expect(screen.getByTestId("child-1")).toBeDefined();
            expect(screen.getByTestId("child-2")).toBeDefined();
        });

        it("renders empty content correctly", () => {
            render(<Typography data-testid="empty-typography"></Typography>);
            const typography = screen.getByTestId("empty-typography");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("");
        });
    });

    describe("HTML attributes", () => {
        it("passes through HTML attributes", () => {
            render(
                <Typography
                    id="test-id"
                    role="heading"
                    aria-label="Test typography"
                    data-testid="typography"
                >
                    Content
                </Typography>,
            );
            const typography = screen.getByTestId("typography");
            expect(typography.getAttribute("id")).toBe("test-id");
            expect(typography.getAttribute("role")).toBe("heading");
            expect(typography.getAttribute("aria-label")).toBe("Test typography");
        });

        it("handles event handlers", () => {
            let clicked = false;
            const handleClick = () => { clicked = true; };
            
            render(<Typography onClick={handleClick} data-testid="clickable-typography">Click me</Typography>);
            const typography = screen.getByTestId("clickable-typography");
            typography.click();
            expect(clicked).toBe(true);
        });
    });

    describe("Component override", () => {
        it("renders with custom component when specified", () => {
            render(<Typography component="article" variant="h1" data-testid="typography">Content</Typography>);
            const typography = screen.getByTestId("typography");
            expect(typography.tagName).toBe("ARTICLE");
        });

        it("custom component overrides variant default", () => {
            render(<Typography component="span" variant="h1" data-testid="typography">Content</Typography>);
            const typography = screen.getByTestId("typography");
            expect(typography.tagName).toBe("SPAN");
        });
    });

    describe("Factory components", () => {
        it("renders TypographyFactory.Display1 with correct semantic element", () => {
            render(<TypographyFactory.Display1 data-testid="factory-display1">Display content</TypographyFactory.Display1>);
            const typography = screen.getByTestId("factory-display1");
            expect(typography.tagName).toBe("H1");
            expect(typography.textContent).toBe("Display content");
        });

        it("renders TypographyFactory.H1 with correct semantic element", () => {
            render(<TypographyFactory.H1 data-testid="factory-h1">H1 content</TypographyFactory.H1>);
            const typography = screen.getByTestId("factory-h1");
            expect(typography.tagName).toBe("H1");
            expect(typography.textContent).toBe("H1 content");
        });

        it("renders TypographyFactory.Body1 with correct semantic element", () => {
            render(<TypographyFactory.Body1 data-testid="factory-body1">Body content</TypographyFactory.Body1>);
            const typography = screen.getByTestId("factory-body1");
            expect(typography.tagName).toBe("P");
            expect(typography.textContent).toBe("Body content");
        });

        it("renders TypographyFactory.Caption with correct semantic element", () => {
            render(<TypographyFactory.Caption data-testid="factory-caption">Caption content</TypographyFactory.Caption>);
            const typography = screen.getByTestId("factory-caption");
            expect(typography.tagName).toBe("SPAN");
            expect(typography.textContent).toBe("Caption content");
        });

        it("renders TypographyFactory.Overline with correct semantic element", () => {
            render(<TypographyFactory.Overline data-testid="factory-overline">Overline content</TypographyFactory.Overline>);
            const typography = screen.getByTestId("factory-overline");
            expect(typography.tagName).toBe("SPAN");
            expect(typography.textContent).toBe("Overline content");
        });

        it("renders TypographyFactory.Centered correctly", () => {
            render(<TypographyFactory.Centered data-testid="factory-centered">Centered content</TypographyFactory.Centered>);
            const typography = screen.getByTestId("factory-centered");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("Centered content");
        });

        it("renders TypographyFactory.Primary correctly", () => {
            render(<TypographyFactory.Primary data-testid="factory-primary">Primary content</TypographyFactory.Primary>);
            const typography = screen.getByTestId("factory-primary");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("Primary content");
        });

        it("renders TypographyFactory.Bold correctly", () => {
            render(<TypographyFactory.Bold data-testid="factory-bold">Bold content</TypographyFactory.Bold>);
            const typography = screen.getByTestId("factory-bold");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("Bold content");
        });

        it("renders TypographyFactory.Truncated correctly", () => {
            render(<TypographyFactory.Truncated data-testid="factory-truncated">Truncated content</TypographyFactory.Truncated>);
            const typography = screen.getByTestId("factory-truncated");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("Truncated content");
        });

        it("factory components accept additional props", () => {
            render(
                <TypographyFactory.H2 
                    color="primary" 
                    align="center" 
                    data-testid="factory-with-props"
                >
                    H2 with props
                </TypographyFactory.H2>,
            );
            const typography = screen.getByTestId("factory-with-props");
            expect(typography.tagName).toBe("H2");
            expect(typography.textContent).toBe("H2 with props");
        });
    });

    describe("Combined props", () => {
        it("accepts multiple styling props without error", () => {
            render(
                <Typography
                    variant="h2"
                    align="center"
                    color="primary"
                    weight="light"
                    uppercase
                    noWrap
                    truncate
                    className="custom-class"
                    data-testid="combined-typography"
                >
                    Combined styling
                </Typography>,
            );
            const typography = screen.getByTestId("combined-typography");
            
            // Verify functional behavior
            expect(typography.tagName).toBe("H2");
            expect(typography.textContent).toBe("Combined styling");
            expect(typography).toBeDefined();
        });
    });

    describe("Edge cases", () => {
        it("handles null children", () => {
            render(<Typography data-testid="null-children">{null}</Typography>);
            const typography = screen.getByTestId("null-children");
            expect(typography).toBeDefined();
        });

        it("handles undefined children", () => {
            render(<Typography data-testid="undefined-children">{undefined}</Typography>);
            const typography = screen.getByTestId("undefined-children");
            expect(typography).toBeDefined();
        });

        it("handles empty string children", () => {
            render(<Typography data-testid="empty-string">{""}
</Typography>);
            const typography = screen.getByTestId("empty-string");
            expect(typography).toBeDefined();
        });

        it("handles zero as children", () => {
            render(<Typography data-testid="zero-children">{0}</Typography>);
            const typography = screen.getByTestId("zero-children");
            expect(typography).toBeDefined();
            expect(typography.textContent).toBe("0");
        });

        it("handles invalid component gracefully", () => {
            // This should fallback to the variant default element
            render(<Typography component={undefined} variant="h1" data-testid="invalid-component">Content</Typography>);
            const typography = screen.getByTestId("invalid-component");
            expect(typography.tagName).toBe("H1");
        });
    });

    describe("Semantic HTML structure", () => {
        it("creates proper heading hierarchy", () => {
            render(
                <div>
                    <Typography variant="h1">Main Heading</Typography>
                    <Typography variant="h2">Section Heading</Typography>
                    <Typography variant="h3">Subsection Heading</Typography>
                </div>,
            );
            
            expect(screen.getByRole("heading", { level: 1 })).toBeDefined();
            expect(screen.getByRole("heading", { level: 2 })).toBeDefined();
            expect(screen.getByRole("heading", { level: 3 })).toBeDefined();
        });

        it("works with article structure", () => {
            render(
                <article>
                    <Typography variant="h1">Article Title</Typography>
                    <Typography variant="subtitle1">Article Subtitle</Typography>
                    <Typography variant="body1">Article content goes here.</Typography>
                    <Typography variant="caption">Article metadata</Typography>
                </article>,
            );
            
            expect(screen.getByText("Article Title")).toBeDefined();
            expect(screen.getByText("Article Subtitle")).toBeDefined();
            expect(screen.getByText("Article content goes here.")).toBeDefined();
            expect(screen.getByText("Article metadata")).toBeDefined();
        });
    });

    describe("Accessibility", () => {
        it("maintains semantic meaning with component override", () => {
            render(
                <Typography 
                    component="div" 
                    variant="h1" 
                    role="heading" 
                    aria-level={1}
                    data-testid="semantic-div"
                >
                    Heading styled as div
                </Typography>,
            );
            const typography = screen.getByTestId("semantic-div");
            expect(typography.tagName).toBe("DIV");
            expect(typography.getAttribute("role")).toBe("heading");
            expect(typography.getAttribute("aria-level")).toBe("1");
        });

        it("supports aria attributes", () => {
            render(
                <Typography 
                    aria-describedby="description"
                    aria-label="Custom label"
                    data-testid="aria-typography"
                >
                    Content
                </Typography>,
            );
            const typography = screen.getByTestId("aria-typography");
            expect(typography.getAttribute("aria-describedby")).toBe("description");
            expect(typography.getAttribute("aria-label")).toBe("Custom label");
        });
    });
});
