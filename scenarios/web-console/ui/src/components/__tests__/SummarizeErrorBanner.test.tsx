import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import SummarizeErrorBanner, { type SummarizeErrorState } from "../SummarizeErrorBanner";
import { strings } from "../../consts/strings";

function makeState(overrides?: Partial<SummarizeErrorState>): SummarizeErrorState {
  return {
    sessionId: "sess-1",
    eventId: "ev-1",
    message: "Ollama timed out",
    source: "auto",
    status: "failed",
    ...overrides,
  };
}

describe("SummarizeErrorBanner", () => {
  it("renders the message and source label for auto failures", () => {
    render(<SummarizeErrorBanner state={makeState({ source: "auto" })} onRetry={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("summarize-error-banner")).toBeInTheDocument();
    expect(screen.getByText(strings.summarizeError.autoFailed)).toBeInTheDocument();
    expect(screen.getByText("Ollama timed out")).toBeInTheDocument();
  });

  it("renders a different source label for on-demand failures", () => {
    render(<SummarizeErrorBanner state={makeState({ source: "on-demand" })} onRetry={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByText(strings.summarizeError.failed)).toBeInTheDocument();
  });

  it("clicking retry fires onRetry", () => {
    const onRetry = vi.fn();
    render(<SummarizeErrorBanner state={makeState()} onRetry={onRetry} onDismiss={vi.fn()} />);
    fireEvent.click(screen.getByTestId("summarize-error-retry"));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("clicking dismiss fires onDismiss", () => {
    const onDismiss = vi.fn();
    render(<SummarizeErrorBanner state={makeState()} onRetry={vi.fn()} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByTestId("summarize-error-dismiss"));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("disables retry and dismiss while retrying and shows a spinner", () => {
    render(<SummarizeErrorBanner state={makeState({ status: "retrying" })} onRetry={vi.fn()} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("summarize-error-retry")).toBeDisabled();
    expect(screen.getByTestId("summarize-error-dismiss")).toBeDisabled();
    expect(screen.getByText(strings.summarizeError.retrying)).toBeInTheDocument();
  });

  it("banner carries data attributes for source and status", () => {
    render(<SummarizeErrorBanner state={makeState({ source: "on-demand", status: "retrying" })} onRetry={vi.fn()} onDismiss={vi.fn()} />);
    const banner = screen.getByTestId("summarize-error-banner");
    expect(banner.getAttribute("data-source")).toBe("on-demand");
    expect(banner.getAttribute("data-status")).toBe("retrying");
  });
});
