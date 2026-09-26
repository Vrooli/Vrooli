import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { selectors } from "../../consts/selectors";
import { Tooltip } from "./tooltip";

describe("Tooltip", () => {
  it("hides the tooltip by default", () => {
    render(
      <Tooltip label="help">
        <button>trigger</button>
      </Tooltip>,
    );
    expect(screen.queryByRole("tooltip")).toBeNull();
  });

  it("shows the tooltip on hover and hides on leave", () => {
    render(
      <Tooltip label="help">
        <button data-testid="t-trigger">trigger</button>
      </Tooltip>,
    );
    const btn = screen.getByTestId("t-trigger");
    fireEvent.mouseEnter(btn);
    expect(screen.getByTestId(selectors.ui.tooltip.root)).toHaveTextContent("help");
    fireEvent.mouseLeave(btn);
    expect(screen.queryByTestId(selectors.ui.tooltip.root)).toBeNull();
  });

  it("shows the tooltip on focus and wires aria-describedby", () => {
    render(
      <Tooltip label="help">
        <button data-testid="t-trigger">trigger</button>
      </Tooltip>,
    );
    const btn = screen.getByTestId("t-trigger");
    fireEvent.focus(btn);
    const tip = screen.getByTestId(selectors.ui.tooltip.root);
    expect(btn.getAttribute("aria-describedby")).toBe(tip.id);
  });
});
