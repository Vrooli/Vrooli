import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Button } from "./Button";

describe("Button", () => {
  it("renders children and applies primary variant by default", () => {
    render(<Button data-testid="b">Save</Button>);
    const el = screen.getByTestId("b");
    expect(el).toHaveTextContent("Save");
    expect(el.className).toMatch(/bg-app-primary/);
  });

  it("merges custom className", () => {
    render(<Button data-testid="b" className="custom-x">Go</Button>);
    expect(screen.getByTestId("b").className).toMatch(/custom-x/);
  });

  it("applies the requested variant", () => {
    render(<Button data-testid="b" variant="danger">Delete</Button>);
    expect(screen.getByTestId("b").className).toMatch(/bg-app-danger/);
  });

  it("invokes onClick when activated", async () => {
    const onClick = vi.fn();
    render(<Button data-testid="b" onClick={onClick}>Run</Button>);
    await userEvent.click(screen.getByTestId("b"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("disables and exposes aria-busy when loading", () => {
    render(<Button loading data-testid="b">Saving</Button>);
    const el = screen.getByTestId("b");
    expect(el).toBeDisabled();
    expect(el).toHaveAttribute("aria-busy", "true");
  });

  it("forwards ref to the button element", () => {
    let captured: HTMLButtonElement | null = null;
    render(<Button ref={(el) => { captured = el; }}>x</Button>);
    expect(captured).toBeInstanceOf(HTMLButtonElement);
  });

  it("renders as child when asChild is true", () => {
    render(
      <Button asChild>
        <a href="/x" data-testid="link-btn">link-btn</a>
      </Button>,
    );
    const link = screen.getByTestId("link-btn");
    expect(link.tagName).toBe("A");
    expect(link).toHaveAttribute("href", "/x");
  });
});
