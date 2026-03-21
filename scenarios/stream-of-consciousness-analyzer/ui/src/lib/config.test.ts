import { describe, it, expect } from "vitest";
import {
  CANVAS_ZOOM_MIN,
  CANVAS_ZOOM_MAX,
  CANVAS_ZOOM_IN_FACTOR,
  CANVAS_ZOOM_OUT_FACTOR,
  CANVAS_PAN_STEP,
  INFO_PLACEMENT_WIDTH,
  INFO_PLACEMENT_HEIGHT,
  THOUGHT_PLACEMENT_WIDTH,
  THOUGHT_PLACEMENT_HEIGHT,
  EDGE_STROKE_WIDTH,
  GRAPH_MIN_HEIGHT,
  LINK_MODE_WAITING,
  TEXT_CAPTURE_ROWS,
} from "./config";

// [REQ:P0-003] Canvas zoom bounds are sane: min < 1 < max
describe("Canvas zoom config", () => {
  it("zoom min is less than 1 (allows zoom out)", () => {
    expect(CANVAS_ZOOM_MIN).toBeGreaterThan(0);
    expect(CANVAS_ZOOM_MIN).toBeLessThan(1);
  });

  it("zoom max is greater than 1 (allows zoom in)", () => {
    expect(CANVAS_ZOOM_MAX).toBeGreaterThan(1);
  });

  it("zoom in factor is greater than 1", () => {
    expect(CANVAS_ZOOM_IN_FACTOR).toBeGreaterThan(1);
  });

  it("zoom out factor is less than 1", () => {
    expect(CANVAS_ZOOM_OUT_FACTOR).toBeGreaterThan(0);
    expect(CANVAS_ZOOM_OUT_FACTOR).toBeLessThan(1);
  });

  it("zoom factors are close to 1 (smooth scrolling)", () => {
    expect(CANVAS_ZOOM_IN_FACTOR).toBeLessThan(1.5);
    expect(CANVAS_ZOOM_OUT_FACTOR).toBeGreaterThan(0.5);
  });

  it("pan step is positive and reasonable", () => {
    expect(CANVAS_PAN_STEP).toBeGreaterThan(0);
    expect(CANVAS_PAN_STEP).toBeLessThanOrEqual(200);
  });
});

// [REQ:P0-002] Placement areas are positive and large enough for visibility
describe("Placement config", () => {
  it("info placement area is positive", () => {
    expect(INFO_PLACEMENT_WIDTH).toBeGreaterThan(0);
    expect(INFO_PLACEMENT_HEIGHT).toBeGreaterThan(0);
  });

  it("thought placement area is positive", () => {
    expect(THOUGHT_PLACEMENT_WIDTH).toBeGreaterThan(0);
    expect(THOUGHT_PLACEMENT_HEIGHT).toBeGreaterThan(0);
  });
});

// [REQ:P0-004] Graph rendering defaults are reasonable
describe("Graph view config", () => {
  it("edge stroke width is positive", () => {
    expect(EDGE_STROKE_WIDTH).toBeGreaterThan(0);
  });

  it("graph min height is positive", () => {
    expect(GRAPH_MIN_HEIGHT).toBeGreaterThan(0);
  });

  it("link mode sentinel is a non-empty string", () => {
    expect(LINK_MODE_WAITING).toBeTruthy();
    expect(typeof LINK_MODE_WAITING).toBe("string");
  });
});

// [REQ:P0-002] Text capture rows config
describe("Text capture config", () => {
  it("text capture rows is at least 1", () => {
    expect(TEXT_CAPTURE_ROWS).toBeGreaterThanOrEqual(1);
  });
});
