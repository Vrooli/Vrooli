import { describe, expect, it } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { MutationView } from "./MutationView";
import { PROPOSAL_OPS, type ProposalMutation } from "../../types/proposal";
import { archetypeFor } from "../../lib/mutation-archetypes";

/**
 * The contract these tests defend: a mutation card must display the payload
 * that decides it. Asserting "renders without crashing" would have passed
 * against the implementation this replaced, which rendered the op name and an
 * empty target for every creation op.
 */

function renderMutation(mutation: ProposalMutation, base?: Parameters<typeof MutationView>[0]["base"]) {
  return render(<MutationView mutation={mutation} base={base} />);
}

/** Minimal but valid payload per op, so every archetype has a real example. */
const SAMPLE_PAYLOAD: Record<string, Partial<ProposalMutation>> = {
  add_item: { item: { kind: "execute", name: "a", title: "Add title" } },
  create_goal: { goal: { name: "g", title: "Goal title" } },
  create_milestone: { goal_milestone: { name: "m", title: "Milestone title" } },
  update_item: { target: "execute/a", patch: { effort: "M" } },
  update_milestone: { goal_milestone: { name: "m", title: "Milestone title" } },
  change_status: { target: "execute/a", status: "blocked" },
  change_priority: { target: "execute/a", priority: 7 },
  move_milestone: { target: "execute/a", milestone: "next" },
  add_edge: { from: "execute/a", to: "execute/b" },
  remove_edge: { from: "execute/a", to: "execute/b" },
  assign_milestone_items: { milestone_name: "m", items: ["execute/a"] },
  unassign_milestone_items: { milestone_name: "m", items: ["execute/a"] },
  add_goal_target: { target: "g", targets: ["scenario/x"] },
  remove_goal_target: { target: "g", targets: ["scenario/x"] },
  split_item: { target: "execute/a", into: [{ kind: "execute", name: "b", title: "Child title" }] },
  merge_items: { sources: ["execute/a", "execute/b"], item: { kind: "execute", name: "c", title: "Merged title" } },
  archive_item: { target: "execute/a" },
  archive_milestone: { target: "m", milestone_name: "m" },
  interrupt_in_progress: { target: "execute/a" },
  recreate_item: { target: "execute/a" },
  recreate_milestone: { target: "m", milestone_name: "m" },
  reset_artifacts: { target: "execute/a", reset_scope: ["review"] },
};

describe("MutationView — exhaustive op coverage", () => {
  it.each(PROPOSAL_OPS)("renders %s through its archetype", (op) => {
    const payload = SAMPLE_PAYLOAD[op];
    expect(payload, `no sample payload defined for ${op}`).toBeDefined();
    const { container } = renderMutation({ id: "m1", op, ...payload });
    const card = within(container).getByTestId("mutation-view");
    expect(card.dataset.archetype).toBe(archetypeFor(op));
    // No op may fall through to the "cannot display" branch.
    expect(card.textContent).not.toContain("cannot display");
  });

  it("shows a subject for every op, never a blank reference", () => {
    for (const op of PROPOSAL_OPS) {
      const { container, unmount } = renderMutation({ id: "m1", op, ...SAMPLE_PAYLOAD[op] });
      const text = within(container).getByTestId("mutation-view").textContent ?? "";
      expect(text.trim().length, `${op} rendered an empty card`).toBeGreaterThan(0);
      unmount();
    }
  });
});

describe("object preview", () => {
  it("shows the item a creation will produce, including its ref", () => {
    renderMutation({
      id: "m1",
      op: "add_item",
      item: {
        kind: "execute",
        name: "capture-health",
        title: "Operators can inspect capture classification health",
        description: "Add a read-only health command.",
        priority: 4,
        effort: "M",
        tags: ["cli", "captures"],
        acceptance_allow: ["shows the capture id", "shows a recovery command"],
      },
    });
    expect(screen.getByText("Operators can inspect capture classification health")).toBeInTheDocument();
    // add_item carries no `target`; the ref must come from the item spec.
    expect(screen.getByText("execute/capture-health")).toBeInTheDocument();
    expect(screen.getByText("4")).toBeInTheDocument();
    expect(screen.getByText("cli")).toBeInTheDocument();
    expect(screen.getByText(/Acceptance criteria \(2\)/)).toBeInTheDocument();
    expect(screen.getByText("shows the capture id")).toBeInTheDocument();
  });

  it("warns instead of rendering an empty card when a creation has no payload", () => {
    renderMutation({ id: "m1", op: "add_item" });
    expect(screen.getByText(/carries no payload/)).toBeInTheDocument();
  });
});

