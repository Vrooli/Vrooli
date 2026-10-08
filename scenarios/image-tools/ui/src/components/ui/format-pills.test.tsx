import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { FormatPills } from "./format-pills";

describe("FormatPills", () => {
  it("renders uppercased format tokens and marks the active one", () => {
    render(
      <FormatPills
        label="Format"
        value="webp"
        options={["png", "webp"]}
        onChange={vi.fn()}
      />,
    );
    expect(screen.getByRole("radio", { name: "WEBP" })).toHaveAttribute(
      "aria-checked",
      "true",
    );
    expect(screen.getByRole("radio", { name: "PNG" })).toHaveAttribute(
      "aria-checked",
      "false",
    );
  });

  it("emits the lowercase token on click", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <FormatPills label="Format" value="png" options={["png", "webp"]} onChange={onChange} />,
    );
    await user.click(screen.getByRole("radio", { name: "WEBP" }));
    expect(onChange).toHaveBeenCalledWith("webp");
  });
});
