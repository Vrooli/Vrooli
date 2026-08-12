import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import ErrorBoundary from "../components/ErrorBoundary";
import { strings } from "../consts/strings";
// [REQ:P0-002d] Error Boundary — isolates runtime crashes to UI regions
// Component that throws on demand
function ThrowingChild({ shouldThrow }) {
    if (shouldThrow)
        throw new Error("test crash");
    return _jsx("div", { "data-testid": "child", children: "OK" });
}
describe("ErrorBoundary", () => {
    beforeEach(() => {
        // Suppress React error boundary console.error noise in test output
        vi.spyOn(console, "error").mockImplementation(() => { });
    });
    it("renders children when no error occurs", () => {
        render(_jsx(ErrorBoundary, { region: "workspace", children: _jsx(ThrowingChild, { shouldThrow: false }) }));
        expect(screen.getByTestId("child")).toBeTruthy();
    });
    it("renders default fallback panel when child throws", () => {
        render(_jsx(ErrorBoundary, { region: "workspace", children: _jsx(ThrowingChild, { shouldThrow: true }) }));
        // Should show the error boundary panel with region name
        expect(screen.getByTestId("error-boundary-workspace")).toBeTruthy();
        expect(screen.getByText(strings.errorBoundary.somethingWentWrong)).toBeTruthy();
        expect(screen.getByText("test crash")).toBeTruthy();
        expect(screen.getByText(strings.errorBoundary.tryAgain)).toBeTruthy();
    });
    it("renders custom fallback when provided", () => {
        render(_jsx(ErrorBoundary, { region: "terminal", fallback: _jsx("div", { "data-testid": "custom-fallback", children: "Custom" }), children: _jsx(ThrowingChild, { shouldThrow: true }) }));
        expect(screen.getByTestId("custom-fallback")).toBeTruthy();
        // Default panel should NOT be rendered
        expect(screen.queryByTestId("error-boundary-terminal")).toBeNull();
    });
    it("resets error state when Try Again is clicked", () => {
        let shouldThrow = true;
        function ConditionalThrower() {
            if (shouldThrow)
                throw new Error("crash");
            return _jsx("div", { "data-testid": "recovered", children: "Recovered" });
        }
        render(_jsx(ErrorBoundary, { region: "pane", children: _jsx(ConditionalThrower, {}) }));
        // Error panel should be shown
        expect(screen.getByTestId("error-boundary-pane")).toBeTruthy();
        // Stop throwing before clicking reset
        shouldThrow = false;
        fireEvent.click(screen.getByText(strings.errorBoundary.tryAgain));
        // Children should render again
        expect(screen.getByTestId("recovered")).toBeTruthy();
        expect(screen.queryByTestId("error-boundary-pane")).toBeNull();
    });
});
