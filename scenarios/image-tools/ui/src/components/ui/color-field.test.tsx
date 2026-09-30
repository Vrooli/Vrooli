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

  it("hides the Clear button and disables the alpha slider when value is empty", () => {
    render(
      <ColorField
        label="Background"
        value=""
        onChange={vi.fn()}
        clearLabel="Clear"
        alphaLabel="Opacity"
        data-testid="c"
      />,
    );
    // hasColor === false → no Clear button, alpha slider disabled and pinned to 100.
    expect(screen.queryByRole("button", { name: "Clear" })).toBeNull();
    const slider = screen.getByRole("slider", { name: "Opacity" });
    expect(slider).toBeDisabled();
    expect(slider).toHaveValue("100");
  });

  it("shows the Clear button and reflects the parsed alpha when a color is set", () => {
    render(
      <ColorField
        label="Background"
        value="#ff000080"
        onChange={vi.fn()}
        clearLabel="Clear"
        alphaLabel="Opacity"
        data-testid="c"
      />,
    );
    // hasColor === true → Clear shown, slider enabled at the parsed alpha (50%).
    expect(screen.getByRole("button", { name: "Clear" })).toBeInTheDocument();
    const slider = screen.getByRole("slider", { name: "Opacity" });
    expect(slider).toBeEnabled();
    expect(slider).toHaveValue("50");
  });

  it("emits an 8-digit hex when the alpha slider drops below full opacity", () => {
    const onChange = vi.fn();
    render(
      <ColorField
        label="Background"
        value="#112233"
        onChange={onChange}
        clearLabel="Clear"
        alphaLabel="Opacity"
        data-testid="c"
      />,
    );
    fireEvent.input(screen.getByRole("slider", { name: "Opacity" }), {
      target: { value: "50" },
    });
    expect(onChange).toHaveBeenCalledWith("#11223380");
  });

  it("emits the picked base color at the current alpha on swatch change", () => {
    const onChange = vi.fn();
    render(
      <ColorField
        label="Background"
        value="#112233ff"
        onChange={onChange}
        clearLabel="Clear"
        alphaLabel="Opacity"
        data-testid="c"
      />,
    );
    // alpha parses to 100 → composeColor returns the bare 6-digit hex.
    fireEvent.input(screen.getByTestId("c"), { target: { value: "#445566" } });
    expect(onChange).toHaveBeenCalledWith("#445566");
  });
});
