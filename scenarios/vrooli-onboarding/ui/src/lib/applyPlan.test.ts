import { describe, expect, it } from "vitest";

import {
  APPLY_KIND_ACTIONS,
  APPLY_NO_REMOVAL_NOTE,
  type ApplyPlanItem,
  groupByKind,
  itemState,
  kindLabel,
  summarizeApplyPlan,
} from "./applyPlan";

function item(overrides: Partial<ApplyPlanItem> & { id: string }): ApplyPlanItem {
  return { kind: "tool", name: overrides.id, required: false, ...overrides };
}

describe("itemState", () => {
  it("treats a missing state as unknown rather than satisfied", () => {
    // A UI newer than its API must never claim an item is already in place on
    // the strength of an absent field.
    expect(itemState(item({ id: "git" }))).toBe("unknown");
  });

  it("rejects any value the API did not promise", () => {
    expect(itemState(item({ id: "git", state: "deferred" }))).toBe("unknown");
    expect(itemState(item({ id: "git", state: "" }))).toBe("unknown");
  });

  it("passes through the two states that are evidence", () => {
    expect(itemState(item({ id: "git", state: "satisfied" }))).toBe("satisfied");
    expect(itemState(item({ id: "git", state: "pending" }))).toBe("pending");
  });
});

describe("summarizeApplyPlan", () => {
  const items = [
    item({ id: "tool:git", name: "git", required: true, state: "satisfied" }),
    item({ id: "tool:jq", name: "jq", state: "satisfied" }),
    item({ id: "safeguard:hard", kind: "safeguard", name: "host_hardening", required: true, privileged: true, state: "pending" }),
    item({ id: "safeguard:clock", kind: "safeguard", name: "clock", required: true, state: "pending" }),
    item({ id: "resource:pg", kind: "resource", name: "postgres", required: true }),
  ];

  it("splits the desired-state list into what changes and what does not", () => {
    const summary = summarizeApplyPlan(items);
    expect(summary.total).toBe(5);
    expect(summary.pending.map((entry) => entry.name)).toEqual(["host_hardening", "clock"]);
    expect(summary.satisfied.map((entry) => entry.name)).toEqual(["git", "jq"]);
    expect(summary.unknown.map((entry) => entry.name)).toEqual(["postgres"]);
  });

  it("counts elevation separately for the pending group", () => {
    // The number that matters before consent is how many elevated items are
    // about to run, not how many exist in the selection.
    const summary = summarizeApplyPlan(items);
    expect(summary.elevatedPending).toBe(1);
    expect(summary.elevatedTotal).toBe(1);
  });

  it("returns empty groups rather than throwing on an empty plan", () => {
    const summary = summarizeApplyPlan([]);
    expect(summary.total).toBe(0);
    expect(summary.pending).toEqual([]);
  });
});

describe("groupByKind", () => {
  it("orders kinds so host changes precede service starts", () => {
    const groups = groupByKind([
      item({ id: "scenario:a", kind: "scenario", name: "a" }),
      item({ id: "resource:b", kind: "resource", name: "b" }),
      item({ id: "safeguard:c", kind: "safeguard", name: "c" }),
      item({ id: "tool:d", kind: "tool", name: "d" }),
    ]);
    expect(groups.map((group) => group.kind)).toEqual(["tool", "safeguard", "resource", "scenario"]);
  });

  it("keeps an unrecognized kind instead of dropping it", () => {
    const groups = groupByKind([item({ id: "x:1", kind: "mystery", name: "one" })]);
    expect(groups).toHaveLength(1);
    expect(groups[0]?.kind).toBe("mystery");
  });
});

describe("kindLabel", () => {
  it("pluralizes only when there is more than one", () => {
    expect(kindLabel("tool", 1)).toBe("Host tool");
    expect(kindLabel("tool", 2)).toBe("Host tools");
  });

  it("titlecases an unrecognized kind", () => {
    expect(kindLabel("mystery", 1)).toBe("Mystery");
  });
});

describe("apply effect disclosure", () => {
  it("names a real control-plane command for every plan kind", () => {
    // These must stay the commands the executor actually runs; a kind with no
    // entry would leave an operator unable to see what apply does to it.
    expect(APPLY_KIND_ACTIONS.map((action) => action.kind)).toEqual(["tool", "safeguard", "resource", "scenario"]);
    for (const action of APPLY_KIND_ACTIONS) {
      expect(action.command.startsWith("vrooli ")).toBe(true);
    }
  });

  it("states that apply never removes anything", () => {
    expect(APPLY_NO_REMOVAL_NOTE).toContain("Nothing is removed");
  });
});
