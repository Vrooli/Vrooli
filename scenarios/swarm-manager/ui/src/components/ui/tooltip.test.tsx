import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { Tooltip } from "./tooltip";

describe("Tooltip", () => {
  it("shows on hover after a delay and hides on mouse leave", async () => {
    const user = userEvent.setup();

    render(
      <Tooltip content="Helpful context" delayMs={0} testId="tooltip">
        <button type="button">Hover me</button>
      </Tooltip>,
    );

    await user.hover(screen.getByRole("button", { name: "Hover me" }));
    await waitFor(() => expect(screen.getByRole("tooltip")).toHaveTextContent("Helpful context"));

    await user.unhover(screen.getByRole("button", { name: "Hover me" }));
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });

  it("shows on focus and hides on Escape", async () => {
    const user = userEvent.setup();

    render(
      <Tooltip content="Keyboard context" delayMs={0}>
        <button type="button">Focus me</button>
      </Tooltip>,
    );

    await user.tab();

    const trigger = screen.getByRole("button", { name: "Focus me" });
    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip).toHaveTextContent("Keyboard context");
    expect(trigger).toHaveAttribute("aria-describedby");

    await user.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByRole("tooltip")).not.toBeInTheDocument());
  });
});
