import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
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
let current:BacklogItem;
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
    if (path === "/backlog/idea/reviewed-item") return { item:current };
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
beforeEach(() => {
  unexpectedReads.length = 0;current={...item,note:"Original owner note"};
  useBacklogStore.getState().setItems([]);
  useAgentActivitiesStore.setState({ activities: [], isRefreshing: false });
  useBacklogDetailUIStore.getState().reset();
  reads();
  vi.spyOn(defaultApiClient,"post").mockRejectedValue(new Error("Unexpected fixture post"));
  vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new Error("Unexpected fixture patch"));
  vi.spyOn(defaultApiClient, "delete").mockRejectedValue(new Error("Unexpected fixture delete"));
});
afterEach(() => { clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); useBacklogStore.getState().setItems([]); useBacklogDetailUIStore.getState().reset(); expect(unexpectedReads).toEqual([]); });


async function editNote(value:string){fireEvent.click(await screen.findByRole("button",{name:"Original owner note"}));fireEvent.change(screen.getByRole("textbox",{name:""}),{target:{value}});}
describe("BacklogDetailsPage exact owner note saves",()=>{
 it("cancels a draft without a patch or cache change",async()=>{const {client}=mount();await editNote("Unsaved owner note");const before=client.getQueryData(["backlog","idea",item.name]);fireEvent.click(screen.getByRole("button",{name:"Cancel"}));expect(screen.getByRole("button",{name:"Original owner note"})).toBeVisible();expect(defaultApiClient.patch).not.toHaveBeenCalled();expect(client.getQueryData(["backlog","idea",item.name])).toBe(before);expect(defaultApiClient.post).not.toHaveBeenCalled();});
 it("holds one exact upstream note write pending and prevents cancellation or duplicate submission",async()=>{let finish!:(value:unknown)=>void;vi.mocked(defaultApiClient.patch).mockImplementation(()=>new Promise(r=>{finish=r;}));const {client}=mount();await editNote("Pending owner note");const before=client.getQueryData(["backlog","idea",item.name]);fireEvent.click(screen.getByRole("button",{name:"Save"}));await waitFor(()=>expect(defaultApiClient.patch).toHaveBeenCalledOnce());expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/backlog/idea/reviewed-item",{note:"Pending owner note"}]]);const save=screen.getByRole("button",{name:"Saving note"});expect(save).toBeDisabled();expect(screen.getByRole("button",{name:"Cancel"})).toBeDisabled();fireEvent.click(save);fireEvent.click(screen.getByRole("button",{name:"Cancel"}));expect(screen.getByRole("textbox")).toHaveValue("Pending owner note");expect(client.getQueryData(["backlog","idea",item.name])).toBe(before);current={...current,note:"Pending owner note"};await act(async()=>finish({item:current}));expect(await screen.findByRole("button",{name:"Pending owner note"})).toBeVisible();await waitFor(()=>expect(client.getQueryData(["backlog","idea",item.name])).toMatchObject({note:"Pending owner note"}));expect(defaultApiClient.patch).toHaveBeenCalledOnce();expect(defaultApiClient.post).not.toHaveBeenCalled();});
 it("handles an upstream refusal visibly, retains draft and cache, and never refetches or retries it automatically",async()=>{vi.mocked(defaultApiClient.patch).mockRejectedValue(new Error("Owner note refused"));const {client}=mount();await editNote("Unaccepted owner note");const before=client.getQueryData(["backlog","idea",item.name]);const readCount=vi.mocked(defaultApiClient.get).mock.calls.filter(([p])=>p==="/backlog/idea/reviewed-item").length;fireEvent.click(screen.getByRole("button",{name:"Save"}));expect(await screen.findByRole("alert")).toHaveTextContent("Owner note refused");expect(screen.getByRole("textbox")).toHaveValue("Unaccepted owner note");expect(client.getQueryData(["backlog","idea",item.name])).toBe(before);expect(defaultApiClient.patch).toHaveBeenCalledOnce();expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([p])=>p==="/backlog/idea/reviewed-item")).toHaveLength(readCount);expect(defaultApiClient.post).not.toHaveBeenCalled();expect(defaultApiClient.delete).not.toHaveBeenCalled();});
 it("refreshes the owner's authoritative returned value after one successful write",async()=>{vi.mocked(defaultApiClient.patch).mockImplementation(async()=>{current={...current,note:"Canonical owner note"};return{item:current};});const {client}=mount();await editNote("Submitted owner note");fireEvent.click(screen.getByRole("button",{name:"Save"}));await waitFor(()=>expect(client.getQueryData(["backlog","idea",item.name])).toMatchObject({note:"Canonical owner note"}));expect(await screen.findByRole("button",{name:"Canonical owner note"})).toBeVisible();expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/backlog/idea/reviewed-item",{note:"Submitted owner note"}]]);expect(defaultApiClient.post).not.toHaveBeenCalled();});
});
