import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Route, Routes, useLocation } from "react-router-dom";
import { createTestQueryClient, renderWithProviders } from "../../../test-utils";
import type { IPlanService } from "../../../services/plan-service";
import type { IOperationsService } from "../../../services/operations-service";
import {
  resetOperationsStoreService,
  setOperationsStoreService,
  useOperationsStore,
} from "../../../stores/operations-store";
import type { OperationsView } from "../../../types/operations";
import type { PlanBoardData, PlanCardData, PlanCardGroupData } from "../types";
import {
  createPlanDataInitialState,
  resetPlanRequestState,
  resetPlanStoreService,
  setPlanStoreService,
  usePlanDataStore,
} from "../stores/plan-data-store";
import { PlanBoard } from "./PlanBoard";
import { defaultApiClient } from "../../../lib/api-client";

function LocationProbe() {
  const location = useLocation();
  return <span data-testid="location-probe">{location.pathname}{location.search}</span>;
}

function emptyOpsView(): OperationsView {
  return {
    lanes: [
      { lane: "investigate", active: 0, capacity: 6, queue: 0 },
      { lane: "execute", active: 0, capacity: 3, queue: 0 },
      { lane: "review", active: 0, capacity: 8, queue: 0 },
      { lane: "reconcile", active: 0, capacity: 2, queue: 0 },
    ],
    queue: { depth: 0, maxDepth: 50 },
    activities: [],
    recentlyFinished: [],
    generatedAt: "2026-07-02T00:00:00Z",
    windowSeconds: 10800,
  };
}

function stubOpsService(): IOperationsService {
  return {
    fetchOperations: vi.fn().mockResolvedValue(emptyOpsView()),
    bulkStop: vi.fn().mockResolvedValue({ outcomes: [], total: 0, stopped: 0, failed: 0 }),
  };
}

function itemCard(id: string, overrides?: Partial<PlanCardData>): PlanCardData {
  return {
    id: `backlog-item/fix/${id}`,
    cardType: "item",
    action: "run",
    itemKind: "fix",
    itemName: id,
    title: `${id} title`,
    status: "ready",
    priority: 3,
    wave: 0,
    milestone: "",
    effort: "",
    gate: null,
    outcome: "",
    finishedAt: "",
    executionId: "",
    unblocks: 0,
    ...overrides,
  };
}

function gateCard(id: string, kind: "decide" | "review" | "proposal", count: number): PlanCardData {
  return itemCard(id, {
    cardType: "gate",
    action: kind,
    gate: {
      id: `${kind}:backlog/fix/${id}`,
      kind,
      ownerType: "backlog",
      ownerKind: "fix",
      ownerName: id,
      ownerTitle: `${id} title`,
      count,
      blocks: [],
      decidableSince: "",
      suggested: "",
    },
  });
}

function group(id: string, cards: PlanCardData[], overrides?: Partial<PlanCardGroupData>): PlanCardGroupData {
  return { id, label: id, blockerKind: "none", gateId: "", blockerKeys: [], cards, ...overrides };
}

function makeBoard(overrides?: Partial<PlanBoardData>): PlanBoardData {
  return {
    now: {
      activeCount: 2,
      queueDepth: 1,
      maxQueueDepth: 10,
      lanes: [{ lane: "execute", active: 2, capacity: 3 }],
    },
    next: {
      groups: [
        group("gates", [gateCard("questions", "decide", 3)], { label: "Decisions & reviews" }),
        group("ready", [itemCard("runnable")], { label: "Ready to run" }),
      ],
      cardCount: 2,
    },
    later: {
      groups: [
        group("items:fix/runnable", [itemCard("blocked", { wave: 1, action: "workshop" })], {
          label: "after runnable title",
          blockerKind: "items",
          blockerKeys: ["fix/runnable"],
        }),
        group("deep-group", [itemCard("deep", { wave: 7 })], {
          label: "after something deep",
          blockerKind: "items",
        }),
      ],
      cardCount: 2,
    },
    done: {
      groups: [
        group("recent", [
          itemCard("shipped", {
            cardType: "outcome",
            action: "none",
            outcome: "ok",
            finishedAt: "2026-07-02T10:00:00Z",
          }),
        ], { label: "Recent outcomes" }),
      ],
      cardCount: 1,
    },
    meta: { generatedAt: "2026-07-02T12:00:00Z", windowSeconds: 86400, maxWave: 2, cycles: [], eta: null },
    ...overrides,
  };
}

