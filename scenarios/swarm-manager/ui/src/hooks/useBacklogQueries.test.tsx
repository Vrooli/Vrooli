import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { defaultApiClient } from "../lib/api-client";
import { useBacklogStore } from "../stores";
import type { BacklogItem } from "../types";
import { useBacklogQueries, type UseBacklogQueriesOptions } from "./useBacklogQueries";

const item: BacklogItem = { kind: "idea", name: "reviewed-item", title: "Owner version", description: "Reviewed", status: "backlog", priority: 1, tags: [], suggestedSkills: [], created: "2026-01-28T00:00:00Z", updated: "2026-01-28T00:00:00Z" };
const clients: QueryClient[] = [];
function mount(options: UseBacklogQueriesOptions = { backlogKind: "idea", name: item.name, agentRunIsBlocking: false }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } }); clients.push(client);
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  return { ...renderHook(() => useBacklogQueries(options), { wrapper }), client };
}
function ownerReads(rounds: unknown[] = []) {
  return vi.spyOn(defaultApiClient, "get").mockImplementation(async (path) => {
    if (path === "/backlog/idea/reviewed-item") return { item };
    if (path === "/backlog/idea/reviewed-item/files") return { files: [] };
    if (path === "/backlog/idea/reviewed-item/next-action") return { action: { id: "none", enabled: false, reason: "Review required" } };
    if (path === "/backlog/idea/reviewed-item/archive-targets") return { targets: [], requirements: [] };
    if (path === "/backlog/idea/reviewed-item/review") return { rounds };
    if (path === "/execution?backlog_kind=idea&backlog_name=reviewed-item") return { items: [] };
    if (path.startsWith("/backlog?")) return { items: [], blocking: {} };
    throw new Error(`Unexpected fixture read ${path}`);
  });
}
afterEach(() => { clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); useBacklogStore.getState().setItems([]); });
describe("backlog queries owner reads", () => {
  it.each([{ backlogKind: null, name: item.name }, { backlogKind: "idea" as const, name: undefined }])("keeps unresolved target queries disabled without network effects (%j)", async (target) => {
    const get = vi.spyOn(defaultApiClient, "get").mockRejectedValue(new Error("Unexpected fixture read"));
    const { result } = mount({ ...target, agentRunIsBlocking: true });
    await act(async () => {});
    expect(get).not.toHaveBeenCalled(); expect(result.current.item).toBeUndefined(); expect(result.current.isGatheringEvidence).toBe(false); expect(result.current.isAwaitingManualReview).toBe(false);
  });
  it("loads the exact target through real services and keeps empty collections empty", async () => {
    const get = ownerReads(); const { result } = mount();
    await waitFor(() => expect(result.current.item?.title).toBe(item.title));
    await waitFor(() => expect(result.current.spawnedItems).toEqual([]));
    expect(result.current.files).toEqual([]); expect(result.current.executionHistory).toEqual([]); expect(result.current.reviewRounds).toEqual([]);
    expect(get.mock.calls.map(([path]) => path)).toContain("/execution?backlog_kind=idea&backlog_name=reviewed-item");
    expect(get.mock.calls.map(([path]) => path)).toContain("/backlog/idea/reviewed-item/next-action");
    await waitFor(() => expect(result.current.nextAction).toMatchObject({ id: "none", enabled: false, reason: "Review required" }));
  });
  it("shows only the exact cached target while owner refresh is pending", async () => {
    useBacklogStore.getState().setItems([{ ...item, name: "other", title: "Do not leak" }, { ...item, title: "Cached target" }]);
    vi.spyOn(defaultApiClient, "get").mockImplementation(() => new Promise(() => {}));
    const { result } = mount(); expect(result.current.item?.title).toBe("Cached target"); expect(result.current.item?.name).toBe(item.name);
  });
  it("keeps owner read failure visible and avoids mutation or invented data", async () => {
    const refusal = new Error("Owner visibility refused"); vi.spyOn(defaultApiClient, "get").mockRejectedValue(refusal);
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    const { result } = mount(); await waitFor(() => expect(result.current.itemError).toBe(refusal), { timeout: 10000 });
    expect(result.current.item).toBeUndefined(); expect(result.current.filesError).toBe(refusal); expect(post).not.toHaveBeenCalled();
  });
  it.each([
    { rounds: [{ status: "gathering", current_run_status: "running" }], gathering: true, manual: false },
    { rounds: [{ status: "gathering", current_run_status: "needs_review" }], gathering: false, manual: true },
    { rounds: [{ status: "complete", current_run_status: "needs_review" }], gathering: false, manual: false },
  ])("distinguishes active evidence collection from retained manual review ($gathering/$manual)", async ({ rounds, gathering, manual }) => {
    ownerReads(rounds); const { result } = mount(); await waitFor(() => expect(result.current.reviewRounds).toEqual(rounds));
    expect(result.current.isGatheringEvidence).toBe(gathering); expect(result.current.isAwaitingManualReview).toBe(manual);
  });
});
