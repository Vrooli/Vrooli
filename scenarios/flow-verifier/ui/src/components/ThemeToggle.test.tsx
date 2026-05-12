import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { ThemeToggle } from "./ThemeToggle";

describe("ThemeToggle", () => {
  afterEach(() => cleanup());

  it("renders the toggle with an accessible label", () => {
    renderWithProviders(<ThemeToggle />);
    const btn = screen.getByTestId("theme-toggle");
    expect(btn).toHaveAttribute("aria-label");
  });

  it("cycles theme on click", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ThemeToggle />);
    const btn = screen.getByTestId("theme-toggle");
    const labelBefore = btn.getAttribute("aria-label");
    await user.click(btn);
    expect(btn.getAttribute("aria-label")).not.toBe(labelBefore);
  });
});
