import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { useBoardKeyboard } from "./useBoardKeyboard";

describe("board keyboard shortcuts", () => {
  it("preserves native activation keys while a real interactive control has focus", async () => {
    const dispatch = vi.fn();
    const goTo = vi.fn();
    const clicked = vi.fn();

    function Control() {
      useBoardKeyboard("/mission-control", [], dispatch, goTo);
      return <><button type="button" onClick={clicked}>Inspect reading</button><label><input type="checkbox" /> Mark reading</label></>;
    }

    render(<Control />);
    const button = screen.getByRole("button", { name: "Inspect reading" });
    button.focus();
    const user = userEvent.setup();

    await user.keyboard("{Enter}");
    await user.keyboard(" ");

    expect(clicked).toHaveBeenCalledTimes(2);
    const checkbox = screen.getByRole("checkbox", { name: "Mark reading" });
    checkbox.focus();
    await user.keyboard(" ");
    expect(checkbox).toBeChecked();
    expect(dispatch).not.toHaveBeenCalled();
    expect(goTo).not.toHaveBeenCalled();
  });

  it("keeps board shortcuts active when focus is on the board surface", () => {
    const dispatch = vi.fn();
    const goTo = vi.fn();
    function Surface() {
      useBoardKeyboard("/mission-control", [], dispatch, goTo);
      return <div data-testid="surface" tabIndex={0}>Board surface</div>;
    }
    render(<Surface />);
    screen.getByTestId("surface").focus();
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "?", bubbles: true }));
    expect(dispatch).toHaveBeenCalledWith("show-help");
  });
});
