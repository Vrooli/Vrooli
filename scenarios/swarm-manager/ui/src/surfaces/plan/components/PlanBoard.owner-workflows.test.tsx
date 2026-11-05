// Draft-only whole-board owner workflows. Real stores, services, routing and widgets;
// only upstream network methods are controlled. No queue/dispatch/attach/stop is offered.
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { createTestQueryClient, renderWithProviders } from "../../../test-utils";
import { defaultApiClient } from "../../../lib/api-client";
import { resetOperationsStoreService, useOperationsStore } from "../../../stores/operations-store";
import { useBacklogStore, backlogStoreInitialState } from "../../../stores/backlog-store";
import { useAgentActivitiesStore } from "../../../stores";
import { useSnoozeStore } from "../../../stores/snooze-store";
import { createPlanDataInitialState, resetPlanRequestState, resetPlanStoreService, usePlanDataStore } from "../stores/plan-data-store";
import type { PlanBoardData, PlanCardData } from "../types";
import { PlanBoard } from "./PlanBoard";

const now = "2026-10-06T00:00:00Z";
function emptyBoard(): PlanBoardData {
  return {
    now: {activeCount: 0, queueDepth: 0, maxQueueDepth: 50, lanes: []},
    next: {groups: [], cardCount: 0}, later: {groups: [], cardCount: 0}, done: {groups: [], cardCount: 0},
    meta: {generatedAt: now, windowSeconds: 86400, maxWave: 0, cycles: [], eta: null},
  };
}
function card(name: string): PlanCardData {
  return {id: `backlog-item/fix/${name}`, cardType: "item", action: "run", itemKind: "fix", itemName: name,
    title: `${name} title`, status: "ready", priority: 2, wave: 0, milestone: "", effort: "", gate: null,
    outcome: "", finishedAt: "", executionId: "", unblocks: 0};
}
function activity(ownerType: string, ownerName: string, ownerKind = "") {
  return {activity_id: `fixture-${ownerType}`, owner_type: ownerType, owner_kind: ownerKind,
    owner_name: ownerName, owner_title: "Fixture running owner", purpose: "Synthetic read-only observation",
    lane: "execute", status: "running", requested_at: now, started_at: now};
}
function Location() { const l = useLocation(); return <output data-testid="owner-location">{l.pathname}{l.search}</output>; }
function deferred<T>() { let resolve!: (value: T) => void; const promise = new Promise<T>(r => {resolve = r;}); return {promise, resolve}; }
let board: PlanBoardData;
let boardReply: () => Promise<unknown>;
let activities: ReturnType<typeof activity>[];
let feed: Array<Record<string, unknown>>;
let expectedRefs: string[];
let projectedRunRefs: string[];
let unexpected: string[];
let catalogOffers: Array<{method:string;path:string;body:unknown}>;
const views: ReturnType<typeof renderWithProviders>[] = [];
const clients: QueryClient[] = [];
const scrollDescriptor = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollIntoView");
function route() { return new URL(screen.getByTestId("owner-location").textContent!, "http://fixture.invalid"); }
function mount(path = "/plan?retained=fixture") {
  const client = createTestQueryClient(); clients.push(client);
  const view = renderWithProviders(<><Routes><Route path="/plan" element={<PlanBoard/>}/><Route path="*" element={<div data-testid="owner-destination"/>}/></Routes><Location/></>, {queryClient: client, initialEntries: [path]});
  views.push(view); return {client, view};
}
beforeEach(() => {
  resetPlanRequestState(); resetPlanStoreService(); usePlanDataStore.setState(createPlanDataInitialState());
  resetOperationsStoreService(); useOperationsStore.getState().reset();
  useBacklogStore.setState({...backlogStoreInitialState, blockingMap: {}});
  useSnoozeStore.setState({entries: new Map()}); useAgentActivitiesStore.setState({activities:[]}); window.localStorage.clear();
  board = emptyBoard(); boardReply = async () => board; activities = []; feed = [];
  expectedRefs = []; projectedRunRefs = []; unexpected = []; catalogOffers = [];
  Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {configurable: true, value: vi.fn()});
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path.split("?")[0] === "/plan") return boardReply();
    if (path.split("?")[0] === "/operations") return {lanes: [{lane:"execute",active:activities.length,capacity:3,queue:0}],queue:{depth:0,max_depth:50},activities, recently_finished: [], generated_at: now,window_seconds:10800};
    if (path === "/next-actions/feed") return {entries: feed};
    if (path === "/goals") return {items: []};
    if (path === "/backlog/summary") return {pending_questions: {items: []}};
    if (path === "/settings") return {settings: {}};
    if (path.split("?")[0] === "/proposal-sessions") return {sessions: []};
    unexpected.push(`GET ${path}`); throw new Error(`Unexpected owner read ${path}`);
  });
  vi.spyOn(defaultApiClient, "post").mockImplementation(async (path, body) => {
    // This POST is the existing read-only action projection, not an admission.
    if (path === "/backlog/next-actions") {
      expect(body).toEqual({items: expectedRefs});
      return {results: projectedRunRefs.map(item => ({item, action: {id:"run", compact_label:"Run", expanded_label:"Run fixture",enabled:true,blockers:[]}}))};
    }
    unexpected.push(`POST ${path}`); throw new Error(`Unexpected owner write ${path}`);
  });
  vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new Error("Unexpected owner patch"));
  vi.spyOn(defaultApiClient, "put").mockRejectedValue(new Error("Unexpected owner put"));
  vi.spyOn(defaultApiClient, "delete").mockRejectedValue(new Error("Unexpected owner delete"));
  vi.stubGlobal("fetch", vi.fn(async (input:RequestInfo|URL, init?:RequestInit) => {
    const request = new Request(input, init); const url = new URL(request.url);
    const body = request.body ? JSON.parse(await request.text()) : undefined;
    const offer = {method:request.method,path:url.pathname,body}; catalogOffers.push(offer);
    expect(offer).toEqual({method:"POST",path:"/vrooli.swarm_manager.v1.api.TransitionService/ListTransitions",body:{}});
    expect(url.search).toBe("");
    return new Response(JSON.stringify({transitions:[]}), {headers:{"content-type":"application/json"}});
  }));
});
afterEach(() => {
  views.splice(0).forEach(v => v.unmount()); clients.splice(0).forEach(c => c.clear()); getSpatialNav()?.dispose();
  try {
    expect(unexpected).toEqual([]);
    expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.put).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
    for (const offer of catalogOffers) expect(offer).toEqual({method:"POST",path:"/vrooli.swarm_manager.v1.api.TransitionService/ListTransitions",body:{}});
    for (const [path, body] of vi.mocked(defaultApiClient.post).mock.calls) {
      expect({path, body}).toEqual({path:"/backlog/next-actions",body:{items:expectedRefs}});
    }
  } finally {
    resetPlanRequestState(); resetPlanStoreService(); usePlanDataStore.setState(createPlanDataInitialState());
    resetOperationsStoreService(); useOperationsStore.getState().reset();
    useBacklogStore.setState({...backlogStoreInitialState, blockingMap: {}}); useSnoozeStore.setState({entries:new Map()}); useAgentActivitiesStore.setState({activities:[]});
    window.localStorage.clear(); vi.restoreAllMocks(); vi.unstubAllGlobals();
    if (scrollDescriptor) Object.defineProperty(HTMLElement.prototype, "scrollIntoView", scrollDescriptor); else Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
  }
});

