import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { PositionPicker } from "./position-picker";

describe("PositionPicker", () => {
  it("renders nine cells and marks the active token", () => {
    render(
      <PositionPicker
        label="Gravity"
        value="center"
        onChange={vi.fn()}
        cellLabel={(t) => t}
      />,
    );
    const cells = screen.getAllByRole("radio");
    expect(cells).toHaveLength(9);
    expect(screen.getByRole("radio", { name: "center" })).toHaveAttribute(
      "aria-checked",
      "true",
    );
  });

  it("emits the gravity token on click", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <PositionPicker
        label="Gravity"
        value=""
        onChange={onChange}
        cellLabel={(t) => t}
      />,
    );
    await user.click(screen.getByRole("radio", { name: "bottom-right" }));
    expect(onChange).toHaveBeenCalledWith("bottom-right");
  });
});
