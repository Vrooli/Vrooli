import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { BacklogDetailsPage } from "./BacklogDetailsPage";
import { defaultApiClient } from "../lib/api-client";
import { useAgentActivitiesStore, useBacklogDetailUIStore, useBacklogStore } from "../stores";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import type { BacklogItem } from "../types";

// Only the network client is replaced: page, queries, mutations, service
// translation, global stores, dialogs and provider/router composition are real.
const item: BacklogItem = { kind: "idea", name: "reviewed-item", title: "Reviewed item", description: "Owner specification", status: "backlog", priority: 1, tags: [], suggestedSkills: [], created: "2026-01-28T00:00:00Z", updated: "2026-01-28T00:00:00Z" };
const clients: QueryClient[] = [];
const unexpectedReads: string[] = [];
function Location() { const location = useLocation(); return <output data-testid="workflow-location">{location.pathname + location.search}</output>; }
function mount(path = "/backlog/idea/reviewed-item") {
  const client = createTestQueryClient(); clients.push(client);
  const view = renderWithProviders(<Routes><Route path="/backlog/:kind/:name" element={<><BacklogDetailsPage /><Location /></>} /></Routes>, { queryClient: client, initialEntries: [path] });
  return { ...view, client };
}
function reads() {
  return vi.spyOn(defaultApiClient, "get").mockImplementation(async (path) => {
    if (path === "/settings") return { settings: {} };
    if (path === "/goals") return { items: [] };
    if (path === "/backlog/idea/reviewed-item/plan-render") return { markdown: "# Owner plan", planRef: null };
    if (path === "/backlog/idea/reviewed-item") return { item };
    if (path === "/backlog/idea/reviewed-item/files") return { files: [] };
    if (path === "/backlog/idea/reviewed-item/next-action") return { action: { id: "accept_plan", compactLabel: "Accept plan", expandedLabel: "Accept plan", enabled: true, reason: "Owner acceptance required", blockers: [], target: "plan_accept", effect: "state_change" } };
    if (path === "/backlog/idea/reviewed-item/archive/targets") return { targets: [], requirements: [], has_archive: false };
    if (path === "/backlog/idea/reviewed-item/review") return { rounds: [] };
    if (path === "/execution?backlog_kind=idea&backlog_name=reviewed-item") return { items: [] };
    if (path.startsWith("/backlog?")) return { items: [], blocking: {} };
    if (path.startsWith("/agent-activities?")) return { items: [] };
    if (path === "/proposal-sessions?target_type=backlog_item&target_ref=idea%2Freviewed-item") return { sessions: [] };
    unexpectedReads.push(path);
    throw new Error(`Unexpected fixture read ${path}`);
  });
}
async function openAction(label: string) {
  await screen.findByText("Reviewed item");
  fireEvent.click(screen.getByTestId("detail-header-actions"));
  fireEvent.click(await screen.findByRole("menuitem", { name: label }));
  return screen.findByRole("alertdialog");
}
beforeEach(() => {
  unexpectedReads.length = 0;
  useBacklogStore.getState().setItems([]);
  useAgentActivitiesStore.setState({ activities: [], isRefreshing: false });
  useBacklogDetailUIStore.getState().reset();
  reads();
  vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new Error("Unexpected fixture patch"));
  vi.spyOn(defaultApiClient, "delete").mockRejectedValue(new Error("Unexpected fixture delete"));
});
afterEach(() => { clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); useBacklogStore.getState().setItems([]); useBacklogDetailUIStore.getState().reset(); expect(unexpectedReads).toEqual([]); });

