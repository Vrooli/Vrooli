/**
 * ErrorAlert Component Tests
 *
 * Tests for error recovery action decisions - maps error categories
 * to appropriate recovery actions (retry, back, help).
 *
 * [REQ:LD-UI-ERROR] Error handling and recovery UI
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ErrorAlert } from "./ErrorAlert";
import { APIError, type ErrorCategory } from "../lib/api";

// Helper to create APIError instances for testing
function createAPIError(
  category: ErrorCategory,
  message: string,
  code: string = "TEST_ERROR"
): APIError {
  return new APIError(
    {
      error: true,
      category,
      code,
      message,
    },
    category === "not_found" ? 404 : category === "validation" ? 400 : 500
  );
}

describe("ErrorAlert", () => {
  describe("getRecoveryAction decision logic", () => {
    it("returns retry action for internal errors", () => {
      const onRetry = vi.fn();
      render(
        <ErrorAlert
          error={createAPIError("internal", "Server error")}
          onRetry={onRetry}
        />
      );

      const retryButton = screen.getByRole("button", { name: /try again/i });
      expect(retryButton).toBeInTheDocument();
      fireEvent.click(retryButton);
      expect(onRetry).toHaveBeenCalled();
    });

    it("returns retry action for unavailable errors", () => {
      const onRetry = vi.fn();
      render(
        <ErrorAlert
          error={createAPIError("unavailable", "Service unavailable")}
          onRetry={onRetry}
        />
      );

      const retryButton = screen.getByRole("button", { name: /try again/i });
      expect(retryButton).toBeInTheDocument();
      fireEvent.click(retryButton);
      expect(onRetry).toHaveBeenCalled();
    });

    it("returns back action for not_found errors", () => {
      const onBack = vi.fn();
      render(
        <ErrorAlert
          error={createAPIError("not_found", "Resource not found")}
          onBack={onBack}
        />
      );

      const backButton = screen.getByRole("button", { name: /go back/i });
      expect(backButton).toBeInTheDocument();
      fireEvent.click(backButton);
      expect(onBack).toHaveBeenCalled();
    });

    it("returns help action for validation errors", () => {
      render(
        <ErrorAlert
          error={createAPIError("validation", "Invalid input")}
        />
      );

      // Validation errors show "Get Help" link, not a button
      const helpLink = screen.getByRole("link", { name: /get help/i });
      expect(helpLink).toBeInTheDocument();
      expect(helpLink).toHaveAttribute("href", "https://github.com/anthropics/claude-code/issues");
    });

    it("returns retry action for network errors (Failed to fetch)", () => {
      const onRetry = vi.fn();
      render(
        <ErrorAlert
          error={new Error("Failed to fetch")}
          onRetry={onRetry}
        />
      );

      const retryButton = screen.getByRole("button", { name: /try again/i });
      expect(retryButton).toBeInTheDocument();
      fireEvent.click(retryButton);
      expect(onRetry).toHaveBeenCalled();
    });

    it("returns retry action for NetworkError messages", () => {
      const onRetry = vi.fn();
      render(
        <ErrorAlert
          error={new Error("NetworkError when attempting to fetch resource")}
          onRetry={onRetry}
        />
      );

      const retryButton = screen.getByRole("button", { name: /try again/i });
      expect(retryButton).toBeInTheDocument();
    });

    it("defaults to retry action for unknown errors", () => {
      const onRetry = vi.fn();
      render(
        <ErrorAlert
          error={new Error("Unknown error")}
          onRetry={onRetry}
        />
      );

      const retryButton = screen.getByRole("button", { name: /try again/i });
      expect(retryButton).toBeInTheDocument();
    });
  });

  describe("getErrorTitle decision logic", () => {
    it('shows "Invalid Request" for validation errors', () => {
      render(
        <ErrorAlert
          error={createAPIError("validation", "Field is required")}
        />
      );

      expect(screen.getByText("Invalid Request")).toBeInTheDocument();
    });

    it('shows "Not Found" for not_found errors', () => {
      render(
        <ErrorAlert
          error={createAPIError("not_found", "Resource not found")}
        />
      );

      expect(screen.getByText("Not Found")).toBeInTheDocument();
    });

    it('shows "Conflict" for conflict errors', () => {
      render(
        <ErrorAlert
          error={createAPIError("conflict", "State conflict")}
        />
      );

      expect(screen.getByText("Conflict")).toBeInTheDocument();
    });

    it('shows "Service Unavailable" for unavailable errors', () => {
      render(
        <ErrorAlert
          error={createAPIError("unavailable", "Service down")}
        />
      );

      expect(screen.getByText("Service Unavailable")).toBeInTheDocument();
    });

    it('shows "Something Went Wrong" for internal errors', () => {
      render(
        <ErrorAlert
          error={createAPIError("internal", "Internal server error")}
        />
      );

      expect(screen.getByText("Something Went Wrong")).toBeInTheDocument();
    });

    it('shows "Connection Error" for network errors', () => {
      render(
        <ErrorAlert
          error={new Error("Failed to fetch")}
        />
      );

      expect(screen.getByText("Connection Error")).toBeInTheDocument();
    });

    it('shows generic "Error" for other errors', () => {
      render(
        <ErrorAlert
          error={new Error("Some random error")}
        />
      );

      expect(screen.getByText("Error")).toBeInTheDocument();
    });
  });

  describe("error message and recovery hints", () => {
    it("displays error message from APIError", () => {
      render(
        <ErrorAlert
          error={createAPIError("validation", "The domain field is required")}
        />
      );

      expect(screen.getByText("The domain field is required")).toBeInTheDocument();
    });

    it("displays recovery hint from APIError when provided", () => {
      const errorWithRecovery = new APIError(
        {
          error: true,
          category: "validation",
          code: "MISSING_FIELD",
          message: "Missing required field",
          recovery: "Please add the 'name' field",
        },
        400
      );

      render(<ErrorAlert error={errorWithRecovery} />);

      expect(screen.getByText("Please add the 'name' field")).toBeInTheDocument();
    });

    it("displays default hint for validation errors without recovery", () => {
      render(
        <ErrorAlert
          error={createAPIError("validation", "Bad input")}
        />
      );

      expect(screen.getByText("Please check your input and try again")).toBeInTheDocument();
    });

    it("displays network error hint", () => {
      render(
        <ErrorAlert
          error={new Error("Failed to fetch")}
        />
      );

      expect(screen.getByText("Unable to connect to the API")).toBeInTheDocument();
      expect(screen.getByText(/Make sure the scenario is running/)).toBeInTheDocument();
    });
  });

  describe("component rendering", () => {
    it("renders nothing when error is null", () => {
      const { container } = render(<ErrorAlert error={null} />);
      expect(container.firstChild).toBeNull();
    });

    it("applies custom className", () => {
      render(
        <ErrorAlert
          error={new Error("Test error")}
          className="custom-class"
        />
      );

      const alert = screen.getByText("Test error").closest("div")?.parentElement?.parentElement;
      expect(alert?.className).toContain("custom-class");
    });

    it("does not show retry button without onRetry handler", () => {
      render(
        <ErrorAlert
          error={createAPIError("internal", "Server error")}
        />
      );

      expect(screen.queryByRole("button", { name: /try again/i })).not.toBeInTheDocument();
    });

    it("does not show back button without onBack handler", () => {
      render(
        <ErrorAlert
          error={createAPIError("not_found", "Not found")}
        />
      );

      expect(screen.queryByRole("button", { name: /go back/i })).not.toBeInTheDocument();
    });
  });
});
