import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ColorSwatch } from "./color-swatch";

// [REQ:BM-REQ-UI-LIBRARY] [REQ:BM-REQ-UI-DASHBOARD]

describe("ColorSwatch", () => {
  it("renders label and hex value", () => {
    render(<ColorSwatch color="#ff0000" label="Primary" />);
    expect(screen.getByText("Primary")).toBeTruthy();
    expect(screen.getByText("#ff0000")).toBeTruthy();
  });

  it("renders swatch with correct background color", () => {
    const { container } = render(<ColorSwatch color="#00ff00" label="Accent" />);
    const swatch = container.querySelector("[style]");
    expect(swatch).toBeTruthy();
    expect(swatch?.getAttribute("style")).toContain("background-color: rgb(0, 255, 0)");
  });

  it("renders nothing when color is undefined", () => {
    const { container } = render(<ColorSwatch color={undefined} label="Empty" />);
    expect(container.innerHTML).toBe("");
  });

  it("sets data-testid from lowercased label", () => {
    render(<ColorSwatch color="#000" label="Primary" />);
    expect(screen.getByTestId("color-swatch-primary")).toBeTruthy();
  });

  it("merges custom className", () => {
    render(<ColorSwatch color="#000" label="Test" className="extra" />);
    const el = screen.getByTestId("color-swatch-test");
    expect(el.className).toContain("extra");
  });
});
