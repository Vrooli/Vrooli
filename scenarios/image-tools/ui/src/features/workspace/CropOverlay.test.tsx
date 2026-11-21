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

// jsdom ships no PointerEvent constructor, so @testing-library's
// fireEvent.pointer* falls back to a bare Event that drops clientX/pointerId.
// Polyfill a minimal PointerEvent (extending MouseEvent, which carries
// clientX/clientY) so the drag plumbing sees real coordinates + a pointer id.
class PointerEventPolyfill extends MouseEvent {
  pointerId: number;
  constructor(type: string, init: PointerEventInit = {}) {
    super(type, init);
    this.pointerId = init.pointerId ?? 0;
  }
}
if (typeof window.PointerEvent === "undefined") {
  (window as unknown as { PointerEvent: typeof PointerEvent }).PointerEvent =
    PointerEventPolyfill as unknown as typeof PointerEvent;
}

/**
 * jsdom implements neither pointer capture nor layout. Stub both so the drag
 * plumbing (which captures the pointer and reads the parent's box) runs without
 * throwing. The bounding rect is the overlay container at the origin, so a
 * display point equals an image point at 1:1 scale (natural === client).
 */
type PointerStubs = ReturnType<typeof buildPointerStubs>;
const activeStubs: PointerStubs[] = [];

const installPointerStubs = (): PointerStubs => {
  const stubs = buildPointerStubs();
  activeStubs.push(stubs);
  return stubs;
};

const buildPointerStubs = () => {
  const captured: number[] = [];
  const released: number[] = [];
  // jsdom does not implement pointer capture at all (the methods are absent),
  // so vi.spyOn can't wrap them — assign directly and restore in teardown.
  const proto = HTMLElement.prototype as unknown as {
    setPointerCapture?: (id: number) => void;
    releasePointerCapture?: (id: number) => void;
  };
  const priorSet = proto.setPointerCapture;
  const priorRelease = proto.releasePointerCapture;
  proto.setPointerCapture = (id: number) => {
    captured.push(id);
  };
  proto.releasePointerCapture = (id: number) => {
    released.push(id);
  };
  const rect = vi
    .spyOn(HTMLElement.prototype, "getBoundingClientRect")
    .mockReturnValue({
      x: 0,
      y: 0,
      top: 0,
      left: 0,
      right: 200,
      bottom: 100,
      width: 200,
      height: 100,
      toJSON: () => ({}),
    });
  const setCapture = { mockRestore: () => (proto.setPointerCapture = priorSet) };
  const releaseCapture = {
    mockRestore: () => (proto.releasePointerCapture = priorRelease),
  };
  return { captured, released, setCapture, releaseCapture, rect };
};

