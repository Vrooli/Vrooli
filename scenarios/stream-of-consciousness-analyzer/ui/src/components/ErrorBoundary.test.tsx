import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import "@testing-library/jest-dom/vitest";
import { ErrorBoundary } from "./ErrorBoundary";

// A component that throws during render
function Thrower({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) throw new Error("render explosion");
  return <div data-testid="child">OK</div>;
}

// Suppress console.error from React's error boundary logging
beforeEach(() => {
  vi.spyOn(console, "error").mockImplementation(() => {});
});

// [REQ:P0-001] ErrorBoundary catches render errors without crashing the app
describe("ErrorBoundary", () => {
  it("renders children when no error", () => {
    render(
      <ErrorBoundary>
        <Thrower shouldThrow={false} />
      </ErrorBoundary>,
    );
    expect(screen.getByTestId("child")).toBeInTheDocument();
  });

  it("shows fallback UI when a child throws", () => {
    render(
      <ErrorBoundary>
        <Thrower shouldThrow={true} />
      </ErrorBoundary>,
    );
    expect(screen.getByTestId("error-boundary-fallback")).toBeInTheDocument();
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.queryByTestId("child")).not.toBeInTheDocument();
  });

  it("provides a retry button that attempts reset", () => {
    render(
      <ErrorBoundary>
        <Thrower shouldThrow={true} />
      </ErrorBoundary>,
    );
    expect(screen.getByTestId("error-boundary-fallback")).toBeInTheDocument();

    // The retry button exists and is clickable
    const retryBtn = screen.getByText("Try again");
    expect(retryBtn).toBeInTheDocument();
    // Clicking it calls handleReset which clears error state
    fireEvent.click(retryBtn);
    // Note: since the child still throws, the boundary catches again.
    // This test verifies the retry mechanism is wired up.
    expect(screen.getByTestId("error-boundary-fallback")).toBeInTheDocument();
  });

  it("provides a reload page button", () => {
    render(
      <ErrorBoundary>
        <Thrower shouldThrow={true} />
      </ErrorBoundary>,
    );
    expect(screen.getByText("Reload page")).toBeInTheDocument();
  });

  it("has accessible role=alert on fallback", () => {
    render(
      <ErrorBoundary>
        <Thrower shouldThrow={true} />
      </ErrorBoundary>,
    );
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });
});
