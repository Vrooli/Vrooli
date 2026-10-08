// Real sidebar, canonical forms, stores and adapters; transports are disposable.
// Backlog creation responses are fixture bytes, not native work/admission.
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { createTestQueryClient, renderWithProviders } from "../../../../test-utils";
import { defaultApiClient } from "../../../../lib/api-client";
import { DEFAULT_SETTINGS } from "../../../../services/settings-service";
import { useAgentSessionStore, useBacklogFormStore, useBacklogStore, useCaptureStore, useExecutionStore, useScenariosStore } from "../../../../stores";
import { useGraphUIStore } from "../../stores/graph-ui-store";
import { selectors } from "../../../../consts/selectors";
import { Sidebar } from "./Sidebar";

const views: ReturnType<typeof renderWithProviders>[] = []; const clients: QueryClient[] = [];
const body = { name: "fixture-idea", title: "Fixture idea", priority: 5, tags: [], kind: "idea", depends_on: [], acceptance_allow: [], acceptance_deny: [], creates: [], execution_mode: "sliced", continuation: "manual", scope_policy: "fixed" };
const returned = { name: "fixture-idea", title: "Owner returned title", description: "Owner returned detail", priority: 5, tags: [], kind: "idea", status: "backlog", created: "2026-10-01T00:00:00Z", updated: "2026-10-01T00:00:00Z" };
let expectedPosts: unknown[][]; let unexpected: string[]; let postOutcome: () => Promise<unknown>; let aiAvailable: boolean;
function reset() { useBacklogStore.getState().reset(); useCaptureStore.getState().reset(); useExecutionStore.getState().reset(); useScenariosStore.getState().reset(); useAgentSessionStore.getState().reset(); useBacklogFormStore.getState().reset(); useGraphUIStore.getState().setSidebarCollapsed(false); }
function mount(tab = "backlog") {
  window.localStorage.setItem("swarm-manager.sidebar.state.v1", JSON.stringify({ activeTab: tab, searchQuery: "", searchMode: "plain", filters: {}, sorts: {} }));
  const queryClient = createTestQueryClient(); clients.push(queryClient); const view = renderWithProviders(<Sidebar onItemClick={vi.fn()} onSettingsOpen={vi.fn()} onGoHome={vi.fn()} onQuickCapture={vi.fn()} />, { queryClient }); views.push(view); return view;
}
function openCreate() { fireEvent.click(screen.getByTestId("sidebar-create-current")); expect(screen.getByTestId(selectors.backlogForm.dialog)).toBeVisible(); }
function draft() { fireEvent.change(screen.getByTestId(selectors.backlogForm.titleInput), { target: { value: "Fixture idea" } }); }
function submit() { fireEvent.click(screen.getByTestId(selectors.backlogForm.submitButton)); }
beforeEach(() => {
  window.localStorage.clear(); reset(); expectedPosts = []; unexpected = []; aiAvailable = false; postOutcome = async () => ({ item: returned });
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/search/ai/status") return { available: aiAvailable, ollama: aiAvailable, qdrant: aiAvailable, indexedBacklog: 0, indexedGoals: 0, onDiskBacklog: 0, onDiskGoals: 0 };
    if (path === "/backlog/summary") return { pending_questions: { questions: [], total: 0 } };
    if (path === "/execution/strategies") return { strategies: [] };
    if (path === "/next-actions/feed") return { entries: [] };
    if (path === "/backlog?archived=all") return { items: [], blocking: {} };
    if (path === "/goals") return { items: [] };
    if (path === "/plan-import/plans") return { plans: [] };
    if (path === "/settings") { const { deleteConfirmation: _, ...settings } = DEFAULT_SETTINGS; return { settings }; }
    unexpected.push(path); throw new Error(`Unexpected sidebar read ${path}`);
  });
  vi.spyOn(defaultApiClient, "post").mockImplementation(async (path, payload) => {
    if (path === "/backlog/next-actions" && expectedPosts.some(row => row[0] === path)) { expect(payload).toEqual({ items: ["idea/fixture-idea"] }); return { results: [] }; }
    if (path !== "/backlog" || expectedPosts.length === 0) { unexpected.push(`POST ${path}`); throw new Error("Forbidden sidebar mutation"); }
    expect(payload).toEqual(body); return postOutcome();
  });
  for (const method of ["put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Forbidden sidebar ${method}`));
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Forbidden direct sidebar transport")));
});
afterEach(() => {
  views.splice(0).forEach(view => view.unmount()); clients.splice(0).forEach(client => client.clear()); getSpatialNav()?.dispose();
  try { expect(unexpected).toEqual([]); expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual(expectedPosts); for (const method of ["put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); expect(fetch).not.toHaveBeenCalled(); }
  finally { reset(); window.localStorage.clear(); vi.restoreAllMocks(); vi.unstubAllGlobals(); }
});
describe("sidebar owner forms, read choices and cancelled actions", () => {
  it("validates a missing title locally without a create request", () => { mount(); openCreate(); submit(); expect(screen.getByText("Title is required.")).toBeVisible(); expect(useBacklogStore.getState().items).toEqual([]); });
  it("cancels an unsent draft and reopens with the canonical empty form", async () => { mount(); openCreate(); draft(); fireEvent.click(screen.getByTestId(selectors.backlogForm.cancelButton)); await waitFor(() => expect(screen.queryByTestId(selectors.backlogForm.dialog)).toBeNull()); openCreate(); expect(screen.getByTestId(selectors.backlogForm.titleInput)).toHaveValue(""); });
  it("waits for exact owner create completion and blocks duplicate submission", async () => {
    let finish!: (value: unknown) => void; postOutcome = () => new Promise(resolve => { finish = resolve; }); expectedPosts = [["/backlog", body]]; mount(); openCreate(); draft(); submit();
    await waitFor(() => expect(defaultApiClient.post).toHaveBeenCalledTimes(1)); expect(screen.getByTestId(selectors.backlogForm.submitButton)).toBeDisabled(); expect(screen.getByTestId(selectors.backlogForm.cancelButton)).toBeDisabled(); expect(useBacklogStore.getState().items).toEqual([]); submit();
    expectedPosts.push(["/backlog/next-actions", { items: ["idea/fixture-idea"] }]); await act(async () => finish({ item: returned })); await waitFor(() => expect(screen.queryByTestId(selectors.backlogForm.dialog)).toBeNull()); expect(useBacklogStore.getState().items).toMatchObject([{ name: "fixture-idea", title: "Owner returned title", description: "Owner returned detail" }]);
  });
  it("retains refused input and only retries after explicit owner intent", async () => {
    postOutcome = async () => { throw new Error("Fixture owner refused create"); }; expectedPosts = [["/backlog", body]]; mount(); openCreate(); draft(); submit(); expect(await screen.findByText("Fixture owner refused create")).toBeVisible(); expect(screen.getByTestId(selectors.backlogForm.titleInput)).toHaveValue("Fixture idea"); expect(useBacklogStore.getState().items).toEqual([]);
    postOutcome = async () => ({ item: returned }); expectedPosts.push(["/backlog", body], ["/backlog/next-actions", { items: ["idea/fixture-idea"] }]); submit(); await waitFor(() => expect(screen.queryByTestId(selectors.backlogForm.dialog)).toBeNull()); expect(useBacklogStore.getState().items).toHaveLength(1);
  });
  it("clears a refused create reason on cancellation before reopening", async () => { postOutcome = async () => { throw new Error("Fixture owner refused create"); }; expectedPosts = [["/backlog", body]]; mount(); openCreate(); draft(); submit(); await screen.findByText("Fixture owner refused create"); fireEvent.click(screen.getByTestId(selectors.backlogForm.cancelButton)); openCreate(); expect(screen.queryByText("Fixture owner refused create")).toBeNull(); expect(screen.getByTestId(selectors.backlogForm.titleInput)).toHaveValue(""); });
  it("opens plan import catalog from the real empty backlog and cancels without import", async () => { mount(); fireEvent.click(screen.getByRole("button", { name: "Create from plan" })); await waitFor(() => expect(defaultApiClient.get).toHaveBeenCalledWith("/plan-import/plans", { signal: expect.any(AbortSignal) })); fireEvent.click(await screen.findByRole("button", { name: /^Close$/ })); await waitFor(() => expect(screen.queryByText("Create work from plan")).toBeNull()); });
  it("opens the goal owner form from the active goal tab then cancels", async () => { mount("goals"); fireEvent.click(screen.getByTestId("sidebar-create-current")); expect(screen.getByRole("heading", { name: "Create goal" })).toBeVisible(); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); await waitFor(() => expect(screen.queryByRole("heading", { name: "Create goal" })).toBeNull()); });
  it("changes semantic/plain display only while query is empty", async () => { aiAvailable = true; mount(); const toggle = await screen.findByRole("button", { name: "Use AI search" }); fireEvent.click(toggle); expect(screen.getByTestId("sidebar-search")).toHaveAttribute("placeholder", "Semantic search..."); expect(screen.queryByTestId("sidebar-create-current")).toBeNull(); fireEvent.click(screen.getByRole("button", { name: "Use plain search" })); expect(screen.getByTestId("sidebar-search")).toHaveAttribute("placeholder", "Search..."); expect(screen.getByTestId("sidebar-create-current")).toBeVisible(); });
});
