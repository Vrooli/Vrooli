import { describe, it, expect } from "vitest";
import {
  computeDepthMap,
  computeUnblockingMap,
  dependencyAwareSort,
  type DepthItem,
} from "./dependency-sort";

// Regression: dependency-sort must work for any `{kind, name, status, dependsOn}`
// object, not just BacklogItem — milestones rely on this.

const mkMilestone = (
  name: string,
  status = "active",
  dependsOn: string[] = [],
): DepthItem => ({
  kind: "milestone",
  name,
  status,
  dependsOn,
});

describe("dependency-sort (generic DepthItem)", () => {
  it("computes depth across a linear milestone chain", () => {
    const items = [
      mkMilestone("a"),
      mkMilestone("b", "active", ["milestone/a"]),
      mkMilestone("c", "active", ["milestone/b"]),
    ];
    const depths = computeDepthMap(items);
    expect(depths.get("milestone/a")).toBe(0);
    expect(depths.get("milestone/b")).toBe(1);
    expect(depths.get("milestone/c")).toBe(2);
  });

  it("completed status resolves dependencies — depth drops to 0", () => {
    const items = [
      mkMilestone("a", "completed"),
      mkMilestone("b", "active", ["milestone/a"]),
    ];
    const depths = computeDepthMap(items);
    expect(depths.get("milestone/b")).toBe(0);
  });

  it("tolerates cycles without infinite loop", () => {
    const items = [
      mkMilestone("a", "active", ["milestone/b"]),
      mkMilestone("b", "active", ["milestone/a"]),
    ];
    const depths = computeDepthMap(items);
    expect(depths.get("milestone/a")).toBeGreaterThanOrEqual(0);
    expect(depths.get("milestone/b")).toBeGreaterThanOrEqual(0);
  });

  it("counts transitive incomplete dependents", () => {
    const items = [
      mkMilestone("root"),
      mkMilestone("mid", "active", ["milestone/root"]),
      mkMilestone("leaf1", "active", ["milestone/mid"]),
      mkMilestone("leaf2", "active", ["milestone/mid"]),
    ];
    const unblocking = computeUnblockingMap(items);
    expect(unblocking.get("milestone/root")).toBe(3);
    expect(unblocking.get("milestone/mid")).toBe(2);
    expect(unblocking.get("milestone/leaf1")).toBe(0);
  });

  it("sorts a filtered subset by dependency order", () => {
    const all = [
      mkMilestone("a"),
      mkMilestone("b", "active", ["milestone/a"]),
      mkMilestone("c", "active", ["milestone/b"]),
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
