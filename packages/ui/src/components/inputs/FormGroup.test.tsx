import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it } from "vitest";
import { FormGroup } from "./FormGroup";

describe("FormGroup", () => {
    describe("Basic rendering", () => {
        it("renders with default props", () => {
            render(
                <FormGroup data-testid="form-group">
                    <div>Child content</div>
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup).toBeDefined();
            expect(formGroup.getAttribute("role")).toBe("group");
            expect(formGroup.textContent).toContain("Child content");
        });

        it("renders children content correctly", () => {
            render(
                <FormGroup data-testid="form-group">
                    <input type="text" aria-label="Test input" />
                    <button type="button">Test button</button>
                    <div>Text content</div>
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup).toBeDefined();
            
            // Verify children are rendered
            expect(screen.getByLabelText("Test input")).toBeDefined();
            expect(screen.getByRole("button", { name: "Test button" })).toBeDefined();
            expect(screen.getByText("Text content")).toBeDefined();
        });

        it("applies semantic role correctly", () => {
            render(
                <FormGroup data-testid="form-group">
                    <div>Content</div>
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup.getAttribute("role")).toBe("group");
            expect(formGroup.tagName.toLowerCase()).toBe("div");
        });
    });

    describe("Row layout behavior", () => {
        it("renders in column layout by default", () => {
            render(
                <FormGroup data-testid="form-group">
                    <div>Item 1</div>
                    <div>Item 2</div>
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup.getAttribute("data-row")).toBe("false");
            
            // Check for column layout classes
            expect(formGroup.className).toContain("tw-flex-col");
            expect(formGroup.className).toContain("tw-gap-y-3");
            expect(formGroup.className).toContain("tw-items-start");
        });

        it("renders in row layout when row=true", () => {
            render(
                <FormGroup row={true} data-testid="form-group">
                    <div>Item 1</div>
                    <div>Item 2</div>
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup.getAttribute("data-row")).toBe("true");
            
            // Check for row layout classes
            expect(formGroup.className).toContain("tw-flex-row");
            expect(formGroup.className).toContain("tw-flex-wrap");
            expect(formGroup.className).toContain("tw-gap-x-6");
            expect(formGroup.className).toContain("tw-gap-y-2");
            expect(formGroup.className).toContain("tw-items-center");
        });

        it("switches between row and column layouts", () => {
            const { rerender } = render(
                <FormGroup row={false} data-testid="form-group">
                    <div>Item 1</div>
                    <div>Item 2</div>
                </FormGroup>
            );

            let formGroup = screen.getByTestId("form-group");
            expect(formGroup.getAttribute("data-row")).toBe("false");
            expect(formGroup.className).toContain("tw-flex-col");
            
            // Switch to row layout
            rerender(
                <FormGroup row={true} data-testid="form-group">
                    <div>Item 1</div>
                    <div>Item 2</div>
                </FormGroup>
            );

            formGroup = screen.getByTestId("form-group");
            expect(formGroup.getAttribute("data-row")).toBe("true");
            expect(formGroup.className).toContain("tw-flex-row");
            
            // Switch back to column layout
            rerender(
                <FormGroup row={false} data-testid="form-group">
                    <div>Item 1</div>
                    <div>Item 2</div>
                </FormGroup>
            );

            formGroup = screen.getByTestId("form-group");
            expect(formGroup.getAttribute("data-row")).toBe("false");
            expect(formGroup.className).toContain("tw-flex-col");
        });
    });

    describe("Styling and customization", () => {
        it("applies custom className", () => {
            render(
                <FormGroup className="custom-class" data-testid="form-group">
                    <div>Content</div>
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup.className).toContain("custom-class");
            // Should still have base classes
            expect(formGroup.className).toContain("tw-flex");
        });

        it("applies custom style prop", () => {
            const customStyle = {
                backgroundColor: "red",
                padding: "10px",
            };

            render(
                <FormGroup style={customStyle} data-testid="form-group">
                    <div>Content</div>
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup.style.backgroundColor).toBe("red");
            expect(formGroup.style.padding).toBe("10px");
        });

        it("combines custom className with generated classes", () => {
            render(
                <FormGroup 
                    row={true} 
                    className="my-custom-class another-class" 
                    data-testid="form-group"
                >
                    <div>Content</div>
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup.className).toContain("my-custom-class");
            expect(formGroup.className).toContain("another-class");
            expect(formGroup.className).toContain("tw-flex-row");
            expect(formGroup.className).toContain("tw-flex");
        });
    });

    describe("Ref forwarding", () => {
        it("forwards ref to the div element", () => {
            const ref = React.createRef<HTMLDivElement>();
            
            render(
                <FormGroup ref={ref} data-testid="form-group">
                    <div>Content</div>
                </FormGroup>
            );

            expect(ref.current).toBeDefined();
            expect(ref.current?.tagName.toLowerCase()).toBe("div");
            expect(ref.current?.getAttribute("data-testid")).toBe("form-group");
        });

        it("ref provides access to DOM methods", () => {
            const ref = React.createRef<HTMLDivElement>();
            
            render(
                <FormGroup ref={ref} data-testid="form-group">
                    <div>Content</div>
                </FormGroup>
            );

            expect(ref.current?.getAttribute).toBeDefined();
            expect(ref.current?.querySelector).toBeDefined();
            expect(ref.current?.getAttribute("role")).toBe("group");
        });
    });

    describe("Additional props handling", () => {
        it("passes through additional HTML attributes", () => {
            render(
                <FormGroup 
                    data-testid="form-group"
                    aria-label="Test form group"
                    id="my-form-group"
                    title="Form group title"
                >
                    <div>Content</div>
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup.getAttribute("aria-label")).toBe("Test form group");
            expect(formGroup.getAttribute("id")).toBe("my-form-group");
            expect(formGroup.getAttribute("title")).toBe("Form group title");
        });

        it("handles event handlers", () => {
            let clickCount = 0;
            let focusCount = 0;

            const handleClick = () => { clickCount++; };
            const handleFocus = () => { focusCount++; };

            render(
                <FormGroup 
                    data-testid="form-group"
                    onClick={handleClick}
                    onFocus={handleFocus}
                    tabIndex={0}
                >
                    <div>Content</div>
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            
            // Simulate click
            formGroup.click();
            expect(clickCount).toBe(1);

            // Simulate focus
            formGroup.focus();
            expect(focusCount).toBe(1);
        });
    });

    describe("Component display name", () => {
        it("has correct display name", () => {
            expect(FormGroup.displayName).toBe("FormGroup");
        });
    });

    describe("Empty and edge cases", () => {
        it("renders without children", () => {
            render(<FormGroup data-testid="form-group" />);

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup).toBeDefined();
            expect(formGroup.getAttribute("role")).toBe("group");
            expect(formGroup.textContent).toBe("");
        });

        it("renders with null children", () => {
            render(
                <FormGroup data-testid="form-group">
                    {null}
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup).toBeDefined();
            expect(formGroup.textContent).toBe("");
        });

        it("renders with mixed valid and invalid children", () => {
            render(
                <FormGroup data-testid="form-group">
                    <div>Valid child</div>
                    {null}
                    {undefined}
                    <span>Another valid child</span>
                    {false}
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup).toBeDefined();
            expect(formGroup.textContent).toContain("Valid child");
            expect(formGroup.textContent).toContain("Another valid child");
        });
    });

    describe("Accessibility", () => {
        it("maintains proper ARIA structure", () => {
            render(
                <FormGroup data-testid="form-group" aria-labelledby="group-label">
                    <div id="group-label">Group Label</div>
                    <input type="text" aria-label="Input field" />
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup.getAttribute("role")).toBe("group");
            expect(formGroup.getAttribute("aria-labelledby")).toBe("group-label");
        });

        it("works with form controls", () => {
            render(
                <FormGroup data-testid="form-group">
                    <label htmlFor="username">Username:</label>
                    <input type="text" id="username" name="username" />
                    <label htmlFor="password">Password:</label>
                    <input type="password" id="password" name="password" />
                </FormGroup>
            );

            const formGroup = screen.getByTestId("form-group");
            expect(formGroup.getAttribute("role")).toBe("group");
            
            // Verify form controls are properly associated
            expect(screen.getByLabelText("Username:")).toBeDefined();
            expect(screen.getByLabelText("Password:")).toBeDefined();
        });
    });
});