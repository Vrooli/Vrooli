import { describe, expect, it } from "vitest";
import { ledgerLayout } from "./ledgerRiver";

describe("ledger waterline layout", () => {
  it("places the revenue level below burn until the measured values cross", () => {
    const frame = {
      w: 1200,
      h: 700,
      quiet: [],
      data: { readings: { offer_posture: { value: null, ink: "solid", meta: { burnMinor: 1000, revenueMinor: 500 } } }, order: ["offer_posture"] },
    } as any;
    const below = ledgerLayout(frame);
    expect(below.revenueY).toBeGreaterThan(below.burnY);
    expect(below.crossing).toBe(false);
    frame.data.readings.offer_posture.meta.revenueMinor = 1200;
    const above = ledgerLayout(frame);
    expect(above.revenueY).toBeLessThanOrEqual(above.burnY);
    expect(above.crossing).toBe(true);
  });
});
