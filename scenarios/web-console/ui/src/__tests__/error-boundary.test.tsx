import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import ErrorBoundary from "../components/ErrorBoundary";
import { strings } from "../consts/strings";

// [REQ:P0-002d] Error Boundary — isolates runtime crashes to UI regions

// Component that throws on demand
function ThrowingChild({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) throw new Error("test crash");
  return <div data-testid="child">OK</div>;
}

describe("ErrorBoundary", () => {
  beforeEach(() => {
    // Suppress React error boundary console.error noise in test output
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  it("renders children when no error occurs", () => {
    render(
      <ErrorBoundary region="workspace">
        <ThrowingChild shouldThrow={false} />
      </ErrorBoundary>,
    );

    expect(screen.getByTestId("child")).toBeTruthy();
  });

  it("renders default fallback panel when child throws", () => {
    render(
      <ErrorBoundary region="workspace">
        <ThrowingChild shouldThrow={true} />
      </ErrorBoundary>,
    );

    // Should show the error boundary panel with region name
    expect(screen.getByTestId("error-boundary-workspace")).toBeTruthy();
    expect(screen.getByText(strings.errorBoundary.somethingWentWrong)).toBeTruthy();
    expect(screen.getByText("test crash")).toBeTruthy();
    expect(screen.getByText(strings.errorBoundary.tryAgain)).toBeTruthy();
  });

  it("renders custom fallback when provided", () => {
    render(
      <ErrorBoundary
        region="terminal"
        fallback={<div data-testid="custom-fallback">Custom</div>}
      >
        <ThrowingChild shouldThrow={true} />
      </ErrorBoundary>,
    );

    expect(screen.getByTestId("custom-fallback")).toBeTruthy();
    // Default panel should NOT be rendered
    expect(screen.queryByTestId("error-boundary-terminal")).toBeNull();
  });

  it("resets error state when Try Again is clicked", () => {
    let shouldThrow = true;

    function ConditionalThrower() {
      if (shouldThrow) throw new Error("crash");
      return <div data-testid="recovered">Recovered</div>;
    }

    render(
      <ErrorBoundary region="pane">
        <ConditionalThrower />
      </ErrorBoundary>,
    );

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
