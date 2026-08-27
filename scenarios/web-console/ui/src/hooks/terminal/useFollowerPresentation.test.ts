import { describe, expect, it } from "vitest";
import { renderHook } from "@testing-library/react";
import { createTerminalStub } from "../../test-utils";
import { FALLBACK_CELL_ASPECT, FRAME_ASPECT } from "../../lib/followerViewport";
import { measureCellAspect, useFollowerPresentation } from "./useFollowerPresentation";

const PANE = { width: 1000, height: 700 };

type PresentationOptions = Parameters<typeof useFollowerPresentation>[0];

// `??` would swallow an explicitly-null serverSize, which is one of the cases
// under test, so defaults are applied by key presence instead.
function present(overrides: Partial<PresentationOptions> = {}) {
  const options: PresentationOptions = {
    terminal: createTerminalStub({ cols: 80, rows: 24, screen: { width: 800, height: 480 } }),
    serverSize: { cols: 80, rows: 24 },
    isFollower: true,
    paneSize: PANE,
    ...overrides,
  };
  const { result } = renderHook(() => useFollowerPresentation(options));
  return result.current;
}

describe("measureCellAspect", () => {
  it("derives the cell aspect from the rendered screen", () => {
    const terminal = createTerminalStub({ cols: 80, rows: 24, screen: { width: 800, height: 480 } });
    // 800/80 = 10 wide, 480/24 = 20 tall.
    expect(measureCellAspect(terminal)).toBeCloseTo(0.5, 6);
  });

  it("falls back when xterm has not rendered yet", () => {
    expect(measureCellAspect(null)).toBe(FALLBACK_CELL_ASPECT);
    expect(measureCellAspect(createTerminalStub({ screen: null }))).toBe(FALLBACK_CELL_ASPECT);
    expect(measureCellAspect(createTerminalStub({ screen: { width: 0, height: 0 } }))).toBe(FALLBACK_CELL_ASPECT);
    expect(measureCellAspect(createTerminalStub({ cols: 0, rows: 0, screen: { width: 10, height: 10 } }))).toBe(FALLBACK_CELL_ASPECT);
  });
});

describe("useFollowerPresentation", () => {
  it("produces no frame for a leader, an unknown size, or an unmeasured pane", () => {
    expect(present({ isFollower: false })).toBeNull();
    expect(present({ serverSize: null })).toBeNull();
    expect(present({ paneSize: { width: 0, height: 0 } })).toBeNull();
  });

  it("frames the leader's declared device, not the grid it is showing", () => {
    const frame = present({ leaderClass: "phone", serverSize: { cols: 120, rows: 30 } });
    expect(frame?.archetype).toBe("phone");
    expect(frame && frame.rect.width / frame.rect.height).toBeCloseTo(FRAME_ASPECT.phone, 5);
  });

  // The regression this whole change exists to prevent.
  it("keeps the silhouette identical when a keyboard halves the leader's rows", () => {
    const closed = present({ leaderClass: "phone", serverSize: { cols: 46, rows: 26 } });
    const open = present({ leaderClass: "phone", serverSize: { cols: 46, rows: 13 }, leaderKbOpen: true });
    expect(open?.archetype).toBe(closed?.archetype);
    expect(open?.tier).toBe(closed?.tier);
    expect(open?.rect.width).toBeCloseTo(closed?.rect.width ?? 0, 5);
    expect(open?.rect.height).toBeCloseTo(closed?.rect.height ?? 0, 5);
    // Only the grid inside it moves, and the keyboard claims a believable
    // slice of the screen rather than whatever the grid left over.
    expect(closed?.keyboardShare).toBe(0);
    expect(open?.keyboardShare).toBeGreaterThan(0.15);
    expect(open?.keyboardShare).toBeLessThan(0.35);
    expect(open?.kbOpen).toBe(true);
    expect(closed?.kbOpen).toBe(false);
  });

  it("falls back to grid geometry when the leader declares no class", () => {
    expect(present({ serverSize: { cols: 240, rows: 30 } })?.archetype).toBe("ultrawide");
    expect(present({ leaderClass: "", serverSize: { cols: 45, rows: 30 } })?.archetype).toBe("phone");
  });

  it("ignores a class it does not recognise", () => {
    const frame = present({ leaderClass: "smartwatch", serverSize: { cols: 240, rows: 30 } });
    expect(frame?.archetype).toBe("ultrawide");
  });

  it("computes without touching xterm, so rendering owns every mutation", () => {
    const terminal = createTerminalStub({ cols: 40, rows: 12, screen: { width: 400, height: 240 } });
    const { rerender } = renderHook(({ paneSize }) => useFollowerPresentation({
      terminal, serverSize: { cols: 80, rows: 24 }, isFollower: true, paneSize,
    }), { initialProps: { paneSize: PANE } });
    rerender({ paneSize: { width: 500, height: 400 } });
    expect(terminal.element?.style.cssText).toBe("");
    expect(terminal.resize).not.toHaveBeenCalled();
    expect(terminal.options.fontSize).toBeUndefined();
  });

  it("degrades to the caption strip in a pane too small for a silhouette", () => {
    const frame = present({ leaderClass: "phone", paneSize: { width: 120, height: 120 } });
    expect(frame?.tier).toBe("strip");
  });
});
