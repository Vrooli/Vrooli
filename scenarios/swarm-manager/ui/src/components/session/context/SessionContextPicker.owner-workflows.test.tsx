// Real local context picker and canonical cards/stores/services. Only catalog reads
// cross a controlled network seam. Never Attach, start/create a session, or launch work.
import { useState } from "react";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import type { QueryClient } from "@tanstack/react-query";
import { createTestQueryClient, renderWithProviders } from "../../../test-utils";
import { defaultApiClient } from "../../../lib/api-client";
import { DEFAULT_SETTINGS } from "../../../services/settings-service";
import { useAgentActivitiesStore, useAgentSessionStore, useBacklogStore, useCaptureStore, useExecutionStore, useScenariosStore } from "../../../stores";
import { resetAgentSessionStoreService } from "../../../stores/agent-session-store";
import type { AgentSessionContextType, AgentSessionKind } from "../../../types";
import type { SessionContextOption } from "./session-context-refs";
import { SessionContextPicker } from "./SessionContextPicker";
const views: ReturnType<typeof renderWithProviders>[] = [];
const clients: QueryClient[] = [];
let applied: ReturnType<typeof vi.fn>; let closed: ReturnType<typeof vi.fn>;
let goalRefused: boolean; let unexpected: string[];
const offeredSelection: SessionContextOption = {type:"scenario",ref:"fixture-alpha",title:"Alpha context",subtitle:"Unsent parent choice",nodeId:"scenario/fixture-alpha"};
const scenario = (name:string,title:string) => ({name,display_name:title,description:"Disposable context owner",status:"stopped",priority:2,tags:["fixture"],is_greenfield:false});
function Harness({kind="workflow_authoring",initialType="scenario",selected=[],initiallyOpen=true}:{kind?:AgentSessionKind;initialType?:AgentSessionContextType;selected?:SessionContextOption[];initiallyOpen?:boolean}) {
 const [open,setOpen]=useState(initiallyOpen);
 return <><button onClick={()=>setOpen(true)}>Choose context</button><SessionContextPicker isOpen={open} sessionKind={kind} initialType={initialType} selected={selected} onApply={applied} onClose={()=>{closed();setOpen(false);}}/></>;
}
function mount(props:Parameters<typeof Harness>[0]={}) { const client=createTestQueryClient();clients.push(client);const view=renderWithProviders(<Harness {...props}/>,{queryClient:client});views.push(view);return view; }
function list(){return screen.getByTestId("session-context-entity-list");}
function search(value:string){fireEvent.change(screen.getByTestId("session-context-search"),{target:{value}});}
function tab(type:string){fireEvent.click(screen.getByTestId(`session-context-tab-${type}`));}
async function ready(){await waitFor(()=>expect(useScenariosStore.getState().scenarios).toHaveLength(2));}
function reset(){useBacklogStore.getState().reset();useCaptureStore.getState().reset();useExecutionStore.getState().reset();useScenariosStore.getState().reset();useAgentSessionStore.getState().reset();useAgentActivitiesStore.setState({activities:[],isRefreshing:false});}
beforeEach(()=>{
 window.localStorage.clear();reset();resetAgentSessionStoreService();applied=vi.fn();closed=vi.fn();goalRefused=false;unexpected=[];
 vi.spyOn(defaultApiClient,"get").mockImplementation(async path=>{
  if(path==="/backlog?archived=all")return{items:[],blocking:{}};
  if(path==="/goals"){if(goalRefused)throw new Error("Fixture goals owner refused");return{items:[]};}
  if(path==="/captures")return{captures:[]};
  if(path==="/execution")return{items:[]};
  if(path==="/agent-activities?active=false")return{items:[]};
  if(path==="/scenarios")return{scenarios:[scenario("fixture-alpha","Alpha owner"),scenario("fixture-beta","Beta owner")]};
  if(path==="/agent-sessions?limit=100")return{sessions:[]};
  if(path==="/settings"){const{deleteConfirmation:_,...settings}=DEFAULT_SETTINGS;return{settings};}
  unexpected.push(path);throw new Error(`Unexpected context owner read ${path}`);
 });
 for(const method of ["post","put","patch","delete"] as const)vi.spyOn(defaultApiClient,method).mockRejectedValue(new Error(`Forbidden context ${method}`));
 vi.stubGlobal("fetch",vi.fn().mockRejectedValue(new Error("Forbidden direct context transport")));
});
afterEach(()=>{
 views.splice(0).forEach(v=>v.unmount());clients.splice(0).forEach(c=>c.clear());getSpatialNav()?.dispose();
 try{expect(unexpected).toEqual([]);for(const method of ["post","put","patch","delete"] as const)expect(defaultApiClient[method]).not.toHaveBeenCalled();expect(fetch).not.toHaveBeenCalled();expect(applied).not.toHaveBeenCalled();}
 finally{reset();resetAgentSessionStoreService();window.localStorage.clear();vi.restoreAllMocks();vi.unstubAllGlobals();}
});
describe("local context owner exploration and cancellation",()=>{
 it("does not acquire context catalogs when closed",()=>{mount({initiallyOpen:false});expect(screen.queryByTestId("session-context-picker")).toBeNull();expect(defaultApiClient.get).not.toHaveBeenCalled();});
 it("reads the actual owner catalogs on explicit opening and retains unsent parent context when cancelled",async()=>{mount({initiallyOpen:false,selected:[offeredSelection]});fireEvent.click(screen.getByRole("button",{name:"Choose context"}));await ready();expect(screen.getByTestId("session-context-selected-tray")).toHaveTextContent("Alpha context");fireEvent.click(screen.getByRole("button",{name:"Cancel"}));await waitFor(()=>expect(screen.queryByTestId("session-context-picker")).toBeNull());expect(closed).toHaveBeenCalledTimes(1);expect(offeredSelection.title).toBe("Alpha context");});
 it("uses workflow-authoring tabs without offering execution or activity contexts",async()=>{mount();await ready();expect(screen.getByTestId("session-context-tab-scenario")).toHaveAttribute("aria-selected","true");expect(screen.queryByTestId("session-context-tab-execution")).toBeNull();expect(screen.queryByTestId("session-context-tab-agent_activity")).toBeNull();expect(screen.queryByTestId("session-context-tab-capture")).toBeNull();});
 it("offers the operations briefing only for the operations kind and remains locally cancelable",async()=>{mount({kind:"swarm_operations",initialType:"operations_briefing"});await ready();expect(within(list()).getByText("Current operations briefing")).toBeVisible();expect(screen.queryByTestId("session-context-tab-scenario")).toBeNull();expect(screen.getByTestId("session-context-tab-execution")).toBeVisible();fireEvent.click(screen.getByRole("button",{name:"Cancel"}));expect(closed).toHaveBeenCalledTimes(1);});
 it("falls back from an unsupported initial type to the kind's startup brief",async()=>{mount({kind:"workflow_authoring",initialType:"execution"});await ready();expect(screen.getByTestId("session-context-tab-startup_brief")).toHaveAttribute("aria-selected","true");expect(within(list()).getByText("Startup brief")).toBeVisible();});
 it.each([["  ALPHA  ","Alpha owner"],["fixture-beta","Beta owner"]])("filters actual scenario context by title or exact owner ref (%s)",async(needle,title)=>{mount();await ready();search(needle);expect(within(list()).getByText(title)).toBeVisible();expect(within(list()).queryByText(title==="Alpha owner"?"Beta owner":"Alpha owner")).toBeNull();search("unrelated-no-owner");expect(list()).toHaveTextContent("No matching context.");search("");expect(within(list()).getByText("Alpha owner")).toBeVisible();expect(within(list()).getByText("Beta owner")).toBeVisible();});
 it("keeps a tentative scenario selection local and discards it on cancel",async()=>{mount();await ready();const boxes=within(list()).getAllByRole("checkbox");expect(boxes).toHaveLength(2);fireEvent.click(boxes[0]!);expect(boxes[0]!).toBeChecked();expect(screen.getByTestId("session-context-selected-tray")).toHaveTextContent("fixture-alpha");expect(screen.getByRole("button",{name:"Remove fixture-alpha"})).toBeVisible();fireEvent.click(screen.getByRole("button",{name:"Cancel"}));expect(closed).toHaveBeenCalledTimes(1);fireEvent.click(screen.getByRole("button",{name:"Choose context"}));await ready();expect(screen.queryByTestId("session-context-selected-tray")).toBeNull();});
 it("removes an offered context only from the local draft while preserving the parent's immutable selection",async()=>{const selected=[offeredSelection];mount({selected});await ready();fireEvent.click(screen.getByRole("button",{name:"Remove Alpha context"}));expect(screen.queryByTestId("session-context-selected-tray")).toBeNull();expect(selected).toEqual([offeredSelection]);fireEvent.click(screen.getByRole("button",{name:"Cancel"}));fireEvent.click(screen.getByRole("button",{name:"Choose context"}));await ready();expect(screen.getByTestId("session-context-selected-tray")).toHaveTextContent("Alpha context");});
 it("clears local search on reopening while retaining the offered selection",async()=>{mount({selected:[offeredSelection]});await ready();search("fixture-beta");expect(within(list()).queryByText("Alpha owner")).toBeNull();fireEvent.click(screen.getByRole("button",{name:"Cancel"}));fireEvent.click(screen.getByRole("button",{name:"Choose context"}));await ready();expect(screen.getByTestId("session-context-search")).toHaveValue("");expect(within(list()).getByText("Alpha owner")).toBeVisible();expect(screen.getByTestId("session-context-selected-tray")).toHaveTextContent("Alpha context");});
 it("preserves local selected context and cancelability when the goals catalog is refused",async()=>{goalRefused=true;mount({initialType:"goal",selected:[offeredSelection]});await waitFor(()=>expect(vi.mocked(defaultApiClient.get).mock.calls.some(([path])=>path==="/goals")).toBe(true));await ready();expect(screen.getByTestId("session-context-selected-tray")).toHaveTextContent("Alpha context");fireEvent.click(screen.getByRole("button",{name:"Cancel"}));expect(closed).toHaveBeenCalledTimes(1);});
 it("shows a preselected scenario cap, then frees capacity through explicit local removal",async()=>{const selected=[1,2,3].map(i=>({...offeredSelection,ref:`fixture-${i}`,title:`Offered ${i}`}));mount({selected});await ready();const capReason=screen.getAllByText("Scenarios allows 3 selections.").find(element=>element.tagName==="P");expect(capReason).toBeDefined();expect(capReason!).toBeVisible();fireEvent.click(screen.getByRole("button",{name:"Remove Offered 3"}));expect(screen.queryAllByText("Scenarios allows 3 selections.")).toHaveLength(0);expect(screen.getByText("2/12 context items selected.")).toBeVisible();expect(selected).toHaveLength(3);});
 it("keeps plan context inspection separate from existing entity browsing and cancels without applying",async()=>{const selected:SessionContextOption[]=[{type:"plan_eta",ref:"plan_eta/latest",title:"Offered ETA",summary:"Owner snapshot only"}];mount({kind:"swarm_operations",initialType:"plan_eta",selected});await ready();expect(list()).toHaveTextContent("No matching context.");expect(screen.getByTestId("session-context-selected-tray")).toHaveTextContent("Offered ETA");tab("startup_brief");expect(within(list()).getByText("Startup brief")).toBeVisible();fireEvent.click(screen.getByRole("button",{name:"Cancel"}));expect(closed).toHaveBeenCalledTimes(1);});
});
