import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { renderWithProviders } from "../../test-utils";
import { defaultApiClient } from "../../lib/api-client";
import { MutationView, type MutationViewProps } from "./MutationView";
import type { ProposalMutation } from "../../types/proposal";

// Actual public renderer, supported proposal ops and canonical providers.
// Optional/missing fields are read-display compatibility, never apply approval.
const views: { unmount: () => void }[] = [];
function mount(mutation: ProposalMutation, base?: MutationViewProps["base"]) {
  const view = renderWithProviders(<MutationView mutation={mutation} base={base} />); views.push(view); return view;
}
beforeEach(() => {
  for (const method of ["get", "post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Forbidden mutation display ${method}`));
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Forbidden mutation display transport")));
});
afterEach(() => {
  try { views.splice(0).forEach(view => view.unmount()); cleanup(); getSpatialNav()?.dispose(); for (const method of ["get", "post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); expect(fetch).not.toHaveBeenCalled(); }
  finally { vi.restoreAllMocks(); vi.unstubAllGlobals(); }
});
describe("Mutation display supported payload compatibility", () => {
  it("shows rich creation metadata and expands complete criteria and prose as safe literal text", async () => {
    const long = "Owner prose ".repeat(30) + "<script>neverExecute()</script>";
    const { container } = mount({ id: "fixture-create", op: "add_item", rationale: "Operator must see the complete proposal", item: { kind: "fix", name: "fixture-safe", title: "<img src=x onerror=neverExecute()>", description: long, priority: 0, effort: "S", milestone: "goal/proof", tags: ["retained"], note: "Literal operator note", acceptance_allow: ["Criterion A", "Criterion B", "Criterion C", "Criterion D"], acceptance_deny: ["Protected A", "Protected B", "Protected C", "Protected D"], depends_on: ["fix/dependency"] } });
    expect(screen.getByText("<img src=x onerror=neverExecute()>")).toBeVisible(); expect(container.querySelector("img")).toBeNull(); expect(container.querySelector("script")).toBeNull();
    expect(screen.getByText("priority").parentElement).toHaveTextContent("0"); expect(screen.getByText("goal/proof")).toBeVisible(); expect(screen.getByText("Note: Literal operator note")).toBeVisible(); expect(screen.getByText("Depends on: fix/dependency")).toBeVisible();
    const prose = screen.getByText("Show full text").closest("details")!; await userEvent.click(within(prose).getByText("Show full text")); expect(prose).toHaveAttribute("open"); expect(within(prose).getByText(long)).toBeVisible();
    for (const summary of screen.getAllByText("Show 1 more")) { const details = summary.closest("details")!; await userEvent.click(summary); expect(details).toHaveAttribute("open"); }
    expect(screen.getByText("Criterion D")).toBeVisible(); expect(screen.getByText("Protected D")).toBeVisible(); expect(screen.getByText(/Operator must see the complete proposal/)).toBeVisible();
  });
  it("uses an honest untitled label when an item read omits optional display metadata", () => {
    mount({ id: "fixture-minimal", op: "add_item", item: { kind: "idea", name: "fixture-minimal", title: "" } });
    expect(screen.getByText("Untitled item")).toBeVisible(); expect(screen.getByText("idea/fixture-minimal")).toBeVisible();
    expect(screen.queryByText("priority")).toBeNull(); expect(screen.queryByText(/Acceptance criteria/)).toBeNull(); expect(screen.queryByText(/Must not/)).toBeNull(); expect(screen.queryByText(/^Why:/)).toBeNull();
  });
  it("renders rich goal and milestone scope without offering a creation action", () => {
    mount({ id: "fixture-goal", op: "create_goal", goal: { name: "fixture-goal", title: "Goal scope preview", description: "Literal goal description", priority: 0, targets: ["execute/a", "fix/b"], milestones: [{ name: "foundation", title: "Foundation", description: "Dependency boundary", items: ["fix/b"], acceptance_criteria: ["Owner proof"], depends_on: ["previous"] }, { name: "minimal", title: "" }] } });
    expect(screen.getByText("Goal scope preview")).toBeVisible(); expect(screen.getByText("priority").parentElement).toHaveTextContent("0"); expect(screen.getByText("milestones").parentElement).toHaveTextContent("2");
    expect(screen.getByText("Targets: execute/a, fix/b")).toBeVisible(); expect(screen.getByText("1 member item(s): fix/b")).toBeVisible(); expect(screen.getByText("Depends on: previous")).toBeVisible(); expect(screen.getByText("Owner proof")).toBeVisible(); expect(screen.getByText("minimal", { selector: "p" })).toBeVisible(); expect(screen.queryByRole("button")).toBeNull();
  });
  it("keeps a minimal goal readable through its name without fabricated targets or milestones", () => {
    mount({ id: "fixture-empty-goal", op: "create_goal", goal: { name: "fixture-named", title: "" } });
    expect(screen.getByText("fixture-named", { selector: "p" })).toBeVisible(); expect(screen.queryByText(/^Targets:/)).toBeNull(); expect(screen.queryByText("priority")).toBeNull(); expect(screen.queryByText(/Acceptance criteria/)).toBeNull();
  });
  it("renders an updated milestone as its complete supported incoming scope", () => {
    mount({ id: "fixture-update-milestone", op: "update_milestone", milestone_name: "proof", goal_milestone: { name: "proof", title: "Updated proof", description: "Owner-revised description", items: ["execute/a", "execute/b"], acceptance_criteria: ["Read every payload"], depends_on: ["foundation"] } });
    expect(screen.getByText("Updated proof")).toBeVisible(); expect(screen.getByText("Owner-revised description")).toBeVisible(); expect(screen.getByText("2 member item(s): execute/a, execute/b")).toBeVisible(); expect(screen.getByText("Read every payload")).toBeVisible(); expect(screen.queryByText(/current values unavailable/)).toBeNull();
  });
  it("distinguishes explicit array clears from unchanged fields and preserves supplied base context", () => {
    mount({ id: "fixture-clear", op: "update_item", target: "execute/fixture", patch: { tags: [], acceptance_deny: [], priority: 0 } }, { patch: { tags: ["old tag"], acceptance_deny: ["protected/**"], priority: 4 } });
    expect(screen.getByText("3 fields change")).toBeVisible(); expect(screen.getAllByText("Cleared — this field is emptied, not left alone.")).toHaveLength(2); expect(screen.getByText("old tag")).toBeVisible(); expect(screen.getByText("protected/**")).toBeVisible(); expect(screen.getByText("4")).toBeVisible(); expect(screen.getByText("0")).toBeVisible(); expect(screen.getByText(/Unchanged:/)).toHaveTextContent("description");
  });
  it("shows zero incoming priority with unavailable current context rather than treating zero as absent", () => {
    mount({ id: "fixture-zero", op: "change_priority", target: "execute/fixture", priority: 0 });
    expect(screen.getByText("current value unavailable")).toBeVisible(); expect(screen.getByText("0")).toBeVisible(); expect(screen.queryByText("unset")).toBeNull();
  });
  it("distinguishes an empty previous milestone from explicit detachment", () => {
    mount({ id: "fixture-detach", op: "move_milestone", target: "execute/fixture", milestone: "" }, { milestone: "" });
    expect(screen.getByText("unset")).toBeVisible(); expect(screen.getByText("detached from milestone")).toBeVisible(); expect(screen.queryByText("current value unavailable")).toBeNull();
  });
  it("lists plural goal-target removals with exact targets instead of empty or inferred payload", () => {
    mount({ id: "fixture-targets", op: "remove_goal_target", target: "fixture-goal", targets: ["execute/first", "fix/second"] });
    expect(screen.getByText("Removing 2 entries")).toBeVisible(); expect(screen.getByText("execute/first")).toBeVisible(); expect(screen.getByText("fix/second")).toBeVisible(); expect(screen.queryByText(/carries no payload/)).toBeNull();
  });
  it("warns about missing goal-target payload instead of displaying a silent removal", () => {
    mount({ id: "fixture-missing-targets", op: "remove_goal_target", target: "fixture-goal" });
    expect(screen.getByText(/carries no payload\. Applying it may do nothing\./)).toBeVisible(); expect(screen.queryByText(/^Removing /)).toBeNull();
  });
  it("renders two split children with exact refs and explicit source/dependency warning", () => {
    mount({ id: "fixture-split", op: "split_item", target: "execute/source", into: [{ kind: "execute", name: "child-a", title: "Child A" }, { kind: "fix", name: "child-b", title: "Child B", note: "Retained child note" }] });
    expect(screen.getByText("2 items created")).toBeVisible(); expect(screen.getByText("execute/child-a")).toBeVisible(); expect(screen.getByText("fix/child-b")).toBeVisible(); expect(screen.getByText(/1 source item will be archived/)).toBeVisible(); expect(screen.getByText(/Dependents of the source are not retargeted automatically/)).toBeVisible(); expect(screen.getByText("Note: Retained child note")).toBeVisible();
  });
  it("shows exact destructive milestone detachment intent and supplied owner title without acting", () => {
    mount({ id: "fixture-archive-display", op: "archive_milestone", target: "fixture-goal/proof", milestone_name: "proof", detach_open: true }, { title: "Owner milestone title" });
    expect(screen.getByText("destructive")).toBeVisible(); expect(screen.getByText("Owner milestone title")).toBeVisible(); expect(screen.getByText(/open member items will be detached first/)).toBeVisible(); expect(screen.queryByText(/Archiving fails if/)).toBeNull(); expect(screen.getByText(/Reversible with the unarchive endpoint/)).toBeVisible(); expect(screen.queryByRole("button")).toBeNull();
  });
});
