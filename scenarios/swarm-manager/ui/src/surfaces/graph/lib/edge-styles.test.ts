import { describe, it, expect } from "vitest";
import { getEdgeStyle, EDGE_STYLES, STRAIGHT_EDGE_THRESHOLD, FILTER_SUGGESTION_THRESHOLD } from "./edge-styles";

describe("getEdgeStyle", () => {
  it("returns correct style for depends_on", () => {
    const style = getEdgeStyle("depends_on");
    expect(style.stroke).toBe("rgb(148 163 184)");
    expect(style.strokeDasharray).toBeUndefined(); // "none" -> undefined
  });

  it("returns dashed style for member_of", () => {
    const style = getEdgeStyle("member_of");
    expect(style.stroke).toBe("rgb(56 189 248)");
    expect(style.strokeDasharray).toBe("6 3");
  });

  it("returns dotted style for classified_as", () => {
    const style = getEdgeStyle("classified_as");
    expect(style.stroke).toBe("rgb(52 211 153)");
    expect(style.strokeDasharray).toBe("2 3");
  });

  it("returns solid violet for targets", () => {
    const style = getEdgeStyle("targets");
    expect(style.stroke).toBe("rgb(167 139 250)");
    expect(style.strokeDasharray).toBeUndefined();
  });

  it("returns dashed teal for activity_for", () => {
    const style = getEdgeStyle("activity_for");
    expect(style.stroke).toBe("rgb(45 212 191)");
    expect(style.strokeDasharray).toBe("7 3");
  });

  it("returns dotted indigo for continued_run", () => {
    const style = getEdgeStyle("continued_run");
    expect(style.stroke).toBe("rgb(129 140 248)");
    expect(style.strokeDasharray).toBe("1 3");
  });

  it("returns default style for unknown edge types", () => {
    const style = getEdgeStyle("unknown_type");
    expect(style.stroke).toBe("rgb(100 116 139 / 0.5)");
  });

  it("returns default style for undefined", () => {
    const style = getEdgeStyle(undefined);
    expect(style.stroke).toBe("rgb(100 116 139 / 0.5)");
  });
});

describe("EDGE_STYLES", () => {
  it("has entries for the graph edge types surfaced in the workspace", () => {
    expect(Object.keys(EDGE_STYLES)).toEqual(
      expect.arrayContaining([
        "depends_on",
        "member_of",
        "classified_as",
        "targets",
        "activity_for",
        "records_activity",
        "spawned_run",
        "continued_run",
      ]),
    );
  });

  it("each entry has stroke, strokeDasharray, and label", () => {
    for (const config of Object.values(EDGE_STYLES)) {
      expect(config.stroke).toBeTruthy();
      expect(typeof config.strokeDasharray).toBe("string");
      expect(config.label).toBeTruthy();
    }
  });
});

describe("thresholds", () => {
  it("straight edge threshold is 300", () => {
    expect(STRAIGHT_EDGE_THRESHOLD).toBe(300);
  });

  it("filter suggestion threshold is 500", () => {
    expect(FILTER_SUGGESTION_THRESHOLD).toBe(500);
  });
});
