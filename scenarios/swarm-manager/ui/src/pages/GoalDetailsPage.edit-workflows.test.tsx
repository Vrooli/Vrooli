import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { GoalDetailsPage } from "./GoalDetailsPage";
import { defaultApiClient } from "../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import { useBacklogStore } from "../stores";
import type { GoalWithScope } from "../types/goal";

// The page, edit drawer, queries/mutation and goal-service translation are real.
// Only network methods are spied; unknown reads and alternate writes refuse.
const initial: GoalWithScope = {
  goal: { name: "owner-goal", title: "Original outcome", description: "Original description", status: "active", priority: 4, targets: [], milestones: [], seeded: false, scopeHistory: [], created: "2026-01-28T00:00:00Z", updated: "2026-01-28T00:00:00Z" },
  eta: null,
  scope: { targets: [], closure: [], completed: [], ready: [], blocked: [], total: 0, completedCount: 0, blockedCount: 0, progressPct: 0 },
};
const clients: QueryClient[] = [];
const unexpectedReads: string[] = [];
const permittedWrites = new Set<string>();
let current: GoalWithScope;
function mount() {
  const client = createTestQueryClient(); clients.push(client);
  const view = renderWithProviders(<Routes><Route path="/goals/:name" element={<GoalDetailsPage />} /></Routes>, { queryClient: client, initialEntries: ["/goals/owner-goal"] });
  return { ...view, client };
}
async function edit() {
  await screen.findByText("Original outcome"); fireEvent.click(screen.getByTestId("detail-header-actions")); fireEvent.click(await screen.findByTestId("goal-edit"));
  return screen.findByRole("dialog", { name: "Edit goal" });
}
function change(dialog: HTMLElement, title = "Updated outcome", description = "Updated evidence scope", priority = "7") {
  fireEvent.change(within(dialog).getByRole("textbox", { name: "Title" }), { target: { value: title } });
  fireEvent.change(within(dialog).getByRole("textbox", { name: "Description" }), { target: { value: description } });
  fireEvent.change(within(dialog).getByRole("spinbutton", { name: "Priority" }), { target: { value: priority } });
}
function alternateWritesAbsent() { if (!permittedWrites.has("post")) expect(defaultApiClient.post).not.toHaveBeenCalled(); if (!permittedWrites.has("patch")) expect(defaultApiClient.patch).not.toHaveBeenCalled(); if (!permittedWrites.has("delete")) expect(defaultApiClient.delete).not.toHaveBeenCalled(); }
beforeEach(() => {
  current = structuredClone(initial); unexpectedReads.length = 0; permittedWrites.clear(); useBacklogStore.getState().setItems([]);
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/goals/owner-goal") return current;
    if (path === "/next-actions/feed") return { entries: [] };
    unexpectedReads.push(path); throw new Error(`Unexpected fixture read ${path}`);
  });
  vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture post"));
  vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new Error("Unexpected fixture patch"));
  vi.spyOn(defaultApiClient, "delete").mockRejectedValue(new Error("Unexpected fixture delete"));
});
afterEach(() => { alternateWritesAbsent(); clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); useBacklogStore.getState().setItems([]); expect(unexpectedReads).toEqual([]); });

