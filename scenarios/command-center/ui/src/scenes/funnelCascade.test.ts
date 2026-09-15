import { describe, expect, it } from "vitest";
import { visibleRungs } from "./funnelCascade";

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
