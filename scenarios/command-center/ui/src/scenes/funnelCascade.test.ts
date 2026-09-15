import { describe, expect, it } from "vitest";
import { ladderSpan, visibleRungs } from "./funnelCascade";

const row = (value: number, detail: string) => ({ value, label: `rung-${value}`, detail });

describe("funnel cascade rungs", () => {
  it("starts one shipped rung below the next release and flags the next one", () => { // [REQ:CC-P1-016]
    const { rungs, above } = visibleRungs([row(1, "SHIPPED"), row(2, "SHIPPED"), row(3, "TRIGGER_MET"), row(4, "IDEA")], 7);
    expect(rungs.map((rung) => rung.rank)).toEqual([2, 3, 4]);
    expect(rungs.find((rung) => rung.next)?.label).toBe("rung-3");
    expect(above).toBe(0);
  });
  it("caps the window and counts what it leaves above", () => {
    const rows = Array.from({ length: 14 }, (_, index) => row(index + 1, "IDEA"));
    const { rungs, above } = visibleRungs(rows, 7);
    expect(rungs).toHaveLength(7);
    expect(rungs[0]?.next).toBe(true);
    expect(above).toBe(7);
  });
  it("draws nothing labelled when there are no rows", () => {
    expect(visibleRungs([])).toEqual({ rungs: [], above: 0 });
  });
});

describe("funnel cascade span", () => { // [REQ:CC-P1-017]
  it("keeps the rails, labels and the count above between the eyebrow and the bottom edge", () => {
    const cases: Array<[number, number, number]> = [[1280, 720, 120], [1920, 1080, 540], [1280, 720, 700], [3840, 2160, 300]];
    for (const [w, h, centerY] of cases) {
      const count = 7;
      const { gap, footY } = ladderSpan(count, centerY, w, h);
      const pad = Math.min(52, Math.max(16, w * 0.026));
      expect(footY - count * gap).toBeGreaterThanOrEqual(pad * 3.2 - 0.01);
      expect(footY + gap * 0.5).toBeLessThanOrEqual(h - pad * 1.6 + 0.01);
    }
  });
  it("centres on the focal band when the band has room", () => {
    const { gap, footY } = ladderSpan(3, 540, 1920, 1080);
    expect(footY - gap).toBeCloseTo(540);
  });
});
