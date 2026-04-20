import { describe, it, expect } from "vitest";
import {
  computeDepthMap,
  computeUnblockingMap,
  dependencyAwareSort,
  type DepthItem,
} from "./dependency-sort";

// Regression: dependency-sort must work for any `{kind, name, status, dependsOn}`
// object, not just BacklogItem — initiatives rely on this.

const mkInit = (
  name: string,
  status = "active",
  dependsOn: string[] = [],
): DepthItem => ({
  kind: "initiative",
  name,
  status,
  dependsOn,
});

describe("dependency-sort (generic DepthItem)", () => {
  it("computes depth across a linear initiative chain", () => {
    const items = [
      mkInit("a"),
      mkInit("b", "active", ["initiative/a"]),
      mkInit("c", "active", ["initiative/b"]),
    ];
    const depths = computeDepthMap(items);
    expect(depths.get("initiative/a")).toBe(0);
    expect(depths.get("initiative/b")).toBe(1);
    expect(depths.get("initiative/c")).toBe(2);
  });

  it("completed status resolves dependencies — depth drops to 0", () => {
    const items = [
      mkInit("a", "completed"),
      mkInit("b", "active", ["initiative/a"]),
    ];
    const depths = computeDepthMap(items);
    expect(depths.get("initiative/b")).toBe(0);
  });

  it("tolerates cycles without infinite loop", () => {
    const items = [
      mkInit("a", "active", ["initiative/b"]),
      mkInit("b", "active", ["initiative/a"]),
    ];
    const depths = computeDepthMap(items);
    expect(depths.get("initiative/a")).toBeGreaterThanOrEqual(0);
    expect(depths.get("initiative/b")).toBeGreaterThanOrEqual(0);
  });

  it("counts transitive incomplete dependents", () => {
    const items = [
      mkInit("root"),
      mkInit("mid", "active", ["initiative/root"]),
      mkInit("leaf1", "active", ["initiative/mid"]),
      mkInit("leaf2", "active", ["initiative/mid"]),
    ];
    const unblocking = computeUnblockingMap(items);
    expect(unblocking.get("initiative/root")).toBe(3);
    expect(unblocking.get("initiative/mid")).toBe(2);
    expect(unblocking.get("initiative/leaf1")).toBe(0);
  });

  it("sorts a filtered subset by dependency order", () => {
    const all = [
      mkInit("a"),
      mkInit("b", "active", ["initiative/a"]),
      mkInit("c", "active", ["initiative/b"]),
    ];
    // Sort c then a — expect a first (lowest depth).
    const sorted = dependencyAwareSort(
      [all[2]!, all[0]!],
      (x, y) => x.name.localeCompare(y.name),
      all,
    );
    expect(sorted.map((i) => i.name)).toEqual(["a", "c"]);
  });
});