describe("BacklogDetailsPage owner-bound lifecycle workflows", () => {
  it("accepts the exact plan, refreshes owner data and routes to Plan only after success", async () => {
    let resolve!: (value: unknown) => void;
    const post = vi.spyOn(defaultApiClient, "post").mockImplementation(() => new Promise(r => { resolve = r; }));
    const { client } = mount();
    fireEvent.click(await screen.findByRole("button", { name: /Accept plan/i }));
    await waitFor(() => expect(post.mock.calls).toEqual([["/backlog/idea/reviewed-item/plan-accept", {}]]));
    expect(screen.getByTestId("workflow-location").textContent).not.toContain("tab=prompt");
    const pendingButton = screen.getByRole("button", { name: /^Accept plan$/i });
    expect(pendingButton).toBeDisabled(); fireEvent.click(pendingButton);
    expect(post.mock.calls).toHaveLength(1);
    await act(async () => resolve({ plan_acceptance: { actor: "owner", plan_content_hash: "exact-hash", subject_version: "revision-1", accepted_at: "2026-01-28T00:00:00Z" } }));
    await waitFor(() => expect(screen.getByTestId("workflow-location")).toHaveTextContent("tab=prompt"));
    expect(client.getQueryData(["backlog", "idea", item.name])).toEqual(item);
    expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/backlog/idea/reviewed-item").length).toBeGreaterThan(1);
  });
  it("keeps owner acceptance refusal visible without routing, patching or deleting", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Exact revision no longer accepted"));
    mount(); fireEvent.click(await screen.findByRole("button", { name: /Accept plan/i }));
    expect(await screen.findByText("Exact revision no longer accepted")).toBeVisible();
    expect(post.mock.calls).toEqual([["/backlog/idea/reviewed-item/plan-accept", {}]]);
    expect(screen.getByTestId("workflow-location").textContent).not.toContain("tab=prompt");
    expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
  it("requires exact typed confirmation before recreating and permits cancellation with zero writes", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    mount(); const dialog = await openAction("Recreate item");
    expect(within(dialog).getByRole("button", { name: "Recreate item" })).toBeDisabled();
    fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "other-item" } });
    expect(within(dialog).getByRole("button", { name: "Recreate item" })).toBeDisabled();
    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull()); expect(post).not.toHaveBeenCalled();
  });
  it("recreates only the confirmed owner target and refreshes the item after success", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockResolvedValue({});
    mount(); const dialog = await openAction("Recreate item");
    fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: item.name } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Recreate item" }));
    await waitFor(() => expect(post.mock.calls).toEqual([["/backlog/idea/reviewed-item/recreate", {}]]));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
    await waitFor(() => expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/backlog/idea/reviewed-item").length).toBeGreaterThan(1));
  });
  it("sends exactly selected reset scopes and keeps a refused reset open for review", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Scope reset refused"));
    mount(); const dialog = await openAction("Reset derived artifacts");
    fireEvent.click(within(dialog).getByRole("checkbox", { name: "Handoff data and executions" }));
    fireEvent.click(within(dialog).getByRole("button", { name: "Reset selected artifacts" }));
    expect(await within(dialog).findByText("Scope reset refused")).toBeVisible();
    expect(post.mock.calls).toEqual([["/backlog/idea/reviewed-item/reset-artifacts", { scope: ["review", "handoff_executions"] }]]);
    expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
  it("refuses an empty reset selection without a request or cache replacement", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    const { client } = mount(); const dialog = await openAction("Reset derived artifacts");
    const before = client.getQueryData(["backlog", "idea", item.name]);
    fireEvent.click(within(dialog).getByRole("checkbox", { name: "Review rounds" }));
    fireEvent.click(within(dialog).getByRole("button", { name: "Reset selected artifacts" }));
    await act(async () => {}); expect(post).not.toHaveBeenCalled(); expect(client.getQueryData(["backlog", "idea", item.name])).toBe(before); expect(dialog).toBeVisible();
  });
  it("resets only the chosen plan binding, disables resubmission while pending, then closes and refreshes", async () => {
    let resolve!: (value: unknown) => void;
    const post = vi.spyOn(defaultApiClient, "post").mockImplementation(() => new Promise(r => { resolve = r; }));
    mount(); const dialog = await openAction("Reset derived artifacts");
    fireEvent.click(within(dialog).getByRole("checkbox", { name: "Review rounds" }));
    fireEvent.click(within(dialog).getByRole("checkbox", { name: "Plan binding" }));
    fireEvent.click(within(dialog).getByRole("button", { name: "Reset selected artifacts" }));
    await waitFor(() => expect(post.mock.calls).toEqual([["/backlog/idea/reviewed-item/reset-artifacts", { scope: ["plan_unbind"] }]]));
    expect(within(dialog).getByRole("button", { name: /Resetting|Reset selected artifacts|Processing/i })).toBeDisabled();
    await act(async () => resolve({}));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
    await waitFor(() => expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/backlog/idea/reviewed-item").length).toBeGreaterThan(1));
  });
  it("shows an invalid kind without target reads, mutations or stale cached item leakage", async () => {
    useBacklogStore.getState().setItems([item]);
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    const { client } = mount("/backlog/invalid/reviewed-item");
    expect(await screen.findByText("Invalid URL")).toBeVisible();
    expect(screen.queryByText("Reviewed item")).toBeNull();
    expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path.startsWith("/backlog/"))).toEqual([]);
    expect(post).not.toHaveBeenCalled(); expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
    expect(client.getQueryData(["backlog", "invalid", item.name])).toBeUndefined(); expect(useBacklogStore.getState().items).toEqual([item]);
  });
  it("holds recreate confirmation during the exact pending request and refreshes only after completion", async () => {
    let resolve!: (v: unknown) => void;
    const post = vi.spyOn(defaultApiClient, "post").mockImplementation(() => new Promise(r => { resolve = r; }));
    const { client } = mount(); const dialog = await openAction("Recreate item");
    const before = client.getQueryData(["backlog", "idea", item.name]);
    fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: item.name } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Recreate item" }));
    await waitFor(() => expect(post.mock.calls).toEqual([["/backlog/idea/reviewed-item/recreate", {}]]));
    const pending = within(dialog).getByRole("button", { name: /Processing|Recreating|Recreate item/i });
    expect(pending).toBeDisabled(); fireEvent.click(pending); expect(post).toHaveBeenCalledTimes(1);
    expect(client.getQueryData(["backlog", "idea", item.name])).toBe(before);
    await act(async () => resolve({})); await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
    await waitFor(() => expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([p]) => p === "/backlog/idea/reviewed-item").length).toBeGreaterThan(1));
  });
  it("keeps rejected recreate input and owner cache intact, then permits an explicit same-target retry", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValueOnce(new Error("Owner recreate refused")).mockResolvedValue({});
    const { client } = mount(); const dialog = await openAction("Recreate item");
    const before = client.getQueryData(["backlog", "idea", item.name]);
    const confirmation = within(dialog).getByRole("textbox");
    fireEvent.change(confirmation, { target: { value: item.name } }); fireEvent.click(within(dialog).getByRole("button", { name: "Recreate item" }));
    expect(await within(dialog).findByText("Owner recreate refused")).toBeVisible(); expect(confirmation).toHaveValue(item.name);
    expect(client.getQueryData(["backlog", "idea", item.name])).toBe(before);
    expect(post.mock.calls).toEqual([["/backlog/idea/reviewed-item/recreate", {}]]);
    expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
    fireEvent.click(within(dialog).getByRole("button", { name: "Recreate item" }));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull()); expect(post.mock.calls).toEqual([["/backlog/idea/reviewed-item/recreate", {}], ["/backlog/idea/reviewed-item/recreate", {}]]);
  });
  it("cancels a populated reset without writes or cache invalidation", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    const { client } = mount(); const dialog = await openAction("Reset derived artifacts");
    const before = client.getQueryData(["backlog", "idea", item.name]);
    fireEvent.click(within(dialog).getByRole("checkbox", { name: "Plan binding" })); fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
    expect(post).not.toHaveBeenCalled(); expect(client.getQueryData(["backlog", "idea", item.name])).toBe(before);
  });
  it("retains selected reset scope and cache after refusal and retries only on a second explicit confirmation", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValueOnce(new Error("Reset grant refused")).mockResolvedValue({});
    const { client } = mount(); const dialog = await openAction("Reset derived artifacts");
    const before = client.getQueryData(["backlog", "idea", item.name]);
    fireEvent.click(within(dialog).getByRole("checkbox", { name: "Plan binding" }));
    fireEvent.click(within(dialog).getByRole("button", { name: "Reset selected artifacts" }));
    expect(await within(dialog).findByText("Reset grant refused")).toBeVisible();
    expect(within(dialog).getByRole("checkbox", { name: "Review rounds" })).toBeChecked(); expect(within(dialog).getByRole("checkbox", { name: "Plan binding" })).toBeChecked();
    expect(client.getQueryData(["backlog", "idea", item.name])).toBe(before);
    expect(post.mock.calls).toEqual([["/backlog/idea/reviewed-item/reset-artifacts", { scope: ["review", "plan_unbind"] }]]);
    fireEvent.click(within(dialog).getByRole("button", { name: "Reset selected artifacts" }));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
    expect(post.mock.calls).toEqual([["/backlog/idea/reviewed-item/reset-artifacts", { scope: ["review", "plan_unbind"] }], ["/backlog/idea/reviewed-item/reset-artifacts", { scope: ["review", "plan_unbind"] }]]);
  });
});

