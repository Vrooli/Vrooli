import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";

import { MicReadinessIndicator } from "./MicReadinessIndicator";

describe("MicReadinessIndicator", () => {
  it("renders the default label for each documented state via role=status", () => {
    const { rerender } = render(<MicReadinessIndicator state="granted" />);
    expect(screen.getByRole("status")).toHaveTextContent("Microphone ready");

    rerender(<MicReadinessIndicator state="denied" />);
    expect(screen.getByRole("status")).toHaveTextContent("Microphone denied");

    rerender(<MicReadinessIndicator state="prompt" />);
    expect(screen.getByRole("status")).toHaveTextContent("Microphone permission required");

    rerender(<MicReadinessIndicator state="unknown" />);
    expect(screen.getByRole("status")).toHaveTextContent("Microphone status unknown");
  });

  it("uses caller-supplied per-state labels when provided", () => {
    render(
      <MicReadinessIndicator
        state="granted"
        labels={{ granted: "Mic OK", denied: "No access" }}
      />,
    );
    expect(screen.getByRole("status")).toHaveTextContent("Mic OK");
  });

  it("sets aria-live=polite and data-state for assistive tech", () => {
    render(<MicReadinessIndicator state="prompt" />);
    const status = screen.getByRole("status");
    expect(status).toHaveAttribute("aria-live", "polite");
    expect(status).toHaveAttribute("data-state", "prompt");
  });
});
