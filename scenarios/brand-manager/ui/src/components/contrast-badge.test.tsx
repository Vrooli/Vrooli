import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ContrastBadge } from "./contrast-badge";

// [REQ:BM-REQ-WCAG-CALC] [REQ:BM-REQ-WCAG-VALIDATE] [REQ:BM-REQ-UI-DASHBOARD]

describe("ContrastBadge", () => {
  it("renders passing badge with ratio and AA label", () => {
    render(<ContrastBadge ratio={5.2} passes={true} />);
    const badge = screen.getByTestId("contrast-badge");
    expect(badge.textContent).toContain("5.2:1");
    expect(badge.textContent).toContain("✓ AA");
  });

  it("renders failing badge with Fail label", () => {
    render(<ContrastBadge ratio={2.1} passes={false} />);
    const badge = screen.getByTestId("contrast-badge");
    expect(badge.textContent).toContain("2.1:1");
    expect(badge.textContent).toContain("✗ Fail");
  });

  it("applies emerald classes when passing", () => {
    render(<ContrastBadge ratio={7.0} passes={true} />);
    const badge = screen.getByTestId("contrast-badge");
    expect(badge.className).toContain("emerald");
  });

  it("applies red classes when failing", () => {
    render(<ContrastBadge ratio={1.5} passes={false} />);
    const badge = screen.getByTestId("contrast-badge");
    expect(badge.className).toContain("red");
  });

  it("merges custom className", () => {
    render(<ContrastBadge ratio={4.5} passes={true} className="my-class" />);
    const badge = screen.getByTestId("contrast-badge");
    expect(badge.className).toContain("my-class");
  });
});
