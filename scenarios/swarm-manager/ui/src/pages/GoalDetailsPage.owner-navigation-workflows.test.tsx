import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { GoalDetailsPage } from "./GoalDetailsPage";
import { ApiError, defaultApiClient } from "../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import { DEFAULT_SETTINGS } from "../services/settings-service";
import { useBacklogStore } from "../stores";
import type { GoalWithScope } from "../types/goal";

// Outside-only draft: actual routed page/hooks/services/providers. No writes,
// proposal decisions, launch, attachment, stop or delete confirmation.
const initial: GoalWithScope = {
  goal: { name: "fixture-goal", title: "Fixture owner outcome", description: "Owner supplied description", priority: 3, status: "active", targets: ["fix/fixture-item"], milestones: [{ name: "fixture-milestone", title: "Fixture milestone", description: "Bounded criterion", items: ["fix/fixture-item"], acceptanceCriteria: ["Inspect owner outcome"], dependsOn: [] }], seeded: false, scopeHistory: [], created: "2026-01-01T00:00:00Z", updated: "2026-01-01T00:00:00Z" },
  scope: { targets: ["fix/fixture-item"], closure: ["fix/fixture-item"], completed: [], ready: [], blocked: ["fix/fixture-item"], total: 1, completedCount: 0, blockedCount: 1, progressPct: 0 }, eta: null,
};
const clients: QueryClient[] = [];
const unmounts: Array<() => void> = [];
const unexpected: string[] = [];
const transport: {path:string;method:string;body:string}[] = [];
const savedDisclosure = new Map<string, string | null>();
let current: GoalWithScope;
let goalRefused: boolean;
let filesRefused: boolean;
function Location() { const value = useLocation(); return <output data-testid="goal-owner-location">{value.pathname + value.search}</output>; }
function mount(query = "") {
  const client = createTestQueryClient(); clients.push(client);
  client.setQueryDefaults(["goal", "unrelated-goal"], { gcTime: Infinity }); client.setQueryData(["goal", "unrelated-goal"], { sentinel: "unchanged" });
  const view = renderWithProviders(<><Routes><Route path="/goals/:name" element={<GoalDetailsPage />} /><Route path="/plan" element={<h1>Goal scoped plan destination</h1>} /><Route path="/stats" element={<h1>Goal scoped stats destination</h1>} /><Route path="/backlog/:kind/:name" element={<h1>Selected owner item destination</h1>} /></Routes><Location /></>, { queryClient: client, initialEntries: [`/goals/fixture-goal${query}`] }); unmounts.push(view.unmount); return { ...view, client };
}
async function loaded() { return screen.findByRole("heading", { name: initial.goal.title }); }
function noWrites() { for (const method of ["post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); expect(transport.every(x => x.path === "/vrooli.swarm_manager.v1.api.TransitionService/ListTransitions" && x.method === "POST" && x.body === "{}")).toBe(true); }
async function openMenu(label: "Edit" | "Archive" | "Delete") { await loaded(); fireEvent.click(screen.getByTestId("detail-header-actions")); fireEvent.click(await screen.findByRole("menuitem", { name: label })); }
beforeEach(() => {
  current = structuredClone(initial); goalRefused = false; filesRefused = false; unexpected.length = 0; transport.length = 0; useBacklogStore.getState().setItems([]);
  for (const key of ["goal.scope-progress", "goal.blocked-work", "goal.milestones", "goal.targets", "goal.milestone.fixture-milestone"]) { const storage = `swarm-manager.section.${key}`; savedDisclosure.set(storage, localStorage.getItem(storage)); localStorage.removeItem(storage); }
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/settings") { const { deleteConfirmation: _, ...settings } = DEFAULT_SETTINGS; return { settings }; }
    if (path === "/goals/fixture-goal") { if (goalRefused) throw new ApiError("http", "Fixture goal read refused", { status: 403 }); return current; }
    if (path === "/next-actions/feed") return { entries: [] };
    if (path === "/backlog?archived=all") return { items: [], blocking: {} };
    if (path === "/proposal-sessions?target_type=goal&target_ref=fixture-goal") return { sessions: [] };
    if (path === "/goals/fixture-goal/files") { if (filesRefused) throw new ApiError("http", "Fixture files read refused", { status: 403 }); return { files: [{ name: "notes.md", path: "notes.md", type: "file", size: "12" }] }; }
    if (path === "/goals/fixture-goal/files/notes.md") return "# Owner read-only fixture notes";
    unexpected.push(path); throw new Error(`Unexpected goal owner read ${path}`);
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected goal owner ${method}`));
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => { const request = input instanceof Request ? new Request(input, init) : new Request(String(input), init); const path = new URL(request.url).pathname; const body=await request.text(); transport.push({path,method:request.method,body}); if (path!=="/vrooli.swarm_manager.v1.api.TransitionService/ListTransitions" || request.method!=="POST" || body!=="{}") { unexpected.push(path); throw new Error(`Unexpected goal owner transport ${path}`); } return new Response(JSON.stringify({ transitions: [] }), { status: 200, headers: { "Content-Type": "application/json" } }); }));
});
afterEach(() => {
  unmounts.splice(0).forEach(unmount => unmount()); cleanup(); getSpatialNav()?.dispose();
  try { expect(unexpected).toEqual([]); noWrites(); for (const client of clients) expect(client.getQueryData(["goal", "unrelated-goal"])).toEqual({ sentinel: "unchanged" }); }
  finally { clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); vi.unstubAllGlobals(); useBacklogStore.getState().setItems([]); for (const [key, value] of savedDisclosure) { if (value === null) localStorage.removeItem(key); else localStorage.setItem(key, value); } savedDisclosure.clear(); }
});

describe("Goal details owner navigation and read recovery", () => {
  it("retries a refused detail read through the same owner route and restores the exact goal", async () => { goalRefused = true; const { client } = mount(); expect(await screen.findByText("Unable to load goal", {}, { timeout: 12000 })).toBeVisible(); expect(client.getQueryData(["goal", "fixture-goal"])).toBeUndefined(); const attempts = vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/goals/fixture-goal").length; goalRefused = false; fireEvent.click(screen.getByRole("button", { name: "Try again" })); await loaded(); await waitFor(() => expect(client.getQueryData<GoalWithScope>(["goal", "fixture-goal"])?.goal.name).toBe("fixture-goal")); expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/goals/fixture-goal").length).toBeGreaterThan(attempts); noWrites(); }, 15000);
  it.each([["Progress", "goal-scope"], ["ETA Band", "goal-scope"], ["Blocked", "goal-blocked"]] as const)("%s reveals its collapsed owner section without a mutation", async (label, section) => { mount(); await loaded(); const toggle = screen.getByTestId(`${section}-toggle`); if (toggle.getAttribute("aria-expanded") === "true") fireEvent.click(toggle); expect(toggle).toHaveAttribute("aria-expanded", "false"); fireEvent.click(within(screen.getByTestId("goal-overview")).getByRole("button", { name: new RegExp(`^${label}`) })); await waitFor(() => expect(toggle).toHaveAttribute("aria-expanded", "true")); expect(screen.getByTestId(section)).toBeVisible(); noWrites(); });
  it("drills to the plan scoped to the exact goal and unmounts its detail view", async () => { mount(); await loaded(); fireEvent.click(screen.getByTestId("lens-bar-plan")); expect(await screen.findByText("Goal scoped plan destination")).toBeVisible(); expect(screen.getByTestId("goal-owner-location")).toHaveTextContent("/plan?goal=fixture-goal"); expect(screen.queryByRole("heading", { name: initial.goal.title })).toBeNull(); noWrites(); });
  it("opens the owner-scoped stats breakdown without launching work", async () => { mount(); await loaded(); fireEvent.click(screen.getByRole("button", { name: "Full stats breakdown" })); expect(await screen.findByText("Goal scoped stats destination")).toBeVisible(); expect(screen.getByTestId("goal-owner-location")).toHaveTextContent("/stats?goal=fixture-goal"); noWrites(); });
  it("moves between compact tabs while keeping the selected owner route", async () => { mount("?tab=activity"); await loaded(); expect(screen.getByText("Goal activity is recorded with milestone and proposal decisions.")).toBeVisible(); expect(screen.queryByTestId("lens-bar")).toBeNull(); fireEvent.click(screen.getByTestId("goal-details-tab-related")); await waitFor(() => expect(screen.getByTestId("goal-owner-location")).toHaveTextContent("?tab=related")); expect(screen.getByTestId("goal-ref-fix/fixture-item")).toBeVisible(); fireEvent.click(screen.getByTestId("goal-details-tab-overview")); await waitFor(() => expect(screen.getByTestId("goal-owner-location").textContent).toBe("/goals/fixture-goal")); expect(screen.getByTestId("lens-bar-plan")).toBeVisible(); noWrites(); });
  it("opens a returned related target through its exact backlog route", async () => { mount("?tab=related"); await loaded(); fireEvent.click(screen.getByTestId("goal-ref-fix/fixture-item")); expect(await screen.findByText("Selected owner item destination")).toBeVisible(); expect(screen.getByTestId("goal-owner-location")).toHaveTextContent("/backlog/fix/fixture-item"); noWrites(); });
  it("retries a refused file list through the same owner adapter without preservation writes", async () => { filesRefused = true; const { client } = mount("?tab=files"); await loaded(); expect(await screen.findByText("Unable to load files", {}, { timeout: 12000 })).toBeVisible(); expect(client.getQueryData(["goal", "fixture-goal", "files"])).toBeUndefined(); const attempts = vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/goals/fixture-goal/files").length; filesRefused = false; fireEvent.click(screen.getByRole("button", { name: "Try again" })); expect(await screen.findByTestId("file-tree-button-notes.md")).toBeVisible(); expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/goals/fixture-goal/files").length).toBeGreaterThan(attempts); expect(screen.getByTestId("goal-owner-location")).toHaveTextContent("?tab=files"); noWrites(); }, 15000);
  it("closes changed goal description using drawer chrome without writing the draft", async () => { const { client } = mount(); await loaded(); const before = client.getQueryData(["goal", "fixture-goal"]); fireEvent.click(screen.getByRole("button", { name: "Edit goal description" })); const dialog = await screen.findByRole("dialog", { name: "Edit goal" }); fireEvent.change(within(dialog).getByRole("textbox", { name: "Description" }), { target: { value: "Unsubmitted navigation draft" } }); fireEvent.click(within(dialog).getByRole("button", { name: "Close drawer" })); await waitFor(() => expect(screen.queryByRole("dialog", { name: "Edit goal" })).toBeNull()); expect(client.getQueryData(["goal", "fixture-goal"])).toBe(before); noWrites(); });
  it("closes target selection through drawer chrome without selecting or attaching anything", async () => { mount(); await loaded(); fireEvent.click(within(screen.getByTestId("goal-targets")).getByRole("button", { name: /Add/ })); const dialog = await screen.findByRole("dialog", { name: "Add target" }); fireEvent.click(within(dialog).getByRole("button", { name: "Close drawer" })); await waitFor(() => expect(screen.queryByRole("dialog", { name: "Add target" })).toBeNull()); noWrites(); });
  it("closes an edited milestone through drawer chrome and preserves the owner record", async () => { const { client } = mount("?tab=milestones"); await loaded(); const before = client.getQueryData(["goal", "fixture-goal"]); fireEvent.click(screen.getByRole("button", { name: "Edit Fixture milestone" })); const dialog = await screen.findByRole("dialog", { name: "Edit milestone" }); fireEvent.change(within(dialog).getByRole("textbox", { name: "Title" }), { target: { value: "Unsubmitted milestone title" } }); fireEvent.click(within(dialog).getByRole("button", { name: "Close drawer" })); await waitFor(() => expect(screen.queryByRole("dialog", { name: "Edit milestone" })).toBeNull()); expect(client.getQueryData(["goal", "fixture-goal"])).toBe(before); noWrites(); });
  it("closes milestone membership through drawer chrome without saving local changes", async () => { const { client } = mount("?tab=milestones"); await loaded(); const before = client.getQueryData(["goal", "fixture-goal"]); fireEvent.click(screen.getByRole("button", { name: "Manage Fixture milestone items" })); const dialog = await screen.findByRole("dialog", { name: "Manage milestone items" }); fireEvent.click(within(dialog).getByRole("checkbox")); fireEvent.click(within(dialog).getByRole("button", { name: "Close drawer" })); await waitFor(() => expect(screen.queryByRole("dialog", { name: "Manage milestone items" })).toBeNull()); expect(client.getQueryData(["goal", "fixture-goal"])).toBe(before); noWrites(); });
  it.each([["Archive", "goal-archive-confirm"], ["Delete", "goal-delete-confirm"]] as const)("dismisses the %s confirmation via Escape without confirming owner effects", async (action, testId) => { const { client } = mount(); await loaded(); const before = client.getQueryData(["goal", "fixture-goal"]); await openMenu(action); expect(await screen.findByTestId(testId)).toBeVisible(); fireEvent.keyDown(document, { key: "Escape" }); await waitFor(() => expect(screen.queryByTestId(testId)).toBeNull()); expect(client.getQueryData(["goal", "fixture-goal"])).toBe(before); noWrites(); });
});
