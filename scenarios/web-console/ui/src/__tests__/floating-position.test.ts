import { describe, expect, it } from "vitest";
import { renderHook } from "@testing-library/react";
import { computeAnchoredFloatingPosition, useFloatingPosition } from "../hooks/useFloatingPosition";

describe("computeAnchoredFloatingPosition", () => {
  it("flips a submenu left when the preferred right side would overflow", () => {
    const position = computeAnchoredFloatingPosition({
      anchor: { left: 260, right: 300, top: 40, bottom: 72, width: 40, height: 32 },
      size: { width: 140, height: 120 },
      viewport: { width: 320, height: 480 },
      placements: ["right-start", "left-start", "bottom-start"],
      margin: 12,
      gap: 4,
    });

    expect(position.placement).toBe("left-start");
    expect(position.x).toBe(116);
    expect(position.y).toBe(40);
  });

  it("places top-end above the anchor, end-aligned", () => {
    const position = computeAnchoredFloatingPosition({
      anchor: { left: 200, right: 240, top: 400, bottom: 430, width: 40, height: 30 },
      size: { width: 100, height: 80 },
      viewport: { width: 320, height: 480 },
      placements: ["top-end"],
      margin: 12,
      gap: 4,
    });

    expect(position.placement).toBe("top-end");
    expect(position.x).toBe(140); // anchor.right - width
    expect(position.y).toBe(316); // anchor.top - height - gap
  });

  it("places bottom-end below the anchor, end-aligned, and falls back upward when it overflows", () => {
    const below = computeAnchoredFloatingPosition({
      anchor: { left: 200, right: 240, top: 40, bottom: 70, width: 40, height: 30 },
      size: { width: 100, height: 80 },
      viewport: { width: 320, height: 480 },
      placements: ["bottom-end", "top-end"],
      margin: 12,
      gap: 4,
    });
    expect(below.placement).toBe("bottom-end");
    expect(below.x).toBe(140);
    expect(below.y).toBe(74); // anchor.bottom + gap

    const flipped = computeAnchoredFloatingPosition({
      anchor: { left: 200, right: 240, top: 420, bottom: 450, width: 40, height: 30 },
      size: { width: 100, height: 80 },
      viewport: { width: 320, height: 480 },
      placements: ["bottom-end", "top-end"],
      margin: 12,
      gap: 4,
    });
    expect(flipped.placement).toBe("top-end");
  });

  it("clamps to viewport when no preferred placement fully fits", () => {
    const position = computeAnchoredFloatingPosition({
      anchor: { left: 280, right: 310, top: 180, bottom: 210, width: 30, height: 30 },
      size: { width: 280, height: 220 },
      viewport: { width: 320, height: 240 },
      placements: ["right-start"],
      margin: 12,
      gap: 4,
    });

    expect(position.placement).toBe("right-start");
    expect(position.x).toBe(28);
    expect(position.y).toBe(12);
  });
});


it("keeps persisted and dragged toolbar positions below reserved navigation after resizing", () => {
  const { result, rerender } = renderHook(({ topInset }) => useFloatingPosition({ topInset }), { initialProps: { topInset: 88 } });
  expect(result.current.clampPosition(1400, 12, { width: 220, height: 54 }, { width: 1440, height: 900 })).toEqual({ x: 1208, y: 100 });
  rerender({ topInset: 140 });
  expect(result.current.clampPosition(1400, 12, { width: 220, height: 54 }, { width: 800, height: 600 })).toEqual({ x: 568, y: 152 });
});
