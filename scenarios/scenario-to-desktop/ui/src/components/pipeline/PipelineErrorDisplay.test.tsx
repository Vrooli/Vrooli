/**
 * Tests for PipelineErrorDisplay components.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@/test-utils";
import { PipelineErrorDisplay, PipelineErrorRecovery, InlineError } from "./PipelineErrorDisplay";
import { suggestRecovery } from "./pipelineUtils";
import type { PipelineErrorInfo } from "../../store/pipelineTypes";

// Mock writeToClipboard
vi.mock("../../lib/browser", () => ({
  writeToClipboard: vi.fn().mockResolvedValue({ success: true }),
}));

describe("suggestRecovery", () => {
  it("suggests scenario check for 404 errors", () => {
    const suggestion = suggestRecovery("Resource not found (404)", "my-scenario");
    expect(suggestion).toContain("my-scenario");
    expect(suggestion).toContain("exists");
  });

  it("suggests UI build for UI not built errors", () => {
    const suggestion = suggestRecovery("UI not built", "test-app");
    expect(suggestion).toContain("Build the scenario UI first");
  });

  it("suggests permission check for EACCES errors", () => {
    const suggestion = suggestRecovery("EACCES permission denied");
    expect(suggestion).toContain("permissions");
  });

  it("suggests disk space for ENOSPC errors", () => {
    const suggestion = suggestRecovery("ENOSPC: no space left on device");
    expect(suggestion).toContain("disk space");
  });

  it("suggests port resolution for port errors", () => {
    const suggestion = suggestRecovery("EADDRINUSE: port already in use");
    expect(suggestion).toContain("port");
  });

  it("returns null for unknown errors", () => {
    const suggestion = suggestRecovery("Some random error");
    expect(suggestion).toBeNull();
  });
});

describe("PipelineErrorDisplay", () => {
  it("renders error message", () => {
    render(<PipelineErrorDisplay errorMessage="Test error" />);
    expect(screen.getByText("Test error")).toBeInTheDocument();
  });

  it("renders custom title", () => {
    render(<PipelineErrorDisplay title="Custom Title" errorMessage="Error" />);
    expect(screen.getByText("Custom Title")).toBeInTheDocument();
  });

  it("renders suggestion when provided", () => {
    render(
      <PipelineErrorDisplay
        errorMessage="Error"
        suggestion="Try this to fix it"
      />
    );
    expect(screen.getByText("Try this to fix it")).toBeInTheDocument();
  });

  it("renders retry button when onRetry provided", () => {
    const onRetry = vi.fn();
    render(<PipelineErrorDisplay errorMessage="Error" onRetry={onRetry} />);
    const retryButton = screen.getByRole("button", { name: /retry/i });
    expect(retryButton).toBeInTheDocument();
    fireEvent.click(retryButton);
    expect(onRetry).toHaveBeenCalled();
  });

  it("does not render retry button when onRetry not provided", () => {
    render(<PipelineErrorDisplay errorMessage="Error" />);
    expect(screen.queryByRole("button", { name: /retry/i })).not.toBeInTheDocument();
  });
});

describe("InlineError", () => {
  it("renders error message", () => {
    render(<InlineError message="Inline error message" />);
    expect(screen.getByText("Inline error message")).toBeInTheDocument();
  });

  it("renders retry button when onRetry provided", () => {
    const onRetry = vi.fn();
    render(<InlineError message="Error" onRetry={onRetry} />);
    const retryButton = screen.getByRole("button", { name: /retry/i });
    expect(retryButton).toBeInTheDocument();
    fireEvent.click(retryButton);
    expect(onRetry).toHaveBeenCalled();
  });
});

describe("PipelineErrorRecovery", () => {
  const baseErrorInfo: PipelineErrorInfo = {
    message: "Test error message",
    category: "network",
  };

  it("renders error message", () => {
    render(<PipelineErrorRecovery errorInfo={baseErrorInfo} />);
    expect(screen.getByText("Test error message")).toBeInTheDocument();
  });

  it("renders error category", () => {
    render(<PipelineErrorRecovery errorInfo={baseErrorInfo} />);
    expect(screen.getByText(/network/i)).toBeInTheDocument();
  });

  it("renders suggestions from errorInfo when available", () => {
    const errorWithSuggestions: PipelineErrorInfo = {
      ...baseErrorInfo,
      suggestions: ["Try restarting", "Check connection"],
    };
    render(<PipelineErrorRecovery errorInfo={errorWithSuggestions} />);
    expect(screen.getByText("Try restarting")).toBeInTheDocument();
    expect(screen.getByText("Check connection")).toBeInTheDocument();
  });

  it("renders category-based suggestions when errorInfo has no suggestions", () => {
    render(<PipelineErrorRecovery errorInfo={baseErrorInfo} />);
    // Network category suggestions include "Check your internet connection"
    expect(screen.getByText(/Check your internet connection/i)).toBeInTheDocument();
  });

  it("renders retry button when onRetry provided", () => {
    const onRetry = vi.fn();
    render(<PipelineErrorRecovery errorInfo={baseErrorInfo} onRetry={onRetry} />);
    const retryButton = screen.getByRole("button", { name: /retry/i });
    expect(retryButton).toBeInTheDocument();
    fireEvent.click(retryButton);
    expect(onRetry).toHaveBeenCalled();
  });

  it("renders dismiss button when onDismiss provided", () => {
    const onDismiss = vi.fn();
    render(<PipelineErrorRecovery errorInfo={baseErrorInfo} onDismiss={onDismiss} />);
    const dismissButton = screen.getByRole("button", { name: /dismiss/i });
    expect(dismissButton).toBeInTheDocument();
    fireEvent.click(dismissButton);
    expect(onDismiss).toHaveBeenCalled();
  });

  it("renders copy button", () => {
    render(<PipelineErrorRecovery errorInfo={baseErrorInfo} />);
    expect(screen.getByRole("button", { name: /copy/i })).toBeInTheDocument();
  });

  it("handles validation category", () => {
    const validationError: PipelineErrorInfo = {
      message: "Validation failed",
      category: "validation",
    };
    render(<PipelineErrorRecovery errorInfo={validationError} />);
    expect(screen.getByText(/Review your form inputs/i)).toBeInTheDocument();
  });

  it("handles permission category", () => {
    const permissionError: PipelineErrorInfo = {
      message: "Permission denied",
      category: "permission",
    };
    render(<PipelineErrorRecovery errorInfo={permissionError} />);
    expect(screen.getByText(/Check file and directory permissions/i)).toBeInTheDocument();
  });

  it("handles unknown category", () => {
    const unknownError: PipelineErrorInfo = {
      message: "Unknown error",
      category: "unknown",
    };
    render(<PipelineErrorRecovery errorInfo={unknownError} />);
    expect(screen.getByText(/An unexpected error occurred/i)).toBeInTheDocument();
  });

  it("handles error without category", () => {
    const errorWithoutCategory: PipelineErrorInfo = {
      message: "Error without category",
    };
    render(<PipelineErrorRecovery errorInfo={errorWithoutCategory} />);
    expect(screen.getByText("Error without category")).toBeInTheDocument();
    // Should not render suggestions section when no category
    expect(screen.queryByText("Suggested actions")).not.toBeInTheDocument();
  });
});
