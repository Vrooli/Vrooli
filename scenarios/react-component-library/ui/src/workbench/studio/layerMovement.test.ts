import { describe, expect, it } from "vitest";
import { node, findNode } from "../templates/spec";
import { reparentLayer } from "./layerMovement";

const tree = () =>
  node("root", "page", {}, [
    node("left", "stack", {}, [node("a", "text"), node("b", "text")]),
    node("right", "card", {}, [node("c", "text")]),
    node("leaf", "text"),
  ]);

describe("reparentLayer", () => {
  it("reorders siblings and reparents layers into any container", () => {
    let result = reparentLayer(tree(), "a", "b", "after");
    expect(result.children?.[0]?.children?.map((item) => item.id)).toEqual(["b", "a"]);
    result = reparentLayer(result, "leaf", "right", "inside");
    expect(findNode(result, "right")?.children?.map((item) => item.id)).toEqual(["c", "leaf"]);
    expect(findNode(result, "root")?.children?.map((item) => item.id)).toEqual(["left", "right"]);
  });

  it("rejects moving the root, cycles, missing targets and inside drops on leaves", () => {
    const original = tree();
    expect(reparentLayer(original, "root", "left", "inside")).toBe(original);
    expect(reparentLayer(original, "left", "a", "inside")).toBe(original);
    expect(reparentLayer(original, "a", "leaf", "inside")).toBe(original);
    expect(reparentLayer(original, "a", "missing", "after")).toBe(original);
    expect(reparentLayer(original, "right", "c", "after")).toBe(original);
  });

  it("supports moving a nested layer to the frame root", () => {
    const result = reparentLayer(tree(), "c", "root", "inside");
    expect(result.children?.map((item) => item.id)).toEqual(["left", "right", "leaf", "c"]);
    expect(findNode(result, "right")?.children).toEqual([]);
  });
});
