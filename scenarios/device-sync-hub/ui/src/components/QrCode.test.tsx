import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

import { QrCode } from "./QrCode";

describe("QrCode", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("renders an SVG with a non-empty module path for a valid payload", () => {
    render(<QrCode value="PAIR-1234" data-testid="qr" aria-label="scan me" />);
    const svg = screen.getByTestId("qr");
    expect(svg.tagName.toLowerCase()).toBe("svg");
    expect(svg).toHaveAttribute("aria-label", "scan me");
    const path = svg.querySelector("path");
    expect(path?.getAttribute("d")?.length).toBeGreaterThan(0);
  });

  it("honours the size override", () => {
    render(<QrCode value="X" size={64} data-testid="qr" />);
    expect(screen.getByTestId("qr")).toHaveAttribute("width", "64");
  });

  it("renders nothing when encoding fails", () => {
    // An over-long payload makes encodeQr throw; the component swallows it.
    const { container } = render(<QrCode value={"Z".repeat(500)} />);
    expect(container.querySelector("svg")).toBeNull();
  });
});
