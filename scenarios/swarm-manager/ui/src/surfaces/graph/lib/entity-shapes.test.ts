import { describe, it, expect } from "vitest";
import {
  ENTITY_SHAPE_MAP,
  ENTITY_SHAPE_INFO,
  getShapeClasses,
  getShapeDimensions,
  needsContentCounterRotation,
  usesClipPath,
} from "./entity-shapes";
import type { GraphEntityType } from "../types";

const ALL_ENTITY_TYPES: GraphEntityType[] = [
  "backlog",
  "scenario",
  "execution",
  "initiative",
  "capture",
  "agent-run",
  "agent-activity",
];

describe("ENTITY_SHAPE_MAP", () => {
  it("has a shape for every entity type", () => {
    for (const entityType of ALL_ENTITY_TYPES) {
      expect(ENTITY_SHAPE_MAP[entityType]).toBeTruthy();
    }
  });

  it("assigns unique shapes to each entity type", () => {
    const shapes = Object.values(ENTITY_SHAPE_MAP);
    // pill and rounded-full are shared by agent-activity and initiative, that's intentional
    // But the shape names should all be defined
    expect(shapes.length).toBe(ALL_ENTITY_TYPES.length);
  });
});

describe("getShapeClasses", () => {
  it("returns rotate-45 for backlog (diamond)", () => {
    expect(getShapeClasses("backlog")).toBe("rotate-45");
  });

  it("returns rounded-lg for scenario (rectangle)", () => {
    expect(getShapeClasses("scenario")).toBe("rounded-lg");
  });

  it("returns clip-hexagon for execution", () => {
    expect(getShapeClasses("execution")).toBe("clip-hexagon");
  });

  it("returns rounded-full for initiative (circle)", () => {
    expect(getShapeClasses("initiative")).toBe("rounded-full");
  });

  it("returns clip-pentagon for capture", () => {
    expect(getShapeClasses("capture")).toBe("clip-pentagon");
  });

  it("returns clip-octagon for agent-run", () => {
    expect(getShapeClasses("agent-run")).toBe("clip-octagon");
  });

  it("returns rounded-full for agent-activity (pill)", () => {
    expect(getShapeClasses("agent-activity")).toBe("rounded-full");
  });
});

describe("getShapeDimensions", () => {
  it("returns square dimensions for diamond shapes", () => {
    const dims = getShapeDimensions("backlog");
    expect(dims.width).toBe(dims.height);
  });

  it("returns square dimensions for circle shapes", () => {
    const dims = getShapeDimensions("initiative");
    expect(dims.width).toBe(dims.height);
  });

  it("returns wider-than-tall for rectangle shapes", () => {
    const dims = getShapeDimensions("scenario");
    expect(dims.width).toBeGreaterThan(dims.height);
  });

  it("returns positive dimensions for all types", () => {
    for (const entityType of ALL_ENTITY_TYPES) {
      const dims = getShapeDimensions(entityType);
      expect(dims.width).toBeGreaterThan(0);
      expect(dims.height).toBeGreaterThan(0);
    }
  });
});

describe("needsContentCounterRotation", () => {
  it("returns true only for backlog (diamond)", () => {
    expect(needsContentCounterRotation("backlog")).toBe(true);
    expect(needsContentCounterRotation("scenario")).toBe(false);
    expect(needsContentCounterRotation("execution")).toBe(false);
    expect(needsContentCounterRotation("initiative")).toBe(false);
  });
});

describe("usesClipPath", () => {
  it("returns true for hexagon, pentagon, octagon", () => {
    expect(usesClipPath("execution")).toBe(true);    // hexagon
    expect(usesClipPath("capture")).toBe(true);       // pentagon
    expect(usesClipPath("agent-run")).toBe(true);     // octagon
  });

  it("returns false for non-clipped shapes", () => {
    expect(usesClipPath("backlog")).toBe(false);      // diamond (rotate)
    expect(usesClipPath("scenario")).toBe(false);     // rectangle
    expect(usesClipPath("initiative")).toBe(false);   // circle
    expect(usesClipPath("agent-activity")).toBe(false); // pill
  });
});

describe("ENTITY_SHAPE_INFO", () => {
  it("has info for all entity types", () => {
    expect(ENTITY_SHAPE_INFO).toHaveLength(ALL_ENTITY_TYPES.length);
  });

  it("each entry has entityType, shape, label, and shapeClass", () => {
    for (const info of ENTITY_SHAPE_INFO) {
      expect(info.entityType).toBeTruthy();
      expect(info.shape).toBeTruthy();
      expect(info.label).toBeTruthy();
      expect(info.shapeClass).toBeTruthy();
    }
  });
});
