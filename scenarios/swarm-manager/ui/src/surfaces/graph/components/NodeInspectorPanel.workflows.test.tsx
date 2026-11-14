import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { NodeInspectorPanel } from "./NodeInspectorPanel";
import { defaultApiClient } from "../../../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../../../test-utils";
import { useBacklogStore } from "../../../stores";
import type { BacklogItem } from "../../../types";
import { cloneGraphDataInitialState, useGraphDataStore } from "../stores/graph-data-store";
import { cloneGraphUIInitialState, useGraphUIStore } from "../stores/graph-ui-store";
import { makeBacklogNode } from "../test-helpers";

// The inspector, mutations, service translation and stores are real. Only the
// API client boundary is replaced; queue acceptance does not prove a run launch.
const id = "backlog-item/idea/owner-item";
const initial: BacklogItem = { kind: "idea", name: "owner-item", title: "Owner item", description: "Scoped item", status: "backlog", priority: 2, tags: [], suggestedSkills: [], created: "2026-01-01T00:00:00Z", updated: "2026-01-01T00:00:00Z" };
let owner: BacklogItem;
const clients: QueryClient[] = [];
const unexpected: string[] = [];
function Location() { const location = useLocation(); return <output data-testid="inspector-location">{location.pathname + location.search}</output>; }
function mount(status: BacklogItem["status"] = "backlog", selected = id) {
  owner = { ...initial, status };
  useBacklogStore.getState().setItems([owner]);
  useGraphDataStore.setState({ ...cloneGraphDataInitialState(), lens: "plan", nodes: [makeBacklogNode(id, { title: owner.title, status })] });
  useGraphUIStore.setState({ ...cloneGraphUIInitialState(), selectedNodeId: selected, highlightState: { highlighted: new Set([id]), mode: "normal" } });
  const client = createTestQueryClient(); clients.push(client);
  renderWithProviders(<><NodeInspectorPanel /><Location /></>, { queryClient: client, initialEntries: [`/graph?lens=plan&select=${encodeURIComponent(selected)}&keep=owner`] });
  return client;
}
beforeEach(() => {
  unexpected.length = 0;
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/goals") return { items: [] };
    if (path === "/backlog" || path.startsWith("/backlog?")) return { items: [owner], blocking: {} };
    unexpected.push(path); throw new Error(`Unexpected inspector read ${path}`);
  });
  vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected inspector POST"));
  vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new Error("Unexpected inspector PATCH"));
  vi.spyOn(defaultApiClient, "put").mockRejectedValue(new Error("Unexpected inspector PUT"));
  vi.spyOn(defaultApiClient, "delete").mockRejectedValue(new Error("Unexpected inspector DELETE"));
});
afterEach(() => {
  clients.splice(0).forEach(client => client.clear());
  expect(defaultApiClient.put).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled(); expect(unexpected).toEqual([]);
  vi.restoreAllMocks(); useGraphDataStore.setState(cloneGraphDataInitialState()); useGraphUIStore.setState(cloneGraphUIInitialState()); useBacklogStore.getState().setItems([]);
});
async function chooseStatus(status: string) { fireEvent.click(screen.getByTestId("status-badge")); fireEvent.click(await screen.findByTestId(`status-badge-option-${status}`)); }
function noWrites() { expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(defaultApiClient.patch).not.toHaveBeenCalled(); }
describe("NodeInspectorPanel scoped navigation and owner mutations", () => {
  it("closes selection and highlights while preserving unrelated location state", async () => {
    mount(); fireEvent.click(screen.getByRole("button", { name: "Close panel" }));
    await waitFor(() => expect(screen.getByTestId("inspector-location")).toHaveTextContent("/graph?lens=plan&keep=owner"));
    expect(useGraphUIStore.getState().selectedNodeId).toBeNull(); expect(useGraphUIStore.getState().highlightState.highlighted.size).toBe(0); noWrites();
  });
  it("opens only the selected backlog detail route without a mutation", async () => {
    mount(); fireEvent.click(screen.getByTestId("inspector-open-details"));
    await waitFor(() => expect(screen.getByTestId("inspector-location")).toHaveTextContent("/backlog/idea/owner-item")); noWrites();
  });
  it("drills to a different lens with the exact selected entity as focus", async () => {
    mount(); fireEvent.click(screen.getByTestId("inspector-lens-focus"));
    await waitFor(() => { const location = screen.getByTestId("inspector-location").textContent ?? ""; const parsed = new URL(location, "https://fixture.invalid"); expect(parsed.pathname).toBe("/graph"); expect(parsed.searchParams.get("mode")).toBe("focus"); expect(parsed.searchParams.get("focus")).toBe(id); expect(parsed.searchParams.get("select")).toBe(id); }); noWrites();
  });
  it("does not offer actions for a selected entity absent from the graph", () => {
    mount("backlog", "backlog-item/idea/missing"); expect(screen.queryByTestId("inspector-open-details")).toBeNull(); expect(screen.queryByTestId("status-badge")).toBeNull(); noWrites();
  });
  it("selecting the current status leaves the owner record unchanged", async () => {
    mount(); const before = useBacklogStore.getState().items; await chooseStatus("backlog"); noWrites(); expect(useBacklogStore.getState().items).toBe(before);
  });
  it("waits for the exact status PATCH before refreshing authoritative backlog state", async () => {
    let complete!: (value: unknown) => void; vi.mocked(defaultApiClient.patch).mockImplementation(() => new Promise(resolve => { complete = resolve; }));
    mount(); const before = useBacklogStore.getState().items; await chooseStatus("ready");
    await waitFor(() => expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/backlog/idea/owner-item", { status: "ready" }]]));
    fireEvent.click(screen.getByTestId("status-badge")); expect(screen.queryByTestId("status-badge-popover")).toBeNull(); expect(defaultApiClient.patch).toHaveBeenCalledOnce(); expect(useBacklogStore.getState().items).toBe(before);
    owner = { ...owner, status: "ready" }; await act(async () => complete({ item: owner }));
    await waitFor(() => expect(useBacklogStore.getState().items[0]?.status).toBe("ready")); expect(await screen.findByText("owner-item set to ready")).toBeVisible(); expect(defaultApiClient.post).not.toHaveBeenCalled();
  });
  it("shows status refusal without refreshing or performing an alternate write", async () => {
    vi.mocked(defaultApiClient.patch).mockRejectedValue(new Error("Status owner refused")); mount(); const before = useBacklogStore.getState().items;
    await chooseStatus("ready"); expect(await screen.findByText("Status owner refused")).toBeVisible(); expect(defaultApiClient.patch).toHaveBeenCalledOnce(); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(useBacklogStore.getState().items).toBe(before); expect(vi.mocked(defaultApiClient.get).mock.calls.some(([path]) => path.startsWith("/backlog"))).toBe(false);
  });
  it("queues only the selected ready item once and waits for acceptance before refresh", async () => {
    let complete!: (value: unknown) => void; vi.mocked(defaultApiClient.post).mockImplementation(() => new Promise(resolve => { complete = resolve; })); mount("ready"); const before = useBacklogStore.getState().items;
    fireEvent.click(screen.getByTestId("inspector-run-button")); await waitFor(() => expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/backlog/idea/owner-item/queue", {}]])); expect(screen.getByTestId("inspector-run-button")).toBeDisabled(); fireEvent.click(screen.getByTestId("inspector-run-button")); expect(defaultApiClient.post).toHaveBeenCalledOnce(); expect(useBacklogStore.getState().items).toBe(before);
    owner = { ...owner, status: "queued" }; await act(async () => complete({})); await waitFor(() => expect(useBacklogStore.getState().items[0]?.status).toBe("queued")); expect(await screen.findByText("owner-item queued for an agent run")).toBeVisible(); expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
  it("preserves ready state when the queue owner refuses without fallback", async () => {
    vi.mocked(defaultApiClient.post).mockRejectedValue(new Error("Queue owner refused")); mount("ready"); const before = useBacklogStore.getState().items; fireEvent.click(screen.getByTestId("inspector-run-button")); expect(await screen.findByText("Queue owner refused")).toBeVisible(); expect(defaultApiClient.post).toHaveBeenCalledOnce(); expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(useBacklogStore.getState().items).toBe(before);
  });
  it.each(["queued", "in_progress"] as const)("keeps %s lifecycle state read-only", status => {
    mount(status); expect(screen.queryByTestId("inspector-run-button")).toBeNull(); expect(screen.queryByTestId("status-badge")).toBeNull(); noWrites();
  });
});
