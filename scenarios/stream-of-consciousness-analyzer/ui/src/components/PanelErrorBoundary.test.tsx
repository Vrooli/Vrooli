import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import "@testing-library/jest-dom/vitest";
import { PanelErrorBoundary } from "./PanelErrorBoundary";

// Suppress console.error for intentional throws in error boundary tests
beforeEach(() => {
  vi.spyOn(console, "error").mockImplementation(() => {});
});

let throwOnRender = false;

function ThrowingChild() {
  if (throwOnRender) throw new Error("Test crash");
  return <div data-testid="child-content">Content</div>;
}

describe("PanelErrorBoundary", () => {
  beforeEach(() => {
    throwOnRender = false;
  });

  it("renders children when no error occurs", () => {
    render(
      <PanelErrorBoundary panelName="Canvas">
        <ThrowingChild />
      </PanelErrorBoundary>,
    );
    expect(screen.getByTestId("child-content")).toBeInTheDocument();
  });

  it("renders fallback UI with panel name when child throws", () => {
    throwOnRender = true;
    render(
      <PanelErrorBoundary panelName="Canvas">
        <ThrowingChild />
      </PanelErrorBoundary>,
    );
    expect(screen.queryByTestId("child-content")).not.toBeInTheDocument();
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("Canvas encountered an error.")).toBeInTheDocument();
  });

  it("recovers when Retry is clicked after underlying issue resolves", () => {
    throwOnRender = true;
    render(
      <PanelErrorBoundary panelName="Graph">
        <ThrowingChild />
      </PanelErrorBoundary>,
    );
    expect(screen.getByText("Graph encountered an error.")).toBeInTheDocument();

    // Simulate the underlying issue being resolved
    throwOnRender = false;

    // Click retry — boundary resets, child no longer throws
    fireEvent.click(screen.getByText("Retry"));
    expect(screen.getByTestId("child-content")).toBeInTheDocument();
  });

  it("isolates crash from sibling panels", () => {
    throwOnRender = true;

    function SafeChild() {
      return <div data-testid="safe-content">Safe</div>;
    }

    render(
      <div>
        <PanelErrorBoundary panelName="Canvas">
          <ThrowingChild />
        </PanelErrorBoundary>
        <PanelErrorBoundary panelName="Text Capture">
          <SafeChild />
        </PanelErrorBoundary>
      </div>,
    );
    // Canvas crashed but Text Capture is still alive
    expect(screen.getByText("Canvas encountered an error.")).toBeInTheDocument();
    expect(screen.getByTestId("safe-content")).toBeInTheDocument();
  });
});
