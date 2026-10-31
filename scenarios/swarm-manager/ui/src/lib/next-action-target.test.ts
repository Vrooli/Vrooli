import { describe, expect, it } from "vitest";
import { MILESTONE_TARGET_PREFIXES, actionTargetSuffix, milestoneTargetOf } from "./next-action-target";

describe("actionTargetSuffix", () => {
  it("returns the value under a matching prefix", () => {
    expect(actionTargetSuffix("milestone_review:release-1", "milestone_review:")).toBe("release-1");
  });

  it("returns empty for a different prefix", () => {
    expect(actionTargetSuffix("milestone_criteria:release-1", "milestone_review:")).toBe("");
  });

  it("returns empty for an absent target", () => {
    expect(actionTargetSuffix(undefined, "milestone_review:")).toBe("");
  });

  it("treats a prefix with no value as no target", () => {
    // A bare prefix would otherwise resolve to "" and be passed to the API as
    // a milestone name, producing a confusing 404 instead of a local no-op.
    expect(actionTargetSuffix("milestone_review:", "milestone_review:")).toBe("");
    expect(actionTargetSuffix("milestone_review:   ", "milestone_review:")).toBe("");
  });

  it("keeps values that themselves contain a colon", () => {
    expect(actionTargetSuffix("milestone_review:a:b", "milestone_review:")).toBe("a:b");
  });
});

describe("milestoneTargetOf", () => {
  it("resolves both milestone-scoped prefixes", () => {
    expect(milestoneTargetOf("milestone_review:delivery")).toBe("delivery");
    expect(milestoneTargetOf("milestone_criteria:foundation")).toBe("foundation");
  });

  it("returns empty for non-milestone actions", () => {
    expect(milestoneTargetOf("goal_plan")).toBe("");
    expect(milestoneTargetOf("proposal_decision")).toBe("");
    expect(milestoneTargetOf(undefined)).toBe("");
  });

  it("covers every declared prefix", () => {
    // Guards the failure this module exists to prevent: a prefix added to the
    // list but not actually resolvable leaves a button that clicks into
    // nothing.
    for (const prefix of MILESTONE_TARGET_PREFIXES) {
      expect(milestoneTargetOf(`${prefix}some-milestone`)).toBe("some-milestone");
    }
  });
});
