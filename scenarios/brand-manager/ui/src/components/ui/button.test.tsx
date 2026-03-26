import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Button } from "./button";

// [REQ:BM-REQ-UI-DASHBOARD] [REQ:BM-REQ-UI-CREATE]

describe("Button", () => {
  it("renders with default variant and size", () => {
    render(<Button>Click me</Button>);
    const btn = screen.getByRole("button", { name: "Click me" });
    expect(btn).toBeTruthy();
    expect(btn.className).toContain("bg-slate-50");
    expect(btn.className).toContain("h-11");
  });

  it("renders outline variant", () => {
    render(<Button variant="outline">Outline</Button>);
    const btn = screen.getByRole("button", { name: "Outline" });
    expect(btn.className).toContain("border");
  });

  it("renders sm size", () => {
    render(<Button size="sm">Small</Button>);
    const btn = screen.getByRole("button", { name: "Small" });
    expect(btn.className).toContain("h-9");
  });

  it("forwards onClick handler", () => {
    const handler = vi.fn();
    render(<Button onClick={handler}>Act</Button>);
    fireEvent.click(screen.getByRole("button", { name: "Act" }));
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("is disabled when disabled prop is set", () => {
    render(<Button disabled>Disabled</Button>);
    const btn = screen.getByRole("button", { name: "Disabled" });
    expect(btn.getAttribute("disabled")).toBeDefined();
    expect(btn.className).toContain("disabled:opacity-60");
  });

  it("merges custom className", () => {
    render(<Button className="custom-class">Custom</Button>);
    const btn = screen.getByRole("button", { name: "Custom" });
    expect(btn.className).toContain("custom-class");
  });

  it("forwards data-testid", () => {
    render(<Button data-testid="test-btn">Test</Button>);
    expect(screen.getByTestId("test-btn")).toBeTruthy();
  });

  it("renders as child slot when asChild is true", () => {
    render(
      <Button asChild>
        <a href="#test">Link</a>
      </Button>,
    );
    const link = screen.getByRole("link", { name: "Link" });
    expect(link).toBeTruthy();
    expect(link.className).toContain("inline-flex");
  });
});
