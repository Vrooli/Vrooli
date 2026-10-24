import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { DecisionDrawer } from "./DecisionDrawer";
import { defaultApiClient } from "../../../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../../../test-utils";
import { useBacklogStore } from "../../../stores";
import { useSnoozeStore } from "../../../stores/snooze-store";
import { NEXT_ACTION_FEED_QUERY_KEY } from "../../../hooks/usePendingDecisionCount";
import type { NextActionFeedEntry } from "../../../services/next-action-service";

// Outside-only desired workflow draft. Real drawer/hooks/stores/services/routes;
// only upstream reads substituted. Mutation channels reject and remain unused.
function entry(name: string): NextActionFeedEntry { return { entity_kind: "backlog_item", entity_ref: `fix/${name}`, entity_title: `Owner ${name}`, tier: 1, action: { id: "resolve_dependencies", compact_label: "Inspect dependencies", expanded_label: `Inspect ${name} dependencies`, enabled: true, reason: "Fixture owner dependencies require inspection", effect: "none", blockers: [{ code: "pending_dependency", message: "Fixture prerequisite remains pending" }] } }; }
const clients: QueryClient[] = [];
const unmounts: Array<() => void> = [];
const unexpected: string[] = [];
const transport: {path:string;method:string;body:string}[] = [];
let entries: NextActionFeedEntry[];
let feedRefused: boolean;
let deferFeed: boolean;
let releaseFeed: ((value: unknown) => void) | undefined;
let storedSnooze: string | null;
function Location() { const location = useLocation(); return <output data-testid="drawer-owner-location">{location.pathname + location.search}</output>; }
function mount(options: { open?: boolean; route?: string; scope?: string | null } = {}) {
  const client = createTestQueryClient(); clients.push(client); const close = vi.fn(); const completed = vi.fn();
  function Parent() { const [open, setOpen] = useState(options.open ?? true); return <DecisionDrawer isOpen={open} onClose={() => { close(); setOpen(false); }} scopeItemKey={options.scope ?? null} currentQuestionId={null} onCurrentQuestionChange={vi.fn()} onCompleted={completed} />; }
  const view = renderWithProviders(<><Routes><Route path="/plan" element={<Parent />} /><Route path="/backlog/:kind/:name" element={<h1>Exact backlog owner destination</h1>} /><Route path="/goals/:name" element={<h1>Exact goal owner destination</h1>} /></Routes><Location /></>, { queryClient: client, initialEntries: [options.route ?? "/plan?keep=fixture"] }); unmounts.push(view.unmount); return { ...view, client, close, completed };
}
function noWrites() { for (const method of ["post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); expect(transport.every(x=>x.path==="/vrooli.swarm_manager.v1.api.TransitionService/ListTransitions"&&x.method==="POST"&&x.body==="{}")).toBe(true); }
function reads() { return vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/next-actions/feed").length; }
function closeChrome() { const button = within(screen.getByRole("dialog")).getAllByRole("button", { name: "Close drawer" })[0]; if (!button) throw new Error("Drawer close chrome missing"); return button; }
async function ready() { return screen.findByTestId("decision-queue-header"); }
async function openFocusedNavigator(firstOpen = true) {
  if (firstOpen) await waitFor(() => expect(screen.getByTestId("plan-decision-drawer.grabber")).toHaveFocus());
  const counter = screen.getByTestId("decision-queue-counter"); counter.focus(); fireEvent.click(counter);
  await screen.findByTestId("decision-queue-navigator");
  await waitFor(() => expect(within(screen.getByTestId("decision-queue-navigator")).getByRole("textbox", { name: "Filter decisions" })).toHaveFocus());
  return screen.getByTestId("decision-queue-navigator");
}
beforeEach(() => {
  entries = [entry("fixture-first"), entry("fixture-second")]; feedRefused = false; deferFeed = false; releaseFeed = undefined; unexpected.length = 0; transport.length = 0; useBacklogStore.getState().setItems([]); useSnoozeStore.setState({ entries: new Map() }); storedSnooze = localStorage.getItem("swarm-manager.snooze.v1"); localStorage.removeItem("swarm-manager.snooze.v1");
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/next-actions/feed") { if (feedRefused) throw new Error("Fixture decision owner refused feed"); if (deferFeed) return new Promise(resolve => { releaseFeed = resolve; }); return { entries }; }
    if (path === "/integrations") return { integrations: [] };
    if (path === "/backlog/summary") return { pending_questions: { items: [] } };
    if (path === "/proposal-sessions" || path === "/proposal-sessions?target_type=backlog_item&target_ref=fix%2Ffixture-second") return { sessions: [] };
    if (path === "/goals/fixture-goal") return { goal: { name: "fixture-goal", title: "Fixture goal", status: "active", priority: 2, milestones: [{ name: "fixture-milestone", title: "Fixture milestone", items: [], acceptance_criteria: ["Inspect synthetic evidence"], depends_on: [] }], targets: [], scope_history: [], seeded: false, created: "2026-01-01T00:00:00Z", updated: "2026-01-01T00:00:00Z" } };
    unexpected.push(path); throw new Error(`Unexpected drawer owner read ${path}`);
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected drawer owner ${method}`));
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => { const request = input instanceof Request ? new Request(input, init) : new Request(String(input), init); const path = new URL(request.url).pathname; const body=await request.text();transport.push({path,method:request.method,body});if (request.method !== "POST" || path!=="/vrooli.swarm_manager.v1.api.TransitionService/ListTransitions" || body!=="{}") { unexpected.push(path); throw new Error(`Unexpected drawer owner transport ${path}`); } return new Response(JSON.stringify({ transitions: [] }), { status: 200, headers: { "Content-Type": "application/json" } }); }));
});
afterEach(() => { unmounts.splice(0).forEach(unmount => unmount()); cleanup(); getSpatialNav()?.dispose(); try { expect(unexpected).toEqual([]); noWrites(); } finally { clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); vi.unstubAllGlobals(); useBacklogStore.getState().setItems([]); useSnoozeStore.setState({ entries: new Map() }); if (storedSnooze === null) localStorage.removeItem("swarm-manager.snooze.v1"); else localStorage.setItem("swarm-manager.snooze.v1", storedSnooze); } });

describe("Decision drawer owner reads and canonical navigation", () => {
  it("holds a pending owner feed without claiming there are no decisions", async () => { deferFeed = true; const view = mount(); expect(await screen.findByText("Loading operator inbox…")).toBeVisible(); expect(screen.queryByText("Nothing needs your decision.")).toBeNull(); await waitFor(() => expect(releaseFeed).toBeDefined()); await act(async () => releaseFeed?.({ entries })); await ready(); expect(screen.getByTestId("decision-queue-title")).toHaveTextContent("Owner fixture-first"); expect(view.completed).not.toHaveBeenCalled(); noWrites(); });
  it("renders a resolved empty owner feed and closes through the actual parent callback", async () => { entries = []; const view = mount(); expect(await screen.findByText("Nothing needs your decision.")).toBeVisible(); expect(screen.queryByTestId("decision-queue-nav")).toBeNull(); fireEvent.click(closeChrome()); await waitFor(() => expect(screen.queryByTestId("plan-decision-drawer")).toBeNull()); expect(view.close).toHaveBeenCalledOnce(); expect(view.completed).not.toHaveBeenCalled(); expect(screen.getByTestId("drawer-owner-location")).toHaveTextContent("/plan?keep=fixture"); noWrites(); });
  it("does not acquire owner data or catalog while closed", () => { mount({ open: false }); expect(defaultApiClient.get).not.toHaveBeenCalled(); expect(fetch).not.toHaveBeenCalled(); expect(screen.queryByTestId("plan-decision-drawer")).toBeNull(); noWrites(); });
  it("navigates the current backlog owner and carries the queue position while unmounting the drawer", async () => { const view = mount({ route: "/plan?keep=fixture&decisionPosition=1" }); await ready(); fireEvent.click(screen.getByTestId("decision-queue-title")); expect(await screen.findByText("Exact backlog owner destination")).toBeVisible(); expect(screen.getByTestId("drawer-owner-location")).toHaveTextContent("/backlog/fix/fixture-second?drawer=decisions&decisionPosition=1"); expect(screen.queryByTestId("plan-decision-drawer")).toBeNull(); expect(view.completed).not.toHaveBeenCalled(); noWrites(); });
  it("opens the exact goal milestone deep link without starting its review", async () => { entries = [{ ...entry("fixture-goal"), entity_kind: "goal", entity_ref: "fixture-goal", action: { id: "review", compact_label: "Inspect milestone", expanded_label: "Inspect fixture milestone", enabled: true, target: "milestone_review:fixture-milestone", effect: "agent_run" } }]; mount(); await ready(); expect(await screen.findByText("Inspect synthetic evidence")).toBeVisible(); fireEvent.click(screen.getByTestId("decision-queue-title")); expect(await screen.findByText("Exact goal owner destination")).toBeVisible(); expect(screen.getByTestId("drawer-owner-location")).toHaveTextContent("/goals/fixture-goal?drawer=decisions&decisionPosition=0&tab=milestones&milestone=fixture-milestone"); noWrites(); });
  it("does not invent navigation for a capture entry", async () => { entries = [{ ...entry("fixture-capture"), entity_kind: "capture", entity_ref: "fixture-capture" }]; mount(); await ready(); fireEvent.click(screen.getByTestId("decision-queue-title")); expect(screen.getByTestId("drawer-owner-location")).toHaveTextContent("/plan?keep=fixture"); expect(screen.getByTestId("plan-decision-drawer")).toBeVisible(); noWrites(); });
  it("leaves a malformed backlog reference in the drawer rather than navigating a guessed owner", async () => { entries = [{ ...entry("fixture-invalid"), entity_ref: "fix/" }]; mount(); await ready(); fireEvent.click(screen.getByTestId("decision-queue-title")); expect(screen.getByTestId("drawer-owner-location")).toHaveTextContent("/plan?keep=fixture"); expect(screen.getByTestId("plan-decision-drawer")).toBeVisible(); noWrites(); });
  it("reconciles a shortened owner queue to its remaining entry without changing unrelated URL state", async () => { const view = mount({ route: "/plan?keep=fixture&decisionPosition=1" }); await ready(); expect(screen.getByTestId("decision-queue-title")).toHaveTextContent("Owner fixture-second"); entries = [entry("fixture-first")]; await act(async () => { await view.client.invalidateQueries({ queryKey: NEXT_ACTION_FEED_QUERY_KEY }); }); await waitFor(() => expect(screen.getByTestId("decision-queue-counter")).toHaveTextContent("1 of 1")); expect(screen.getByTestId("decision-queue-title")).toHaveTextContent("Owner fixture-first"); expect(screen.getByTestId("drawer-owner-location")).toHaveTextContent("keep=fixture"); expect(view.completed).not.toHaveBeenCalled(); noWrites(); });
  it("closes the navigator with Escape without closing the containing drawer", async () => { const view = mount(); await ready(); await waitFor(() => expect(screen.getByTestId("plan-decision-drawer.grabber")).toHaveFocus()); screen.getByTestId("decision-queue-counter").focus(); fireEvent.click(screen.getByTestId("decision-queue-counter")); expect(await screen.findByTestId("decision-queue-navigator")).toBeVisible(); fireEvent.keyDown(document, { key: "Escape" }); await waitFor(() => expect(screen.queryByTestId("decision-queue-navigator")).toBeNull()); expect(screen.getByTestId("plan-decision-drawer")).toBeVisible(); expect(view.close).not.toHaveBeenCalled(); await waitFor(() => expect(screen.getByTestId("decision-queue-counter")).toHaveFocus()); fireEvent.keyDown(document, { key: "Escape" }); await waitFor(() => expect(screen.queryByTestId("plan-decision-drawer")).toBeNull()); expect(view.close).toHaveBeenCalledOnce(); noWrites(); });
  it("closes an unmatched filter without changing queue position or submitting decisions", async () => { mount(); await ready(); const navigator = await openFocusedNavigator(); fireEvent.change(within(navigator).getByRole("textbox", { name: "Filter decisions" }), { target: { value: "No fixture owner matches" } }); expect(within(navigator).getByText("No decisions match that filter.")).toBeVisible(); fireEvent.keyDown(document, { key: "Escape" }); await waitFor(() => expect(screen.queryByTestId("decision-queue-navigator")).toBeNull()); expect(screen.getByTestId("decision-queue-title")).toHaveTextContent("Owner fixture-first"); expect(screen.getByTestId("drawer-owner-location")).toHaveTextContent("/plan?keep=fixture"); noWrites(); });
  it("snoozes only the selected queue entry and allows parent close without owner writes", async () => { const view = mount({ scope: "fix/fixture-second" }); await ready(); fireEvent.click(screen.getByTestId("decision-queue-snooze")); expect([...useSnoozeStore.getState().entries.keys()]).toEqual(["backlog_item:fix/fixture-second"]); expect(view.completed).not.toHaveBeenCalled(); fireEvent.click(closeChrome()); await waitFor(() => expect(screen.queryByTestId("plan-decision-drawer")).toBeNull()); expect(view.close).toHaveBeenCalledOnce(); noWrites(); });
  it("keeps the navigator filter across close and reopening and jumps to the original owner index", async () => {
    const view = mount(); await ready(); const navigator = await openFocusedNavigator();
    fireEvent.change(within(navigator).getByRole("textbox", { name: "Filter decisions" }), { target: { value: "fixture-second" } });
    expect(within(navigator).getAllByTestId("decision-queue-navigator-row")).toHaveLength(1);
    fireEvent.keyDown(document, { key: "Escape" }); await waitFor(() => expect(screen.queryByTestId("decision-queue-navigator")).toBeNull());
    const reopened = await openFocusedNavigator(false);
    expect(within(reopened).getByRole("textbox", { name: "Filter decisions" })).toHaveValue("fixture-second");
    fireEvent.click(within(reopened).getByTestId("decision-queue-navigator-row"));
    await waitFor(() => expect(screen.queryByTestId("decision-queue-navigator")).toBeNull());
    await waitFor(() => {
      expect(screen.getByTestId("decision-queue-title")).toHaveTextContent("Owner fixture-second");
      expect(screen.getByTestId("drawer-owner-location")).toHaveTextContent("/plan?keep=fixture&decisionPosition=1");
    });
    expect(view.close).not.toHaveBeenCalled(); noWrites();
  });

});

// Desired regressions: current source ignores feedQuery.error and renders false
// empty success. Retain these predicates for baseline/independent repair review.
describe("Decision drawer refused-owner feed regressions", () => {
  it("shows owner refusal instead of claiming the decision queue is empty", async () => { feedRefused = true; const view = mount(); expect(await screen.findByText("Fixture decision owner refused feed", {}, { timeout: 12000 })).toBeVisible(); expect(screen.queryByText("Nothing needs your decision.")).toBeNull(); expect(view.completed).not.toHaveBeenCalled(); expect(screen.getByTestId("drawer-owner-location")).toHaveTextContent("/plan?keep=fixture"); noWrites(); }, 15000);
  it("offers explicit same-route retry after refused feed and resolves only on the new owner response", async () => { feedRefused = true; const view = mount(); expect(await screen.findByText("Fixture decision owner refused feed", {}, { timeout: 12000 })).toBeVisible(); const before = reads(); feedRefused = false; fireEvent.click(screen.getByRole("button", { name: "Try again" })); await ready(); expect(reads()).toBeGreaterThan(before); expect(screen.getByTestId("decision-queue-title")).toHaveTextContent("Owner fixture-first"); expect(view.completed).not.toHaveBeenCalled(); noWrites(); }, 15000);
  it("keeps explicit retry pending and refuses a same-turn duplicate owner read", async () => { feedRefused = true; const view = mount(); await screen.findByText("Fixture decision owner refused feed", {}, { timeout: 12000 }); const attempts = reads(); feedRefused = false; deferFeed = true; const retry = screen.getByRole("button", { name: "Try again" }); fireEvent.click(retry); fireEvent.click(retry); await waitFor(() => expect(releaseFeed).toBeDefined()); expect(reads()).toBe(attempts + 1); expect(retry).toBeDisabled(); expect(screen.queryByTestId("decision-queue-header")).toBeNull(); expect(view.completed).not.toHaveBeenCalled(); await act(async () => releaseFeed?.({ entries })); await ready(); expect(reads()).toBe(attempts + 1); noWrites(); }, 15000);
  it("replaces a refused stale owner feed with an error instead of keeping its action offered", async () => { const view = mount(); await ready(); const priorAction = screen.getByTestId("next-action-primary"); const before = view.client.getQueryData(NEXT_ACTION_FEED_QUERY_KEY); feedRefused = true; await act(async () => { await view.client.invalidateQueries({ queryKey: NEXT_ACTION_FEED_QUERY_KEY }); }); expect(await screen.findByText("Fixture decision owner refused feed", {}, { timeout: 12000 })).toBeVisible(); expect(screen.queryByTestId("next-action-primary")).toBeNull(); fireEvent.click(priorAction); expect(view.client.getQueryData(NEXT_ACTION_FEED_QUERY_KEY)).toBe(before); expect(view.completed).not.toHaveBeenCalled(); noWrites(); }, 15000);
  it("distinguishes authoritative empty response after explicit retry from the prior owner refusal", async () => { feedRefused = true; const view = mount(); await screen.findByText("Fixture decision owner refused feed", {}, { timeout: 12000 }); feedRefused = false; entries = []; fireEvent.click(screen.getByRole("button", { name: "Try again" })); expect(await screen.findByText("Nothing needs your decision.")).toBeVisible(); expect(screen.queryByText("Fixture decision owner refused feed")).toBeNull(); expect(screen.queryByTestId("decision-queue-nav")).toBeNull(); expect(view.completed).not.toHaveBeenCalled(); noWrites(); }, 15000);

  it("hides cached entry and footer throughout explicit refused-feed retry", async () => {
    const view = mount({ route: "/plan?keep=fixture&decisionPosition=1" }); await ready();
    const before = view.client.getQueryData(NEXT_ACTION_FEED_QUERY_KEY);
    feedRefused = true; await act(async () => { await view.client.invalidateQueries({ queryKey: NEXT_ACTION_FEED_QUERY_KEY }); });
    await screen.findByText("Fixture decision owner refused feed", {}, { timeout: 12000 });
    feedRefused = false; deferFeed = true; const attempts = reads();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    await waitFor(() => expect(releaseFeed).toBeDefined());
    expect(screen.getByText("Fixture decision owner refused feed")).toBeVisible();
    expect(screen.queryByTestId("decision-queue-header")).toBeNull();
    expect(screen.queryByTestId("decision-queue-nav")).toBeNull();
    expect(screen.queryByTestId("next-action-primary")).toBeNull();
    expect(view.client.getQueryData(NEXT_ACTION_FEED_QUERY_KEY)).toBe(before);
    expect(screen.getByTestId("drawer-owner-location")).toHaveTextContent("keep=fixture&decisionPosition=1");
    await act(async () => releaseFeed?.({ entries })); await ready();
    expect(screen.getByTestId("decision-queue-title")).toHaveTextContent("Owner fixture-second");
    expect(reads()).toBe(attempts + 1); expect(view.completed).not.toHaveBeenCalled(); noWrites();
  }, 15000);
  it("allows parent close after owner refusal without acquiring another feed", async () => {
    feedRefused = true; const view = mount(); await screen.findByText("Fixture decision owner refused feed", {}, { timeout: 12000 });
    const attempts = reads(); fireEvent.click(closeChrome());
    await waitFor(() => expect(screen.queryByTestId("plan-decision-drawer")).toBeNull());
    expect(view.close).toHaveBeenCalledOnce(); expect(view.completed).not.toHaveBeenCalled(); expect(reads()).toBe(attempts); noWrites();
  }, 15000);

  it("does not cancel or restart a background owner read while refusal remains visible", async () => {
    const view = mount(); await ready(); feedRefused = true;
    await act(async () => { await view.client.invalidateQueries({ queryKey: NEXT_ACTION_FEED_QUERY_KEY }); });
    await screen.findByText("Fixture decision owner refused feed", {}, { timeout: 12000 });
    feedRefused = false; deferFeed = true; const attempts = reads();
    let refresh: Promise<unknown> | undefined;
    act(() => { refresh = view.client.invalidateQueries({ queryKey: NEXT_ACTION_FEED_QUERY_KEY }); });
    await waitFor(() => expect(releaseFeed).toBeDefined());
    const retry = screen.getByRole("button", { name: "Try again" }); expect(retry).toBeDisabled();
    fireEvent.click(retry); fireEvent.click(retry);
    expect(reads()).toBe(attempts + 1); expect(screen.queryByTestId("next-action-primary")).toBeNull();
    expect(screen.queryByTestId("decision-queue-nav")).toBeNull();
    await act(async () => { releaseFeed?.({ entries }); await refresh; }); await ready();
    expect(reads()).toBe(attempts + 1); expect(view.completed).not.toHaveBeenCalled(); noWrites();
  }, 15000);
  it("reconciles retained explicit-retry refusal after a later successful background owner feed", async () => {
    feedRefused = true; const view = mount();
    await screen.findByText("Fixture decision owner refused feed", {}, { timeout: 12000 });
    const attempts = reads(); fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    await waitFor(() => expect(reads()).toBeGreaterThan(attempts));
    await waitFor(() => expect(screen.getByRole("button", { name: "Try again" })).not.toBeDisabled(), { timeout: 12000 });
    expect(screen.getByText("Fixture decision owner refused feed")).toBeVisible();
    feedRefused = false;
    await act(async () => { await view.client.invalidateQueries({ queryKey: NEXT_ACTION_FEED_QUERY_KEY }); });
    await ready(); expect(screen.getByTestId("decision-queue-title")).toHaveTextContent("Owner fixture-first");
    expect(screen.queryByText("Fixture decision owner refused feed")).toBeNull();
    expect(view.completed).not.toHaveBeenCalled(); noWrites();
  }, 20000);

});
