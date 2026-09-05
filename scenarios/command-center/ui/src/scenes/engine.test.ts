import { describe, expect, it } from "vitest";
import type { Reading } from "../lib/api";
import { focalPoint, freeBand, inQuiet, mulberry32, read, sceneData, type Frame } from "./engine";
import { authoredSample, makeReading } from "../test-utils/readings";

const frame = (w: number, h: number, quiet: Frame["quiet"]): Frame => ({ w, h, quiet } as Frame);

describe("quiet zones", () => {
  it("keeps the composition in the band to the right of a landscape hero", () => {
    const focal = focalPoint(frame(1600, 1000, [{ x: 40, y: 300, w: 500, h: 350 }, { x: 40, y: 800, w: 1520, h: 160 }]));
    expect(focal.x).toBeGreaterThan(900);
    expect(focal.y).toBeLessThan(800);
  });
  it("uses the band between the hero and the readings in portrait", () => {
    const focal = focalPoint(frame(390, 844, [{ x: 16, y: 100, w: 358, h: 220 }, { x: 16, y: 520, w: 358, h: 300 }]));
    expect(focal.x).toBe(195);
    expect(focal.y).toBeGreaterThan(320);
    expect(focal.y).toBeLessThan(520);
  });
  it("reports the largest free band", () => {
    expect(freeBand(1000, [{ x: 0, y: 0, w: 10, h: 100 }, { x: 0, y: 700, w: 10, h: 300 }])).toMatchObject({ top: 100, bottom: 700, size: 600 });
  });
  it("answers membership with padding", () => {
    const quiet = [{ x: 100, y: 100, w: 50, h: 50 }];
    expect(inQuiet(quiet, 125, 125)).toBe(true);
    expect(inQuiet(quiet, 160, 125)).toBe(false);
    expect(inQuiet(quiet, 160, 125, 20)).toBe(true);
  });
});

describe("scene data", () => {
  const reading = (id: string, overrides: Partial<Reading>): Reading => makeReading({ id, label: id, ...overrides });
  it("carries the figure and its ink so the field is the data", () => {
    const data = sceneData([reading("a", { value: 58 }), reading("b", { coverage: "MISSING", trust: "UNAVAILABLE", sample: authoredSample(5) })]);
    expect(data.readings.a).toEqual({ value: 58, ink: "solid" });
    expect(data.readings.b).toEqual({ value: 5, ink: "dotted" });
    expect(read(data, "a", 0)).toBe(58);
    expect(read(data, "missing", 7)).toBe(7);
  });
  it("seeds deterministically so adjacent displays can be offset on purpose", () => {
    const a = mulberry32(42);
    const b = mulberry32(42);
    expect([a(), a(), a()]).toEqual([b(), b(), b()]);
  });
});
