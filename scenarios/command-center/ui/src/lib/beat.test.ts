import { describe, expect, it } from "vitest";
import { readingsForBeat } from "./beat";

const readings = ["hero", "first", "second", "unassigned"].map((id) => ({ id }));

describe("beat reading groups", () => {
  it("renders only the authored group and always retains its hero", () => {
    expect(readingsForBeat({ hero: "hero", readingIds: ["second"] }, readings).map((reading) => reading.id)).toEqual(["hero", "second"]);
  });

  it("preserves registry order instead of authored group order", () => {
    expect(readingsForBeat({ hero: "hero", readingIds: ["second", "first"] }, readings).map((reading) => reading.id)).toEqual(["hero", "first", "second"]);
  });

  it("keeps legacy whole-room behavior when a beat has no group", () => {
    expect(readingsForBeat({ hero: "hero" }, readings)).toBe(readings);
  });
});
