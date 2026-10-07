import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { BacklogDetailsPage } from "./BacklogDetailsPage";
import { defaultApiClient } from "../lib/api-client";
import { useAgentActivitiesStore, useBacklogDetailUIStore, useBacklogStore } from "../stores";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import type { BacklogItem } from "../types";

// Only the network client is replaced: page, queries, mutations, service
// translation, global stores, dialogs and provider/router composition are real.
const item: BacklogItem = { kind: "idea", name: "reviewed-item", title: "Reviewed item", description: "Owner specification", status: "backlog", priority: 1, tags: [], suggestedSkills: [], created: "2026-01-28T00:00:00Z", updated: "2026-01-28T00:00:00Z" };
let current:BacklogItem;
let expectedWrites:{method:string;args:unknown[]}[];
const requirement={id:"REQ-ORIGINAL",title:"Original requirement",description:"Original requirement evidence",status:"pending",category:"functional",prd_ref:"PRD-OWNER"};
const siblingRequirement={id:"REQ-SIBLING",title:"Sibling requirement",description:"Retained sibling evidence",status:"complete",category:"compatibility",prd_ref:"PRD-SIBLING"};
const target={id:"OT-ORIGINAL",title:"Original target",criticality:"P1",status:"pending",notes:"Original notes",linked_requirement_ids:[requirement.id]};
const clients: QueryClient[] = [];
const mountedViews:{unmount:()=>void}[]=[];
const unexpectedReads: string[] = [];
function Location() { const location = useLocation(); return <output data-testid="workflow-location">{location.pathname + location.search}</output>; }
function mount(path = "/backlog/idea/reviewed-item") {
  const client = createTestQueryClient(); clients.push(client);
  const view = renderWithProviders(<Routes><Route path="/backlog/:kind/:name" element={<><BacklogDetailsPage /><Location /></>} /></Routes>, { queryClient: client, initialEntries: [path] });
  mountedViews.push(view);return { ...view, client };
}
function reads() {
  return vi.spyOn(defaultApiClient, "get").mockImplementation(async (path) => {
    if (path === "/settings") return { settings: {} };
    if(path === "/execution/strategies")return{items:[]};
    if (path === "/goals") return { items: [] };
    if (path === "/backlog/idea/reviewed-item/plan-render") return { markdown: "# Owner plan", planRef: null };
    if (path === "/backlog/idea/reviewed-item") return { item:current };
    if (path === "/backlog/idea/reviewed-item/files") return { files: [] };
    if (path === "/backlog/idea/reviewed-item/next-action") return { action: { id: "accept_plan", compactLabel: "Accept plan", expandedLabel: "Accept plan", enabled: true, reason: "Owner acceptance required", blockers: [], target: "plan_accept", effect: "state_change" } };
    if (path === "/backlog/idea/reviewed-item/archive/targets") return { targets:[target],requirements:[{id:"module-owner",name:"Original module",requirements:[requirement,siblingRequirement],children:[]}],has_archive:true };
    if (path === "/backlog/idea/reviewed-item/review") return { rounds: [] };
    if (path === "/execution?backlog_kind=idea&backlog_name=reviewed-item") return { items: [] };
    if (path.startsWith("/backlog?")) return { items: [], blocking: {} };
    if (path.startsWith("/agent-activities?")) return { items: [] };
    if (path === "/proposal-sessions?target_type=backlog_item&target_ref=idea%2Freviewed-item") return { sessions: [] };
    unexpectedReads.push(path);
    throw new Error(`Unexpected fixture read ${path}`);
  });
}
beforeEach(() => {
  unexpectedReads.length = 0;expectedWrites=[];current={...item,note:"Original owner note"};
  useBacklogStore.getState().setItems([]);
  useAgentActivitiesStore.setState({ activities: [], isRefreshing: false });
  useBacklogDetailUIStore.getState().reset();
  reads();
  vi.spyOn(defaultApiClient,"put").mockRejectedValue(new Error("Unexpected fixture put"));
  vi.spyOn(defaultApiClient,"post").mockRejectedValue(new Error("Unexpected fixture post"));
  vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new Error("Unexpected fixture patch"));
  vi.spyOn(defaultApiClient, "delete").mockRejectedValue(new Error("Unexpected fixture delete"));
});
afterEach(() => { mountedViews.splice(0).forEach(view=>view.unmount());cleanup();getSpatialNav()?.dispose(); clients.splice(0).forEach(client => client.clear()); useBacklogStore.getState().setItems([]); useBacklogDetailUIStore.getState().reset(); expect(unexpectedReads).toEqual([]);for(const method of ["post","put","patch","delete"] as const)expect(vi.mocked(defaultApiClient[method]).mock.calls).toEqual(expectedWrites.filter(w=>w.method===method).map(w=>w.args));vi.restoreAllMocks(); });


