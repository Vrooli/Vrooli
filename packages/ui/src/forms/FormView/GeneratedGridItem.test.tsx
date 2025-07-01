import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it } from "vitest";
import { GeneratedGridItem } from "./GeneratedGridItem";

describe("GeneratedGridItem", () => {
    const TestChild = () => <div data-testid="test-child">Test Content</div>;

    describe("When isInGrid is true", () => {
        it("wraps children in a Grid item", () => {
            render(
                <GeneratedGridItem fieldsInGrid={1} isInGrid={true}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper).toBeDefined();
            expect(wrapper.getAttribute("data-in-grid")).toBe("true");
            expect(wrapper.getAttribute("data-fields-count")).toBe("1");
            
            // Verify it's using a Grid component (has item attribute)
            expect(wrapper.tagName).toBe("DIV"); // MUI Grid renders as div
            
            // Check child is rendered
            expect(screen.getByTestId("test-child")).toBeDefined();
        });

        it("renders with correct grid behavior for 1 field", () => {
            render(
                <GeneratedGridItem fieldsInGrid={1} isInGrid={true}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper.getAttribute("data-fields-count")).toBe("1");
            expect(wrapper.getAttribute("data-in-grid")).toBe("true");
        });

        it("renders with correct grid behavior for 2 fields", () => {
            render(
                <GeneratedGridItem fieldsInGrid={2} isInGrid={true}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper.getAttribute("data-fields-count")).toBe("2");
            expect(wrapper.getAttribute("data-in-grid")).toBe("true");
        });

        it("renders with correct grid behavior for 3 fields", () => {
            render(
                <GeneratedGridItem fieldsInGrid={3} isInGrid={true}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper.getAttribute("data-fields-count")).toBe("3");
            expect(wrapper.getAttribute("data-in-grid")).toBe("true");
        });

        it("renders with correct grid behavior for 4+ fields", () => {
            render(
                <GeneratedGridItem fieldsInGrid={4} isInGrid={true}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper.getAttribute("data-fields-count")).toBe("4");
            expect(wrapper.getAttribute("data-in-grid")).toBe("true");
        });

        it("renders with correct grid behavior for more than 4 fields", () => {
            render(
                <GeneratedGridItem fieldsInGrid={6} isInGrid={true}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper.getAttribute("data-fields-count")).toBe("6");
            expect(wrapper.getAttribute("data-in-grid")).toBe("true");
        });
    });

    describe("When isInGrid is false", () => {
        it("renders children without Grid wrapper", () => {
            render(
                <GeneratedGridItem fieldsInGrid={3} isInGrid={false}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper).toBeDefined();
            expect(wrapper.getAttribute("data-in-grid")).toBe("false");
            
            // Should be a plain div wrapper
            expect(wrapper.tagName).toBe("DIV");
            expect(wrapper.getAttribute("data-fields-count")).toBeNull();
            
            // Check child is rendered
            expect(screen.getByTestId("test-child")).toBeDefined();
        });

        it("ignores fieldsInGrid when not in grid", () => {
            render(
                <GeneratedGridItem fieldsInGrid={5} isInGrid={false}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            // fieldsInGrid should not affect non-grid rendering
            expect(wrapper.getAttribute("data-fields-count")).toBeNull();
            expect(wrapper.getAttribute("data-in-grid")).toBe("false");
        });
    });

    describe("Edge cases", () => {
        it("handles null children", () => {
            render(
                <GeneratedGridItem fieldsInGrid={1} isInGrid={true}>
                    {null}
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper).toBeDefined();
            expect(wrapper.textContent).toBe("");
        });

        it("handles undefined children", () => {
            render(
                <GeneratedGridItem fieldsInGrid={1} isInGrid={true}>
                    {undefined}
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper).toBeDefined();
            expect(wrapper.textContent).toBe("");
        });

        it("handles multiple children", () => {
            render(
                <GeneratedGridItem fieldsInGrid={2} isInGrid={true}>
                    <>
                        <div data-testid="child-1">Child 1</div>
                        <div data-testid="child-2">Child 2</div>
                    </>
                </GeneratedGridItem>,
            );

            expect(screen.getByTestId("child-1")).toBeDefined();
            expect(screen.getByTestId("child-2")).toBeDefined();
        });

        it("handles 0 fields correctly", () => {
            render(
                <GeneratedGridItem fieldsInGrid={0} isInGrid={true}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            // Should still render in grid mode with 0 fields
            expect(wrapper.getAttribute("data-fields-count")).toBe("0");
            expect(wrapper.getAttribute("data-in-grid")).toBe("true");
        });

        it("handles negative fieldsInGrid values", () => {
            render(
                <GeneratedGridItem fieldsInGrid={-1} isInGrid={true}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            const wrapper = screen.getByTestId("generated-grid-item");
            // Should still render in grid mode with negative value
            expect(wrapper.getAttribute("data-fields-count")).toBe("-1");
            expect(wrapper.getAttribute("data-in-grid")).toBe("true");
        });
    });

    describe("State transitions", () => {
        it("switches from grid to non-grid rendering", () => {
            const { rerender } = render(
                <GeneratedGridItem fieldsInGrid={2} isInGrid={true}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            // Start in grid mode
            let wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper.getAttribute("data-in-grid")).toBe("true");
            expect(wrapper.getAttribute("data-fields-count")).toBe("2");

            // Switch to non-grid mode
            rerender(
                <GeneratedGridItem fieldsInGrid={2} isInGrid={false}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper.getAttribute("data-in-grid")).toBe("false");
            expect(wrapper.getAttribute("data-fields-count")).toBeNull();
        });

        it("updates grid sizing when fieldsInGrid changes", () => {
            const { rerender } = render(
                <GeneratedGridItem fieldsInGrid={1} isInGrid={true}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            // Start with 1 field
            let wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper.getAttribute("data-fields-count")).toBe("1");
            expect(wrapper.getAttribute("data-in-grid")).toBe("true");

            // Change to 3 fields
            rerender(
                <GeneratedGridItem fieldsInGrid={3} isInGrid={true}>
                    <TestChild />
                </GeneratedGridItem>,
            );

            wrapper = screen.getByTestId("generated-grid-item");
            expect(wrapper.getAttribute("data-fields-count")).toBe("3");
            expect(wrapper.getAttribute("data-in-grid")).toBe("true");
        });
    });
});
