import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { BacklogDetailsPage } from "./BacklogDetailsPage";
import { defaultApiClient } from "../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import { useAgentActivitiesStore, useBacklogDetailUIStore, useBacklogStore } from "../stores";
import type { BacklogItem } from "../types";
import type { WorkFeedEntry } from "../components/backlog/activity-surface/work-feed-list";

// Real page, hooks, owner read services, stores and widgets. No subject mocks or
// server writes; read metadata never establishes product acceptance or a run.
const owner = "/backlog/idea/fixture-rich";
const clients: QueryClient[] = [];
const views: { unmount: () => void }[] = [];
const unexpected: string[] = [];
const readTransports: { method: string; path: string; body: unknown }[] = [];
let current: BacklogItem;
let entries: WorkFeedEntry[];
let feedRefused: boolean;
let detailRefused: boolean;
const episode = "/fixture-read-only/episode";
function Location() { const location = useLocation(); return <output data-testid="backlog-rich-location">{location.pathname + location.search}</output>; }
function mount(query = "") { const client = createTestQueryClient(); clients.push(client); const view = renderWithProviders(<><Routes><Route path="/backlog/:kind/:name" element={<BacklogDetailsPage />} /></Routes><Location /></>, { queryClient: client, initialEntries: [`${owner}?keep=fixture${query}`] }); views.push(view); return client; }
async function loaded() { await screen.findByText("Rich owner item"); }
async function activity() { mount("&tab=activity"); await loaded(); return screen.findByRole("region", { name: "Work history" }); }
function closeEpisode(dialog: HTMLElement) { const close = within(dialog).getAllByRole("button", { name: "Close drawer" }); expect(close.length).toBeGreaterThan(0); fireEvent.click(close[0]!); }
beforeEach(() => {
  current = { kind: "idea", name: "fixture-rich", title: "Rich owner item", description: "Owner read-only specification", status: "backlog", priority: 1, tags: [], suggestedSkills: [], created: "2026-01-01T00:00:00Z", updated: "2026-01-02T00:00:00Z" };
  entries = []; feedRefused = false; detailRefused = false; unexpected.length = 0; readTransports.length = 0;
  useBacklogStore.getState().setItems([]); useBacklogDetailUIStore.getState().reset(); useAgentActivitiesStore.setState({ activities: [], isRefreshing: false });
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/settings") return { settings: {} };
    if (path === "/goals") return { items: [] };
    if (path === owner) return { item: current };
    if (path === `${owner}/files`) return { files: [] };
    if (path === `${owner}/plan-render`) return { markdown: "", planRef: null };
    if (path === `${owner}/next-action`) return { action: { id: "none", compactLabel: "No action", expandedLabel: "No action", enabled: false, blockers: [] } };
    if (path === `${owner}/archive/targets`) return { targets: [], requirements: [], has_archive: false };
    if (path === `${owner}/review`) return { rounds: [] };
    if (path === `${owner}/work-feed`) { if (feedRefused) throw new Error("Owner work-feed read refused"); return { items: entries }; }
    if (path === "/execution?backlog_kind=idea&backlog_name=fixture-rich") return { items: [] };
    if (path.startsWith("/backlog?")) return { items: [], blocking: {} };
    if (path.startsWith("/agent-activities?")) return { items: [] };
    if (path === "/proposal-sessions?target_type=backlog_item&target_ref=idea%2Ffixture-rich") return { sessions: [] };
    if (path === episode) { if (detailRefused) throw new Error("Owner episode read refused"); return { status: "needs_fixup", strategy: "bounded fixture strategy", agent_assessment: "Owner assessment is incomplete" }; }
    unexpected.push(path); throw new Error(`Unexpected rich owner read ${path}`);
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Forbidden rich owner ${method}`));
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? new Request(input, init) : new Request(String(input), init); const path = new URL(request.url).pathname; const body: unknown = JSON.parse(await request.text());
    readTransports.push({ method: request.method, path, body });
    if (path === "/vrooli.swarm_manager.v1.api.TransitionService/ListTransitions" && request.method === "POST") { expect(body).toEqual({}); return new Response(JSON.stringify({ transitions: [] }), { status: 200, headers: { "Content-Type": "application/json" } }); }
    unexpected.push(path); throw new Error("Forbidden alternate transport");
  }));
});
afterEach(() => {
  try {
    views.splice(0).forEach(view => view.unmount()); cleanup(); getSpatialNav()?.dispose(); expect(unexpected).toEqual([]);
    for (const method of ["post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled();
    expect(readTransports.every(value => value.method === "POST" && value.path === "/vrooli.swarm_manager.v1.api.TransitionService/ListTransitions" && JSON.stringify(value.body) === "{}")).toBe(true);
  } finally { clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); vi.unstubAllGlobals(); useBacklogStore.getState().setItems([]); useBacklogDetailUIStore.getState().reset(); useAgentActivitiesStore.setState({ activities: [], isRefreshing: false }); }
});
describe("Backlog rich owner read and local display workflows", () => {
  it("shows prospective auto-filer intent without accepting or dismissing the suggestion", async () => {
    current = { ...current, status: "suggested" }; mount(); await loaded();
    expect(screen.getByText("Auto-filer suggestion")).toBeVisible();
    expect(screen.getByRole("button", { name: "Accept" })).toBeEnabled(); expect(screen.getByRole("button", { name: "Dismiss" })).toBeEnabled();
    expect(screen.getByTestId("backlog-rich-location")).toHaveTextContent("keep=fixture");
  });
  it("distinguishes archived owner metadata from a fresh suggestion without restoring it", async () => {
    current = { ...current, status: "suggested", archivedAt: "2026-01-03T00:00:00Z" }; mount(); await loaded();
    expect(screen.queryByText("Auto-filer suggestion")).toBeNull(); expect(screen.getByRole("button", { name: "Unarchive" })).toBeEnabled(); expect(screen.getByText(/^Archived /)).toBeVisible();
  });
  it("expands and collapses long canonical description without editing or mutating it", async () => {
    current = { ...current, description: "Owner description\n\n" + "Retained detail. ".repeat(12) }; mount(); await loaded();
    fireEvent.click(screen.getByRole("button", { name: "Show more…" }));
    expect(screen.getByRole("button", { name: "Show less" })).toBeVisible(); expect(screen.getByText(/Retained detail\./)).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Show less" })); expect(screen.getByRole("button", { name: "Show more…" })).toBeVisible();
  });
  it("independently expands owner allow and deny patterns without editing acceptance scope", async () => {
    current = { ...current, acceptanceAllow: ["a/**", "b/**", "c/**", "owner-extra/**"], acceptanceDeny: ["x/**", "y/**", "z/**", "protected-extra/**"] }; mount(); await loaded();
    const allow = screen.getByText("Allow").parentElement!; const deny = screen.getByText("Deny").parentElement!;
    expect(screen.queryByText("owner-extra/**")).toBeNull(); expect(screen.queryByText("protected-extra/**")).toBeNull();
    fireEvent.click(within(allow).getByRole("button", { name: "Show more… (1 more)" })); expect(within(allow).getByText("owner-extra/**")).toBeVisible(); expect(screen.queryByText("protected-extra/**")).toBeNull();
    fireEvent.click(within(deny).getByRole("button", { name: "Show more… (1 more)" })); expect(within(deny).getByText("protected-extra/**")).toBeVisible();
    fireEvent.click(within(allow).getByRole("button", { name: "Show less" })); expect(screen.queryByText("owner-extra/**")).toBeNull(); expect(within(deny).getByText("protected-extra/**")).toBeVisible();
  });
  it("retains a refused work-feed read as unavailable history without an alternate write", async () => {
    feedRefused = true; mount("&tab=activity"); await loaded();
    expect(await screen.findByText("Work history could not be loaded. Try again shortly.", {}, { timeout: 12000 })).toBeVisible();
    fireEvent.click(screen.getByTestId("backlog-details-tab-info")); await waitFor(() => expect(screen.getByTestId("backlog-rich-location")).toHaveTextContent("keep=fixture"));
    expect(screen.queryByRole("region", { name: "Work history" })).toBeNull();
  }, 15000);
  it("shows synthetic execution completion separately from acceptance and closes its read-only source context", async () => {
    entries = [{ id: "execution/fixture", kind: "execution", title: "Owner execution episode", outcome: "failed", actor: "Fixture operator", started_at: "2026-01-01T00:00:00Z", ended_at: "2026-01-01T00:02:00Z", cost_estimate: 0.4, correlation: { execution_id: "fixture-execution" }, detail_ref: "/executions/fixture-execution", detail_api_ref: episode }];
    const history = await activity(); fireEvent.click(within(history).getByRole("button", { name: /Owner execution episode/ }));
    const dialog = await screen.findByRole("dialog", { name: "Owner execution episode" });
    expect(await within(dialog).findByText("Agent session ended; product acceptance is separate.")).toBeVisible();
    expect(within(dialog).getByText("bounded fixture strategy")).toBeVisible();
    expect(within(dialog).getByRole("link", { name: "Open execution" })).toHaveAttribute("href", "/executions/fixture-execution");
    closeEpisode(dialog); await waitFor(() => expect(screen.queryByRole("dialog", { name: "Owner execution episode" })).toBeNull());
  });
  it("keeps refused source-detail evidence cancelable without treating it as accepted review", async () => {
    detailRefused = true; entries = [{ id: "review/fixture", kind: "review", title: "Owner review episode", started_at: "2026-01-01T00:00:00Z", detail_ref: "/reviews/fixture", detail_api_ref: episode }];
    const history = await activity(); fireEvent.click(within(history).getByRole("button", { name: /Owner review episode/ }));
    const dialog = await screen.findByRole("dialog", { name: "Owner review episode" });
    expect(await within(dialog).findByText("Source detail is temporarily unavailable.")).toBeVisible();
    expect(within(dialog).queryByText("Owner assessment is incomplete")).toBeNull(); closeEpisode(dialog);
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Owner review episode" })).toBeNull());
  });
  it("filters finished and active owner episodes locally without changing owner queries or creating work", async () => {
    entries = [{ id: "workflow/fixture", kind: "workflow", title: "Owner active workflow", started_at: "2026-01-01T00:00:00Z", detail_ref: "/workflows/fixture" }, { id: "workshop/fixture", kind: "workshop", title: "Owner finished workshop", outcome: "completed", started_at: "2026-01-01T00:00:00Z", ended_at: "2026-01-01T00:00:30Z", detail_ref: "/workshops/fixture" }, { id: "event/fixture", kind: "event", title: "Owner item update", outcome: "changed", started_at: "2026-01-01T00:00:00Z", ended_at: "2026-01-01T00:01:00Z", detail_ref: "/events/fixture" }];
    const history = await activity(); const before = vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === `${owner}/work-feed`).length;
    fireEvent.change(within(history).getByLabelText("Filter work outcome"), { target: { value: "terminal" } }); expect(within(history).queryByText("Owner active workflow")).toBeNull(); expect(within(history).getByText("Owner finished workshop")).toBeVisible();
    fireEvent.change(within(history).getByLabelText("Filter work history"), { target: { value: "event" } }); expect(within(history).queryByText("Owner finished workshop")).toBeNull(); expect(within(history).getByText("Owner item update")).toBeVisible();
    expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === `${owner}/work-feed`).length).toBe(before);
  });
});
