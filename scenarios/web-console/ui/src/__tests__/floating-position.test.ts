import { describe, expect, it } from "vitest";
import { computeAnchoredFloatingPosition } from "../hooks/useFloatingPosition";

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
