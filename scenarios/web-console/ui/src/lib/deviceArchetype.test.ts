import { describe, expect, it } from "vitest";
import { DEVICE_ARCHETYPES, archetypeForGrid, isDeviceArchetype, resolveArchetype } from "./deviceArchetype";

describe("resolveArchetype", () => {
  // The invariant the previous golden-value test could not express, and the
  // regression it therefore never caught: a leader's virtual keyboard halves
  // its row count, which used to reclassify a phone as a laptop mid-session.
  it("does not change when only the row count changes", () => {
    const declaredClass = "phone";
    const keyboardClosed = resolveArchetype({ declaredClass, cols: 46, rows: 26, cellAspect: 0.5 });
    const keyboardOpen = resolveArchetype({ declaredClass, cols: 46, rows: 13, cellAspect: 0.5 });
    expect(keyboardOpen).toBe(keyboardClosed);
    expect(keyboardOpen).toBe("phone");
  });

  it("holds every declared class steady across a halved grid", () => {
    for (const declaredClass of DEVICE_ARCHETYPES) {
      const wide = resolveArchetype({ declaredClass, cols: 120, rows: 40, cellAspect: 0.5 });
      const short = resolveArchetype({ declaredClass, cols: 120, rows: 12, cellAspect: 0.5 });
      expect(short).toBe(declaredClass);
      expect(wide).toBe(declaredClass);
    }
  });

  it("ignores a class it does not recognise rather than trusting it", () => {
    // The class is self-declared by the leader, so an unknown value is either
    // an older client or a malformed one. Neither may reach the frame table.
    const resolved = resolveArchetype({ declaredClass: "smartwatch", cols: 46, rows: 26, cellAspect: 0.5 });
    expect(DEVICE_ARCHETYPES).toContain(resolved);
    expect(resolved).toBe(archetypeForGrid(46, 26, 0.5));
  });

  it("falls back to grid geometry when the leader declares nothing", () => {
    expect(resolveArchetype({ cols: 45, rows: 30, cellAspect: 0.5 })).toBe("phone");
    expect(resolveArchetype({ declaredClass: "", cols: 240, rows: 30, cellAspect: 0.5 })).toBe("ultrawide");
  });
});

describe("archetypeForGrid", () => {
  // Retained as the fallback contract for leaders that declare no class. It is
  // explicitly not the path a modern leader takes.
  it.each([
    [45, 30, "phone"],
    [100, 38, "tablet"],
    [100, 24, "laptop"],
    [150, 30, "monitor"],
    [240, 30, "ultrawide"],
  ] as const)("classifies %ix%i as %s", (cols, rows, expected) => {
    expect(archetypeForGrid(cols, rows, 0.5)).toBe(expected);
  });

  it("only ever returns a known archetype", () => {
    for (const [cols, rows] of [[1, 200], [200, 1], [80, 24], [4, 4]] as const) {
      expect(isDeviceArchetype(archetypeForGrid(cols, rows, 0.5))).toBe(true);
    }
  });
});
