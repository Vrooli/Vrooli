import { render, screen, within } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { ScrollBox } from "../../styles.js";

describe("ScrollBox", () => {
    describe("Basic rendering", () => {
        it("renders as a scrollable container", () => {
            render(<ScrollBox data-testid="scroll-box" />);

            const scrollBox = screen.getByTestId("scroll-box");
            expect(scrollBox).toBeDefined();
            expect(scrollBox.tagName).toBe("DIV");
        });

        it("applies scroll-box CSS class", () => {
            render(<ScrollBox data-testid="scroll-box" />);

            const scrollBox = screen.getByTestId("scroll-box");
            expect(scrollBox.className).toContain("scroll-box");
        });

        it("has proper MUI Box structure", () => {
            render(<ScrollBox data-testid="scroll-box" />);

            const scrollBox = screen.getByTestId("scroll-box");
            // MUI Box components have MuiBox-root class
            expect(scrollBox.className).toContain("MuiBox-root");
        });
    });

    describe("Content rendering", () => {
        it("renders children content", () => {
            render(
                <ScrollBox data-testid="scroll-box">
                    <div data-testid="child-content">Test content</div>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            const childContent = screen.getByTestId("child-content");

            expect(scrollBox).toBeDefined();
            expect(childContent).toBeDefined();
            
            // Check that child is within the scroll box
            expect(within(scrollBox).getByTestId("child-content")).toBeDefined();
        });

        it("renders multiple children", () => {
            render(
                <ScrollBox data-testid="scroll-box">
                    <div data-testid="child-1">First child</div>
                    <div data-testid="child-2">Second child</div>
                    <div data-testid="child-3">Third child</div>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");

            // Check that all children are within the scroll box
            expect(within(scrollBox).getByTestId("child-1")).toBeDefined();
            expect(within(scrollBox).getByTestId("child-2")).toBeDefined();
            expect(within(scrollBox).getByTestId("child-3")).toBeDefined();
        });

        it("renders text content directly", () => {
            render(
                <ScrollBox data-testid="scroll-box">
                    Plain text content
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            expect(scrollBox.textContent).toBe("Plain text content");
        });

        it("renders empty when no children provided", () => {
            render(<ScrollBox data-testid="scroll-box" />);

            const scrollBox = screen.getByTestId("scroll-box");
            expect(scrollBox.textContent).toBe("");
            expect(scrollBox.children.length).toBe(0);
        });
    });

    describe("Props forwarding", () => {
        it("forwards standard Box props", () => {
            render(
                <ScrollBox
                    data-testid="scroll-box"
                    id="custom-id"
                    role="region"
                    aria-label="Scrollable content"
                />,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            expect(scrollBox.id).toBe("custom-id");
            expect(scrollBox.getAttribute("role")).toBe("region");
            expect(scrollBox.getAttribute("aria-label")).toBe("Scrollable content");
        });

        it("accepts sx styles prop", () => {
            render(
                <ScrollBox
                    data-testid="scroll-box"
                    sx={{ backgroundColor: "red", border: "1px solid blue" }}
                />,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            // Should render without errors and maintain basic structure
            expect(scrollBox).toBeDefined();
            expect(scrollBox.className).toContain("MuiBox-root");
        });

        it("accepts custom className while preserving base classes", () => {
            render(
                <ScrollBox
                    data-testid="scroll-box"
                    className="custom-class another-class"
                />,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            expect(scrollBox.className).toContain("scroll-box");
            expect(scrollBox.className).toContain("custom-class");
            expect(scrollBox.className).toContain("another-class");
        });
    });

    describe("Scrolling behavior", () => {
        it("provides scroll container structure", () => {
            render(
                <ScrollBox
                    data-testid="scroll-box"
                    style={{ height: "100px" }}
                >
                    <div style={{ height: "200px" }} data-testid="tall-content">
                        Tall content that should cause scrolling
                    </div>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            
            // Should render as a container that can potentially scroll
            expect(scrollBox).toBeDefined();
            expect(scrollBox.tagName).toBe("DIV");
            expect(within(scrollBox).getByTestId("tall-content")).toBeDefined();
        });

        it("renders content within container", () => {
            render(
                <ScrollBox
                    data-testid="scroll-box"
                    style={{ height: "200px" }}
                >
                    <div style={{ height: "100px" }} data-testid="short-content">
                        Short content
                    </div>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            
            // Content should be present within container
            expect(within(scrollBox).getByTestId("short-content")).toBeDefined();
            expect(scrollBox.scrollTop).toBe(0);
        });

        it("supports scroll API methods", () => {
            render(
                <ScrollBox
                    data-testid="scroll-box"
                    style={{ height: "100px" }}
                >
                    <div style={{ height: "300px" }} data-testid="scrollable-content">
                        <div style={{ height: "100px" }}>Section 1</div>
                        <div style={{ height: "100px" }}>Section 2</div>
                        <div style={{ height: "100px" }}>Section 3</div>
                    </div>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            
            // Should have scroll API available
            expect(typeof scrollBox.scrollTo).toBe("function");
            expect(typeof scrollBox.scrollTop).toBe("number");
            expect(typeof scrollBox.scrollHeight).toBe("number");
            expect(typeof scrollBox.clientHeight).toBe("number");
            
            // Initial position
            expect(scrollBox.scrollTop).toBe(0);
        });
    });

    describe("Event handling", () => {
        it("handles scroll events", async () => {
            const onScroll = vi.fn();
            const user = userEvent.setup();

            render(
                <ScrollBox
                    data-testid="scroll-box"
                    onScroll={onScroll}
                    style={{ height: "100px" }}
                >
                    <div style={{ height: "300px" }} data-testid="scrollable-content">
                        Long scrollable content
                    </div>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            
            // Trigger scroll event
            scrollBox.scrollTo({ top: 50 });
            scrollBox.dispatchEvent(new Event("scroll"));

            expect(onScroll).toHaveBeenCalled();
        });

        it("handles mouse events when provided", async () => {
            const onClick = vi.fn();
            const onMouseEnter = vi.fn();
            const onMouseLeave = vi.fn();
            const user = userEvent.setup();

            render(
                <ScrollBox
                    data-testid="scroll-box"
                    onClick={onClick}
                    onMouseEnter={onMouseEnter}
                    onMouseLeave={onMouseLeave}
                >
                    <div>Interactive content</div>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");

            await user.click(scrollBox);
            expect(onClick).toHaveBeenCalledTimes(1);

            await user.hover(scrollBox);
            expect(onMouseEnter).toHaveBeenCalledTimes(1);

            await user.unhover(scrollBox);
            expect(onMouseLeave).toHaveBeenCalledTimes(1);
        });

        it("handles keyboard events when focused and provided", async () => {
            const onKeyDown = vi.fn();
            const user = userEvent.setup();

            render(
                <ScrollBox
                    data-testid="scroll-box"
                    onKeyDown={onKeyDown}
                    tabIndex={0}
                >
                    <div>Focusable scrollable content</div>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            
            // Focus the scroll box first
            scrollBox.focus();
            expect(document.activeElement).toBe(scrollBox);

            // Press a key
            await user.keyboard("{ArrowDown}");
            expect(onKeyDown).toHaveBeenCalled();
        });
    });

    describe("Accessibility", () => {
        it("supports ARIA attributes for accessibility", () => {
            render(
                <ScrollBox
                    data-testid="scroll-box"
                    role="region"
                    aria-label="Scrollable content area"
                    aria-describedby="scroll-description"
                />,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            expect(scrollBox.getAttribute("role")).toBe("region");
            expect(scrollBox.getAttribute("aria-label")).toBe("Scrollable content area");
            expect(scrollBox.getAttribute("aria-describedby")).toBe("scroll-description");
        });

        it("supports keyboard navigation when focusable", () => {
            render(
                <ScrollBox
                    data-testid="scroll-box"
                    tabIndex={0}
                    style={{ height: "100px" }}
                >
                    <div style={{ height: "300px" }}>
                        Focusable scrollable content
                    </div>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            
            // Should be focusable
            scrollBox.focus();
            expect(document.activeElement).toBe(scrollBox);
            
            // Should support keyboard events
            const initialScrollTop = scrollBox.scrollTop;
            scrollBox.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown" }));
            
            // Note: Actual keyboard scrolling behavior depends on browser implementation
            // We're just testing that the element can receive focus and keyboard events
        });

        it("maintains semantic structure with proper container role", () => {
            render(
                <ScrollBox data-testid="scroll-box">
                    <h2>Section Title</h2>
                    <p>Section content</p>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            const heading = screen.getByRole("heading", { level: 2 });
            
            expect(within(scrollBox).getByRole("heading", { level: 2 })).toBeDefined();
            expect(heading.textContent).toBe("Section Title");
        });
    });

    describe("Performance and refs", () => {
        it("forwards ref to underlying element", () => {
            const ref = React.createRef<HTMLDivElement>();

            render(
                <ScrollBox ref={ref} data-testid="scroll-box">
                    Test content
                </ScrollBox>,
            );

            expect(ref.current).toBeDefined();
            expect(ref.current?.tagName).toBe("DIV");
            expect(ref.current?.getAttribute("data-testid")).toBe("scroll-box");
        });

        it("allows ref-based scroll control", () => {
            const ref = React.createRef<HTMLDivElement>();

            render(
                <ScrollBox ref={ref} style={{ height: "100px" }}>
                    <div style={{ height: "300px" }}>Scrollable content</div>
                </ScrollBox>,
            );

            expect(ref.current).toBeDefined();
            
            if (ref.current) {
                // Use ref to control scrolling
                ref.current.scrollTo({ top: 50 });
                expect(ref.current.scrollTop).toBe(50);
            }
        });
    });

    describe("Dynamic content handling", () => {
        it("adapts to content changes", () => {
            const { rerender } = render(
                <ScrollBox data-testid="scroll-box" style={{ height: "100px" }}>
                    <div style={{ height: "50px" }}>Short content</div>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            
            // Initially should have one content div
            expect(scrollBox.children.length).toBe(1);

            // Add more content
            rerender(
                <ScrollBox data-testid="scroll-box" style={{ height: "100px" }}>
                    <div style={{ height: "50px" }}>Short content</div>
                    <div style={{ height: "100px" }}>Additional content</div>
                </ScrollBox>,
            );

            // Now should have two content divs
            expect(scrollBox.children.length).toBe(2);
        });

        it("handles empty to content transitions", () => {
            const { rerender } = render(
                <ScrollBox data-testid="scroll-box" />,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            expect(scrollBox.children.length).toBe(0);

            // Add content
            rerender(
                <ScrollBox data-testid="scroll-box">
                    <div data-testid="new-content">New content added</div>
                </ScrollBox>,
            );

            expect(scrollBox.children.length).toBe(1);
            expect(screen.getByTestId("new-content")).toBeDefined();
        });

        it("handles content to empty transitions", () => {
            const { rerender } = render(
                <ScrollBox data-testid="scroll-box">
                    <div data-testid="content-to-remove">Content to be removed</div>
                </ScrollBox>,
            );

            const scrollBox = screen.getByTestId("scroll-box");
            expect(scrollBox.children.length).toBe(1);
            expect(screen.getByTestId("content-to-remove")).toBeDefined();

            // Remove content
            rerender(<ScrollBox data-testid="scroll-box" />);

            expect(scrollBox.children.length).toBe(0);
            expect(screen.queryByTestId("content-to-remove")).toBeNull();
        });
    });
});
