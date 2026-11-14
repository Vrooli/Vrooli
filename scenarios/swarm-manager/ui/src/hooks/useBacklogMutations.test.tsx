import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ToastProvider } from "../components/ui/toast-provider";
import { ApiError, defaultApiClient } from "../lib/api-client";
import { useBacklogStore } from "../stores";
import type { BacklogItem, BacklogFile } from "../types";
import { useBacklogMutations, type UseBacklogMutationsOptions } from "./useBacklogMutations";

const original: BacklogItem = { name: "reviewed-item", kind: "idea", title: "Original", description: "Retained edits", status: "backlog", priority: 1, tags: [], suggestedSkills: [], created: "2026-01-28T00:00:00Z", updated: "2026-01-28T00:00:00Z" };
const file: BacklogFile = { name: "notes.md", path: "notes.md", type: "file", size: 42 };
function mount(options: UseBacklogMutationsOptions = { backlogKind: "idea", name: original.name }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}><ToastProvider>{children}</ToastProvider></QueryClientProvider>;
  const hook = renderHook(() => useBacklogMutations(options), { wrapper });
  return { ...hook, client, invalidate };
}
describe("backlog mutation request and reconciliation", () => {
  beforeEach(() => { useBacklogStore.getState().setItems([original]); });
  afterEach(() => { vi.restoreAllMocks(); useBacklogStore.getState().setItems([]); localStorage.clear(); });
  it("saves reviewed editor fields through the actual service and refreshes only the item/frontier", async () => {
    const updated = { ...original, title: "Reviewed", description: "Saved", status: "ready" as const, priority: 2, tags: ["approved"] };
    const patch = vi.spyOn(defaultApiClient, "patch").mockResolvedValue({ item: updated });
    const { result, invalidate } = mount();
    await act(async () => { await result.current.updateMutation.mutateAsync({ title: updated.title, description: updated.description, status: updated.status, priority: updated.priority, tags: updated.tags }); });
    expect(patch).toHaveBeenCalledOnce();
    expect(patch).toHaveBeenCalledWith("/backlog/idea/reviewed-item", { title: "Reviewed", description: "Saved", status: "ready", priority: 2, tags: ["approved"] });
    expect(useBacklogStore.getState().items).toEqual([updated]);
    expect(invalidate.mock.calls.map(([request]) => request?.queryKey)).toEqual([["backlog", "idea", original.name], ["backlog", "idea", original.name, "next-action"]]);
  });
  it("preserves explicit cleared acceptance arrays and reconciles the returned owner state", async () => {
    const patch = vi.spyOn(defaultApiClient, "patch").mockResolvedValue({ item: original });
    const { result, invalidate } = mount();
    await act(async () => { await result.current.acceptanceGlobMutation.mutateAsync({ acceptanceAllow: [], acceptanceDeny: ["private/**"] }); });
    expect(patch).toHaveBeenCalledOnce();
    expect(patch).toHaveBeenCalledWith("/backlog/idea/reviewed-item", { acceptance_allow: [], acceptance_deny: ["private/**"] });
    expect(invalidate).toHaveBeenCalledTimes(2);
  });
  it("keeps existing item data and inline errors when the owner refuses an edit", async () => {
    vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new ApiError("http", "reviewed revision changed", { status: 409 }));
    const { result, invalidate } = mount();
    await act(async () => { await expect(result.current.updateMutation.mutateAsync({ title: "Unsaved", description: "Keep these edits", status: "ready", priority: 1, tags: [] })).rejects.toThrow("reviewed revision changed"); });
    await waitFor(() => expect(result.current.updateError).toContain("reviewed revision changed"));
    expect(useBacklogStore.getState().items).toEqual([original]);
    expect(invalidate).not.toHaveBeenCalled();
  });
  it.each([{ backlogKind: null, name: original.name }, { backlogKind: "idea" as const, name: undefined }])("refuses an unresolved target before any service request (%j)", async (options) => {
    const patch = vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new Error("Unexpected fixture network request")); const remove = vi.spyOn(defaultApiClient, "delete").mockRejectedValue(new Error("Unexpected fixture network request"));
    const { result, invalidate } = mount(options);
    await act(async () => {
      await expect(result.current.statusMutation.mutateAsync("ready")).rejects.toThrow("Backlog kind and name are required");
      await expect(result.current.archiveMutation.mutateAsync()).rejects.toThrow("Backlog kind and name are required");
      await expect(result.current.deleteMutation.mutateAsync()).rejects.toThrow("Backlog kind and name are required");
    });
    expect(patch).not.toHaveBeenCalled(); expect(remove).not.toHaveBeenCalled(); expect(invalidate).not.toHaveBeenCalled();
    expect(useBacklogStore.getState().items).toEqual([original]);
  });
  it("updates a dependency's exact identity and refreshes the owning item/list frontier", async () => {
    const patch = vi.spyOn(defaultApiClient, "patch").mockResolvedValue({ item: { ...original, name: "dependency" } });
    const { result, invalidate } = mount();
    await act(async () => { await result.current.depStatusMutation.mutateAsync({ kind: "idea", depName: "dependency", newStatus: "ready" }); });
    expect(patch).toHaveBeenCalledOnce();
    expect(patch).toHaveBeenCalledWith("/backlog/idea/dependency", { status: "ready" });
    expect(invalidate.mock.calls.map(([request]) => request?.queryKey)).toEqual([["backlog", "idea", original.name], ["backlog", "idea", original.name, "next-action"], ["backlog-list"]]);
    expect(useBacklogStore.getState().items).toEqual([original]);
  });
  it("archives and restores without dropping the store item and refreshes list membership", async () => {
    const patch = vi.spyOn(defaultApiClient, "patch").mockResolvedValue({ item: original });
    const remove = vi.spyOn(defaultApiClient, "delete").mockResolvedValue({ item: original });
    const { result, invalidate } = mount();
    await act(async () => { await result.current.archiveMutation.mutateAsync(); await result.current.unarchiveMutation.mutateAsync(); });
    expect(patch).toHaveBeenCalledOnce();
    expect(patch).toHaveBeenCalledWith("/backlog/idea/reviewed-item/archive-item", {});
    expect(remove).toHaveBeenCalledOnce();
    expect(remove).toHaveBeenCalledWith("/backlog/idea/reviewed-item/archive-item");
    expect(invalidate.mock.calls.filter(([request]) => request?.queryKey?.[0] === "backlog-list")).toHaveLength(2);
    expect(useBacklogStore.getState().items).toEqual([original]);
  });
  it("deletes only the successfully removed item from the actual store", async () => {
    const other = { ...original, name: "keep" }; useBacklogStore.getState().setItems([original, other]);
    const remove = vi.spyOn(defaultApiClient, "delete").mockResolvedValue(undefined);
    const { result } = mount();
    await act(async () => { await result.current.deleteMutation.mutateAsync(); });
    expect(remove).toHaveBeenCalledOnce();
    expect(remove).toHaveBeenCalledWith("/backlog/idea/reviewed-item");
    expect(useBacklogStore.getState().items).toEqual([other]);
  });
  it("refuses a missing file destination without a request or cache effects", async () => {
    const patch = vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new Error("Unexpected fixture network request")); const { result, invalidate } = mount();
    await act(async () => { await expect(result.current.fileActionMutation.mutateAsync({ action: "move", target: file })).rejects.toThrow("Destination path is required"); });
    expect(patch).not.toHaveBeenCalled(); expect(invalidate).not.toHaveBeenCalled();
  });
  it.each(["rename", "move", "copy", "delete"] as const)("performs an exact typed file %s and refreshes only the file tree", async (action) => {
    const patch = vi.spyOn(defaultApiClient, "patch").mockResolvedValue(action === "delete" ? { deleted_path: file.path } : { file: { ...file, path: "reviewed/notes.md" } });
    const { result, invalidate } = mount();
    await act(async () => { await result.current.fileActionMutation.mutateAsync({ action, target: file, destinationPath: "reviewed/notes.md" }); });
    expect(patch).toHaveBeenCalledOnce();
    expect(patch).toHaveBeenCalledWith("/backlog/idea/reviewed-item/files", action === "delete" ? { operation: action, source_path: file.path } : { operation: action, source_path: file.path, destination_path: "reviewed/notes.md" });
    expect(invalidate.mock.calls.map(([request]) => request?.queryKey)).toEqual([["backlog", "idea", original.name, "files"]]);
  });
});