describe("CropOverlay", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    // Restore only the prototype spies this file installs; the process-wide
    // canvas getContext stub in test-setup must survive (it is deliberately
    // never restored), so we never call vi.restoreAllMocks() here.
    activeStubs.forEach((stub) => {
      stub.setCapture.mockRestore();
      stub.releaseCapture.mockRestore();
      stub.rect.mockRestore();
    });
    activeStubs.length = 0;
  });

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

  it("ignores arrow keys that are not directional nudges", () => {
    const onChange = vi.fn();
    renderWithProviders(
      <CropOverlay
        natural={natural}
        client={client}
        rect={{ x: 10, y: 10, width: 80, height: 40 }}
        onChange={onChange}
      />,
    );
    fireEvent.keyDown(screen.getByTestId(selectors.workspace.crop.box), { key: "Enter" });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("nudges down, left and up for the other arrow keys", () => {
    const onChange = vi.fn();
    renderWithProviders(
      <CropOverlay
        natural={natural}
        client={client}
        rect={{ x: 10, y: 10, width: 80, height: 40 }}
        onChange={onChange}
      />,
    );
    const box = screen.getByTestId(selectors.workspace.crop.box);
    fireEvent.keyDown(box, { key: "ArrowDown" });
    expect(onChange).toHaveBeenLastCalledWith({ x: 10, y: 11, width: 80, height: 40 });
    fireEvent.keyDown(box, { key: "ArrowLeft" });
    expect(onChange).toHaveBeenLastCalledWith({ x: 9, y: 10, width: 80, height: 40 });
    fireEvent.keyDown(box, { key: "ArrowUp" });
    expect(onChange).toHaveBeenLastCalledWith({ x: 10, y: 9, width: 80, height: 40 });
  });

  it("drags the box body: capture, a window pointermove commits, pointerup releases", () => {
    const stubs = installPointerStubs();
    const onChange = vi.fn();
    renderWithProviders(
      <CropOverlay
        natural={natural}
        client={client}
        rect={{ x: 20, y: 20, width: 60, height: 30 }}
        onChange={onChange}
      />,
    );
    const box = screen.getByTestId(selectors.workspace.crop.box);

    // Grab at (20,20) — the box origin — so the drag delta is the pointer delta.
    fireEvent.pointerDown(box, { pointerId: 7, clientX: 20, clientY: 20 });
    expect(stubs.captured).toContain(7);

    // Move 10px right / 5px down (1:1 scale → same image delta).
    fireEvent.pointerMove(window, { pointerId: 7, clientX: 30, clientY: 25 });
    expect(onChange).toHaveBeenCalled();
    const moved = onChange.mock.calls.at(-1)?.[0] as { x: number; y: number };
    expect(moved.x).toBe(30);
    expect(moved.y).toBe(25);

    fireEvent.pointerUp(window, { pointerId: 7 });
    expect(stubs.released).toContain(7);

    // After release the window listener is gone — a further move is a no-op.
    onChange.mockClear();
    fireEvent.pointerMove(window, { pointerId: 7, clientX: 60, clientY: 60 });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("resizes from each corner handle, mutating the matching edges", () => {
    const stubs = installPointerStubs();
    for (const corner of ["nw", "ne", "sw", "se"] as const) {
      const onChange = vi.fn();
      const { unmount } = renderWithProviders(
        <CropOverlay
          natural={natural}
          client={client}
          rect={{ x: 40, y: 30, width: 60, height: 30 }}
          onChange={onChange}
        />,
      );
      const handle = screen.getByTestId(selectors.workspace.crop.handle({ corner }));
      fireEvent.pointerDown(handle, { pointerId: 3, clientX: 40, clientY: 30 });
      fireEvent.pointerMove(window, { pointerId: 3, clientX: 55, clientY: 45 });
      expect(onChange).toHaveBeenCalled();
      const next = onChange.mock.calls.at(-1)?.[0] as {
        x: number;
        y: number;
        width: number;
        height: number;
      };
      // Every corner produces a valid (clamped, ≥1) rect inside the image.
      expect(next.width).toBeGreaterThanOrEqual(1);
      expect(next.height).toBeGreaterThanOrEqual(1);
      expect(next.x).toBeGreaterThanOrEqual(0);
      expect(next.y).toBeGreaterThanOrEqual(0);
      fireEvent.pointerUp(window, { pointerId: 3 });
      unmount();
    }
    expect(stubs.captured.length).toBeGreaterThan(0);
  });

  it("snaps using the image's own ratio for the 'original' aspect preset", () => {
    installPointerStubs();
    const onChange = vi.fn();
    renderWithProviders(
      <CropOverlay
        natural={{ width: 400, height: 200 }}
        client={{ width: 400, height: 200 }}
        rect={{ x: 0, y: 0, width: 120, height: 200 }}
        onChange={onChange}
      />,
    );
    const aspect = screen.getByTestId(selectors.workspace.crop.aspect);
    const original = [...aspect.querySelectorAll('[role="radio"]')].at(-1);
    fireEvent.click(original as Element);
    expect(onChange).toHaveBeenCalled();
    const shaped = onChange.mock.calls.at(-1)?.[0] as { width: number; height: number };
    // 400:200 = 2:1 → snapped height is half the width.
    expect(shaped.width / shaped.height).toBeCloseTo(2, 1);
  });
});