function resetStore() {
  resetPlanRequestState();
  usePlanDataStore.setState({ ...createPlanDataInitialState() });
}

function stubService(board: PlanBoardData | Error): IPlanService {
  return {
    getBoard: board instanceof Error
      ? vi.fn().mockRejectedValue(board)
      : vi.fn().mockResolvedValue(board),
    listCanonicalPlans: vi.fn().mockResolvedValue([]),
    importPlan: vi.fn().mockRejectedValue(new Error("not implemented in PlanBoard tests")),
  };
}

let readService:IPlanService;let opsService:IOperationsService;let unexpected:string[];const views:{unmount:()=>void}[]=[];const clients:ReturnType<typeof createTestQueryClient>[]=[];
function mount(board:PlanBoardData=makeBoard(),path="/plan"){readService=stubService(board);setPlanStoreService(readService);const client=createTestQueryClient();clients.push(client);const v=renderWithProviders(<><Routes><Route path="/plan" element={<PlanBoard/>}/><Route path="*" element={<div data-testid="destination"/>}/></Routes><LocationProbe/></>,{queryClient:client,initialEntries:[path]});views.push(v);return v;}
function route(){return new URL(screen.getByTestId("location-probe").textContent??"/","https://fixture.invalid");}
function cycles(entries:string[]){return makeBoard({meta:{generatedAt:"2026-10-05T00:00:00Z",windowSeconds:86400,maxWave:2,cycles:entries,eta:null}});}
function eta(p50Label="~5 days",p80Label="~10 days",remainingItems=6){return makeBoard({meta:{generatedAt:"2026-10-05T00:00:00Z",windowSeconds:86400,maxWave:2,cycles:[],eta:{p50Hours:120,p80Hours:240,p50Label,p80Label,basis:"live",basisLabel:"27 samples",confidence:"high",remainingItems,laneCapacity:3}}});}
beforeEach(()=>{unexpected=[];resetStore();useOperationsStore.getState().reset();opsService=stubOpsService();setOperationsStoreService(opsService);vi.spyOn(defaultApiClient,"get").mockImplementation(async path=>{if(path==="/goals")return{items:[]};if(path==="/backlog/summary")return{pending_questions:{items:[]}};if(path==="/next-actions/feed")return{entries:[]};if(path==="/settings")return{settings:{}};unexpected.push(path);throw new Error("Unexpected plan read "+path);});vi.spyOn(defaultApiClient,"post").mockImplementation(async(path,body)=>{expect(path).toBe("/backlog/next-actions");expect(body).toEqual({items:["fix/questions","fix/runnable","fix/blocked","fix/deep","fix/shipped"]});return {results:[]};});for(const method of ["put","patch","delete"] as const)vi.spyOn(defaultApiClient,method).mockRejectedValue(new Error("Unexpected plan write"));});
afterEach(()=>{views.splice(0).forEach(v=>v.unmount());cleanup();getSpatialNav()?.dispose();clients.splice(0).forEach(c=>c.clear());resetPlanStoreService();resetOperationsStoreService();useOperationsStore.getState().reset();resetStore();expect(unexpected).toEqual([]);expect(defaultApiClient.post).toHaveBeenCalled();for(const [path,body] of vi.mocked(defaultApiClient.post).mock.calls){expect(path).toBe("/backlog/next-actions");expect(body).toEqual({items:["fix/questions","fix/runnable","fix/blocked","fix/deep","fix/shipped"]});}for(const method of ["put","patch","delete"] as const)expect(defaultApiClient[method]).not.toHaveBeenCalled();expect(opsService.bulkStop).not.toHaveBeenCalled();expect(readService.importPlan).not.toHaveBeenCalled();vi.restoreAllMocks();});
describe("actual plan board readonly owner navigation",()=>{
 it("discloses cycle chain once and opens the exact canonical backlog owner",async()=>{mount(cycles(["fix/runnable -> fix/blocked -> fix/runnable"]));fireEvent.click(await screen.findByTestId("plan-cycle-warning"));expect(screen.getByTestId("plan-cycle-popover")).toHaveTextContent("Remove one of these dependencies");expect(screen.getAllByRole("button",{name:"runnable title"})).toHaveLength(1);fireEvent.click(screen.getByTestId("plan-cycle-resolve"));expect(route().pathname).toBe("/backlog/fix/runnable");expect(screen.queryByTestId("plan-cycle-popover")).toBeNull();});
 it("opens a secondary cycle chip's exact backlog owner",async()=>{mount(cycles(["fix/runnable -> fix/blocked -> fix/runnable"]));fireEvent.click(await screen.findByTestId("plan-cycle-warning"));fireEvent.click(screen.getByRole("button",{name:"blocked title"}));expect(route().pathname).toBe("/backlog/fix/blocked");});
 it.each([{entity:"fix/runnable",node:"backlog-item/fix/runnable"},{entity:"scenario/fixture-scenario",node:"scenario/fixture-scenario"},{entity:"goal/fixture-goal",node:"goal/fixture-goal"},{entity:"opaque-fixture-owner",node:"opaque-fixture-owner"}])("inspects supplied cycle $entity through exact graph focus",async({entity,node})=>{mount(cycles([entity+" -> fix/blocked -> "+entity]));fireEvent.click(await screen.findByTestId("plan-cycle-warning"));fireEvent.click(screen.getByRole("button",{name:"View in graph"}));expect(route().pathname).toBe("/graph");expect(route().searchParams.get("mode")).toBe("focus");expect(route().searchParams.get("focus")).toBe(node);expect(route().searchParams.get("select")).toBe(node);expect(screen.queryByTestId("plan-cycle-popover")).toBeNull();});
 it("falls back to exact graph inspection when a cycle has no current backlog card",async()=>{mount(cycles(["scenario/fixture-scenario -> opaque-fixture-owner"]));fireEvent.click(await screen.findByTestId("plan-cycle-warning"));fireEvent.click(screen.getByTestId("plan-cycle-resolve"));expect(route().pathname).toBe("/graph");expect(route().searchParams.get("focus")).toBe("scenario/fixture-scenario");});
 it("discloses authoritative ETA context then navigates only on explicit stats intent",async()=>{mount(eta());fireEvent.click(await screen.findByTestId("plan-eta-strip"));expect(screen.getByTestId("plan-eta-popover")).toHaveTextContent("27 samples");expect(route().pathname).toBe("/plan");fireEvent.click(screen.getByTestId("plan-eta-stats-link"));expect(route().pathname).toBe("/stats");expect(route().search).toBe("");expect(screen.queryByTestId("plan-eta-popover")).toBeNull();});
 it.each([{left:"~5 days",right:"~5 days",text:"~5 days"},{left:"~5 days",right:"~2 weeks",text:"~5 days – ~2 weeks"}])("preserves authoritative ETA labels $left to $right",async({left,right,text})=>{mount(eta(left,right));expect(await screen.findByTestId("plan-eta-label")).toHaveTextContent(text);});
 it("does not manufacture an ETA when the owner reports no remaining items",async()=>{mount(eta("~5 days","~10 days",0));await screen.findByTestId("plan-board");expect(screen.queryByTestId("plan-eta-strip")).toBeNull();});
 it("changes the Done time window through the owner read without offering effects",async()=>{mount();await screen.findByTestId("plan-board");fireEvent.click(screen.getByTestId("plan-done-window-6h"));await waitFor(()=>expect(usePlanDataStore.getState().windowSeconds).toBe(21600));await waitFor(()=>expect(readService.getBoard).toHaveBeenCalledTimes(2));});
});
