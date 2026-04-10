import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ErrorState } from "./error-state";
import { ApiError } from "../../lib/api-client";
import { selectors } from "../../consts/selectors";

/**
 * ErrorState Component Tests
 *
 * Tests verify the component:
 * - Auto-detects error types from ApiError instances
 * - Displays appropriate icons, titles, and messages for each variant
 * - Shows/hides retry button based on error recoverability
 * - Allows custom title/message overrides
 * - Uses user-friendly messages (no technical details)
 *
 * [REQ:PHASE5] Test graceful degradation patterns
 */

describe("ErrorState", () => {
  describe("auto-detection from ApiError", () => {
    it("renders network error variant for network ApiError", () => {
      const error = new ApiError("network", "Failed to fetch");

      render(<ErrorState error={error} />);

      expect(screen.getByTestId(selectors.error.title)).toHaveTextContent(
        "Connection problem"
      );
      expect(screen.getByTestId(selectors.error.message)).toHaveTextContent(
        "check your internet connection"
      );
    });

    it("renders timeout error variant for timeout ApiError", () => {
      const error = new ApiError("timeout", "Request aborted");

      render(<ErrorState error={error} />);

      expect(screen.getByTestId(selectors.error.title)).toHaveTextContent(
        "Request timed out"
      );
      expect(screen.getByTestId(selectors.error.message)).toHaveTextContent(
        "server is taking too long"
      );
    });

    it("renders server error variant for 5xx ApiError", () => {
      const error = new ApiError("http", "Internal Server Error", { status: 500 });

      render(<ErrorState error={error} />);

      expect(screen.getByTestId(selectors.error.title)).toHaveTextContent(
        "Server error"
      );
      expect(screen.getByTestId(selectors.error.message)).toHaveTextContent(
        "server encountered a problem"
      );
    });

    it("renders notFound variant for 404 ApiError", () => {
      const error = new ApiError("http", "Not Found", { status: 404 });

      render(<ErrorState error={error} />);

      expect(screen.getByTestId(selectors.error.title)).toHaveTextContent(
        "Not found"
      );
      expect(screen.getByTestId(selectors.error.message)).toHaveTextContent(
        "could not be found"
      );
    });

    it("renders generic variant for parse ApiError", () => {
      const error = new ApiError("parse", "Invalid JSON");

      render(<ErrorState error={error} />);

      expect(screen.getByTestId(selectors.error.title)).toHaveTextContent(
        "Server error"
      );
    });
  });

  describe("retry button behavior", () => {
    it("shows retry button for retryable errors", () => {
      const error = new ApiError("network", "Connection failed");
      const onRetry = vi.fn();

      render(<ErrorState error={error} onRetry={onRetry} />);

      const retryButton = screen.getByTestId(selectors.error.retryButton);
      expect(retryButton).toBeInTheDocument();
      expect(retryButton).toHaveTextContent("Try again");
    });

    it("calls onRetry when retry button clicked", () => {
      const error = new ApiError("network", "Connection failed");
      const onRetry = vi.fn();

      render(<ErrorState error={error} onRetry={onRetry} />);

      fireEvent.click(screen.getByTestId(selectors.error.retryButton));
      expect(onRetry).toHaveBeenCalledTimes(1);
    });

    it("hides retry button for notFound variant", () => {
      const error = new ApiError("http", "Not Found", { status: 404 });
      const onRetry = vi.fn();

      render(<ErrorState error={error} onRetry={onRetry} />);

      expect(screen.queryByTestId(selectors.error.retryButton)).not.toBeInTheDocument();
    });

    it("hides retry button when hideRetry is true", () => {
      const error = new ApiError("network", "Connection failed");
      const onRetry = vi.fn();

      render(<ErrorState error={error} onRetry={onRetry} hideRetry={true} />);

      expect(screen.queryByTestId(selectors.error.retryButton)).not.toBeInTheDocument();
    });

    it("hides retry button when no onRetry callback provided", () => {
      const error = new ApiError("network", "Connection failed");

      render(<ErrorState error={error} />);

      expect(screen.queryByTestId(selectors.error.retryButton)).not.toBeInTheDocument();
    });
  });

  describe("custom overrides", () => {
    it("allows custom title override", () => {
      const error = new ApiError("network", "Failed to fetch");

      render(
        <ErrorState error={error} title="Unable to load backlog" />
      );

      expect(screen.getByTestId(selectors.error.title)).toHaveTextContent(
        "Unable to load backlog"
      );
    });

    it("allows custom message override", () => {
      const error = new ApiError("network", "Failed to fetch");

      render(
        <ErrorState error={error} message="Please check your VPN connection." />
      );

      expect(screen.getByTestId(selectors.error.message)).toHaveTextContent(
        "Please check your VPN connection."
      );
    });

    it("allows explicit variant override", () => {
      render(<ErrorState variant="timeout" />);

      expect(screen.getByTestId(selectors.error.title)).toHaveTextContent(
        "Request timed out"
      );
    });
  });

  describe("generic error handling", () => {
    it("renders generic variant for standard Error", () => {
      const error = new Error("Something went wrong");

      render(<ErrorState error={error} />);

      expect(screen.getByTestId(selectors.error.title)).toHaveTextContent(
        "Something went wrong"
      );
    });

    it("renders generic variant when no error provided", () => {
      render(<ErrorState />);

      expect(screen.getByTestId(selectors.error.title)).toHaveTextContent(
        "Something went wrong"
      );
    });

    it("uses ApiError.userMessage for generic display", () => {
      const error = new ApiError("http", "Unauthorized", { status: 401 });

      render(<ErrorState error={error} />);

      // The ApiError.userMessage for 401 should mention session expired
      expect(screen.getByTestId(selectors.error.message)).toHaveTextContent(
        "session"
      );
    });
  });

  describe("accessibility and styling", () => {
    it("renders with test selectors for automation", () => {
      render(<ErrorState variant="generic" />);

      expect(screen.getByTestId(selectors.error.container)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.error.icon)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.error.title)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.error.message)).toBeInTheDocument();
    });

    it("applies custom className", () => {
      render(<ErrorState variant="generic" className="custom-class" />);

      expect(screen.getByTestId(selectors.error.container)).toHaveClass(
        "custom-class"
      );
    });
  });
});
