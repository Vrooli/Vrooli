import { describe, expect, it } from "vitest";

import GestureTokens, {
  resolveGestureFeel,
} from "../../../../components/GestureTokens/versions/1.0.1/GestureTokens.tsx";

describe("GestureTokens", () => {
  it("publishes the complete shared feel vocabulary", () => {
    expect(GestureTokens).toEqual({
      axisSlop: 8,
      flickVelocity: 0.5,
      resistance: 0.32,
      dismissThreshold: 96,
      hoverOpenDelay: 280,
      hoverCloseDelay: 100,
      safePolygonFuse: 300,
    });
    expect(Object.isFrozen(GestureTokens)).toBe(true);
  });

  it("merges an override without dropping sibling tokens", () => {
    expect(resolveGestureFeel({ axisSlop: 12 })).toEqual({
      ...GestureTokens,
      axisSlop: 12,
    });
  });
});
