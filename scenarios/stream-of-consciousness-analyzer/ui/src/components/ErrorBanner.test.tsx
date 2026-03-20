import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import { ErrorBanner } from "./ErrorBanner";
import { ApiRequestError } from "../lib/api";

// Mock @vrooli/api-base (required by api.ts import)
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:3000/api/v1",
  buildApiUrl: (path: string) => `http://localhost:3000/api/v1${path}`,
}));

// [REQ:P0-001] ErrorBanner displays structured errors with recovery actions
describe("ErrorBanner", () => {
  it("renders nothing when error is null", () => {
    const { container } = render(<ErrorBanner error={null} />);
    expect(container.innerHTML).toBe("");
  });

  it("shows generic error message for unknown errors", () => {
    render(<ErrorBanner error={new Error("unknown")} />);
    expect(screen.getByText("Something went wrong. Please try again.")).toBeInTheDocument();
  });

  it("shows network error message for fetch failures", () => {
    render(<ErrorBanner error={new Error("Failed to fetch")} />);
    expect(screen.getByText(/Unable to reach the server/)).toBeInTheDocument();
  });

  it("shows structured message from ApiRequestError", () => {
    const err = new ApiRequestError(400, {
      category: "validation",
      message: "name is required",
      retryable: false,
    });
    render(<ErrorBanner error={err} />);
    expect(screen.getByText("name is required")).toBeInTheDocument();
  });

  it("shows retry button for retryable errors", () => {
    const err = new ApiRequestError(503, {
      category: "dependency",
      message: "LLM unavailable",
      retryable: true,
    });
    const onRetry = vi.fn();
    render(<ErrorBanner error={err} onRetry={onRetry} />);
    const retryBtn = screen.getByTestId("error-retry-btn");
    fireEvent.click(retryBtn);
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("hides retry button for non-retryable errors", () => {
    const err = new ApiRequestError(400, {
      category: "validation",
      message: "bad input",
      retryable: false,
    });
    render(<ErrorBanner error={err} onRetry={vi.fn()} />);
    expect(screen.queryByTestId("error-retry-btn")).not.toBeInTheDocument();
  });

  it("calls onDismiss when dismiss button clicked", () => {
    const onDismiss = vi.fn();
    render(<ErrorBanner error={new Error("oops")} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByLabelText("Dismiss error"));
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it("has accessible role=alert", () => {
    render(<ErrorBanner error={new Error("fail")} />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });
});
