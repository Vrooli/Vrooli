import { describe, it, expect, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { MicReadinessIndicator } from "./MicReadinessIndicator";

afterEach(() => {
  cleanup();
});

describe("MicReadinessIndicator", () => {
  it("renders with role=status and aria-live=polite", () => {
    render(<MicReadinessIndicator state="unknown" />);
    const el = screen.getByRole("status");
    expect(el).toHaveAttribute("aria-live", "polite");
  });

  it("sets data-state to the given state", () => {
    render(<MicReadinessIndicator state="granted" />);
    expect(screen.getByRole("status")).toHaveAttribute("data-state", "granted");
  });

  it("shows the default label for 'granted'", () => {
    render(<MicReadinessIndicator state="granted" />);
    expect(screen.getByText(/Microphone ready/)).toBeInTheDocument();
  });

  it("shows the default label for 'denied'", () => {
    render(<MicReadinessIndicator state="denied" />);
    expect(screen.getByText(/Microphone denied/)).toBeInTheDocument();
  });

  it("shows the default label for 'prompt'", () => {
    render(<MicReadinessIndicator state="prompt" />);
    expect(screen.getByText(/Microphone permission required/)).toBeInTheDocument();
  });

  it("shows the default label for 'unknown'", () => {
    render(<MicReadinessIndicator state="unknown" />);
    expect(screen.getByText(/Microphone status unknown/)).toBeInTheDocument();
  });

  it("uses a caller-supplied label override when provided", () => {
    render(
      <MicReadinessIndicator
        state="granted"
        labels={{ granted: "Mic OK" }}
      />,
    );
    expect(screen.getByText(/Mic OK/)).toBeInTheDocument();
  });

  it("falls back to the default label when the caller label for that state is undefined", () => {
    // Only the 'denied' label is overridden; 'granted' still uses the default.
    render(
      <MicReadinessIndicator
        state="granted"
        labels={{ denied: "No mic" }}
      />,
    );
    expect(screen.getByText(/Microphone ready/)).toBeInTheDocument();
  });

  it("applies the audio-tools-embed-mic-readiness class", () => {
    render(<MicReadinessIndicator state="prompt" />);
    expect(screen.getByRole("status")).toHaveClass("audio-tools-embed-mic-readiness");
  });
});
