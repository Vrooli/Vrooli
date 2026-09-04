import { describe, expect, it } from "vitest";

import {
  axisOf,
  coordinateOf,
  type PhysicalEdge,
} from "../../../../foundations/GestureDirection/versions/1.1.0/GestureDirection.ts";

describe("logical gesture axes", () => {
  it.each([
    ["left", "inline", "clientX"],
    ["right", "inline", "clientX"],
    ["top", "block", "clientY"],
    ["bottom", "block", "clientY"],
  ] as const)("resolves %s to %s/%s", (edge, axis, coordinate) => {
    expect(axisOf(edge as PhysicalEdge)).toBe(axis);
    expect(coordinateOf(edge as PhysicalEdge)).toBe(coordinate);
  });
});
