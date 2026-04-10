import { describe, it, expect } from "vitest";
import {
  ENTITY_REGISTRY,
  ENTITY_SHAPE_INFO,
  GRAPH_ENTITY_TYPES,
  getClipPathStyle,
  getEntityBadgeLabel,
  getEntityIcon,
  getEntityLabel,
  getShapeClasses,
  getShapeDimensions,
  usesClipPath,
} from "./entity-shapes";
import type { GraphEntityType } from "../types";

/** Known entity types — used to verify the registry is complete. */
const EXPECTED_ENTITY_TYPES: GraphEntityType[] = [
  "backlog",
  "scenario",
  "execution",
  "initiative",
  "capture",
  "agent-run",
  "agent-activity",
];

describe("ENTITY_REGISTRY completeness", () => {
  it("has an entry for every known entity type", () => {
    for (const et of EXPECTED_ENTITY_TYPES) {
      expect(ENTITY_REGISTRY[et]).toBeDefined();
    }
  });

  it("GRAPH_ENTITY_TYPES contains all expected types", () => {
    expect(GRAPH_ENTITY_TYPES).toHaveLength(EXPECTED_ENTITY_TYPES.length);
    for (const et of EXPECTED_ENTITY_TYPES) {
      expect(GRAPH_ENTITY_TYPES).toContain(et);
    }
  });
});

describe("shape uniqueness", () => {
  it("assigns a unique shape to every entity type", () => {
    const shapes = GRAPH_ENTITY_TYPES.map((et) => ENTITY_REGISTRY[et].shape);
    expect(new Set(shapes).size).toBe(shapes.length);
  });
});

describe("dimensions", () => {
  it("all shapes are wider than tall", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      const dims = ENTITY_REGISTRY[et].dimensions;
      expect(dims.width).toBeGreaterThan(dims.height);
    }
  });

  it("all dimensions are positive", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      const dims = ENTITY_REGISTRY[et].dimensions;
      expect(dims.width).toBeGreaterThan(0);
      expect(dims.height).toBeGreaterThan(0);
    }
  });
});

describe("clip-path validity", () => {
  // Valid polygon coordinate pair: "N% N%" where N is a number
  const COORD_PAIR = /^\d+%\s+\d+%$/;

  it("clip-path entries are valid polygon coordinate lists", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      const cp = ENTITY_REGISTRY[et].clipPath;
      if (cp === null) continue;

      const pairs = cp.split(",").map((p) => p.trim());
      expect(pairs.length).toBeGreaterThanOrEqual(3); // minimum polygon
      for (const pair of pairs) {
        expect(pair).toMatch(COORD_PAIR);
      }
    }
  });

  it("non-clipped shapes have a non-empty cssClass", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      if (ENTITY_REGISTRY[et].clipPath === null) {
        expect(ENTITY_REGISTRY[et].cssClass).not.toBe("");
      }
    }
  });
});

describe("getShapeClasses", () => {
  it("returns the cssClass from the registry", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      expect(getShapeClasses(et)).toBe(ENTITY_REGISTRY[et].cssClass);
    }
  });
});

describe("getShapeDimensions", () => {
  it("returns the dimensions from the registry", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      expect(getShapeDimensions(et)).toEqual(ENTITY_REGISTRY[et].dimensions);
    }
  });
});

describe("usesClipPath", () => {
  it("returns true when clipPath is non-null", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      expect(usesClipPath(et)).toBe(ENTITY_REGISTRY[et].clipPath !== null);
    }
  });
});

describe("getClipPathStyle", () => {
  it("returns a polygon style for clipped shapes", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      const style = getClipPathStyle(et);
      if (ENTITY_REGISTRY[et].clipPath !== null) {
        expect(style).toBeDefined();
        expect(style?.clipPath).toMatch(/^polygon\(.+\)$/);
      } else {
        expect(style).toBeUndefined();
      }
    }
  });
});

describe("getEntityIcon", () => {
  it("returns a component for every entity type", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      const icon = getEntityIcon(et);
      // Lucide icons are forwardRef objects with a render function
      expect(icon).toBeDefined();
      expect(typeof icon === "function" || typeof icon === "object").toBe(true);
    }
  });
});

describe("getEntityBadgeLabel", () => {
  it("returns a non-empty string for every entity type", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      const label = getEntityBadgeLabel(et);
      expect(label).toBeTruthy();
      expect(typeof label).toBe("string");
    }
  });

  it("returns short labels (no hyphens)", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      expect(getEntityBadgeLabel(et)).not.toContain("-");
    }
  });
});

describe("getEntityLabel", () => {
  it("returns a non-empty string for every entity type", () => {
    for (const et of GRAPH_ENTITY_TYPES) {
      expect(getEntityLabel(et)).toBeTruthy();
    }
  });
});

describe("ENTITY_SHAPE_INFO", () => {
  it("has one entry per entity type", () => {
    expect(ENTITY_SHAPE_INFO).toHaveLength(GRAPH_ENTITY_TYPES.length);
  });

  it("each entry has the required fields", () => {
    for (const info of ENTITY_SHAPE_INFO) {
      expect(info.entityType).toBeTruthy();
      expect(info.shape).toBeTruthy();
      expect(info.label).toBeTruthy();
      // cssClass may be empty for clip-path shapes
      expect(typeof info.cssClass).toBe("string");
    }
  });
});

describe("needsContentCounterRotation is removed", () => {
  it("is not exported from entity-shapes", async () => {
    const mod = await import("./entity-shapes");
    expect("needsContentCounterRotation" in mod).toBe(false);
  });
});
