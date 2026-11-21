import { describe, expect, it } from "vitest";
import type { SceneGroup } from "./engine";
import { layoutAtlas, skyField, spanningEdges } from "./panoramaConstellation";

const group = (id: string, starIds: string[]): SceneGroup => ({ id, title: id, stars: starIds.map((starId) => ({ id: starId, state: "measured", cached: false })) });
const board = [group("mission-control", ["a", "b", "c"]), group("hive", ["d", "e"]), group("forge", ["f", "g", "h", "i"]), group("ledger", ["j"]), group("broadcast", ["k", "l", "m", "n", "o"])];

describe("sky field", () => {
  it("sits beside a landscape hero, above the readings and below the room header", () => {
    const hero = { x: 40, y: 280, w: 560, h: 360 };
    const readings = { x: 40, y: 780, w: 1520, h: 180 };
    const field = skyField({ w: 1600, h: 1000, quiet: [hero, readings] });
    expect(field.x).toBeGreaterThan(hero.x + hero.w);
    expect(field.x + field.w).toBeLessThanOrEqual(1600);
    expect(field.y).toBeGreaterThanOrEqual(120);
    expect(field.y + field.h).toBeLessThan(readings.y);
  });
  it("uses the band between hero and readings in portrait", () => {
    const field = skyField({ w: 390, h: 844, quiet: [{ x: 16, y: 100, w: 358, h: 220 }, { x: 16, y: 560, w: 358, h: 260 }] });
    expect(field.y).toBeGreaterThan(320);
    expect(field.y + field.h).toBeLessThan(560);
  });
});

describe("atlas layout", () => {
  const field = { x: 700, y: 120, w: 840, h: 620 };
  it("draws one constellation per room with every star inside the field", () => {
    const figures = layoutAtlas(board, field, 40);
    expect(figures.map((figure) => figure.id)).toEqual(board.map((entry) => entry.id));
    for (const figure of figures) {
      expect(figure.edges).toHaveLength(Math.max(0, figure.stars.length - 1));
      for (const star of figure.stars) {
        expect(star.x).toBeGreaterThanOrEqual(field.x);
        expect(star.x).toBeLessThanOrEqual(field.x + field.w);
        expect(star.y).toBeGreaterThanOrEqual(field.y);
        expect(star.y).toBeLessThanOrEqual(field.y + field.h);
      }
    }
  });
  it("is deterministic, and a new signal leaves existing stars where they were", () => {
    const before = layoutAtlas(board, field, 40);
    expect(layoutAtlas(board, field, 40)).toEqual(before);
    const grown = board.map((entry) => (entry.id === "hive" ? group("hive", ["d", "e", "z"]) : entry));
    const after = layoutAtlas(grown, field, 40);
    expect(after[1]?.stars.slice(0, 2)).toEqual(before[1]?.stars);
    expect(after[1]?.stars).toHaveLength(3);
  });
});

describe("constellation lines", () => {
  it("joins every star with the shortest set of lines", () => {
    const edges = spanningEdges([{ x: 0, y: 0 }, { x: 10, y: 0 }, { x: 100, y: 0 }, { x: 11, y: 1 }]);
    expect(edges).toHaveLength(3);
    expect(edges).toContainEqual([1, 3]);
    expect(edges).not.toContainEqual([0, 2]);
  });
});
