import { describe, expect, it } from "vitest";
import { archetypeForGrid, orientationForGrid } from "./deviceArchetype";
describe("device archetypes", () => {
  it("derives frames only from the grid", () => {
    expect(archetypeForGrid(45, 30, 0.5)).toBe("phone");
    expect(orientationForGrid(45, 30, 0.5)).toBe("portrait");
    expect(archetypeForGrid(100, 24, 0.5)).toBe("laptop");
    expect(archetypeForGrid(240, 30, 0.5)).toBe("ultrawide");
  });
});