describe("GoalDetailsPage exact owner edit workflows", () => {
  it("initializes the editor from the owner response and cancels local edits without network or cache effects", async () => {
    const put = vi.spyOn(defaultApiClient, "put").mockRejectedValue(new Error("Unexpected fixture update")); const { client } = mount(); const dialog = await edit();
    expect(within(dialog).getByRole("textbox", { name: "Title" })).toHaveValue(initial.goal.title); expect(within(dialog).getByRole("textbox", { name: "Description" })).toHaveValue(initial.goal.description); expect(within(dialog).getByRole("spinbutton", { name: "Priority" })).toHaveValue(4);
    const cached = client.getQueryData(["goal", initial.goal.name]); change(dialog); fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Edit goal" })).toBeNull()); expect(put).not.toHaveBeenCalled(); expect(client.getQueryData(["goal", initial.goal.name])).toBe(cached);
    const reopened = await edit(); expect(within(reopened).getByRole("textbox", { name: "Title" })).toHaveValue(initial.goal.title); expect(within(reopened).getByRole("spinbutton", { name: "Priority" })).toHaveValue(4);
  });
  it.each(["", "   "])("refuses an empty title %j without a request or optimistic cache update", async title => {
    const put = vi.spyOn(defaultApiClient, "put").mockRejectedValue(new Error("Unexpected fixture update")); const { client } = mount(); const dialog = await edit(); const cached = client.getQueryData(["goal", initial.goal.name]);
    change(dialog, title); const save = within(dialog).getByRole("button", { name: "Save goal" }); expect(save).toBeDisabled(); fireEvent.click(save); await act(async () => {});
    expect(put).not.toHaveBeenCalled(); expect(client.getQueryData(["goal", initial.goal.name])).toBe(cached); expect(dialog).toBeVisible();
  });
  it("saves the exact current goal payload, blocks duplicate save and cancellation until owner completion, then refreshes", async () => {
    let resolve!: (value: unknown) => void; const put = vi.spyOn(defaultApiClient, "put").mockImplementation(() => new Promise(r => { resolve = r; })); const { client } = mount(); const dialog = await edit();
    const other = { ...initial, goal: { ...initial.goal, name: "other-goal", title: "Other cached goal" } };
    // Retain only this unobserved fixture sentinel; default test gcTime is zero.
    client.setQueryDefaults(["goal", "other-goal"], { gcTime: Infinity }); client.setQueryData(["goal", "other-goal"], other);
    change(dialog); fireEvent.click(within(dialog).getByRole("button", { name: "Save goal" }));
    await waitFor(() => expect(put.mock.calls).toEqual([["/goals/owner-goal", { title: "Updated outcome", description: "Updated evidence scope", priority: 7 }]]));
    expect(within(dialog).getByRole("button", { name: "Saving..." })).toBeDisabled(); expect(within(dialog).getByRole("button", { name: "Cancel" })).toBeDisabled();
    fireEvent.click(within(dialog).getByRole("button", { name: "Saving..." })); expect(put.mock.calls).toHaveLength(1); expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.title).toBe(initial.goal.title);
    current = { ...current, goal: { ...current.goal, title: "Updated outcome", description: "Updated evidence scope", priority: 7 } }; await act(async () => resolve(current));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Edit goal" })).toBeNull()); expect(await screen.findByText("Updated outcome")).toBeVisible();
    await waitFor(() => expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.priority).toBe(7)); expect(client.getQueryData(["goal", "other-goal"])).toBe(other);
    expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/goals/owner-goal").length).toBeGreaterThan(1);
  });
  it("retains draft input and owner cache on refused update without claiming success", async () => {
    const put = vi.spyOn(defaultApiClient, "put").mockRejectedValue(new Error("Goal revision refused")); const { client } = mount(); const dialog = await edit(); const cached = client.getQueryData(["goal", initial.goal.name]);
    change(dialog); fireEvent.click(within(dialog).getByRole("button", { name: "Save goal" })); expect(await within(dialog).findByRole("alert")).toHaveTextContent("Goal revision refused");
    expect(within(dialog).getByRole("textbox", { name: "Title" })).toHaveValue("Updated outcome"); expect(within(dialog).getByRole("textbox", { name: "Description" })).toHaveValue("Updated evidence scope"); expect(within(dialog).getByRole("spinbutton", { name: "Priority" })).toHaveValue(7); expect(within(dialog).getByRole("button", { name: "Save goal" })).toBeEnabled();
    expect(client.getQueryData(["goal", initial.goal.name])).toBe(cached); expect(put.mock.calls).toEqual([["/goals/owner-goal", { title: "Updated outcome", description: "Updated evidence scope", priority: 7 }]]); expect(screen.queryByText("Saved goal")).toBeNull();
  });
  it("allows an explicit corrected retry after refusal and refreshes from the owner rather than the local draft", async () => {
    const put = vi.spyOn(defaultApiClient, "put").mockRejectedValueOnce(new Error("Expected version refused")).mockImplementation(async () => {
      current = { ...current, goal: { ...current.goal, title: "Owner canonical title", description: "Owner canonical description", priority: 6 } }; return current;
    }); const { client } = mount(); const dialog = await edit(); change(dialog); fireEvent.click(within(dialog).getByRole("button", { name: "Save goal" })); expect(await within(dialog).findByRole("alert")).toHaveTextContent("Expected version refused");
    change(dialog, "Corrected outcome", "Corrected evidence", "6"); fireEvent.click(within(dialog).getByRole("button", { name: "Save goal" }));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Edit goal" })).toBeNull()); expect(await screen.findByText("Owner canonical title")).toBeVisible();
    expect(put.mock.calls).toEqual([["/goals/owner-goal", { title: "Updated outcome", description: "Updated evidence scope", priority: 7 }], ["/goals/owner-goal", { title: "Corrected outcome", description: "Corrected evidence", priority: 6 }]]);
    await waitFor(() => expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.title).toBe("Owner canonical title"));
  });
});

