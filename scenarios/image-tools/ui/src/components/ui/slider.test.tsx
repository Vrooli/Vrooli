import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Slider } from "./slider";

describe("Slider", () => {
  it("hides the reset button when the value equals the default", () => {
    render(
      <Slider
        label="Brightness"
        value={0}
        min={-100}
        max={100}
        defaultValue={0}
        resetLabel="Reset"
        onChange={vi.fn()}
        data-testid="s"
      />,
    );
    expect(screen.queryByRole("button", { name: "Reset" })).not.toBeInTheDocument();
  });

  it("shows reset when dirty and restores the default on click", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Slider
        label="Brightness"
        value={25}
        min={-100}
        max={100}
        defaultValue={0}
        resetLabel="Reset"
        onChange={onChange}
        data-testid="s"
      />,
    );
    const reset = screen.getByRole("button", { name: "Reset" });
    await user.click(reset);
    expect(onChange).toHaveBeenCalledWith(0);
  });

  it("emits the numeric value on range change", () => {
    const onChange = vi.fn();
    render(
      <Slider
        label="Brightness"
        value={0}
        min={-100}
        max={100}
        defaultValue={0}
        resetLabel="Reset"
        onChange={onChange}
        data-testid="s"
      />,
    );
    fireEvent.change(screen.getByTestId("s"), { target: { value: "42" } });
    expect(onChange).toHaveBeenCalledWith(42);
  });
});