describe("PlanBoard whole-owner navigation and cancellation", () => {
  it.each([
    ["backlog", "fixture-backlog", "fix", "backlog-item/fix/fixture-backlog"],
    ["milestone", "fixture-milestone", "", "milestone/fixture-milestone"],
    ["scenario", "fixture-scenario", "", "scenario/fixture-scenario"],
    ["capture", "fixture-capture", "", "capture/fixture-capture"],
  ])("reconciles a link to an actually returned running %s owner without a false miss", async (type, name, kind, node) => {
    activities = [activity(type, name, kind)]; const reply = deferred<unknown>(); boardReply = () => reply.promise;
    mount(`/plan?select=${encodeURIComponent(node)}&focus=old&retained=fixture`);
    await waitFor(() => expect(useOperationsStore.getState().view?.activities[0]?.ownerName).toBe(name));
    expect(route().searchParams.get("select")).toBe(node);
    await act(async () => reply.resolve(board)); await screen.findByTestId("plan-board");
    await waitFor(() => expect(route().searchParams.has("select")).toBe(false));
    expect(route().searchParams.has("focus")).toBe(false); expect(route().searchParams.get("retained")).toBe("fixture");
    expect(screen.queryByTestId("plan-select-miss")).toBeNull();
    expect(within(screen.getByTestId("plan-column-now")).getByText("Fixture running owner")).toBeVisible();
  });
  it.each([
    ["backlog", "", "backlog-item/fix/fixture-unknown"],
    ["unsupported", "fix", "unsupported/fixture-unknown"],
  ])("does not guess a selectable identity for an incomplete %s activity", async (type, kind, node) => {
    activities = [activity(type, "fixture-unknown", kind)]; const reply = deferred<unknown>(); boardReply = () => reply.promise;
    mount(`/plan?select=${encodeURIComponent(node)}&retained=fixture`);
    await waitFor(() => expect(useOperationsStore.getState().view?.activities).toHaveLength(1));
    await act(async () => reply.resolve(board));
    expect(await screen.findByTestId("plan-select-miss")).toHaveTextContent("fixture-unknown");
    await waitFor(() => expect(route().searchParams.has("select")).toBe(false));
    expect(route().searchParams.get("retained")).toBe("fixture");
  });
  it("reconciles an initially missing running capture after a later authoritative operations refresh", async () => {
    mount("/plan?select=capture%2Ffixture-late&retained=fixture"); await screen.findByTestId("plan-select-miss");
    activities = [activity("capture", "fixture-late")];
    fireEvent.click(screen.getByTestId("plan-now-refresh"));
    await waitFor(() => expect(useOperationsStore.getState().view?.activities[0]?.ownerName).toBe("fixture-late"));
    await waitFor(() => expect(screen.queryByTestId("plan-select-miss")).toBeNull());
    expect(route().searchParams.get("retained")).toBe("fixture");
  });
  it("closes dependency diagnostics with Escape and reopens the same owner snapshot without attaching", async () => {
    board.meta.cycles = ["scenario/fixture-a -> scenario/fixture-b -> scenario/fixture-a"];
    mount(); const trigger = await screen.findByTestId("plan-cycle-warning"); fireEvent.click(trigger);
    expect(screen.getByTestId("plan-cycle-popover")).toHaveTextContent("Loops back to scenario/fixture-a");
    fireEvent.keyDown(document, {key:"Escape"}); await waitFor(() => expect(screen.queryByTestId("plan-cycle-popover")).toBeNull());
    expect(trigger).toHaveAttribute("aria-expanded", "false"); fireEvent.click(trigger);
    expect(screen.getByTestId("plan-cycle-popover")).toHaveTextContent("scenario/fixture-b");
    expect(route().pathname).toBe("/plan"); expect(route().searchParams.get("retained")).toBe("fixture");
  });
  it("dismisses ETA by clicking outside and retains the band when reopened without navigating or attaching", async () => {
    board.meta.eta = {p50Hours:1,p80Hours:2,p50Label:"~1 hours",p80Label:"~2 hours",basis:"live",basisLabel:"27 samples",confidence:"medium",laneCapacity:3,remainingItems:4};
    mount(); const trigger = await screen.findByTestId("plan-eta-strip"); fireEvent.click(trigger);
    expect(screen.getByTestId("plan-eta-popover")).toHaveTextContent("27 samples");
    fireEvent.mouseDown(screen.getByTestId("plan-column-next"));
    await waitFor(() => expect(screen.queryByTestId("plan-eta-popover")).toBeNull());
    fireEvent.click(trigger); expect(screen.getByTestId("plan-eta-popover")).toHaveTextContent("27 samples");
    expect(route().pathname).toBe("/plan"); expect(route().searchParams.get("retained")).toBe("fixture");
  });
  it("closes and reopens the board filter panel without clearing its canonical URL filters", async () => {
    mount("/plan?q=fixture&lane=execute&retained=fixture"); await screen.findByTestId("plan-board");
    act(() => usePlanDataStore.getState().setFilterDrawerOpen(true));
    const panel = await screen.findByTestId("plan-filter-drawer"); expect(within(panel).getByTestId("plan-filter-search")).toHaveValue("fixture");
    fireEvent.click(within(panel).getByRole("button", {name:"Close panel"}));
    await waitFor(() => expect(usePlanDataStore.getState().filterDrawerOpen).toBe(false));
    expect(screen.queryByTestId("plan-filter-drawer")).toBeNull();
    expect(route().searchParams.get("q")).toBe("fixture"); expect(route().searchParams.get("lane")).toBe("execute");
    act(() => usePlanDataStore.getState().setFilterDrawerOpen(true));
    expect(within(await screen.findByTestId("plan-filter-drawer")).getByTestId("plan-filter-lane")).toHaveValue("execute");
  });
  it("opens all decisions from the actual Next header and closes only its owned URL state", async () => {
    feed = [{entity_kind:"backlog_item",entity_ref:"fix/fixture-question",entity_title:"Fixture question",action:{id:"decide",compact_label:"Decide",expanded_label:"Answer fixture",enabled:true,effect:"state_change"},tier:1}];
    mount("/plan?decisionScope=fix%2Fold&decisionQuestion=old&q=fixture&retained=fixture");
    fireEvent.click(await screen.findByTestId("plan-next-answer-all"));
    const drawer = await screen.findByTestId("plan-decision-drawer");
    await waitFor(() => expect(route().searchParams.get("drawer")).toBe("decisions"));
    expect(route().searchParams.has("decisionScope")).toBe(false); expect(route().searchParams.has("decisionQuestion")).toBe(false);
    const closeControls = within(drawer).getAllByRole("button", {name:"Close drawer"});
    expect(closeControls.length).toBeGreaterThan(0); fireEvent.click(closeControls[0]!);
    await waitFor(() => expect(screen.queryByTestId("plan-decision-drawer")).toBeNull());
    await waitFor(() => expect(route().searchParams.has("drawer")).toBe(false));
    expect(route().searchParams.get("q")).toBe("fixture"); expect(route().searchParams.get("retained")).toBe("fixture");
  });
  it("uses only owner-projected ready cards for bulk confirmation and cancels before any queue offer", async () => {
    const cards = ["fixture-1","fixture-2","fixture-3","fixture-4","fixture-unqualified"].map(card);
    board.next = {groups:[{id:"ready",label:"Ready",blockerKind:"none",gateId:"",blockerKeys:[],cards}],cardCount:5};
    expectedRefs = cards.map(c => `fix/${c.itemName}`); projectedRunRefs = expectedRefs.slice(0,4);
    mount(); const trigger = await screen.findByTestId("plan-next-run-all");
    await waitFor(() => expect(trigger).toHaveAttribute("title", "Run all 4 ready items"));
    fireEvent.click(trigger); expect(await screen.findByText("Run all ready (4 items)")).toBeVisible();
    expect(screen.queryByTestId("run-sheet")).toBeNull();
    fireEvent.click(screen.getByRole("button", {name:"Cancel"}));
    await waitFor(() => expect(screen.queryByText("Run all ready (4 items)")).toBeNull());
    expect(screen.queryByTestId("run-sheet")).toBeNull(); expect(route().pathname).toBe("/plan");
  });
});