// Real lifecycle drawers/buttons and service translation. Allowed writes below
// are explicit per test; the original edit cases retain their zero-alternate-write checks.
describe("GoalDetailsPage lifecycle and explicit target actions", () => {
  async function menu(id: string) {
    await screen.findByText(initial.goal.title);
    fireEvent.click(screen.getByTestId("detail-header-actions"));
    fireEvent.click(await screen.findByTestId(id));
  }
  it.each([0, 10])("keeps priority at the authored limit %i without a write", async priority => {
    current.goal.priority = priority;
    const put = vi.spyOn(defaultApiClient, "put").mockRejectedValue(new Error("Unexpected limit update"));
    const { client } = mount(); await screen.findByText(initial.goal.title);
    const cache = client.getQueryData(["goal", initial.goal.name]);
    const button = screen.getByRole("button", { name: priority === 0 ? "Lower goal priority" : "Raise goal priority" });
    expect(button).toBeDisabled(); fireEvent.click(button); await act(async () => {});
    expect(put).not.toHaveBeenCalled(); expect(client.getQueryData(["goal", initial.goal.name])).toBe(cache);
  });
  it("changes priority only on owner completion and prevents duplicate pending actions", async () => {
    let resolve!: (value: unknown) => void;
    const put = vi.spyOn(defaultApiClient, "put").mockImplementation(() => new Promise(r => { resolve = r; }));
    const { client } = mount(); await screen.findByText(initial.goal.title);
    fireEvent.click(screen.getByRole("button", { name: "Raise goal priority" }));
    await waitFor(() => expect(put.mock.calls).toEqual([["/goals/owner-goal", { priority: 5 }]]));
    expect(screen.getByRole("button", { name: "Raise goal priority" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Lower goal priority" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Raise goal priority" })); expect(put).toHaveBeenCalledOnce();
    expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.priority).toBe(4);
    current = { ...current, goal: { ...current.goal, priority: 5 } }; await act(async () => resolve(current));
    await waitFor(() => expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.priority).toBe(5));
    expect(screen.getByRole("button", { name: "Lower goal priority" })).toBeEnabled();
  });
  it("cancels archive without changing owner cache or issuing a write", async () => {
    const { client } = mount(); await menu("goal-archive"); const cache = client.getQueryData(["goal", initial.goal.name]);
    const dialog = screen.getByRole("alertdialog", { name: "Archive goal" });
    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull()); expect(client.getQueryData(["goal", initial.goal.name])).toBe(cache);
  });
  it("holds the archive dialog and original owner state on refusal", async () => {
    permittedWrites.add("patch"); vi.mocked(defaultApiClient.patch).mockRejectedValue(new Error("Current goal archive refused"));
    const { client } = mount(); await menu("goal-archive"); const cache = client.getQueryData(["goal", initial.goal.name]);
    fireEvent.click(screen.getByTestId("goal-archive-confirm-submit"));
    const dialog = screen.getByRole("alertdialog", { name: "Archive goal" });
    await within(dialog).findByText("Current goal archive refused");
    expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/goals/owner-goal/archive-item", {}]]);
    expect(client.getQueryData(["goal", initial.goal.name])).toBe(cache); expect(dialog).toBeVisible();
  });
  it("archives the exact current goal once and refreshes only after the owner responds", async () => {
    permittedWrites.add("patch"); let resolve!: (value: unknown) => void;
    vi.mocked(defaultApiClient.patch).mockImplementation(() => new Promise(r => { resolve = r; }));
    const { client } = mount(); await menu("goal-archive"); fireEvent.click(screen.getByTestId("goal-archive-confirm-submit"));
    await waitFor(() => expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/goals/owner-goal/archive-item", {}]]));
    expect(screen.getByTestId("goal-archive-confirm-submit")).toBeDisabled();
    expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.status).toBe("active");
    current = { ...current, goal: { ...current.goal, status: "archived" } }; await act(async () => resolve({}));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
    await waitFor(() => expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.status).toBe("archived"));
  });
  it("requires the exact goal name for deletion and cancellation has zero owner effects", async () => {
    const { client } = mount(); await menu("goal-delete"); const cache = client.getQueryData(["goal", initial.goal.name]);
    const dialog = screen.getByRole("alertdialog", { name: "Delete goal" });
    fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "other-goal" } });
    expect(screen.getByTestId("goal-delete-confirm-submit")).toBeDisabled(); fireEvent.click(screen.getByTestId("goal-delete-confirm-submit"));
    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull()); expect(defaultApiClient.delete).not.toHaveBeenCalled(); expect(client.getQueryData(["goal", initial.goal.name])).toBe(cache);
  });
  it("retains typed delete confirmation and current owner cache when deletion is refused", async () => {
    permittedWrites.add("delete"); vi.mocked(defaultApiClient.delete).mockRejectedValue(new Error("Goal deletion refused"));
    const { client } = mount(); await menu("goal-delete"); const cache = client.getQueryData(["goal", initial.goal.name]);
    const dialog = screen.getByRole("alertdialog", { name: "Delete goal" }); fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: initial.goal.name } }); fireEvent.click(screen.getByTestId("goal-delete-confirm-submit"));
    await within(dialog).findByText("Goal deletion refused"); expect(vi.mocked(defaultApiClient.delete).mock.calls).toEqual([["/goals/owner-goal"]]); expect(within(dialog).getByRole("textbox")).toHaveValue(initial.goal.name); expect(client.getQueryData(["goal", initial.goal.name])).toBe(cache);
  });
  it("filters archived and already selected targets, and closes cancellation without writes", async () => {
    useBacklogStore.getState().setItems([
      { kind: "idea", name: "chosen", title: "Already chosen", description: "", status: "backlog", priority: 1, tags: [], suggestedSkills: [], created: "", updated: "" },
      { kind: "idea", name: "archived", title: "Archived target", description: "", status: "backlog", priority: 1, tags: [], suggestedSkills: [], created: "", updated: "", archivedAt: "2026-01-28" },
      { kind: "execute", name: "eligible", title: "Eligible target", description: "", status: "ready", priority: 1, tags: [], suggestedSkills: [], created: "", updated: "" },
    ]); current.goal.targets = ["idea/chosen"];
    const { client } = mount(); await screen.findByText(initial.goal.title); const cache = client.getQueryData(["goal", initial.goal.name]);
    fireEvent.click(within(screen.getByTestId("goal-targets")).getByRole("button", { name: "Add" }));
    const dialog = screen.getByRole("dialog", { name: "Add target" }); expect(within(dialog).queryByText("Already chosen")).toBeNull(); expect(within(dialog).queryByText("Archived target")).toBeNull(); expect(within(dialog).getByText("Eligible target")).toBeVisible();
    fireEvent.click(within(dialog).getByRole("button", { name: "Close drawer" }));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Add target" })).toBeNull()); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(client.getQueryData(["goal", initial.goal.name])).toBe(cache);
  });
});

