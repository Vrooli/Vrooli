import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Input, Textarea } from "./input";

// [REQ:BM-REQ-UI-CREATE] [REQ:BM-REQ-UI-GENERATE]

describe("Input", () => {
  it("renders default variant", () => {
    render(<Input data-testid="inp" placeholder="Type here" />);
    const inp = screen.getByTestId("inp");
    expect(inp).toBeTruthy();
    expect(inp.getAttribute("placeholder")).toBe("Type here");
    expect(inp.className).toContain("rounded-lg");
  });

  it("renders search variant with extra padding", () => {
    render(<Input variant="search" data-testid="search" />);
    const inp = screen.getByTestId("search");
    expect(inp.className).toContain("pl-10");
  });

  it("renders sm size", () => {
    render(<Input inputSize="sm" data-testid="sm" />);
    const inp = screen.getByTestId("sm");
    expect(inp.className).toContain("text-xs");
  });

  it("forwards onChange handler", () => {
    const handler = vi.fn();
    render(<Input data-testid="inp" onChange={handler} />);
    fireEvent.change(screen.getByTestId("inp"), { target: { value: "hello" } });
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("merges custom className", () => {
    render(<Input className="my-custom" data-testid="inp" />);
    expect(screen.getByTestId("inp").className).toContain("my-custom");
  });

  it("forwards disabled attribute", () => {
    render(<Input disabled data-testid="inp" />);
    expect(screen.getByTestId("inp").getAttribute("disabled")).toBeDefined();
  });
});

describe("Textarea", () => {
  it("renders with placeholder", () => {
    render(<Textarea data-testid="ta" placeholder="Notes..." />);
    const ta = screen.getByTestId("ta");
    expect(ta).toBeTruthy();
    expect(ta.getAttribute("placeholder")).toBe("Notes...");
  });

  it("forwards onChange handler", () => {
    const handler = vi.fn();
    render(<Textarea data-testid="ta" onChange={handler} />);
    fireEvent.change(screen.getByTestId("ta"), { target: { value: "text" } });
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("merges custom className", () => {
    render(<Textarea className="extra" data-testid="ta" />);
    expect(screen.getByTestId("ta").className).toContain("extra");
  });
});
