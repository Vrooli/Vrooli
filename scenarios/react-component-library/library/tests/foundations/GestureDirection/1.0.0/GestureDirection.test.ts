import { describe, expect, it } from "vitest";

import {
  edgeSign,
  gestureSign,
  normalizeWritingDirection,
  oppositeEdge,
  resolveAnchorEdge,
  resolveGestureDirection,
  type AnchorEdge,
  type WritingDirection,
} from "../../../../foundations/GestureDirection/versions/1.0.0/GestureDirection.ts";

const DIRECTIONS: WritingDirection[] = ["ltr", "rtl"];
const ANCHORS: AnchorEdge[] = ["inline-start", "inline-end"];

describe("resolveAnchorEdge", () => {
  it("places a start-anchored panel on the left when text runs left to right", () => {
    expect(resolveAnchorEdge("ltr", "inline-start")).toBe("left");
  });

  it("mirrors a start-anchored panel with the locale", () => {
    expect(resolveAnchorEdge("rtl", "inline-start")).toBe("right");
  });

  it("moves an end-anchored panel without touching the locale", () => {
    expect(resolveAnchorEdge("ltr", "inline-end")).toBe("right");
  });

  it("lands back on the left when both inputs are flipped", () => {
    expect(resolveAnchorEdge("rtl", "inline-end")).toBe("left");
  });

  // The regression this module exists to prevent: an implementation that reads
  // only `dir` cannot tell these two apart, and an implementation that reads
  // only the anchor cannot either. Both must be consulted.
  it("reaches the same side by two different routes", () => {
    expect(resolveAnchorEdge("rtl", "inline-start")).toBe("right");
    expect(resolveAnchorEdge("ltr", "inline-end")).toBe("right");
  });

  it("is not determined by either input alone", () => {
    const byDirection = new Set(DIRECTIONS.map((d) => resolveAnchorEdge(d, "inline-start")));
    const byAnchor = new Set(ANCHORS.map((a) => resolveAnchorEdge("ltr", a)));
    expect(byDirection.size).toBe(2);
    expect(byAnchor.size).toBe(2);
  });
});

describe("resolveGestureDirection", () => {
  it("dismisses toward the anchored edge", () => {
    for (const direction of DIRECTIONS) {
      for (const anchor of ANCHORS) {
        expect(resolveGestureDirection(direction, anchor, "dismiss")).toBe(
          resolveAnchorEdge(direction, anchor),
        );
      }
    }
  });

  it("reveals away from the anchored edge", () => {
    for (const direction of DIRECTIONS) {
      for (const anchor of ANCHORS) {
        expect(resolveGestureDirection(direction, anchor, "reveal")).toBe(
          oppositeEdge(resolveAnchorEdge(direction, anchor)),
        );
      }
    }
  });

  // This opposition is the whole arbitration strategy: a row's reveal and its
  // drawer's dismiss never contend because they can never share a sign.
  it("keeps dismiss and reveal on opposite signs in every configuration", () => {
    for (const direction of DIRECTIONS) {
      for (const anchor of ANCHORS) {
        const dismiss = gestureSign(direction, anchor, "dismiss");
        const reveal = gestureSign(direction, anchor, "reveal");
        expect(dismiss).toBe(-reveal);
      }
    }
  });
});

describe("edgeSign", () => {
  it("treats leftward travel as negative", () => {
    expect(edgeSign("left")).toBe(-1);
    expect(edgeSign("right")).toBe(1);
  });
});

describe("normalizeWritingDirection", () => {
  it("only treats an explicit rtl as right to left", () => {
    expect(normalizeWritingDirection("rtl")).toBe("rtl");
  });

  it.each(["", "auto", "ltr", null, undefined])("treats %o as left to right", (value) => {
    expect(normalizeWritingDirection(value)).toBe("ltr");
  });
});
