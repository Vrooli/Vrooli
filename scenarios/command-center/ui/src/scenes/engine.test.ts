import { describe, expect, it } from "vitest";
import type { Reading } from "../lib/api";
import { drawGlow, focalPoint, freeBand, inQuiet, mulberry32, read, sceneData, type Frame } from "./engine";
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

describe("drawGlow", () => {
  const recording = () => {
    const calls: number[] = [];
    const ctx = { globalAlpha: 1, drawImage: (...args: unknown[]) => { calls.push(args.length); } } as unknown as CanvasRenderingContext2D;
    return { ctx, calls };
  };
  it("draws a positive-radius glow", () => {
    const { ctx, calls } = recording();
    drawGlow({ ctx } as Frame, 10, 20, 5, "#fff", 0.5);
    expect(calls).toHaveLength(1);
  });
  it("ignores a non-positive radius instead of throwing on the canvas", () => {
    const { ctx, calls } = recording();
    expect(() => drawGlow({ ctx } as Frame, 10, 20, -0.8, "#fff", 0.5)).not.toThrow();
    expect(calls).toHaveLength(0);
  });
});

describe("scene data", () => {
  const reading = (id: string, overrides: Partial<Reading>): Reading => makeReading({ id, label: id, ...overrides });
  it("carries the figure and its ink so the field is the data", () => {
    const data = sceneData([reading("a", { value: 58 }), reading("b", { coverage: "MISSING", trust: "UNAVAILABLE", sample: authoredSample(5) })]);
    expect(data.readings.a).toEqual({ value: 58, ink: "solid" });
    expect(data.readings.b).toEqual({ value: 5, ink: "dotted" });
    expect(read(data, "a")).toBe(58);
    expect(read(data, "missing")).toBeNull();
  });
  it("seeds deterministically so adjacent displays can be offset on purpose", () => {
    const a = mulberry32(42);
    const b = mulberry32(42);
    expect([a(), a(), a()]).toEqual([b(), b(), b()]);
  });
});