describe("field diff", () => {
  it("names the changed field and lists what stays untouched", () => {
    renderMutation({ id: "m1", op: "update_item", target: "execute/a", patch: { effort: "L" } });
    expect(screen.getByText("effort")).toBeInTheDocument();
    expect(screen.getByText(/Unchanged:/)).toBeInTheDocument();
    expect(screen.getByText(/description/)).toBeInTheDocument();
  });

  it("shows before and after when base state is supplied", () => {
    renderMutation(
      { id: "m1", op: "update_item", target: "execute/a", patch: { priority: 6 } },
      { patch: { priority: 3 } },
    );
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("6")).toBeInTheDocument();
  });

  it("says so when current values could not be resolved", () => {
    renderMutation({ id: "m1", op: "update_item", target: "execute/a", patch: { priority: 6 } });
    expect(screen.getByText(/current values unavailable/)).toBeInTheDocument();
  });

  it("calls out an explicit clear, which is not the same as leaving a field alone", () => {
    renderMutation({ id: "m1", op: "update_item", target: "execute/a", patch: { note: "" } });
    expect(screen.getByText(/emptied, not left alone/)).toBeInTheDocument();
  });

  it("word-diffs long prose against its previous value", () => {
    const before = `Remove the boot call from startup. ${"context ".repeat(20)}`;
    const after = `Remove the package and its tests. ${"context ".repeat(20)}`;
    const { container } = renderMutation(
      { id: "m1", op: "update_item", target: "execute/a", patch: { description: after } },
      { patch: { description: before } },
    );
    expect(container.textContent).toContain("boot call");
    expect(container.textContent).toContain("package and its tests");
  });
});

describe("scalar transition", () => {
  it("renders an empty move_milestone destination as a detach, not a blank", () => {
    renderMutation({ id: "m1", op: "move_milestone", target: "execute/a", milestone: "" });
    expect(screen.getByText("detached from milestone")).toBeInTheDocument();
  });

  it("shows the incoming status", () => {
    renderMutation({ id: "m1", op: "change_status", target: "execute/a", status: "blocked" }, { status: "backlog" });
    expect(screen.getByText("backlog")).toBeInTheDocument();
    expect(screen.getByText("blocked")).toBeInTheDocument();
  });
});

describe("edge delta", () => {
  it("names both endpoints", () => {
    renderMutation({ id: "m1", op: "add_edge", from: "execute/a", to: "execute/b" });
    expect(screen.getByText("execute/b")).toBeInTheDocument();
    expect(screen.getByText("depends on")).toBeInTheDocument();
  });

  it("flags a half-specified edge", () => {
    renderMutation({ id: "m1", op: "add_edge", from: "execute/a" });
    expect(screen.getByText(/One endpoint is missing/)).toBeInTheDocument();
  });
});

describe("fan out and in", () => {
  it("states that split archives its source and previews the children", () => {
    renderMutation({
      id: "m1",
      op: "split_item",
      target: "execute/parent",
      into: [{ kind: "execute", name: "child-a", title: "Child A" }],
    });
    expect(screen.getByText(/1 source item will be archived/)).toBeInTheDocument();
    expect(screen.getByText("Child A")).toBeInTheDocument();
    expect(screen.getByText(/Dependents of the source are not retargeted/)).toBeInTheDocument();
  });

  it("states that merge archives all sources and retargets external edges", () => {
    renderMutation({
      id: "m1",
      op: "merge_items",
      sources: ["execute/a", "execute/b"],
      item: { kind: "execute", name: "merged", title: "Merged item" },
    });
    expect(screen.getByText(/2 source items will be archived/)).toBeInTheDocument();
    expect(screen.getByText(/External edges are retargeted/)).toBeInTheDocument();
  });
});

describe("destructive ops", () => {
  it("marks destructive ops and states how to reverse them", () => {
    renderMutation({ id: "m1", op: "archive_item", target: "execute/a" });
    expect(screen.getByText("destructive")).toBeInTheDocument();
    expect(screen.getByText(/Reversible with the unarchive endpoint/)).toBeInTheDocument();
  });

  it("says plainly when an op cannot be reversed", () => {
    renderMutation({ id: "m1", op: "interrupt_in_progress", target: "execute/a" });
    expect(screen.getByText(/Not reversible/)).toBeInTheDocument();
  });

  it("marks split and merge destructive because they archive their sources", () => {
    const { container, unmount } = renderMutation({ id: "m1", op: "split_item", target: "execute/a", into: [] });
    expect(within(container).getByText("destructive")).toBeInTheDocument();
    unmount();
    const merged = renderMutation({ id: "m2", op: "merge_items", sources: ["execute/a"], item: { kind: "execute", name: "c", title: "C" } });
    expect(within(merged.container).getByText("destructive")).toBeInTheDocument();
  });
});

describe("scope checklist", () => {
  it("spells out which artifacts are removed and that the spec survives", () => {
    renderMutation({ id: "m1", op: "reset_artifacts", target: "execute/a", reset_scope: ["review", "plan_unbind"] });
    expect(screen.getByText("Review rounds and their evidence")).toBeInTheDocument();
    expect(screen.getByText("Binding to the plan of record")).toBeInTheDocument();
    expect(screen.getByText(/item specification is kept/)).toBeInTheDocument();
  });
});

describe("malformed payloads", () => {
  it("warns rather than throwing when a mutation declares no op", () => {
    // Payloads are agent-authored JSON. One malformed mutation must not be
    // able to blank the whole decision queue.
    const mutation = { id: "m1" } as unknown as ProposalMutation;
    expect(() => renderMutation(mutation)).not.toThrow();
    expect(screen.getByText(/declares no operation/)).toBeInTheDocument();
  });

  it("warns for an op this build does not know", () => {
    const mutation = { id: "m1", op: "teleport_item" } as unknown as ProposalMutation;
    renderMutation(mutation);
    expect(screen.getByText(/cannot display/)).toBeInTheDocument();
  });
});
