import { describe, expect, it } from "vitest";
import { nextActionDetailTab } from "./backlog-next-action";

describe("nextActionDetailTab", () => {
  it.each([
    ["plan_author", "prompt"],
    ["plan_accept", "prompt"],
    ["plan_repair", "prompt"],
    ["review", "decide"],
    ["execution", "activity"],
    ["dependencies", "related"],
  ] as const)("routes %s to the %s detail tab", (target, tab) => {
    expect(nextActionDetailTab({
      id: "author_plan",
      compactLabel: "Plan",
      expandedLabel: "Author plan",
      enabled: true,
      blockers: [],
      target,
    })).toBe(tab);
  });

  it("does not route immediate mutations through a detail tab", () => {
    expect(nextActionDetailTab({
      id: "run",
      compactLabel: "Run",
      expandedLabel: "Run item",
      enabled: true,
      blockers: [],
      target: "run",
    })).toBeUndefined();
  });
});
