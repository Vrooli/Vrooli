import { act, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { QueryClient } from "@tanstack/react-query";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import { backlogService } from "../services";
import { defaultApiClient } from "../lib/api-client";
import { useAgentActivitiesStore, useBacklogStore } from "../stores";
import type { BacklogItem } from "../types";
import { useBacklogDetailData, type UseBacklogDetailDataOptions } from "./useBacklogDetailData";

const item: BacklogItem = { kind: "idea", name: "exact-owner", title: "Original owner", description: "Retained", status: "backlog", priority: 2, tags: [], suggestedSkills: [], acceptanceAllow: ["scenarios/swarm-manager/**"], created: "2026-01-28T00:00:00Z", updated: "2026-01-28T00:00:00Z" };
const options: UseBacklogDetailDataOptions = { backlogKind: "idea", name: item.name, agentRunIsExecuting: false, agentRunIsBlocking: false };
const clients: QueryClient[] = [];
const unexpectedReads: string[] = [];
let owner = structuredClone(item);
let targets: unknown = { targets: [], requirements: [], has_archive: true };
let action: unknown = { id: "none", enabled: false, reason: "Review first", blockers: [] };
function mount(initial = options) {
  const client = createTestQueryClient(); clients.push(client);
  let current!: ReturnType<typeof useBacklogDetailData>;
  function Probe({ value }: { value: UseBacklogDetailDataOptions }) { current = useBacklogDetailData(value); return null; }
  const view = renderWithProviders(<Probe value={initial} />, { queryClient: client });
  return { ...view, client, get current() { return current; }, change(value: UseBacklogDetailDataOptions) { view.rerender(<Probe value={value} />); } };
}
beforeEach(() => {
  owner = structuredClone(item); targets = { targets: [], requirements: [], has_archive: true }; action = { id: "none", enabled: false, reason: "Review first", blockers: [] }; unexpectedReads.length = 0;
  useBacklogStore.getState().setItems([]); useAgentActivitiesStore.setState({ activities: [] });
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/backlog/idea/exact-owner") return { item: owner };
    if (path === "/backlog/idea/exact-owner/files") return { files: [] };
    if (path === "/backlog/idea/exact-owner/next-action") return { action };
    if (path === "/backlog/idea/exact-owner/archive/targets") return targets;
    if (path === "/backlog/idea/exact-owner/review") return { rounds: [] };
    if (path === "/execution?backlog_kind=idea&backlog_name=exact-owner") return { items: [] };
    if (path === "/backlog?spawned_from=idea%2Fexact-owner") return { items: [], blocking: {} };
    unexpectedReads.push(path); throw new Error(`Unexpected fixture read ${path}`);
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected fixture ${method}`));
});
afterEach(() => { clients.splice(0).forEach(c => c.clear()); useBacklogStore.getState().setItems([]); useAgentActivitiesStore.setState({ activities: [] }); vi.restoreAllMocks(); expect(unexpectedReads).toEqual([]); });
function writesAbsent() { for (const method of ["post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); }

describe("backlog detail actual projection and current target", () => {
  it("keeps unresolved identity empty and refuses description updates before network/cache effects", async () => {
    const h = mount({ ...options, backlogKind: null });
    await act(async () => {}); expect(defaultApiClient.get).not.toHaveBeenCalled(); expect(h.current.item).toBeUndefined(); expect(h.current.itemActions).toBeNull(); expect(h.current.reqModuleMap.size).toBe(0); expect(h.current.targetIdSet.size).toBe(0);
    await act(async () => { await expect(h.current.updateDescription("Never written")).rejects.toThrow("Backlog kind and name are required"); });
    expect(h.current.updateDescriptionError).toBe("Backlog kind and name are required"); writesAbsent();
    act(() => h.current.resetDescriptionMutation()); await waitFor(() => expect(h.current.updateDescriptionError).toBeNull());
    expect(h.client.getQueryCache().findAll().every(q => q.state.data === undefined)).toBe(true);
  });
  it("preserves owner action eligibility, nested requirement mapping and explicit target/scenario identities", async () => {
    owner = { ...owner, status: "queued" }; action = { id: "none", enabled: false, reason: "Owner holds admission", blockers: [] };
    targets = { has_archive: true, targets: [{ id: "target-a", title: "A" }], requirements: [{ id: "module-a", requirements: [{ id: "req-a" }], children: [{ id: "module-b", requirements: [{ id: "req-b" }], children: [] }] }] };
    const h = mount({ ...options, agentRunIsExecuting: true, agentRunIsBlocking: true });
    await waitFor(() => expect(h.current.reqModuleMap.size).toBe(2));
    expect([...h.current.reqModuleMap]).toEqual([["req-a", "module-a"], ["req-b", "module-b"]]); expect([...h.current.targetIdSet]).toEqual(["target-a"]); expect(h.current.targetScenarios).toEqual(["swarm-manager"]);
    expect(h.current.isLocked).toBe(true); expect(h.current.isTerminal).toBe(false); expect(h.current.itemActions).toMatchObject({ agentRunning: true, agentExecuting: true, disabledReason: "Owner holds admission", canRun: false }); writesAbsent();
  });
  it("shows only the selected cached target during a pending read and clears stale projections on unresolved selection", async () => {
    useBacklogStore.getState().setItems([{ ...item, name: "other", title: "Other owner" }, item]); vi.mocked(defaultApiClient.get).mockImplementation(() => new Promise(() => {}));
    const h = mount(); expect(h.current.item?.name).toBe(item.name); expect(h.current.item?.title).toBe(item.title);
    act(() => h.change({ ...options, name: undefined })); expect(h.current.item).toBeUndefined(); expect(h.current.itemActions).toBeNull(); expect(h.current.depRelations).toEqual({ parents: [], children: [] }); writesAbsent();
  });
  it("keeps an owner read error visible without inventing cache or mutation success", async () => {
    const error = new Error("Owner read refused"); vi.mocked(defaultApiClient.get).mockRejectedValue(error); const h = mount();
    await waitFor(() => expect(h.current.itemError).toBe(error), { timeout: 10000 }); expect(h.current.item).toBeUndefined(); expect(h.current.filesError).toBe(error); expect(h.current.itemActions).toBeNull(); writesAbsent();
  });
  it("refetches the exact item and file projections without changing mutation state", async () => {
    expect(await backlogService.get("idea", item.name)).toMatchObject({ name: item.name }); const h = mount(); await waitFor(() => { expect(h.current.itemError).toBeNull(); expect(h.current.item?.title).toBe(item.title); });
    const get = vi.mocked(defaultApiClient.get); const prior = get.mock.calls.length;
    act(() => { h.current.refetchItem(); h.current.refetchFiles(); });
    await waitFor(() => expect(get.mock.calls.slice(prior).map(([path]) => path)).toEqual(expect.arrayContaining(["/backlog/idea/exact-owner", "/backlog/idea/exact-owner/files"]))); writesAbsent();
  });
  it("saves the description only to the current owner and reconciles the exact returned store entry", async () => {
    const h = mount(); await waitFor(() => expect(h.current.item?.name).toBe(item.name));
    const other = { ...item, name: "unrelated", title: "Preserve other" }; useBacklogStore.getState().setItems([item, other]);
    owner = { ...item, description: "Owner accepted description" }; vi.mocked(defaultApiClient.patch).mockResolvedValue({ item: owner });
    await act(async () => { await h.current.updateDescription("Owner accepted description"); });
    expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/backlog/idea/exact-owner", { description: "Owner accepted description" }]]); expect(useBacklogStore.getState().items).toEqual([owner, other]); expect(h.current.updateDescriptionError).toBeNull();
  });
});

// Each callback uses the actual mutation/service implementation. Owner refusal
// cannot trigger a different write or alter the existing canonical item.
describe("backlog detail callback refusal and recovery", () => {
  const target = { id: "t-1", criticality: "high", title: "Target", notes: "N", status: "pending", linked_requirement_ids: [] };
  const cases: Array<{ label: string; method: "patch" | "post" | "put" | "delete"; path: string; payload?: unknown; invoke: (h: ReturnType<typeof useBacklogDetailData>) => void; error: (h: ReturnType<typeof useBacklogDetailData>) => unknown; reset: (h: ReturnType<typeof useBacklogDetailData>) => void }> = [
    { label: "editor", method: "patch", path: "/backlog/idea/exact-owner", payload: { title: "Reviewed", description: "D", status: "ready", priority: 3, tags: ["owner"] }, invoke: h => h.updateItem({ title: "Reviewed", description: "D", status: "ready", priority: 3, tags: ["owner"] }), error: h => h.updateError, reset: h => h.resetUpdateMutation() },
    { label: "acceptance globs", method: "patch", path: "/backlog/idea/exact-owner", payload: { acceptance_allow: [], acceptance_deny: ["private/**"] }, invoke: h => h.updateAcceptanceGlob({ acceptanceAllow: [], acceptanceDeny: ["private/**"] }), error: h => h._mutations.acceptanceGlob.error, reset: h => h.resetGlobMutation() },
    { label: "delete", method: "delete", path: "/backlog/idea/exact-owner", invoke: h => h.deleteItem(), error: h => h.deleteError, reset: h => h.resetDeleteMutation() },
    { label: "requirements", method: "put", path: "/backlog/idea/exact-owner/archive/requirements/module-a", payload: { requirements: [] }, invoke: h => h.updateRequirements({ moduleId: "module-a", requirements: [] }), error: h => h.updateReqsError, reset: h => h.resetUpdateReqsMutation() },
    { label: "new module", method: "post", path: "/backlog/idea/exact-owner/archive/requirements", payload: { id: "m-1", title: "M", description: "D", position: 2 }, invoke: h => h.createModule({ id: "m-1", title: "M", description: "D", position: 2 }), error: h => h.createModuleError, reset: h => h.resetCreateModuleMutation() },
    { label: "module metadata", method: "put", path: "/backlog/idea/exact-owner/archive/requirements/module-a/meta", payload: { title: "M", description: "D" }, invoke: h => h.updateModuleMeta({ moduleId: "module-a", payload: { title: "M", description: "D" } }), error: h => h.updateModuleMetaError, reset: h => h.resetUpdateModuleMetaMutation() },
    { label: "new target", method: "post", path: "/backlog/idea/exact-owner/archive/targets", payload: target, invoke: h => h.createTarget(target), error: h => h.createTargetError, reset: h => h.resetCreateTargetMutation() },
    { label: "target update", method: "put", path: "/backlog/idea/exact-owner/archive/targets/t-1", payload: target, invoke: h => h.updateTarget({ targetId: "t-1", target }), error: h => h.updateTargetError, reset: h => h.resetUpdateTargetMutation() },
  ];
  it.each(cases)("preserves canonical state and resets the explicit $label error without retry or alternate effects", async c => {
    useBacklogStore.getState().setItems([item]); const h = mount(); await waitFor(() => expect(h.client.getQueryData<BacklogItem>(["backlog", "idea", item.name])?.name).toBe(item.name));
    const cache = h.client.getQueryData(["backlog", "idea", item.name]); const refusal = new Error(`Owner refused ${c.label}`); vi.mocked(defaultApiClient[c.method]).mockRejectedValue(refusal);
    act(() => c.invoke(h.current)); await waitFor(() => { const error = c.error(h.current); if (error instanceof Error) expect(error).toBe(refusal); else expect(error).toBe(refusal.message); });
    expect(vi.mocked(defaultApiClient[c.method]).mock.calls).toEqual([c.payload === undefined ? [c.path] : [c.path, c.payload]]);
    expect(useBacklogStore.getState().items).toEqual([item]); expect(h.client.getQueryData(["backlog", "idea", item.name])).toBe(cache);
    for (const method of ["post", "put", "patch", "delete"] as const) if (method !== c.method) expect(defaultApiClient[method]).not.toHaveBeenCalled();
    act(() => c.reset(h.current)); await waitFor(() => expect(c.error(h.current)).toBeFalsy()); expect(defaultApiClient[c.method]).toHaveBeenCalledOnce();
  });
});

describe("backlog detail dependent changes and file safety", () => {
  it("changes only the selected dependency status and refreshes the parent's projection", async () => {
    const h = mount(); await waitFor(() => expect(h.current.item?.name).toBe(item.name));
    vi.mocked(defaultApiClient.patch).mockResolvedValue({ item: { ...item, kind: "execute", name: "dependency", status: "ready" } });
    const invalidate = vi.spyOn(h.client, "invalidateQueries");
    act(() => h.current.updateDepStatus({ kind: "execute", depName: "dependency", newStatus: "ready" }));
    await waitFor(() => expect(invalidate.mock.calls.map(([p]) => p?.queryKey)).toEqual([["backlog", "idea", item.name], ["backlog", "idea", item.name, "next-action"], ["backlog-list"]]));
    expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/backlog/execute/dependency", { status: "ready" }]]); expect(h.current.item?.status).toBe("backlog");
  });
  it("changes the exact owner status and keeps a refused follow-up change from altering the cache", async () => {
    const h = mount(); await waitFor(() => expect(h.current.item?.name).toBe(item.name));
    owner = { ...item, status: "ready" }; vi.mocked(defaultApiClient.patch).mockResolvedValueOnce({ item: owner }).mockRejectedValueOnce(new Error("Status changed elsewhere"));
    act(() => h.current.updateStatus("ready")); await waitFor(() => expect(useBacklogStore.getState().items.find(i => i.name === item.name)?.status).toBe("ready"));
    await waitFor(() => expect(h.current.item?.status).toBe("ready")); const cache = h.client.getQueryData(["backlog", "idea", item.name]);
    act(() => h.current.updateStatus("completed")); await waitFor(() => expect(h.current.isUpdatingStatus).toBe(false));
    await waitFor(() => expect(vi.mocked(defaultApiClient.patch).mock.calls).toHaveLength(2));
    expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/backlog/idea/exact-owner", { status: "ready" }], ["/backlog/idea/exact-owner", { status: "completed" }]]); expect(h.client.getQueryData(["backlog", "idea", item.name])).toBe(cache); expect(h.current.item?.status).toBe("ready");
  });
  it.each([
    { label: "module", path: "/backlog/idea/exact-owner/archive/requirements/module-a", invoke: (h: ReturnType<typeof useBacklogDetailData>) => h.deleteModule("module-a") },
    { label: "target", path: "/backlog/idea/exact-owner/archive/targets/target-a", invoke: (h: ReturnType<typeof useBacklogDetailData>) => h.deleteTarget("target-a") },
  ])("deletes only the explicit $label and invalidates archive projection after owner completion", async c => {
    const h = mount(); await waitFor(() => expect(h.current.item?.name).toBe(item.name)); let resolve!: (value: unknown) => void;
    vi.mocked(defaultApiClient.delete).mockImplementation(() => new Promise(r => { resolve = r; })); const invalidate = vi.spyOn(h.client, "invalidateQueries"); act(() => c.invoke(h.current));
    await waitFor(() => expect(vi.mocked(defaultApiClient.delete).mock.calls).toEqual([[c.path]])); expect(invalidate).not.toHaveBeenCalled();
    await act(async () => resolve({})); await waitFor(() => expect(invalidate.mock.calls.map(([p]) => p?.queryKey)).toEqual([["backlog", "idea", item.name, "archive-targets"]])); expect(h.current.item?.name).toBe(item.name);
  });
  it("preserves the exact batch-review decision payload and leaves projection unchanged on refusal", async () => {
    const h = mount(); await waitFor(() => expect(h.current.item?.name).toBe(item.name));
    const payload = [{ id: "req-a", type: "requirement" as const, review_status: "approved" as const, review_comment: "Explicit owner review" }];
    vi.mocked(defaultApiClient.put).mockRejectedValue(new Error("Review generation refused")); const invalidate = vi.spyOn(h.client, "invalidateQueries"); const prior = h.current.archiveTargets;
    act(() => h.current.batchReview(payload)); await waitFor(() => expect(h.current.batchReviewError).toBe("Review generation refused"));
    expect(vi.mocked(defaultApiClient.put).mock.calls).toEqual([["/backlog/idea/exact-owner/archive/review", { items: payload }]]); expect(invalidate).not.toHaveBeenCalled(); expect(h.current.archiveTargets).toBe(prior);
  });
  it("refuses a missing file destination locally without network or file-cache effects", async () => {
    const h = mount(); await waitFor(() => expect(h.current.item?.name).toBe(item.name)); const invalidate = vi.spyOn(h.client, "invalidateQueries"); const files = h.current.files;
    act(() => h.current.fileAction({ action: "rename", target: { path: "notes.md", name: "notes.md", type: "file", size: 12 } }));
    await waitFor(() => { expect(h.current._mutations.fileAction.error).toBeInstanceOf(Error); expect(h.current._mutations.fileAction.error instanceof Error ? h.current._mutations.fileAction.error.message : null).toBe("Destination path is required"); }); writesAbsent(); expect(invalidate).not.toHaveBeenCalled(); expect(h.current.files).toBe(files);
  });
});
