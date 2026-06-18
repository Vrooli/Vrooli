import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ColorField } from "./color-field";
import { composeColor, parseColor } from "./colorMath";

describe("color helpers", () => {
  it("parses 8-digit hex into base + alpha percent", () => {
    expect(parseColor("#ff000080")).toEqual({ base: "#ff0000", alpha: 50 });
  });

  it("treats 6-digit hex as full opacity", () => {
    expect(parseColor("#00ff00")).toEqual({ base: "#00ff00", alpha: 100 });
  });

  it("composes full opacity as a 6-digit hex and partial as 8-digit", () => {
    expect(composeColor("#112233", 100)).toBe("#112233");
    expect(composeColor("#112233", 50)).toBe("#11223380");
  });
});

describe("ColorField", () => {
  it("emits a 6-digit hex when the color picker changes at full opacity", () => {
    const onChange = vi.fn();
    render(
      <ColorField
        label="Background"
        value=""
        onChange={onChange}
        clearLabel="Clear"
        alphaLabel="Opacity"
        data-testid="c"
      />,
    );
    fireEvent.input(screen.getByTestId("c"), { target: { value: "#abcdef" } });
    expect(onChange).toHaveBeenCalledWith("#abcdef");
  });

  it("clears to an empty string", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <ColorField
        label="Background"
        value="#abcdef"
        onChange={onChange}
        clearLabel="Clear"
        alphaLabel="Opacity"
        data-testid="c"
      />,
    );
    await user.click(screen.getByRole("button", { name: "Clear" }));
    expect(onChange).toHaveBeenCalledWith("");
  });
});
