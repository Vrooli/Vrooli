import { describe, expect, it } from "vitest";
import { boardTransitionDuration } from "./boardTransition";

describe("boardTransitionDuration", () => {
  it("uses a zero-duration cut and the existing crossfade rhythm", () => {
    expect(boardTransitionDuration("cut")).toBe(0);
    expect(boardTransitionDuration("crossfade")).toBe(900);
  });
});
