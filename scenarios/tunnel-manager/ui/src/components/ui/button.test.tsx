import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { Button } from "./button";

// provider-free-exception: this leaf primitive has no provider dependencies.

describe("Button", () => {
  it("renders the outline small variant", () => {
    render(<Button variant="outline" size="sm">Inspect</Button>);
    const button = screen.getByRole("button", { name: "Inspect" });
    expect(button.className).toContain("border");
    expect(button.className).toContain("h-9");
  });

  it("renders children through Slot when asChild is set", () => {
    render(<Button asChild><a href="/settings">Settings</a></Button>);
    expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute("href", "/settings");
  });
});
