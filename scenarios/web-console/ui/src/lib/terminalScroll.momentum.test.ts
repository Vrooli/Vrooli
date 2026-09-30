import { describe, expect, it } from "vitest";
import { decayTouchScrollVelocity, touchScrollVelocity } from "./terminalScroll";

function simulatedFlick(frameMs: number): number {
  let velocity = touchScrollVelocity(-240, 16);
  let distance = 0;
  for (let elapsed = 0; Math.abs(velocity) >= 0.05 && elapsed < 3000; elapsed += frameMs) {
    velocity = decayTouchScrollVelocity(velocity, frameMs);
    distance += velocity * (frameMs / 16);
  }
  return Math.abs(distance);
}

describe("elapsed-time momentum", () => {
  it("keeps a fixed flick within ten percent at 60 Hz and 120 Hz", () => {
    const at60Hz = simulatedFlick(1000 / 60);
    const at120Hz = simulatedFlick(1000 / 120);
    expect(Math.abs(at60Hz - at120Hz) / Math.max(at60Hz, at120Hz)).toBeLessThan(0.1);
  });

  it("uses measured elapsed time for velocity samples", () => {
    expect(touchScrollVelocity(16, 16)).toBe(16);
    expect(touchScrollVelocity(16, 8)).toBe(32);
    expect(touchScrollVelocity(16, 0)).toBe(0);
  });
});
