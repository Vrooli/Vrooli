import { describe, expect, it } from "vitest";

import { DEFAULT_TEMPLATE } from "./library";
import {
  appendChild,
  clearSlotPlacement,
  duplicateNode,
  findNode,
  findSlot,
  layerRows,
  moveNode,
  node,
  removeNode,
  setProp,
  setSlotPlacement,
  slotPlacement,
  templateStyle,
  walk,
} from "./spec";

describe("workspace starter composition", () => {
  it("is a typed page with a component slot and unique node ids", () => {
    const ids: string[] = [];
    let slots = 0;
    walk(DEFAULT_TEMPLATE.spec, (current) => {
      ids.push(current.id);
      if (current.type === "slot") slots += 1;
    });
    expect(DEFAULT_TEMPLATE.spec.type).toBe("page");
    expect(slots).toBeGreaterThanOrEqual(1);
    expect(new Set(ids).size).toBe(ids.length);
    expect(DEFAULT_TEMPLATE.slotHint.length).toBeGreaterThan(4);
  });

  it("resolves a palette and typography into CSS custom properties", () => {
    const style = templateStyle(DEFAULT_TEMPLATE) as Record<string, string>;
    expect(style["--t-bg"]).toBe(DEFAULT_TEMPLATE.palette.bg);
    expect(style["--color-primary"]).toBe(DEFAULT_TEMPLATE.palette.accent);
    expect(style["--color-foreground"]).toBe(DEFAULT_TEMPLATE.palette.ink);
  });
});

describe("template spec operations", () => {
  const base = node("root", "page", { pad: 20 }, [
    node("stack", "stack", { gap: 12 }, [node("title", "text", { text: "Hello", variant: "display" })]),
  ]);

  it("flattens the layer tree with depth", () => {
    const rows = layerRows(base);
    expect(rows.map((row) => [row.node.id, row.depth])).toEqual([
      ["root", 0],
      ["stack", 1],
      ["title", 2],
    ]);
  });

  it("sets a prop without mutating the original spec", () => {
    const next = setProp(base, "title", "text", "Changed");
    expect(findNode(next, "title")?.props.text).toBe("Changed");
    expect(findNode(base, "title")?.props.text).toBe("Hello");
  });

  it("appends and removes nodes", () => {
    const added = appendChild(base, "stack", node("body", "text", { text: "Body" }));
    expect(findNode(added, "body")).toBeDefined();
    const removed = removeNode(added, "title");
    expect(findNode(removed, "title")).toBeUndefined();
    expect(findNode(removed, "body")).toBeDefined();
  });

  it("does not remove the page root through the tree helper", () => {
    const removed = removeNode(base, "root");
    expect(removed.id).toBe("root");
  });

  it("places, reads and clears the component slot without touching flow props", () => {
    const withSlot = appendChild(base, "stack", node("slot", "slot", { hint: "Action", minH: 48 }));
    expect(findSlot(withSlot)?.id).toBe("slot");
    expect(slotPlacement(findSlot(withSlot))).toBeUndefined();

    const placed = setSlotPlacement(withSlot, "slot", { x: 32, y: 120, w: 260, h: 64 });
    expect(slotPlacement(findSlot(placed))).toEqual({ x: 32, y: 120, w: 260, h: 64 });
    expect(findNode(placed, "slot")?.props.hint).toBe("Action");

    const cleared = clearSlotPlacement(placed, "slot");
    expect(slotPlacement(findSlot(cleared))).toBeUndefined();
    expect(findNode(cleared, "slot")?.props.minH).toBe(48);
  });

  it("clamps a placement to a usable minimum", () => {
    const withSlot = appendChild(base, "stack", node("slot", "slot", {}));
    const placed = setSlotPlacement(withSlot, "slot", { x: -20, y: -4, w: 10, h: 2 });
    expect(slotPlacement(findSlot(placed))).toEqual({ x: 0, y: 0, w: 80, h: 24 });
  });

  it("moves a layer within its siblings and stops at the edges", () => {
    const page = node("root", "page", {}, [
      node("stack", "stack", {}, [
        node("first", "text", { text: "First" }),
        node("second", "text", { text: "Second" }),
        node("third", "text", { text: "Third" }),
      ]),
    ]);
    const moved = moveNode(page, "third", -1);
    expect(moved.children?.[0]?.children?.map((child) => child.id)).toEqual([
      "first",
      "third",
      "second",
    ]);
    const atEdge = moveNode(moved, "first", -1);
    expect(atEdge.children?.[0]?.children?.map((child) => child.id)).toEqual([
      "first",
      "third",
      "second",
    ]);
  });

  it("duplicates a layer with fresh ids and keeps the original", () => {
    const duplicated = duplicateNode(base, "title");
    const stack = duplicated.children?.[0];
    expect(stack?.children).toHaveLength(2);
    expect(stack?.children?.[0]?.id).toBe("title");
    expect(stack?.children?.[1]?.id).toContain("title-copy");
    expect(stack?.children?.[1]?.props.text).toBe("Hello");
  });
});