// Actual page, parent dialog composition, forms, stores and services are used.
// Injected API client is the upstream boundary; no native/real server effects.
async function loaded(){mount();await screen.findByRole("button",{name:"Original owner note"});await waitFor(()=>expect(defaultApiClient.get).toHaveBeenCalledWith("/backlog/idea/reviewed-item/archive/targets"));}
async function open(action:()=>void){await loaded();act(action);}
function change(label:string,value:string){fireEvent.change(screen.getByLabelText(label),{target:{value}});}
function state(){return useBacklogDetailUIStore.getState();}
describe("actual backlog parent dialog composition",()=>{
 it("cancels item edit without offering a write and reopens original owner data",async()=>{await open(()=>state().openEdit());change("Title","Abandoned operator title");fireEvent.click(screen.getByRole("button",{name:"Cancel"}));expect(state().showEdit).toBe(false);expect(defaultApiClient.patch).not.toHaveBeenCalled();act(()=>state().openEdit());expect(screen.getByLabelText("Title")).toHaveValue(item.title);expect(screen.getByLabelText("Name")).toHaveValue(item.name);});
 it("passes only allowed item editor fields to the exact adapter and closes on canonical success",async()=>{expectedWrites=[{method:"patch",args:["/backlog/idea/reviewed-item",{title:"Submitted item title",description:item.description,status:item.status,priority:item.priority,tags:[]}]}];vi.mocked(defaultApiClient.patch).mockImplementation(async()=>{current={...current,title:"Canonical item title"};return{item:current};});await open(()=>state().openEdit());change("Title"," Submitted item title ");fireEvent.click(screen.getByRole("button",{name:"Save Changes"}));await waitFor(()=>expect(state().showEdit).toBe(false));expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/backlog/idea/reviewed-item",{title:"Submitted item title",description:item.description,status:item.status,priority:item.priority,tags:[]}]]);await waitFor(()=>expect(useBacklogStore.getState().items.find(i=>i.name===item.name)?.title).toBe("Canonical item title"));expect(defaultApiClient.post).not.toHaveBeenCalled();});
 it("retains item draft during upstream pending and refusal parent rerenders, then resets refusal on Cancel",async()=>{expectedWrites=[{method:"patch",args:["/backlog/idea/reviewed-item",{title:"Retained item draft",description:item.description,status:item.status,priority:item.priority,tags:[]}]}];let refuse!:(e:Error)=>void;vi.mocked(defaultApiClient.patch).mockImplementation(()=>new Promise((_,r)=>{refuse=r;}));await open(()=>state().openEdit());change("Title","Retained item draft");fireEvent.click(screen.getByRole("button",{name:"Save Changes"}));await waitFor(()=>expect(defaultApiClient.patch).toHaveBeenCalledOnce());expect(screen.getByLabelText("Title")).toHaveValue("Retained item draft");expect(screen.getByRole("button",{name:"Saving..."})).toBeDisabled();await act(async()=>refuse(new Error("Exact owner refused edit")));expect(await screen.findByText("Exact owner refused edit")).toBeVisible();expect(screen.getByLabelText("Title")).toHaveValue("Retained item draft");expect(state().showEdit).toBe(true);fireEvent.click(screen.getByRole("button",{name:"Cancel"}));act(()=>state().openEdit());expect(screen.queryByText("Exact owner refused edit")).toBeNull();expect(screen.getByLabelText("Title")).toHaveValue(item.title);expect(defaultApiClient.patch).toHaveBeenCalledOnce();});
 it.each(["module","target","requirement"] as const)("closes %s editor without mutation and clears its local dialog identity",async owner=>{await open(()=>{if(owner==="module")state().openModuleEdit("module-owner");else if(owner==="target")state().openTargetEdit(target);else state().openReqEdit({groupId:"module-owner",req:requirement});});fireEvent.click(screen.getByRole("button",{name:"Cancel"}));const dialog=owner==="module"?state().moduleDialog:owner==="target"?state().targetDialog:state().reqDialog;expect(dialog).toMatchObject({isOpen:false,editing:null});expect(defaultApiClient.post).not.toHaveBeenCalled();expect(defaultApiClient.put).not.toHaveBeenCalled();expect(defaultApiClient.patch).not.toHaveBeenCalled();});
 it("creates a module through exact parent kind/name and closes only after adapter success",async()=>{expectedWrites=[{method:"post",args:["/backlog/idea/reviewed-item/archive/requirements",{id:"new-module",title:"New owner module",description:"New description"}]}];vi.mocked(defaultApiClient.post).mockResolvedValue({});await open(()=>state().openModuleCreate());change("ID","new-module");change("Title"," New owner module ");change("Description"," New description ");fireEvent.click(screen.getByRole("button",{name:"Create Module"}));await waitFor(()=>expect(state().moduleDialog.isOpen).toBe(false));expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/backlog/idea/reviewed-item/archive/requirements",{id:"new-module",title:"New owner module",description:"New description"}]]);expect(defaultApiClient.put).not.toHaveBeenCalled();});
 it("updates only selected module metadata while retaining draft during exact refusal",async()=>{expectedWrites=[{method:"put",args:["/backlog/idea/reviewed-item/archive/requirements/module-owner/meta",{title:"Retained module draft",description:"Reviewed description"}]}];vi.mocked(defaultApiClient.put).mockRejectedValue(new Error("Exact module owner refused"));await open(()=>state().openModuleEdit("module-owner"));expect(screen.getByLabelText("Title")).toHaveValue("Original module");change("Title","Retained module draft");change("Description","Reviewed description");fireEvent.click(screen.getByRole("button",{name:"Save Changes"}));expect(await screen.findByText("Exact module owner refused")).toBeVisible();expect(screen.getByLabelText("Title")).toHaveValue("Retained module draft");expect(state().moduleDialog.isOpen).toBe(true);expect(vi.mocked(defaultApiClient.put).mock.calls).toEqual([["/backlog/idea/reviewed-item/archive/requirements/module-owner/meta",{title:"Retained module draft",description:"Reviewed description"}]]);expect(defaultApiClient.post).not.toHaveBeenCalled();});
 it("creates an exact target through parent adapter and closes after success",async()=>{expectedWrites=[{method:"post",args:["/backlog/idea/reviewed-item/archive/targets",{id:"OT-NEW",title:"New owner target",criticality:"P0",status:"pending",notes:"",linked_requirement_ids:[]}]}];vi.mocked(defaultApiClient.post).mockResolvedValue({});await open(()=>state().openTargetCreate());change("ID","OT-NEW");change("Title"," New owner target ");fireEvent.click(screen.getByRole("button",{name:"Create Target"}));await waitFor(()=>expect(state().targetDialog.isOpen).toBe(false));expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/backlog/idea/reviewed-item/archive/targets",{id:"OT-NEW",title:"New owner target",criticality:"P0",status:"pending",notes:"",linked_requirement_ids:[]}]]);expect(defaultApiClient.put).not.toHaveBeenCalled();});
 it("updates only selected target with exact refusal retained and no alternate adapter",async()=>{expectedWrites=[{method:"put",args:["/backlog/idea/reviewed-item/archive/targets/OT-ORIGINAL",{...target,title:"Retained target draft"}]}];vi.mocked(defaultApiClient.put).mockRejectedValue(new Error("Exact target owner refused"));await open(()=>state().openTargetEdit(target));change("Title","Retained target draft");fireEvent.click(screen.getByRole("button",{name:"Save Changes"}));expect(await screen.findByText("Exact target owner refused")).toBeVisible();expect(screen.getByLabelText("Title")).toHaveValue("Retained target draft");expect(vi.mocked(defaultApiClient.put).mock.calls).toEqual([["/backlog/idea/reviewed-item/archive/targets/OT-ORIGINAL",{...target,title:"Retained target draft"}]]);expect(defaultApiClient.post).not.toHaveBeenCalled();expect(state().targetDialog.isOpen).toBe(true);});
 it("adds a requirement to the selected original module while preserving sibling records",async()=>{expectedWrites=[{method:"put",args:["/backlog/idea/reviewed-item/archive/requirements/module-owner",{requirements:[requirement,siblingRequirement,{id:"REQ-NEW",title:"New owner requirement",description:"",status:"pending",category:"",prd_ref:"",notes:undefined}]}]}];vi.mocked(defaultApiClient.put).mockResolvedValue({});await open(()=>state().openReqCreate("module-owner"));change("ID","REQ-NEW");change("Title"," New owner requirement ");fireEvent.click(screen.getByRole("button",{name:"Create Requirement"}));await waitFor(()=>expect(state().reqDialog.isOpen).toBe(false));expect(vi.mocked(defaultApiClient.put).mock.calls).toEqual([["/backlog/idea/reviewed-item/archive/requirements/module-owner",{requirements:[requirement,siblingRequirement,{id:"REQ-NEW",title:"New owner requirement",description:"",status:"pending",category:"",prd_ref:"",notes:undefined}]}]]);expect(defaultApiClient.post).not.toHaveBeenCalled();});
 it("replaces only the selected requirement and closes after canonical adapter success",async()=>{expectedWrites=[{method:"put",args:["/backlog/idea/reviewed-item/archive/requirements/module-owner",{requirements:[{...requirement,title:"Edited requirement",notes:undefined},siblingRequirement]}]}];vi.mocked(defaultApiClient.put).mockResolvedValue({});await open(()=>state().openReqEdit({groupId:"module-owner",req:requirement}));change("Title","Edited requirement");fireEvent.click(screen.getByRole("button",{name:"Save Changes"}));await waitFor(()=>expect(state().reqDialog.isOpen).toBe(false));expect(vi.mocked(defaultApiClient.put).mock.calls).toEqual([["/backlog/idea/reviewed-item/archive/requirements/module-owner",{requirements:[{...requirement,title:"Edited requirement",notes:undefined},siblingRequirement]}]]);expect(defaultApiClient.post).not.toHaveBeenCalled();});
 it("refuses a missing module's requirement with zero API effects",async()=>{await open(()=>state().openReqCreate("absent-module"));change("ID","REQ-UNOWNED");change("Title","Unowned requirement");fireEvent.click(screen.getByRole("button",{name:"Create Requirement"}));expect(state().reqDialog.isOpen).toBe(true);expect(defaultApiClient.put).not.toHaveBeenCalled();expect(defaultApiClient.post).not.toHaveBeenCalled();expect(defaultApiClient.patch).not.toHaveBeenCalled();});
});
