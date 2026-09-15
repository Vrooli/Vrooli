import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { selectors } from "../consts/selectors";
import { SplitPane } from "./SplitPane";

describe("SplitPane", () => {
  it("renders primary and secondary slots", () => {
    render(
      <SplitPane
        primary={<div data-testid="primary-content">P</div>}
        secondary={<div data-testid="secondary-content">S</div>}
        handleLabel="resize"
      />,
    );
    expect(screen.getByTestId("primary-content")).toBeInTheDocument();
    expect(screen.getByTestId("secondary-content")).toBeInTheDocument();
  });

  it("renders an accessible separator with valuenow", () => {
    render(
      <SplitPane
        primary={<div data-testid="p">P</div>}
        secondary={<div data-testid="s">S</div>}
        handleLabel="resize"
        initialPercent={60}
      />,
    );
    const sep = screen.getByTestId(selectors.shared.splitPane.handle);
    expect(sep).toHaveAttribute("aria-valuenow", "60");
    expect(sep).toHaveAttribute("aria-orientation", "vertical");
  });

  it("keyboard adjusts the split percentage", () => {
    render(
      <SplitPane
        primary={<div data-testid="p">P</div>}
        secondary={<div data-testid="s">S</div>}
        handleLabel="resize"
        initialPercent={50}
      />,
    );
    const sep = screen.getByTestId(selectors.shared.splitPane.handle);
    fireEvent.keyDown(sep, { key: "ArrowRight" });
    expect(Number(sep.getAttribute("aria-valuenow"))).toBeGreaterThan(50);
  });
});