// Additional exact owner description and goal-membership workflows. Existing
// test bytes above are retained. Subjects/providers/hooks remain real.
import type { GoalWithScope } from "../types/goal";

describe("BacklogDetailsPage description and explicit goal membership", () => {
  function currentOwnerReads(owner: () => BacklogItem, goals: () => GoalWithScope[] = () => []) {
    const original = vi.mocked(defaultApiClient.get).getMockImplementation(); if (!original) throw new Error("Missing owner read boundary");
    vi.mocked(defaultApiClient.get).mockImplementation(async path => {
      if (path === "/backlog/idea/reviewed-item") return {item:owner()};
      if (path === "/goals") return {items:goals()};
      return original(path);
    });
  }
  async function description() { await screen.findByText("Reviewed item"); fireEvent.click(screen.getByRole("button",{name:"Edit description"})); return screen.findByRole("dialog",{name:"Edit description"}); }
  function goal(targets: string[]=[]): GoalWithScope { return {goal:{name:"owner-goal",title:"Owner goal",description:"Exact target membership",status:"active",priority:4,targets,milestones:[],seeded:false,scopeHistory:[],created:"2026-01-28T00:00:00Z",updated:"2026-01-28T00:00:00Z"},scope:{targets,closure:targets,completed:[],ready:targets,blocked:[],total:targets.length,completedCount:0,blockedCount:0,progressPct:0},eta:null}; }
  it("initializes a description draft from the exact owner and cancels without request or cache change",async()=>{
    const {client}=mount();const dialog=await description();const before=client.getQueryData(["backlog","idea",item.name]);expect(within(dialog).getByRole("textbox",{name:"Description"})).toHaveValue(item.description);fireEvent.change(within(dialog).getByRole("textbox",{name:"Description"}),{target:{value:"Local unsaved description"}});fireEvent.click(within(dialog).getByRole("button",{name:"Cancel"}));await waitFor(()=>expect(screen.queryByRole("dialog",{name:"Edit description"})).toBeNull());expect(defaultApiClient.patch).not.toHaveBeenCalled();expect(client.getQueryData(["backlog","idea",item.name])).toBe(before);
  });
  it("saves the exact description for the current item only after the owner completes, with pending duplicate/cancel guards",async()=>{
    let current=structuredClone(item);currentOwnerReads(()=>current);let complete!: (r:unknown)=>void;vi.mocked(defaultApiClient.patch).mockImplementation(()=>new Promise(r=>{complete=r;}));const {client}=mount();const dialog=await description();const before=client.getQueryData(["backlog","idea",item.name]);fireEvent.change(within(dialog).getByRole("textbox",{name:"Description"}),{target:{value:"Exact new owner description"}});fireEvent.click(within(dialog).getByRole("button",{name:"Save description"}));await waitFor(()=>expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/backlog/idea/reviewed-item",{description:"Exact new owner description"}]]));expect(within(dialog).getByRole("button",{name:"Cancel"})).toBeDisabled();expect(within(dialog).getByRole("button",{name:"Saving..."})).toBeDisabled();fireEvent.click(within(dialog).getByRole("button",{name:"Saving..."}));expect(defaultApiClient.patch).toHaveBeenCalledOnce();expect(client.getQueryData(["backlog","idea",item.name])).toBe(before);current={...current,description:"Owner canonical description"};await act(async()=>complete({item:current}));await waitFor(()=>expect(screen.queryByRole("dialog",{name:"Edit description"})).toBeNull());expect(client.getQueryData(["backlog","idea",item.name])).toEqual(current);
  });
  it("retains a refused description draft and owner cache with visible feedback and no automatic retry",async()=>{
    vi.mocked(defaultApiClient.patch).mockRejectedValue(new Error("Description owner refused"));const {client}=mount();const dialog=await description();const before=client.getQueryData(["backlog","idea",item.name]);fireEvent.change(within(dialog).getByRole("textbox",{name:"Description"}),{target:{value:"Keep authored description"}});fireEvent.click(within(dialog).getByRole("button",{name:"Save description"}));expect(await within(dialog).findByText("Description owner refused")).toBeVisible();expect(within(dialog).getByRole("textbox",{name:"Description"})).toHaveValue("Keep authored description");expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/backlog/idea/reviewed-item",{description:"Keep authored description"}]]);expect(client.getQueryData(["backlog","idea",item.name])).toBe(before);
  });
  async function attachPicker(){await screen.findByText("Reviewed item");fireEvent.click(screen.getByRole("button",{name:"Attach goal"}));const dialog=await screen.findByRole("dialog",{name:"Attach to goal"});fireEvent.click(within(dialog).getByTestId("plan-goal-picker"));return dialog;}
  it("dismisses goal selection without any target mutation or owner-cache change",async()=>{
    currentOwnerReads(()=>item,()=>[goal()]);const post=vi.spyOn(defaultApiClient,"post").mockRejectedValue(new Error("Unexpected goal target write"));const {client}=mount();const dialog=await attachPicker();await screen.findByRole("option",{name:/Owner goal/});const before=client.getQueryData(["goals"]);fireEvent.click(within(dialog).getByRole("button",{name:"Close drawer"}));await waitFor(()=>expect(screen.queryByRole("dialog",{name:"Attach to goal"})).toBeNull());expect(post).not.toHaveBeenCalled();expect(defaultApiClient.delete).not.toHaveBeenCalled();expect(client.getQueryData(["goals"])).toBe(before);
  });
  it("attaches only the exact current backlog ref and selected goal, then refreshes owner membership",async()=>{
    let currentGoal=goal();currentOwnerReads(()=>item,()=>[currentGoal]);let complete!: (r:unknown)=>void;const post=vi.spyOn(defaultApiClient,"post").mockImplementation(()=>new Promise(r=>{complete=r;}));const {client}=mount();await attachPicker();const before=client.getQueryData(["goals"]);fireEvent.click(await screen.findByRole("option",{name:/Owner goal/}));await waitFor(()=>expect(post.mock.calls).toEqual([["/goals/owner-goal/targets",{targets:["idea/reviewed-item"]}]]));expect(screen.getByRole("dialog",{name:"Attach to goal"})).toBeVisible();expect(client.getQueryData(["goals"])).toBe(before);currentGoal=goal(["idea/reviewed-item"]);await act(async()=>complete(currentGoal));await waitFor(()=>expect(screen.queryByRole("dialog",{name:"Attach to goal"})).toBeNull());expect(await screen.findByRole("button",{name:"Detach from Owner goal"})).toBeVisible();expect(defaultApiClient.patch).not.toHaveBeenCalled();expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
  it("keeps a refused goal attachment visible and preserves owner membership without alternate writes",async()=>{
    currentOwnerReads(()=>item,()=>[goal()]);const post=vi.spyOn(defaultApiClient,"post").mockRejectedValue(new Error("Goal attachment refused"));const {client}=mount();await attachPicker();const before=client.getQueryData(["goals"]);fireEvent.click(await screen.findByRole("option",{name:/Owner goal/}));expect(await screen.findByText("Goal attachment refused")).toBeVisible();expect(screen.getByRole("dialog",{name:"Attach to goal"})).toBeVisible();expect(post.mock.calls).toEqual([["/goals/owner-goal/targets",{targets:["idea/reviewed-item"]}]]);expect(client.getQueryData(["goals"])).toBe(before);expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
  async function detach(){await screen.findByRole("button",{name:"Detach from Owner goal"});fireEvent.click(screen.getByRole("button",{name:"Detach from Owner goal"}));return screen.findByRole("dialog",{name:"Detach from goal"});}
  it("cancels a detach confirmation without deleting the goal or target membership",async()=>{
    currentOwnerReads(()=>item,()=>[goal(["idea/reviewed-item"])]);mount();const dialog=await detach();fireEvent.click(within(dialog).getByRole("button",{name:"Cancel"}));await waitFor(()=>expect(screen.queryByRole("dialog",{name:"Detach from goal"})).toBeNull());expect(defaultApiClient.delete).not.toHaveBeenCalled();expect(screen.getByRole("button",{name:"Detach from Owner goal"})).toBeVisible();
  });
  it("removes only the exact target membership after owner completion with pending duplicate guards",async()=>{
    let currentGoal=goal(["idea/reviewed-item"]);currentOwnerReads(()=>item,()=>[currentGoal]);let complete!: (r:unknown)=>void;vi.mocked(defaultApiClient.delete).mockImplementation(()=>new Promise(r=>{complete=r;}));mount();const dialog=await detach();fireEvent.click(within(dialog).getByRole("button",{name:"Detach"}));await waitFor(()=>expect(vi.mocked(defaultApiClient.delete).mock.calls).toEqual([["/goals/owner-goal/targets",{targets:["idea/reviewed-item"]}]]));expect(within(dialog).getByRole("button",{name:"Cancel"})).toBeDisabled();fireEvent.click(within(dialog).getByRole("button",{name:"Detaching..."}));expect(defaultApiClient.delete).toHaveBeenCalledOnce();currentGoal=goal();await act(async()=>complete(currentGoal));await waitFor(()=>expect(screen.queryByRole("dialog",{name:"Detach from goal"})).toBeNull());expect(await screen.findByRole("button",{name:"Attach goal"})).toBeVisible();expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
  it("retains detach confirmation and membership on owner refusal without deleting the goal itself",async()=>{
    currentOwnerReads(()=>item,()=>[goal(["idea/reviewed-item"])]);vi.mocked(defaultApiClient.delete).mockRejectedValue(new Error("Goal detach refused"));const {client}=mount();const dialog=await detach();const before=client.getQueryData(["goals"]);fireEvent.click(within(dialog).getByRole("button",{name:"Detach"}));expect(await screen.findByText("Goal detach refused")).toBeVisible();expect(dialog).toBeVisible();expect(client.getQueryData(["goals"])).toBe(before);expect(vi.mocked(defaultApiClient.delete).mock.calls).toEqual([["/goals/owner-goal/targets",{targets:["idea/reviewed-item"]}]]);
  });
});
