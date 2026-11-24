import { describe, expect, it } from "vitest";

import type { DesignFrame } from "../design-types";
import { node } from "../templates/spec";
import {
  designCoverage,
  frameCoverage,
  layerCount,
  patternUsage,
  primitiveUsage,
  rclPrimitive,
} from "./rcl";

const spec = node("page", "page", {}, [
  node("card", "card", { pad: 16 }, [
    node("title", "text", { text: "Retention" }),
    node("button", "button", { text: "Open" }),
  ]),
  node("body", "text", { text: "Copy" }),
  node("meter", "meter", { value: 40 }),
]);

function frame(id: string, specNode = spec): DesignFrame {
  return { id, name: id, device: "desktop", spec: specNode };
}

describe("RCL coverage", () => {
  it("maps only real primitives and leaves layout custom", () => {
    expect(rclPrimitive("stat")).toBe("Stat");
    expect(rclPrimitive("card")).toBe("Card");
    expect(rclPrimitive("text")).toBeNull();
    expect(rclPrimitive("row")).toBeNull();
  });

  it("counts RCL layers against every layer below the frame root", () => {
    // card + button + meter = 3 of 5 (card, title, button, body, meter).
    expect(frameCoverage(spec)).toEqual({
      rcl: 3,
      custom: 2,
      total: 5,
      ratio: 3 / 5,
    });
    expect(layerCount(spec)).toBe(5);
  });

  it("does not claim BoundedMeter coverage for compact native meters", () => {
    const compact = node("compact-meter", "meter", { value: 40, compact: true });
    const page = node("page", "page", {}, [compact]);
    expect(frameCoverage(page)).toEqual({
      rcl: 0,
      custom: 1,
      total: 1,
      ratio: 0,
    });
    expect(primitiveUsage([frame("compact", page)], "compact")).toEqual([]);

    const regularPage = node("regular-page", "page", {}, [node("meter", "meter", { value: 60 })]);
    expect(primitiveUsage([frame("regular", regularPage), frame("compact", page)], "regular")).toEqual([
      { type: "meter", name: "BoundedMeter", count: 1, frames: 1 },
    ]);
  });

  it("counts a live catalog component bound into a slot as RCL coverage", () => {
    const bound = node("published-action", "slot", {
      componentId: "asset-button",
      storyId: "primary",
    });
    expect(frameCoverage(node("page", "page", {}, [bound]))).toEqual({
      rcl: 1,
      custom: 0,
      total: 1,
      ratio: 1,
    });
  });

  it("returns an empty ratio for a frame with no layers", () => {
    expect(frameCoverage(node("empty", "page", {}))).toEqual({
      rcl: 0,
      custom: 0,
      total: 0,
      ratio: 0,
    });
  });

  it("sums coverage across frames", () => {
    const frames = [frame("a"), frame("b", node("page2", "page", {}, [node("only", "card", {})]))];
    expect(designCoverage(frames)).toEqual({
      rcl: 4,
      custom: 2,
      total: 6,
      ratio: 4 / 6,
    });
  });

  it("reports primitive counts in the reference frame and the frames that use them", () => {
    const other = node("page2", "page", {}, [
      node("card2", "card", {}, [node("button2", "button", {})]),
    ]);
    const usage = primitiveUsage([frame("a"), frame("b", other)], "a");
    expect(usage).toEqual([
      { type: "meter", name: "BoundedMeter", count: 1, frames: 1 },
      { type: "button", name: "Button", count: 1, frames: 2 },
      { type: "card", name: "Card", count: 1, frames: 2 },
    ]);
  });

  it("counts grouping layers as patterns and skips empty containers", () => {
    const patterns = patternUsage(
      node("page", "page", {}, [
        node("row", "row", {}, [node("rowCard", "card", {}, [node("text", "text", {})])]),
        node("emptyRow", "row", {}),
        node("slot", "slot", {}),
      ]),
    );
    expect(patterns).toEqual([
      { type: "card", count: 1 },
      { type: "row", count: 1 },
      { type: "slot", count: 1 },
    ]);
  });
});