describe("GoalDetailsPage milestone owner completion", () => {
  async function newMilestone() {
    await screen.findByText(initial.goal.title); fireEvent.click(screen.getByTestId("goal-details-tab-milestones"));
    fireEvent.click(within(screen.getByTestId("goal-milestones")).getByRole("button", { name: "Add" }));
    return screen.getByRole("dialog", { name: "Add milestone" });
  }
  function fillMilestone(dialog: HTMLElement) {
    fireEvent.change(within(dialog).getByRole("textbox", { name: "Name" }), { target: { value: "acceptance" } });
    fireEvent.change(within(dialog).getByRole("textbox", { name: "Title" }), { target: { value: "Owner acceptance" } });
    fireEvent.change(within(dialog).getByRole("textbox", { name: "Description" }), { target: { value: "Retain original scope" } });
  }
  it("refuses incomplete milestone drafts and cancels a complete local draft without writes", async () => {
    mount(); const dialog = await newMilestone(); expect(within(dialog).getByRole("button", { name: "Save milestone" })).toBeDisabled();
    fireEvent.change(within(dialog).getByRole("textbox", { name: "Name" }), { target: { value: "acceptance" } }); expect(within(dialog).getByRole("button", { name: "Save milestone" })).toBeDisabled();
    fillMilestone(dialog); fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Add milestone" })).toBeNull()); expect(defaultApiClient.post).not.toHaveBeenCalled();
  });
  it("creates the authored milestone payload, blocks duplicate save, and refreshes the owner response", async () => {
    permittedWrites.add("post"); let resolve!: (value: unknown) => void;
    vi.mocked(defaultApiClient.post).mockImplementation(() => new Promise(r => { resolve = r; }));
    const { client } = mount(); const dialog = await newMilestone(); fillMilestone(dialog); fireEvent.click(within(dialog).getByRole("button", { name: "Save milestone" }));
    const milestone = { name: "acceptance", title: "Owner acceptance", description: "Retain original scope", items: [], acceptanceCriteria: [], dependsOn: [] };
    await waitFor(() => expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/goals/owner-goal/milestones", milestone]]));
    expect(within(dialog).getByRole("button", { name: "Saving..." })).toBeDisabled(); expect(within(dialog).getByRole("button", { name: "Cancel" })).toBeDisabled();
    fireEvent.click(within(dialog).getByRole("button", { name: "Saving..." })); expect(defaultApiClient.post).toHaveBeenCalledOnce();
    expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.milestones).toEqual([]);
    current = { ...current, goal: { ...current.goal, milestones: [milestone] } }; await act(async () => resolve(current));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Add milestone" })).toBeNull()); expect(await screen.findByText("Owner acceptance")).toBeVisible();
  });
  it("retains the complete milestone draft and original scope when creation is refused", async () => {
    permittedWrites.add("post"); vi.mocked(defaultApiClient.post).mockRejectedValue(new Error("Milestone revision refused"));
    const { client } = mount(); const dialog = await newMilestone(); const cache = client.getQueryData(["goal", initial.goal.name]); fillMilestone(dialog); fireEvent.click(within(dialog).getByRole("button", { name: "Save milestone" }));
    expect(await within(dialog).findByRole("alert")).toHaveTextContent("Milestone revision refused"); expect(within(dialog).getByRole("textbox", { name: "Title" })).toHaveValue("Owner acceptance"); expect(client.getQueryData(["goal", initial.goal.name])).toBe(cache); expect(defaultApiClient.post).toHaveBeenCalledOnce();
  });
});

