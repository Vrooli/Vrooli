import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { DEFAULT_SETTINGS } from "../services/settings-service";
import { GoalDetailsPage } from "./GoalDetailsPage";
import { ApiError, defaultApiClient } from "../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import { useBacklogStore } from "../stores";
import type { GoalWithScope } from "../types/goal";
import type { BacklogFile, BacklogItem } from "../types";
import type { NextActionFeedEntry } from "../services/next-action-service";

// Actual page, services, mutations, providers and cache. Network fixtures
// qualify current typed requests and owner outcomes, not actual agent dispatch.
const initial: GoalWithScope = {
  goal: { name: "owner-goal", title: "Owner outcome", description: "Bounded outcome", status: "active", priority: 4, targets: [], milestones: [], seeded: false, scopeHistory: [], created: "2026-01-28T00:00:00Z", updated: "2026-01-28T00:00:00Z" },
  scope: { targets: [], closure: [], completed: [], ready: [], blocked: [], total: 0, completedCount: 0, blockedCount: 0, progressPct: 0 }, eta: null,
};
const backlog: BacklogItem = { kind: "idea", name: "owner-item", title: "Hydrated owner item", description: "Exact scope", status: "ready", priority: 4, tags: [], suggestedSkills: [], created: "", updated: "" };
let current: GoalWithScope;
let files: BacklogFile[];
let contentRefused = false;
let feed: NextActionFeedEntry[];
let startRefused = false;
let completeStart: ((response: Response) => void) | undefined;
const clients: QueryClient[] = [], unexpected: string[] = [];
const requests: { path: string; body: unknown }[] = [];
const allowed = new Set<string>();
function response(body: unknown, status = 200) { return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }); }
function Location() { const location = useLocation(); return <output data-testid="goal-action-location">{location.pathname + location.search}</output>; }
function mount(query = "") {
  const client = createTestQueryClient(); clients.push(client);
  return { ...renderWithProviders(<><Routes><Route path="/goals/:name" element={<GoalDetailsPage />} /></Routes><Location /></>, { queryClient: client, initialEntries: [`/goals/owner-goal${query}`] }), client };
}
async function loaded() { return screen.findByText("Owner outcome"); }
function action(id: string, label: string, target?: string): NextActionFeedEntry {
  return { entity_kind: "goal", entity_ref: "owner-goal", entity_title: "Owner outcome", tier: 1, action: { id, compact_label: label, expanded_label: label, enabled: true, target, effect: "agent_run" } };
}
function noWrites() { for (const method of ["post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); expect(requests.filter(request => request.path.endsWith("/StartTransition"))).toEqual([]); }
beforeEach(() => {
  files = []; contentRefused = false; current = structuredClone(initial); feed = []; startRefused = false; completeStart = undefined; unexpected.length = 0; requests.length = 0; allowed.clear(); useBacklogStore.getState().setItems([]);
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/goals/owner-goal") return current;
    if (path === "/settings") { const { deleteConfirmation: _, ...settings } = DEFAULT_SETTINGS; return { settings }; }
    if (path === "/next-actions/feed") return { entries: feed };
    if (path === "/proposal-sessions?target_type=goal&target_ref=owner-goal") return { sessions: [] };
    if (path === "/goals/owner-goal/files") return { files };
    if (path === "/goals/owner-goal/files/notes.md") { if (contentRefused) throw new ApiError("http", "Goal file owner refused", { status: 403 }); return "# Authoritative notes\nOriginal content"; }
    if (path === "/goals/owner-goal/files/goal.json") return '{"title":"Canonical goal source"}';
    if (path.startsWith("/goals/owner-goal/files/") && files.some(file => path === `/goals/owner-goal/files/${file.path}`)) return "# Authoritative notes\nOriginal content";
    unexpected.push(path); throw new Error(`Unexpected goal action read ${path}`);
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected goal ${method}`));
  vi.stubGlobal("fetch", vi.fn(async (value: RequestInfo | URL, init?: RequestInit) => {
    const request = value instanceof Request ? new Request(value, init) : new Request(String(value), init);
    const path = new URL(request.url).pathname; const text = await request.text(); requests.push({ path, body: text ? JSON.parse(text) : null });
    if (path.endsWith("/ListTransitions")) return response({ transitions: ["goal.plan", "goal.discover", "goal.close_out"].map(key => ({ key, label: key, subject: "goal", kind: "TRANSITION_KIND_WORKFLOW" })) });
    if (path.endsWith("/StartTransition")) {
      if (startRefused) return response({ code: "permission_denied", message: "Current goal owner refused" }, 403);
      return new Promise<Response>(resolve => { completeStart = resolve; });
    }
    unexpected.push(path); throw new Error(`Unexpected goal action transport ${path}`);
  }));
});
afterEach(() => {
  // File workspace unmount may reacquire its controller while deregistering.
  // Finish real component cleanup, then dispose the remaining fixture controller.
  cleanup(); getSpatialNav()?.dispose();
  expect(unexpected).toEqual([]); for (const method of ["post", "put", "patch", "delete"] as const) if (!allowed.has(method)) expect(defaultApiClient[method]).not.toHaveBeenCalled(); clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); vi.unstubAllGlobals(); useBacklogStore.getState().setItems([]);
});

describe("GoalDetailsPage declared decision workflows", () => {
  it.each([["Plan goal", "goal-plan-action", "goal.plan", "Goal planning started"], ["Discover", "goal-discover-action", "goal.discover", "Discovery started"]] as const)("%s preserves the exact goal subject, blocks a duplicate and waits for owner completion", async (_label, testId, key, confirmation) => {
    const { client } = mount("?tab=decide"); await loaded(); const before = client.getQueryData(["goal", "owner-goal"]); fireEvent.click(screen.getByTestId(testId)); await waitFor(() => expect(completeStart).toBeDefined());
    const starts = requests.filter(request => request.path.endsWith("/StartTransition")); expect(starts).toHaveLength(1); expect(starts[0]?.body).toEqual({ transitionKey: key, subjectRef: { subject: "goal", value: "owner-goal" } });
    expect(screen.getByTestId("goal-plan-action")).toBeDisabled(); expect(screen.getByTestId("goal-discover-action")).toBeDisabled(); fireEvent.click(screen.getByTestId(testId)); expect(requests.filter(request => request.path.endsWith("/StartTransition"))).toHaveLength(1); expect(client.getQueryData(["goal", "owner-goal"])).toBe(before);
    current = { ...current, goal: { ...current.goal, description: "Owner accepted planning snapshot" } }; await act(async () => completeStart?.(response({ executionId: "fixture-admission" }))); expect(await screen.findByText(confirmation)).toBeVisible(); await waitFor(() => expect(client.getQueryData<GoalWithScope>(["goal", "owner-goal"])?.goal.description).toBe("Owner accepted planning snapshot"));
  });
  it.each(["goal-plan-action", "goal-discover-action"])("%s retains current owner state and reports refusal without alternate dispatch", async testId => {
    startRefused = true; const { client } = mount("?tab=decide"); await loaded(); const before = client.getQueryData(["goal", "owner-goal"]); fireEvent.click(screen.getByTestId(testId)); expect(await screen.findByText(/Current goal owner refused/)).toBeVisible(); expect(requests.filter(request => request.path.endsWith("/StartTransition"))).toHaveLength(1); expect(client.getQueryData(["goal", "owner-goal"])).toBe(before); expect(screen.queryByText("Goal planning started")).toBeNull(); expect(screen.queryByText("Discovery started")).toBeNull();
  });
  it("starts close-out only for the exact current goal and keeps completion claims bounded", async () => {
    feed = [action("close_out", "Close out")]; mount(); await loaded(); fireEvent.click(screen.getByRole("button", { name: /Close out/ })); await waitFor(() => expect(completeStart).toBeDefined()); expect(requests.find(request => request.path.endsWith("/StartTransition"))?.body).toEqual({ transitionKey: "goal.close_out", subjectRef: { subject: "goal", value: "owner-goal" } }); await act(async () => completeStart?.(response({ executionId: "fixture-close-out" }))); expect(await screen.findByText("Close-out started")).toBeVisible();
  });
  it("routes an unresolved milestone action to the decision queue without a mutation", async () => {
    feed = [action("define_criteria", "Define criteria", "milestone_criteria:missing")]; mount(); await loaded(); fireEvent.click(screen.getByRole("button", { name: /Define criteria/ })); expect(await screen.findByText("That action can't be completed from this page")).toBeVisible(); noWrites(); fireEvent.click(screen.getByRole("button", { name: "Open decision queue" })); await waitFor(() => expect(screen.getByTestId("goal-action-location")).toHaveTextContent("/plan?drawer=decisions"));
  });
  it("opens only the authored milestone criteria editor without starting work", async () => {
    current.goal.milestones = [{ name: "owner-milestone", title: "Owner milestone", description: "Explicit evidence", items: [], acceptanceCriteria: [], dependsOn: [] }]; feed = [action("define_criteria", "Define criteria", "milestone_criteria:owner-milestone")]; mount(); await loaded(); fireEvent.click(screen.getByRole("button", { name: /Define criteria/ })); const dialog = await screen.findByRole("dialog", { name: "Edit milestone" }); expect(within(dialog).getByRole("textbox", { name: "Name" })).toHaveValue("owner-milestone"); expect(screen.getByTestId("goal-action-location")).toHaveTextContent("tab=milestones&milestone=owner-milestone"); noWrites(); fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" })); await waitFor(() => expect(screen.queryByRole("dialog", { name: "Edit milestone" })).toBeNull());
  });
  it("opens the declared decide surface without dispatch or changes to the current goal", async () => {
    feed = [action("decide", "Decide proposals")]; const { client } = mount(); await loaded(); const before = client.getQueryData(["goal", "owner-goal"]); fireEvent.click(screen.getByRole("button", { name: /Decide proposals/ })); expect(await screen.findByTestId("goal-decide")).toBeVisible(); await waitFor(() => expect(screen.getByTestId("goal-action-location")).toHaveTextContent("?tab=decide")); noWrites(); expect(client.getQueryData(["goal", "owner-goal"])).toBe(before);
  });
});

describe("GoalDetailsPage current scope and completion navigation", () => {
  it("renders hydrated scope work and opens only its exact backlog target", async () => {
    current.scope.ready = ["idea/owner-item"]; current.scope.closure = ["idea/owner-item"]; current.scope.total = 1; current.scopeEntities = { items: { "idea/owner-item": backlog } }; mount(); await loaded(); fireEvent.click(screen.getByTestId("goal-ref-card-idea/owner-item")); await waitFor(() => expect(screen.getByTestId("goal-action-location")).toHaveTextContent("/backlog/idea/owner-item")); noWrites();
  });
  it("keeps unresolved owner scope references readable without inventing item data", async () => {
    current.scope.ready = ["idea/missing"]; current.scopeEntities = { items: {} }; mount(); await loaded(); expect(screen.getByTestId("goal-ref-idea/missing")).toHaveTextContent("missing"); expect(screen.queryByTestId("goal-ref-card-idea/missing")).toBeNull(); noWrites();
  });
  it("opens stats scoped to the exact goal without owner effects", async () => {
    mount(); await loaded(); fireEvent.click(screen.getByRole("button", { name: "Full stats breakdown" })); await waitFor(() => { const location = new URL(screen.getByTestId("goal-action-location").textContent ?? "", "https://fixture.invalid"); expect(location.pathname).toBe("/stats"); expect(location.searchParams.get("goal")).toBe("owner-goal"); }); noWrites();
  });
  it("reads authored files only while their actual tab is selected and displays owner emptiness", async () => {
    mount(); await loaded(); expect(vi.mocked(defaultApiClient.get).mock.calls.some(([path]) => path.endsWith("/files"))).toBe(false); fireEvent.click(screen.getByTestId("goal-details-tab-files")); await waitFor(() => expect(vi.mocked(defaultApiClient.get).mock.calls.some(([path]) => path === "/goals/owner-goal/files")).toBe(true)); expect(screen.getByTestId("goal-action-location")).toHaveTextContent("tab=files"); noWrites();
  });
  it("removes only the exact confirmed goal after owner completion, then returns to Plan", async () => {
    allowed.add("delete"); let complete!: (value: unknown) => void; vi.mocked(defaultApiClient.delete).mockImplementation(() => new Promise(resolve => { complete = resolve; })); const { client } = mount(); await loaded(); client.setQueryDefaults(["goal", "other-goal"], { gcTime: Infinity }); const sentinel = { ...initial, goal: { ...initial.goal, name: "other-goal" } }; client.setQueryData(["goal", "other-goal"], sentinel); fireEvent.click(screen.getByTestId("detail-header-actions")); fireEvent.click(await screen.findByTestId("goal-delete")); const dialog = screen.getByRole("alertdialog", { name: "Delete goal" }); fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "owner-goal" } }); fireEvent.click(screen.getByTestId("goal-delete-confirm-submit")); await waitFor(() => expect(vi.mocked(defaultApiClient.delete).mock.calls).toEqual([["/goals/owner-goal"]])); expect(screen.getByTestId("goal-delete-confirm-submit")).toBeDisabled(); fireEvent.click(screen.getByTestId("goal-delete-confirm-submit")); expect(defaultApiClient.delete).toHaveBeenCalledOnce(); expect(screen.getByTestId("goal-action-location")).toHaveTextContent("/goals/owner-goal"); await act(async () => complete({})); await waitFor(() => expect(screen.getByTestId("goal-action-location")).toHaveTextContent("/plan")); expect(client.getQueryData(["goal", "other-goal"])).toBe(sentinel);
  });
});

const note: BacklogFile = { name: "notes.md", path: "notes.md", type: "file", size: 42 };
async function selectNotes() { mount("?tab=files"); fireEvent.click(await screen.findByTestId("file-tree-button-notes.md")); const editor = await screen.findByTestId("file-preview-editor"); await waitFor(() => expect(editor).toHaveValue("# Authoritative notes\nOriginal content")); return editor; }
async function fileMenu(label: string) { fireEvent.click(await screen.findByTestId("file-header-actions-trigger")); fireEvent.click(await screen.findByRole("button", { name: label })); }
describe("GoalDetailsPage authored file ownership workflows", () => {
  beforeEach(() => { files = [{ ...note }]; });
  it("reads and previews the exact selected goal file without an owner mutation", async () => {
    await selectNotes(); expect(vi.mocked(defaultApiClient.get).mock.calls).toContainEqual(["/goals/owner-goal/files/notes.md", { responseType: "text" }]); noWrites();
  });
  it("reports file-content refusal and keeps the current file list without a fallback write", async () => {
    contentRefused = true; mount("?tab=files"); fireEvent.click(await screen.findByTestId("file-tree-button-notes.md")); await waitFor(() => expect(document.body.textContent).toContain("You don't have permission to access this resource."), { timeout: 12000 }); expect(screen.getByText("You don't have permission to access this resource.")).toBeVisible(); expect(screen.getByTestId("file-tree-button-notes.md")).toBeVisible(); noWrites();
  }, 15000);
  it("keeps canonical goal.json read-only and disables every file mutation menu entry", async () => {
    files = [{ ...note, name: "goal.json", path: "goal.json" }]; mount("?tab=files"); fireEvent.click(await screen.findByTestId("file-tree-button-goal.json")); const editor = await screen.findByTestId("file-preview-editor"); expect(editor).toHaveValue('{"title":"Canonical goal source"}'); expect(editor).toHaveAttribute("data-read-only", "true"); fireEvent.click(screen.getByTestId("file-header-actions-trigger")); for (const label of ["Rename", "Move", "Copy", "Delete"]) { const action = screen.getByRole("button", { name: new RegExp(label) }); expect(action).toBeDisabled(); fireEvent.click(action); } noWrites();
  });
  it("refuses slash-containing rename input and cancellation without a request or cache change", async () => {
    const { client } = mount("?tab=files"); fireEvent.click(await screen.findByTestId("file-tree-button-notes.md")); await waitFor(() => expect(screen.getByTestId("file-preview-editor")).toHaveValue("# Authoritative notes\nOriginal content")); const before = client.getQueryData(["goal", "owner-goal", "files"]); await fileMenu("Rename"); const dialog = screen.getByRole("dialog", { name: "Rename file" }); fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "other/path.md" } }); fireEvent.click(within(dialog).getByRole("button", { name: "Apply" })); expect(await screen.findByText("Rename requires a file or folder name without slashes.")).toBeVisible(); noWrites(); fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" })); expect(client.getQueryData(["goal", "owner-goal", "files"])).toBe(before);
  });
  it.each([["Rename", "renamed.md", "rename"], ["Move", "archive/notes.md", "move"], ["Copy", "copies/notes.md", "copy"]] as const)("%s submits only the current goal/path and refreshes after authoritative completion", async (label, destination, operation) => {
    allowed.add("patch"); let complete!: (value: unknown) => void; vi.mocked(defaultApiClient.patch).mockImplementation(() => new Promise(resolve => { complete = resolve; })); await selectNotes(); await fileMenu(label); const dialog = screen.getByRole("dialog", { name: `${label} file` }); fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: destination } }); fireEvent.click(within(dialog).getByRole("button", { name: "Apply" })); await waitFor(() => expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/goals/owner-goal/files", { operation, source_path: "notes.md", destination_path: destination }]])); expect(screen.getByTestId("file-tree-button-notes.md")).toBeVisible(); const updated = { ...note, path: destination, name: destination.split("/").pop()! }; files = operation === "copy" ? [note, updated] : [updated]; await act(async () => complete({ file: { ...updated, size: "42" } })); await waitFor(() => expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/goals/owner-goal/files").length).toBeGreaterThan(1)); expect(defaultApiClient.patch).toHaveBeenCalledOnce(); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
  it("reports a refused file operation without replacing the owner file list or another write", async () => {
    allowed.add("patch"); vi.mocked(defaultApiClient.patch).mockRejectedValue(new Error("File rename owner refused")); const { client } = mount("?tab=files"); fireEvent.click(await screen.findByTestId("file-tree-button-notes.md")); await waitFor(() => expect(screen.getByTestId("file-preview-editor")).toHaveValue("# Authoritative notes\nOriginal content")); const before = client.getQueryData(["goal", "owner-goal", "files"]); await fileMenu("Rename"); fireEvent.change(within(screen.getByRole("dialog", { name: "Rename file" })).getByRole("textbox"), { target: { value: "renamed.md" } }); fireEvent.click(screen.getByTestId("confirm-file-action")); expect(await screen.findByText("File rename owner refused")).toBeVisible(); expect(defaultApiClient.patch).toHaveBeenCalledOnce(); expect(client.getQueryData(["goal", "owner-goal", "files"])).toBe(before); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
  it("uploads the exact selected file to the current goal and refreshes only after owner completion", async () => {
    allowed.add("post"); let complete!: (value: unknown) => void; vi.mocked(defaultApiClient.post).mockImplementation(() => new Promise(resolve => { complete = resolve; })); mount("?tab=files"); fireEvent.click(await screen.findByRole("button", { name: "Upload files" })); const uploaded = new File(["Exact evidence"], "evidence.txt", { type: "text/plain" }); fireEvent.change(screen.getByTestId("file-upload-input"), { target: { files: [uploaded] } }); await waitFor(() => expect(defaultApiClient.post).toHaveBeenCalledOnce()); const request = vi.mocked(defaultApiClient.post).mock.calls[0]!; expect(request[0]).toBe("/goals/owner-goal/files"); expect(request[1]).toBeInstanceOf(FormData); expect((request[1] as FormData).get("file")).toBe(uploaded); expect((request[1] as FormData).get("path")).toBeNull(); expect(request[2]).toEqual({ headers: {} }); const ownerFile = { ...note, name: "evidence.txt", path: "evidence.txt" }; files = [note, ownerFile]; await act(async () => complete({ file: { ...ownerFile, size: "14" } })); await waitFor(() => expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/goals/owner-goal/files").length).toBeGreaterThan(1)); expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
});
