/**
 * CropOverlay tests. The geometry lives in cropMath (unit-tested separately);
 * here we pin the keyboard fallback (arrow-nudge), the aspect-preset snap, and
 * the rendered handles — the pointer plumbing is exercised by the BAS journey.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { CropOverlay } from "./CropOverlay";

const natural = { width: 200, height: 100 };
const client = { width: 200, height: 100 };

describe("CropOverlay", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => cleanup());

  it("renders the box and four corner handles", () => {
    renderWithProviders(
      <CropOverlay
        natural={natural}
        client={client}
        rect={{ x: 10, y: 10, width: 80, height: 40 }}
        onChange={vi.fn()}
      />,
    );
    expect(screen.getByTestId(selectors.workspace.crop.box)).toBeInTheDocument();
    for (const corner of ["nw", "ne", "sw", "se"] as const) {
      expect(
        screen.getByTestId(selectors.workspace.crop.handle({ corner })),
      ).toBeInTheDocument();
    }
  });

  it("nudges the box right with the ArrowRight key", () => {
    const onChange = vi.fn();
    renderWithProviders(
      <CropOverlay
        natural={natural}
        client={client}
        rect={{ x: 10, y: 10, width: 80, height: 40 }}
        onChange={onChange}
      />,
    );
    fireEvent.keyDown(screen.getByTestId(selectors.workspace.crop.box), { key: "ArrowRight" });
    expect(onChange).toHaveBeenCalledWith({ x: 11, y: 10, width: 80, height: 40 });
  });

  it("snaps to a 1:1 ratio when the square aspect preset is chosen", () => {
    const onChange = vi.fn();
    const { container } = renderWithProviders(
      <CropOverlay
        natural={{ width: 400, height: 400 }}
        client={{ width: 400, height: 400 }}
        rect={{ x: 0, y: 0, width: 120, height: 200 }}
        onChange={onChange}
      />,
    );
    // The "1:1" pill lives inside the aspect SegmentedControl.
    const aspect = screen.getByTestId(selectors.workspace.crop.aspect);
    const squarePill = [...aspect.querySelectorAll('[role="radio"]')].find(
      (el) => el.textContent === "1:1",
    );
    expect(squarePill).toBeDefined();
    fireEvent.click(squarePill as Element);
    expect(onChange).toHaveBeenCalled();
    const last = onChange.mock.calls.at(-1)?.[0] as { width: number; height: number };
    expect(last.width).toBe(last.height);
    expect(container).toBeTruthy();
  });
});
