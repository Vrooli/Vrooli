import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { FormErrorBoundary } from "./FormErrorBoundary";

// Mock the Icon component
vi.mock("../../icons/Icons.js", () => ({
    Icon: ({ "data-testid": testId, className, size }: any) => (
        <svg data-testid={testId} className={className} width={size} height={size}>
            <text>Icon</text>
        </svg>
    ),
}));

// Component that throws an error when shouldThrow is true
const ThrowingComponent = ({ shouldThrow }: { shouldThrow: boolean }) => {
    if (shouldThrow) {
        throw new Error("Test error message");
    }
    return <div data-testid="normal-content">Normal content</div>;
};

// Component that never throws
const NormalComponent = () => <div data-testid="normal-content">Normal content</div>;

// Component with custom error
const CustomErrorComponent = ({ errorMessage }: { errorMessage: string }) => {
    throw new Error(errorMessage);
};

describe("FormErrorBoundary", () => {
    // Mock console.error to prevent cluttering test output
    const originalError = console.error;
    beforeEach(() => {
        console.error = vi.fn();
    });
    afterEach(() => {
        console.error = originalError;
    });

    describe("Normal operation", () => {
        it("renders children normally when no error occurs", () => {
            render(
                <FormErrorBoundary>
                    <NormalComponent />
                </FormErrorBoundary>,
            );

            const boundary = screen.getByTestId("form-error-boundary");
            expect(boundary).toBeDefined();
            expect(boundary.getAttribute("data-has-error")).toBe("false");
            
            const content = screen.getByTestId("normal-content");
            expect(content).toBeDefined();
            expect(content.textContent).toBe("Normal content");
        });

        it("wraps multiple children correctly", () => {
            render(
                <FormErrorBoundary>
                    <div data-testid="child-1">Child 1</div>
                    <div data-testid="child-2">Child 2</div>
                    <div data-testid="child-3">Child 3</div>
                </FormErrorBoundary>,
            );

            expect(screen.getByTestId("child-1")).toBeDefined();
            expect(screen.getByTestId("child-2")).toBeDefined();
            expect(screen.getByTestId("child-3")).toBeDefined();
            expect(screen.getByTestId("form-error-boundary").getAttribute("data-has-error")).toBe("false");
        });

        it("does not show error UI when no error", () => {
            render(
                <FormErrorBoundary>
                    <NormalComponent />
                </FormErrorBoundary>,
            );

            expect(screen.queryByRole("alert")).toBeNull();
            expect(screen.queryByTestId("error-message")).toBeNull();
            expect(screen.queryByTestId("error-details")).toBeNull();
        });
    });

    describe("Error catching", () => {
        it("catches errors and displays error UI", () => {
            render(
                <FormErrorBoundary>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            const boundary = screen.getByTestId("form-error-boundary");
            expect(boundary.getAttribute("data-has-error")).toBe("true");
            expect(boundary.getAttribute("role")).toBe("alert");
        });

        it("displays error header with error icon", () => {
            render(
                <FormErrorBoundary>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            const header = screen.getByTestId("error-header");
            expect(header).toBeDefined();

            const icon = screen.getByTestId("error-icon");
            expect(icon).toBeDefined();
            // Icon component renders SVG, not text content
            expect(icon.tagName.toLowerCase()).toBe("svg");

            const heading = screen.getByRole("heading", { level: 2 });
            expect(heading.textContent).toBe("An error occurred");
        });

        it("displays user-friendly error message", () => {
            render(
                <FormErrorBoundary>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            const message = screen.getByTestId("error-message");
            expect(message.textContent).toBe(
                "There was a problem rendering the form. Please try refreshing the page or contact support if the issue persists.",
            );
        });

        it("displays technical error details", () => {
            render(
                <FormErrorBoundary>
                    <CustomErrorComponent errorMessage="Custom test error" />
                </FormErrorBoundary>,
            );

            const details = screen.getByTestId("error-details");
            expect(details).toBeDefined();
            expect(details.textContent).toContain("Custom test error");
        });

        it("does not render original children when error occurs", () => {
            render(
                <FormErrorBoundary>
                    <div data-testid="should-not-render">This should not be visible</div>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            expect(screen.queryByTestId("should-not-render")).toBeNull();
            expect(screen.queryByTestId("normal-content")).toBeNull();
        });
    });

    describe("Error callback functionality", () => {
        it("calls onError callback when error occurs", () => {
            const onError = vi.fn();
            
            render(
                <FormErrorBoundary onError={onError}>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            expect(onError).toHaveBeenCalledTimes(1);
        });

        it("does not call onError when no error occurs", () => {
            const onError = vi.fn();
            
            render(
                <FormErrorBoundary onError={onError}>
                    <NormalComponent />
                </FormErrorBoundary>,
            );

            expect(onError).not.toHaveBeenCalled();
        });

        it("works without onError callback", () => {
            expect(() => {
                render(
                    <FormErrorBoundary>
                        <ThrowingComponent shouldThrow={true} />
                    </FormErrorBoundary>,
                );
            }).not.toThrow();

            expect(screen.getByTestId("form-error-boundary").getAttribute("data-has-error")).toBe("true");
        });
    });

    describe("Console logging", () => {
        it("logs errors to console when caught", () => {
            const consoleSpy = vi.spyOn(console, "error");
            
            render(
                <FormErrorBoundary>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            expect(consoleSpy).toHaveBeenCalled();
            expect(consoleSpy).toHaveBeenCalledWith(
                "Error caught by Error Boundary: ",
                expect.any(Error),
                expect.objectContaining({
                    componentStack: expect.any(String),
                }),
            );
        });
    });

    describe("Accessibility", () => {
        it("provides proper ARIA role for error state", () => {
            render(
                <FormErrorBoundary>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            const alert = screen.getByRole("alert");
            expect(alert).toBeDefined();
            expect(alert).toBe(screen.getByTestId("form-error-boundary"));
        });

        it("provides proper heading structure", () => {
            render(
                <FormErrorBoundary>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            const heading = screen.getByRole("heading", { level: 2 });
            expect(heading.textContent).toBe("An error occurred");
        });

        it("uses details element for expandable error information", () => {
            render(
                <FormErrorBoundary>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            const details = screen.getByTestId("error-details");
            expect(details.tagName.toLowerCase()).toBe("details");
        });
    });

    describe("Error recovery", () => {
        it("maintains error state after error occurs", () => {
            const { rerender } = render(
                <FormErrorBoundary>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            // Error boundary should show error state
            expect(screen.getByTestId("form-error-boundary").getAttribute("data-has-error")).toBe("true");

            // Re-rendering with same props should maintain error state
            rerender(
                <FormErrorBoundary>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            expect(screen.getByTestId("form-error-boundary").getAttribute("data-has-error")).toBe("true");
            expect(screen.getByTestId("error-message")).toBeDefined();
        });

        it("resets when component is unmounted and remounted", () => {
            const { unmount } = render(
                <FormErrorBoundary>
                    <ThrowingComponent shouldThrow={true} />
                </FormErrorBoundary>,
            );

            expect(screen.getByTestId("form-error-boundary").getAttribute("data-has-error")).toBe("true");
            
            unmount();

            render(
                <FormErrorBoundary>
                    <NormalComponent />
                </FormErrorBoundary>,
            );

            expect(screen.getByTestId("form-error-boundary").getAttribute("data-has-error")).toBe("false");
            expect(screen.getByTestId("normal-content")).toBeDefined();
        });
    });

    describe("Edge cases", () => {
        it("handles errors with no message", () => {
            const NoMessageError = () => {
                throw new Error();
            };

            render(
                <FormErrorBoundary>
                    <NoMessageError />
                </FormErrorBoundary>,
            );

            expect(screen.getByTestId("form-error-boundary").getAttribute("data-has-error")).toBe("true");
            expect(screen.getByTestId("error-message")).toBeDefined();
        });

        it("handles null/undefined children", () => {
            render(
                <FormErrorBoundary>
                    {null}
                    {undefined}
                </FormErrorBoundary>,
            );

            expect(screen.getByTestId("form-error-boundary").getAttribute("data-has-error")).toBe("false");
        });

        it("handles complex nested component structures", () => {
            render(
                <FormErrorBoundary>
                    <div>
                        <div>
                            <NormalComponent />
                            <div>
                                <ThrowingComponent shouldThrow={true} />
                            </div>
                        </div>
                    </div>
                </FormErrorBoundary>,
            );

            expect(screen.getByTestId("form-error-boundary").getAttribute("data-has-error")).toBe("true");
            expect(screen.queryByTestId("normal-content")).toBeNull();
        });
    });
});