describe("GoalDetailsPage target and milestone assignment", () => {
  const milestone = { name: "acceptance", title: "Owner milestone", description: "Original criteria", items: ["idea/original"], acceptanceCriteria: [], dependsOn: [] };
  function withMilestone() { current.goal.milestones = [structuredClone(milestone)]; current.scope.closure = ["idea/original", "fix/next"]; }
  async function milestones() {
    await screen.findByText(initial.goal.title); fireEvent.click(screen.getByTestId("goal-details-tab-milestones")); return screen.findByText("Owner milestone");
  }
  it("opens an existing milestone with immutable name and cancels local edits without writes", async () => {
    withMilestone(); const put = vi.spyOn(defaultApiClient, "put").mockRejectedValue(new Error("Unexpected milestone write")); const { client } = mount(); await milestones();
    const before = client.getQueryData(["goal", initial.goal.name]); fireEvent.click(screen.getByRole("button", { name: "Edit Owner milestone" })); const dialog = await screen.findByRole("dialog", { name: "Edit milestone" });
    expect(within(dialog).getByRole("textbox", { name: "Name" })).toBeDisabled(); expect(within(dialog).getByRole("textbox", { name: "Title" })).toHaveValue("Owner milestone");
    fireEvent.change(within(dialog).getByRole("textbox", { name: "Title" }), { target: { value: "Local revision" } }); fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Edit milestone" })).toBeNull()); expect(put).not.toHaveBeenCalled(); expect(client.getQueryData(["goal", initial.goal.name])).toBe(before);
  });
  it("updates only the existing milestone's full authored record and refreshes after success", async () => {
    withMilestone(); let resolve!: (v: unknown) => void; const put = vi.spyOn(defaultApiClient, "put").mockImplementation(() => new Promise(r => { resolve = r; })); const { client } = mount(); await milestones();
    fireEvent.click(screen.getByRole("button", { name: "Edit Owner milestone" })); const dialog = await screen.findByRole("dialog", { name: "Edit milestone" });
    fireEvent.change(within(dialog).getByRole("textbox", { name: "Title" }), { target: { value: "Owner revised milestone" } }); fireEvent.change(within(dialog).getByRole("textbox", { name: "Description" }), { target: { value: "Exact revised evidence" } }); fireEvent.click(within(dialog).getByRole("button", { name: "Save milestone" }));
    const revised = { ...milestone, title: "Owner revised milestone", description: "Exact revised evidence" };
    await waitFor(() => expect(put.mock.calls).toEqual([["/goals/owner-goal/milestones/acceptance", revised]])); expect(within(dialog).getByRole("button", { name: "Cancel" })).toBeDisabled(); expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.milestones[0]?.title).toBe("Owner milestone");
    current.goal.milestones = [revised]; await act(async () => resolve(current)); await waitFor(() => expect(screen.queryByRole("dialog", { name: "Edit milestone" })).toBeNull()); expect(await screen.findByText("Owner revised milestone")).toBeVisible();
  });
  it("retains refused milestone changes and owner cache without alternate writes", async () => {
    withMilestone(); const put = vi.spyOn(defaultApiClient, "put").mockRejectedValue(new Error("Milestone owner refused")); const { client } = mount(); await milestones(); const before = client.getQueryData(["goal", initial.goal.name]);
    fireEvent.click(screen.getByRole("button", { name: "Edit Owner milestone" })); const dialog = await screen.findByRole("dialog", { name: "Edit milestone" }); fireEvent.change(within(dialog).getByRole("textbox", { name: "Description" }), { target: { value: "Draft acceptance" } }); fireEvent.click(within(dialog).getByRole("button", { name: "Save milestone" }));
    expect(await within(dialog).findByRole("alert")).toHaveTextContent("Milestone owner refused"); expect(within(dialog).getByRole("textbox", { name: "Description" })).toHaveValue("Draft acceptance"); expect(client.getQueryData(["goal", initial.goal.name])).toBe(before); expect(put).toHaveBeenCalledOnce();
  });
  it("archives exactly one existing milestone only after the owner responds", async () => {
    withMilestone(); permittedWrites.add("delete"); let resolve!: (v: unknown) => void; vi.mocked(defaultApiClient.delete).mockImplementation(() => new Promise(r => { resolve = r; })); const { client } = mount(); await milestones(); const before = client.getQueryData(["goal", initial.goal.name]);
    fireEvent.click(screen.getByRole("button", { name: "Archive Owner milestone" })); await waitFor(() => expect(vi.mocked(defaultApiClient.delete).mock.calls).toEqual([["/goals/owner-goal/milestones/acceptance"]])); expect(client.getQueryData(["goal", initial.goal.name])).toBe(before);
    current.goal.milestones = [{ ...milestone, archivedAt: "2026-01-28T01:00:00Z" }]; await act(async () => resolve(current)); await waitFor(() => expect(screen.queryByText("Owner milestone")).toBeNull()); expect(defaultApiClient.delete).toHaveBeenCalledOnce();
  });
  it("retains a refused milestone archive without replacing the cached owner record", async () => {
    withMilestone(); permittedWrites.add("delete"); vi.mocked(defaultApiClient.delete).mockRejectedValue(new Error("Milestone archive denied")); const { client } = mount(); await milestones(); const before = client.getQueryData(["goal", initial.goal.name]); fireEvent.click(screen.getByRole("button", { name: "Archive Owner milestone" }));
    expect(await screen.findByText("Milestone archive denied")).toBeVisible(); expect(screen.getByText("Owner milestone")).toBeVisible(); expect(client.getQueryData(["goal", initial.goal.name])).toBe(before); expect(defaultApiClient.delete).toHaveBeenCalledOnce();
  });
  it("cancels changed milestone membership without saving or invalidating the goal", async () => {
    withMilestone(); const { client } = mount(); await milestones(); const before = client.getQueryData(["goal", initial.goal.name]); fireEvent.click(screen.getByRole("button", { name: "Manage Owner milestone items" })); const dialog = await screen.findByRole("dialog", { name: "Manage milestone items" });
    expect(within(dialog).getByRole("checkbox", { name: "idea/original" })).toBeChecked(); fireEvent.click(within(dialog).getByRole("checkbox", { name: "fix/next" })); fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" })); await waitFor(() => expect(screen.queryByRole("dialog", { name: "Manage milestone items" })).toBeNull()); expect(client.getQueryData(["goal", initial.goal.name])).toBe(before);
  });
  it("applies only the membership delta, in order, and prevents duplicate pending saves", async () => {
    withMilestone(); permittedWrites.add("post"); permittedWrites.add("delete"); let resolveAdd!: (v: unknown) => void; vi.mocked(defaultApiClient.post).mockImplementation(() => new Promise(r => { resolveAdd = r; })); vi.mocked(defaultApiClient.delete).mockImplementation(async () => { current.goal.milestones = [{ ...milestone, items: ["fix/next"] }]; return current; });
    const { client } = mount(); await milestones(); fireEvent.click(screen.getByRole("button", { name: "Manage Owner milestone items" })); const dialog = await screen.findByRole("dialog", { name: "Manage milestone items" }); fireEvent.click(within(dialog).getByRole("checkbox", { name: "idea/original" })); fireEvent.click(within(dialog).getByRole("checkbox", { name: "fix/next" })); fireEvent.click(within(dialog).getByRole("button", { name: "Save items" }));
    await waitFor(() => expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/goals/owner-goal/milestones/acceptance/items", { targets: ["fix/next"] }]])); expect(defaultApiClient.delete).not.toHaveBeenCalled(); expect(within(dialog).getByRole("button", { name: "Saving..." })).toBeDisabled(); fireEvent.click(within(dialog).getByRole("button", { name: "Saving..." })); expect(defaultApiClient.post).toHaveBeenCalledOnce();
    expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.milestones[0]?.items).toEqual(["idea/original"]); await act(async () => resolveAdd(current)); await waitFor(() => expect(screen.queryByRole("dialog", { name: "Manage milestone items" })).toBeNull()); expect(vi.mocked(defaultApiClient.delete).mock.calls).toEqual([["/goals/owner-goal/milestones/acceptance/items", { targets: ["idea/original"] }]]);
  });
  it("stops the remove phase after a refused membership addition and keeps the draft", async () => {
    withMilestone(); permittedWrites.add("post"); vi.mocked(defaultApiClient.post).mockRejectedValue(new Error("Assignment admission refused")); const { client } = mount(); await milestones(); const before = client.getQueryData(["goal", initial.goal.name]); fireEvent.click(screen.getByRole("button", { name: "Manage Owner milestone items" })); const dialog = await screen.findByRole("dialog", { name: "Manage milestone items" }); fireEvent.click(within(dialog).getByRole("checkbox", { name: "idea/original" })); fireEvent.click(within(dialog).getByRole("checkbox", { name: "fix/next" })); fireEvent.click(within(dialog).getByRole("button", { name: "Save items" }));
    expect(await within(dialog).findByRole("alert")).toHaveTextContent("Assignment admission refused"); await waitFor(() => expect(defaultApiClient.post).toHaveBeenCalledOnce()); await waitFor(() => expect(within(dialog).getByRole("button", { name: "Save items" })).toBeEnabled()); expect(dialog).toBeVisible(); expect(within(dialog).getByRole("checkbox", { name: "fix/next" })).toBeChecked(); expect(defaultApiClient.delete).not.toHaveBeenCalled(); expect(client.getQueryData(["goal", initial.goal.name])).toBe(before);
  });
  function targetChoices() {
    useBacklogStore.getState().setItems([{ kind: "fix", name: "next", title: "Next owner item", description: "", status: "backlog", priority: 1, tags: [], suggestedSkills: [], created: "2026-01-28T00:00:00Z", updated: "2026-01-28T00:00:00Z" }]);
  }
  async function targets() { await screen.findByText(initial.goal.title); fireEvent.click(within(screen.getByTestId("goal-targets")).getByRole("button", { name: "Add" })); return screen.findByRole("dialog", { name: "Add target" }); }
  it("cancels target selection without creating a target or invalidating the goal", async () => {
    targetChoices(); const { client } = mount(); const dialog = await targets(); const before = client.getQueryData(["goal", initial.goal.name]); expect(within(dialog).getByRole("button", { name: /Next owner item/ })).toBeVisible(); fireEvent.click(within(dialog).getByRole("button", { name: "Close drawer" })); await waitFor(() => expect(screen.queryByRole("dialog", { name: "Add target" })).toBeNull()); expect(client.getQueryData(["goal", initial.goal.name])).toBe(before);
  });
  it("adds exactly the selected backlog reference and closes only after owner acceptance", async () => {
    targetChoices(); permittedWrites.add("post"); let resolve!: (v: unknown) => void; vi.mocked(defaultApiClient.post).mockImplementation(() => new Promise(r => { resolve = r; })); const { client } = mount(); const dialog = await targets(); fireEvent.click(within(dialog).getByRole("button", { name: /Next owner item/ })); await waitFor(() => expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/goals/owner-goal/targets", { targets: ["fix/next"] }]]));
    expect(within(dialog).getByRole("button", { name: /Next owner item/ })).toBeDisabled(); expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.targets).toEqual([]); current.goal.targets = ["fix/next"]; await act(async () => resolve(current)); await waitFor(() => expect(screen.queryByRole("dialog", { name: "Add target" })).toBeNull()); expect(await screen.findByRole("button", { name: "Remove fix/next from goal" })).toBeVisible();
  });
  it("retains refused target selection with no optimistic target or alternate write", async () => {
    targetChoices(); permittedWrites.add("post"); vi.mocked(defaultApiClient.post).mockRejectedValue(new Error("Target owner refused")); const { client } = mount(); const dialog = await targets(); const before = client.getQueryData(["goal", initial.goal.name]); fireEvent.click(within(dialog).getByRole("button", { name: /Next owner item/ })); expect(await within(dialog).findByRole("alert")).toHaveTextContent("Target owner refused"); expect(client.getQueryData(["goal", initial.goal.name])).toBe(before); expect(dialog).toBeVisible(); expect(defaultApiClient.post).toHaveBeenCalledOnce();
  });
  it("removes exactly one target and leaves owner state unchanged until completion", async () => {
    current.goal.targets = ["fix/next"]; permittedWrites.add("delete"); let resolve!: (v: unknown) => void; vi.mocked(defaultApiClient.delete).mockImplementation(() => new Promise(r => { resolve = r; })); const { client } = mount(); await screen.findByText(initial.goal.title); fireEvent.click(screen.getByRole("button", { name: "Remove fix/next from goal" })); await waitFor(() => expect(vi.mocked(defaultApiClient.delete).mock.calls).toEqual([["/goals/owner-goal/targets", { targets: ["fix/next"] }]])); expect(client.getQueryData<GoalWithScope>(["goal", initial.goal.name])?.goal.targets).toEqual(["fix/next"]); current.goal.targets = []; await act(async () => resolve(current)); await waitFor(() => expect(screen.queryByRole("button", { name: "Remove fix/next from goal" })).toBeNull());
  });
});
