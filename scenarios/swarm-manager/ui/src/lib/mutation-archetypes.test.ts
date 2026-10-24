import { describe, it, expect } from "vitest";
import {
  OP_ARCHETYPE,
  OP_HEADLINE,
  daysSince,
  describePatch,
  headlineFor,
  isDestructiveOp,
  isKnownOp,
  mutationSubject,
  parseProposalPayload,
  proposalMutations,
} from "./mutation-archetypes";
import { PROPOSAL_OPS, type ProposalMutation } from "../types/proposal";

describe("op coverage contract", () => {
  it("maps every op the server accepts to an archetype", () => {
    const unmapped = PROPOSAL_OPS.filter((op) => !OP_ARCHETYPE[op]);
    expect(unmapped).toEqual([]);
  });

  it("gives every op a human headline", () => {
    const unlabelled = PROPOSAL_OPS.filter((op) => !OP_HEADLINE[op]);
    expect(unlabelled).toEqual([]);
  });

  it("does not map ops the server does not accept", () => {
    const extra = Object.keys(OP_ARCHETYPE).filter((op) => !(PROPOSAL_OPS as readonly string[]).includes(op));
    expect(extra).toEqual([]);
  });

  it("recognises known ops and rejects unknown ones", () => {
    expect(isKnownOp("update_item")).toBe(true);
    expect(isKnownOp("teleport_item")).toBe(false);
  });

  it("flags the ops that destroy or interrupt something", () => {
    expect(isDestructiveOp("archive_item")).toBe(true);
    expect(isDestructiveOp("interrupt_in_progress")).toBe(true);
    // split and merge archive their sources even though they render as fans.
    expect(isDestructiveOp("split_item")).toBe(true);
    expect(isDestructiveOp("merge_items")).toBe(true);
    expect(isDestructiveOp("add_item")).toBe(false);
    expect(isDestructiveOp("update_item")).toBe(false);
  });

  it("falls back to a readable headline for an unmapped op", () => {
    expect(headlineFor("mystery_op" as never)).toBe("mystery op");
  });
});

describe("mutationSubject", () => {
  it("resolves add_item to the ref it will create, not an empty target", () => {
    const mutation: ProposalMutation = {
      id: "m1",
      op: "add_item",
      item: { kind: "execute", name: "capture-health", title: "Capture health" },
    };
    expect(mutation.target).toBeUndefined();
    expect(mutationSubject(mutation)).toBe("execute/capture-health");
  });

  it("uses target for ops that act on an existing item", () => {
    expect(mutationSubject({ id: "m1", op: "update_item", target: "execute/thing" })).toBe("execute/thing");
  });

  it("resolves goal creation to the goal name", () => {
    expect(mutationSubject({ id: "m1", op: "create_goal", goal: { name: "g1", title: "Goal" } })).toBe("g1");
  });

  it("resolves milestone ops to the milestone name", () => {
    expect(mutationSubject({ id: "m1", op: "create_milestone", goal_milestone: { name: "m-1", title: "M" } })).toBe("m-1");
    expect(mutationSubject({ id: "m2", op: "assign_milestone_items", milestone_name: "m-2" })).toBe("m-2");
  });

  it("resolves edges to their origin", () => {
    expect(mutationSubject({ id: "m1", op: "add_edge", from: "execute/a", to: "execute/b" })).toBe("execute/a");
  });

  it("resolves merge to the merged item", () => {
    const mutation: ProposalMutation = {
      id: "m1",
      op: "merge_items",
      sources: ["execute/a", "execute/b"],
      item: { kind: "execute", name: "merged", title: "Merged" },
    };
    expect(mutationSubject(mutation)).toBe("execute/merged");
  });

  it("returns an empty string rather than throwing on a malformed mutation", () => {
    expect(mutationSubject({ id: "m1", op: "add_item" })).toBe("");
  });
});

describe("describePatch", () => {
  it("lists changed fields and names the untouched ones", () => {
    const summary = describePatch({ description: "new text" });
    expect(summary.changed.map((change) => change.field)).toEqual(["description"]);
    expect(summary.unchanged).toContain("title");
    expect(summary.unchanged).toContain("priority");
    expect(summary.unchanged).not.toContain("description");
  });

  it("distinguishes an explicit clear from an absent field", () => {
    const summary = describePatch({ note: "" });
    expect(summary.changed).toHaveLength(1);
    expect(summary.changed[0]?.cleared).toBe(true);
    // An absent field is untouched, never reported as cleared.
    expect(summary.unchanged).toContain("description");
  });

  it("treats a patch that restates the current value as unchanged", () => {
    const summary = describePatch({ effort: "M" }, { effort: "M" });
    expect(summary.changed).toHaveLength(0);
    expect(summary.unchanged).toContain("effort");
  });

  it("carries the before side when base state is supplied", () => {
    const summary = describePatch({ priority: 6 }, { priority: 3 });
    expect(summary.changed[0]?.before).toBe("3");
    expect(summary.changed[0]?.after).toBe("6");
  });

  it("omits the before side when base state is absent", () => {
    expect(describePatch({ priority: 6 }).changed[0]?.before).toBeUndefined();
  });

  it("renders list fields as comma-joined values", () => {
    const summary = describePatch({ tags: ["a", "b"] });
    expect(summary.changed[0]?.after).toBe("a, b");
  });

  it("routes long prose to a line diff and short values inline", () => {
    const long = "x".repeat(200);
    expect(describePatch({ description: long }).changed[0]?.presentation).toBe("prose");
    expect(describePatch({ effort: "M" }).changed[0]?.presentation).toBe("inline");
  });

  it("reports every field untouched for an absent patch", () => {
    const summary = describePatch(undefined);
    expect(summary.changed).toHaveLength(0);
    expect(summary.unchanged.length).toBeGreaterThan(0);
  });
});

describe("parseProposalPayload", () => {
  it("parses a well-formed envelope", () => {
    const payload = parseProposalPayload('{"form":"mutation_list","mutations":[{"id":"m1","op":"add_item"}]}');
    expect(payload.mutations).toHaveLength(1);
  });

  it("returns an empty envelope for malformed JSON instead of throwing", () => {
    expect(parseProposalPayload("{not json").mutations).toBeUndefined();
  });

  it("returns an empty envelope for an absent payload", () => {
    expect(parseProposalPayload(undefined).form).toBe("mutation_list");
  });

  it("survives a payload that parses to a non-object", () => {
    expect(parseProposalPayload("42").form).toBe("mutation_list");
  });

  it("drops mutations with no id, which the accept API cannot key on", () => {
    const mutations = proposalMutations('{"form":"mutation_list","mutations":[{"op":"add_item"},{"id":"m2","op":"add_item"}]}');
    expect(mutations.map((mutation) => mutation.id)).toEqual(["m2"]);
  });
});

describe("daysSince", () => {
  it("counts whole days elapsed", () => {
    expect(daysSince("2026-07-21T01:59:36Z", new Date("2026-08-09T12:00:00Z"))).toBe(19);
  });

  it("returns 0 for a future timestamp rather than a negative age", () => {
    expect(daysSince("2026-08-10T00:00:00Z", new Date("2026-08-09T00:00:00Z"))).toBe(0);
  });

  it("returns undefined for missing or unparseable input", () => {
    expect(daysSince(undefined)).toBeUndefined();
    expect(daysSince("not a date")).toBeUndefined();
  });
});
