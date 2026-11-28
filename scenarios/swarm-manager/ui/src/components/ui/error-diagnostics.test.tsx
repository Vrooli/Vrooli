import { screen, fireEvent, waitFor } from "@testing-library/react";
import { renderWithProviders as renderWithCanonicalProviders } from "../../test-utils/renderWithProviders";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ErrorDiagnostics } from "./error-diagnostics";
import { selectors } from "../../consts/selectors";

function makeError(name: string, message: string): Error {
  const err = new Error(message);
  err.name = name;
  return err;
}

const defaultProps = {
  error: makeError("TypeError", "Cannot read properties of undefined"),
  componentStack: "\n    at GraphCanvas\n    at CanvasErrorBoundary\n    at GraphWorkspace",
  errorId: "err_test123_abc456",
  timestamp: "2026-04-01T12:00:00.000Z",
};

describe("ErrorDiagnostics", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("renders collapsed by default", () => {
    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} />);
    expect(screen.getByTestId(selectors.errorBoundary.showDetailsButton)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.errorBoundary.diagnosticsPanel)).not.toBeInTheDocument();
  });

  it("expands when Show Details is clicked", () => {
    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} />);
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.showDetailsButton));
    expect(screen.getByTestId(selectors.errorBoundary.diagnosticsPanel)).toBeInTheDocument();
  });

  it("collapses when Hide Details is clicked", () => {
    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} />);
    const toggle = screen.getByTestId(selectors.errorBoundary.showDetailsButton);
    fireEvent.click(toggle); // expand
    fireEvent.click(toggle); // collapse
    expect(screen.queryByTestId(selectors.errorBoundary.diagnosticsPanel)).not.toBeInTheDocument();
  });

  it("displays error name and sanitized message", () => {
    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} />);
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.showDetailsButton));
    expect(screen.getByTestId(selectors.errorBoundary.errorName)).toHaveTextContent("TypeError");
    expect(screen.getByTestId(selectors.errorBoundary.errorMessage)).toHaveTextContent(
      "Cannot read properties of undefined",
    );
  });

  it("sanitizes URLs in error messages", () => {
    const error = makeError("Error", "Failed to fetch https://api.example.com/secret-token");
    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} error={error} />);
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.showDetailsButton));
    expect(screen.getByTestId(selectors.errorBoundary.errorMessage)).toHaveTextContent(
      "Failed to fetch [URL]",
    );
  });

  it("displays component stack in a pre block", () => {
    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} />);
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.showDetailsButton));
    const stack = screen.getByTestId(selectors.errorBoundary.componentStack);
    expect(stack.tagName).toBe("PRE");
    expect(stack).toHaveTextContent("at GraphCanvas");
  });

  it("handles null componentStack gracefully", () => {
    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} componentStack={null} />);
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.showDetailsButton));
    expect(screen.getByTestId(selectors.errorBoundary.componentStack)).toHaveTextContent(
      "Component stack not available",
    );
  });

  it("displays timestamp and user agent", () => {
    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} />);
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.showDetailsButton));
    expect(screen.getByTestId(selectors.errorBoundary.timestamp)).toHaveTextContent(
      "2026-04-01T12:00:00.000Z",
    );
    expect(screen.getByTestId(selectors.errorBoundary.userAgent)).toBeInTheDocument();
  });

  it("displays error category badge", () => {
    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} />);
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.showDetailsButton));
    expect(screen.getByTestId(selectors.errorBoundary.errorCategory)).toHaveTextContent("RUNTIME");
  });

  it("copies diagnostics to clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} />);
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.showDetailsButton));
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.copyButton));

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledTimes(1);
    });

    const copiedText = String(writeText.mock.calls[0]?.[0]);
    expect(copiedText).toContain("TypeError");
    expect(copiedText).toContain("Cannot read properties of undefined");
    expect(copiedText).toContain("err_test123_abc456");
  });

  it("shows copy confirmation", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} />);
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.showDetailsButton));
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.copyButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.errorBoundary.copyConfirmation)).toHaveTextContent("Copied!");
    });
  });

  it("renders in compact mode with smaller styles", () => {
    renderWithCanonicalProviders(<ErrorDiagnostics {...defaultProps} compact />);
    fireEvent.click(screen.getByTestId(selectors.errorBoundary.showDetailsButton));
    const panel = screen.getByTestId(selectors.errorBoundary.diagnosticsPanel);
    expect(panel.className).toContain("p-3");
  });
});
