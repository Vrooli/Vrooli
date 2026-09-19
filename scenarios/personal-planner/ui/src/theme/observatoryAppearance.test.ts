import { describe, expect, it } from "vitest";
import { appearanceAt, parseClock, readTransitionMinutes } from "./observatoryAppearance";

describe("observatory appearance transitions", () => {
  it("uses configured local-time boundaries", () => {
    expect(appearanceAt(new Date(2026, 8, 19, 6, 59), 7 * 60, 19 * 60)).toBe("night");
    expect(appearanceAt(new Date(2026, 8, 19, 7, 0), 7 * 60, 19 * 60)).toBe("day");
    expect(appearanceAt(new Date(2026, 8, 19, 19, 0), 7 * 60, 19 * 60)).toBe("night");
  });

  it("supports a daylight window that crosses midnight", () => {
    expect(appearanceAt(new Date(2026, 8, 19, 23, 0), 18 * 60, 6 * 60)).toBe("day");
    expect(appearanceAt(new Date(2026, 8, 19, 12, 0), 18 * 60, 6 * 60)).toBe("night");
  });

  it("falls back safely for malformed settings", () => {
    expect(parseClock("bad", 420)).toBe(420);
    expect(parseClock("25:00", 420)).toBe(420);
    expect(parseClock("07:30", 420)).toBe(450);
    expect(parseClock("07:99", 420)).toBe(420);
  });

  it("reads persisted transition settings without coupling the resolver to the UI", () => {
    const storage = window.localStorage;
    storage.setItem("planner.auto-day-start", "06:45");
    storage.setItem("planner.auto-night-start", "20:15");
    expect(readTransitionMinutes(storage)).toEqual({ dayStart: 405, nightStart: 1215 });
    storage.clear();
    expect(readTransitionMinutes(storage)).toEqual({ dayStart: 420, nightStart: 1140 });
  });
});
