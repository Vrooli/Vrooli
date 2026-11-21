import { describe, expect, it } from "vitest";

import { VERDICT_ICON, VERDICT_KEY, asVerdict } from "./verdicts";

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
