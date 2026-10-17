import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Toggle } from "./toggle";

describe("Toggle", () => {
  it("exposes a switch role reflecting the checked state", () => {
    render(<Toggle label="Lossless" checked onChange={vi.fn()} data-testid="t" />);
    const sw = screen.getByTestId("t");
    expect(sw).toHaveAttribute("role", "switch");
    expect(sw).toHaveAttribute("aria-checked", "true");
  });

  it("toggles the value on click", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Toggle label="Lossless" checked={false} onChange={onChange} data-testid="t" />);
    await user.click(screen.getByTestId("t"));
    expect(onChange).toHaveBeenCalledWith(true);
  });
});
