import { describe, expect, it } from "vitest";

import { VERDICT_ICON, VERDICT_KEY, asVerdict, verdictTone } from "./verdicts";

describe("verdict vocabulary", () => {
  it("keeps the four evidence states", () => {
    expect(Object.keys(VERDICT_ICON).sort()).toEqual(["fail", "pass", "unmeasured", "warn"]);
  });

  it("never lets an unknown value read as a pass", () => {
    expect(asVerdict("pass")).toBe("pass");
    expect(asVerdict("info")).toBe("unmeasured");
    expect(asVerdict("")).toBe("unmeasured");
  });

  it("maps every verdict to a catalog key", () => {
    for (const [verdict, key] of Object.entries(VERDICT_KEY)) {
      expect(key).toBe(`workbench.verdict.${verdict}`);
    }
  });
});

describe("asVerdict", () => {
  it("translates every recorded evidence vocabulary", () => {
    expect(asVerdict("pass")).toBe("pass");
    expect(asVerdict("passed")).toBe("pass");
    expect(asVerdict("failed")).toBe("fail");
    expect(asVerdict("warned")).toBe("warn");
    expect(asVerdict("skipped")).toBe("unmeasured");
    expect(asVerdict("")).toBe("unmeasured");
    expect(asVerdict(undefined)).toBe("unmeasured");
  });

  it("maps verdicts to one tone", () => {
    expect(verdictTone("pass")).toBe("green");
    expect(verdictTone("warn")).toBe("amber");
    expect(verdictTone("fail")).toBe("red");
    expect(verdictTone("unmeasured")).toBe("ice");
  });
});
