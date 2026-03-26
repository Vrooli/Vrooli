import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ErrorAlert } from "./error-alert";
import { ApiRequestError } from "../../lib/api";

// [REQ:BM-REQ-UI-DASHBOARD]

describe("ErrorAlert", () => {
  it("renders fallback message for generic errors", () => {
    render(<ErrorAlert error={new Error("boom")} testId="err" />);
    expect(screen.getByTestId("err")).toBeTruthy();
    expect(screen.getByText("An unexpected error occurred.")).toBeTruthy();
  });

  it("renders custom fallback message", () => {
    render(<ErrorAlert error={new Error("x")} fallbackMessage="Something broke" testId="err" />);
    expect(screen.getByText("Something broke")).toBeTruthy();
  });

  it("renders ApiRequestError message from server", () => {
    const err = new ApiRequestError(422, { code: "validation", message: "Name is required" });
    render(<ErrorAlert error={err} testId="err" />);
    expect(screen.getByText("Name is required")).toBeTruthy();
  });

  it("renders recovery hint from ApiRequestError", () => {
    const err = new ApiRequestError(422, {
      code: "validation",
      message: "Invalid",
      recovery: "Check the name field",
    });
    render(<ErrorAlert error={err} testId="err" />);
    expect(screen.getByText("Check the name field")).toBeTruthy();
  });

  it("renders fallback recovery hint for generic errors", () => {
    render(
      <ErrorAlert error={new Error("x")} fallbackRecovery="Try refreshing" testId="err" />,
    );
    expect(screen.getByText("Try refreshing")).toBeTruthy();
  });

  it("shows retry button for retryable ApiRequestError", () => {
    const err = new ApiRequestError(500, { code: "internal", message: "Server error" });
    const onRetry = vi.fn();
    render(<ErrorAlert error={err} onRetry={onRetry} testId="err" />);
    const retryBtn = screen.getByRole("button", { name: /retry/i });
    expect(retryBtn).toBeTruthy();
    fireEvent.click(retryBtn);
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("hides retry button for non-retryable ApiRequestError", () => {
    const err = new ApiRequestError(422, { code: "validation", message: "Bad input" });
    const onRetry = vi.fn();
    render(<ErrorAlert error={err} onRetry={onRetry} testId="err" />);
    expect(screen.queryByRole("button", { name: /retry/i })).toBeNull();
  });

  it("shows retry for generic errors with onRetry (status >= 500 fallback)", () => {
    const onRetry = vi.fn();
    render(<ErrorAlert error={new Error("boom")} onRetry={onRetry} testId="err" />);
    const retryBtn = screen.getByRole("button", { name: /retry/i });
    expect(retryBtn).toBeTruthy();
  });

  it("renders null error gracefully", () => {
    render(<ErrorAlert error={null} testId="err" />);
    expect(screen.getByTestId("err")).toBeTruthy();
    expect(screen.getByText("An unexpected error occurred.")).toBeTruthy();
  });

  it("merges custom className", () => {
    render(<ErrorAlert error={new Error("x")} className="extra" testId="err" />);
    const el = screen.getByTestId("err");
    expect(el.className).toContain("extra");
    expect(el.className).toContain("border-red-500/20");
  });
});
