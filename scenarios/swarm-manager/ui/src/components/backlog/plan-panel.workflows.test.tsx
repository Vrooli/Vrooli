// Actual plan panel, services, cache and HTTP client; all owner replies are disposable.
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { QueryClient } from "@tanstack/react-query";
import { createTestQueryClient, renderWithProviders } from "../../test-utils";
import { PlanPanel } from "./plan-panel";
import type { BacklogNextAction } from "../../services/backlog/types";
let calls:{method:string;path:string;body:unknown}[];let owner:Record<string,unknown>;let replies:Map<string,()=>Response|Promise<Response>>;let unexpected:string[];
const clients:QueryClient[]=[];const stamp="2026-10-05T00:00:00Z";
const acceptance={actor:"fixture-operator",accepted_at:stamp,plan_content_hash:"fixture-plan-hash",subject_version:"fixture-subject"};
const markdown="# Owner plan\n\nExact fixture content.\n\n## Scope\n\nOwned fixture scope.\n\n# Checks\n\nExplicit fixture acceptance.";
let clipboardDescriptor:PropertyDescriptor|undefined;let scrollDescriptor:PropertyDescriptor|undefined;let copy:ReturnType<typeof vi.fn>;let scroll:ReturnType<typeof vi.fn>;
function json(value:unknown,status=200){return new Response(JSON.stringify(value),{status,headers:{"content-type":"application/json"}});}
function rendered(overrides:Record<string,unknown>={}){return json({path:"plan-manager:fixture-plan",markdown,quality_status:"pass",plan_ref:{provider:"plan-manager",planId:"fixture-plan",slug:"fixture-plan",role:"execution_spec"},...overrides});}
function mount(nextAction?:BacklogNextAction){const client=createTestQueryClient();clients.push(client);const ui=renderWithProviders(<PlanPanel backlogKind="fix" backlogName="fixture-owner" nextAction={nextAction}/>,{queryClient:client});return{client,...ui};}
function mutations(){return calls.filter(c=>c.method!=="GET");}
beforeEach(()=>{
 calls=[];unexpected=[];owner={kind:"fix",name:"fixture-owner",title:"Fixture owner",status:"ready",priority:2,tags:[],suggested_skills:[],created:stamp,updated:stamp};
 replies=new Map([["GET /backlog/fix/fixture-owner/plan-render",()=>rendered()],["GET /backlog/fix/fixture-owner",()=>json({item:owner})],["GET /embedded/plan-manager/external-url",()=>json({url:"https://fixture.invalid/plan-manager/"})]]);
 vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{const request=new Request(input,init);const path=new URL(request.url).pathname.replace(/^\/api\/v1/,"");const key=request.method+" "+path;calls.push({method:request.method,path,body:request.body?JSON.parse(await request.text()):undefined});if(!replies.has(key)){unexpected.push(key);throw new Error("Unexpected fixture request "+key);}return replies.get(key)!();}));
 clipboardDescriptor=Object.getOwnPropertyDescriptor(navigator,"clipboard");copy=vi.fn(async()=>{});Object.defineProperty(navigator,"clipboard",{configurable:true,value:{writeText:copy}});
 scrollDescriptor=Object.getOwnPropertyDescriptor(HTMLElement.prototype,"scrollIntoView");scroll=vi.fn();Object.defineProperty(HTMLElement.prototype,"scrollIntoView",{configurable:true,value:scroll});
});
afterEach(()=>{clients.splice(0).forEach(c=>c.clear());expect(unexpected).toEqual([]);if(clipboardDescriptor)Object.defineProperty(navigator,"clipboard",clipboardDescriptor);else Reflect.deleteProperty(navigator,"clipboard");if(scrollDescriptor)Object.defineProperty(HTMLElement.prototype,"scrollIntoView",scrollDescriptor);else Reflect.deleteProperty(HTMLElement.prototype,"scrollIntoView");vi.unstubAllGlobals();vi.restoreAllMocks();});
describe("canonical plan panel owner workflows",()=>{
 it("accepts only the named fixture plan and reconciles its exact item cache",async()=>{
  replies.set("POST /backlog/fix/fixture-owner/plan-accept",()=>{owner={...owner,plan_acceptance:acceptance};return json({plan_acceptance:acceptance});});const{client}=mount();const button=await screen.findByRole("button",{name:"Accept plan"});fireEvent.click(button);
  expect(await screen.findByText("Plan accepted. Queueing will recheck this exact revision and scope.")).toBeVisible();await screen.findByRole("button",{name:"Un-accept"});
  expect(mutations()).toEqual([{method:"POST",path:"/backlog/fix/fixture-owner/plan-accept",body:{}}]);expect(client.getQueryData(["backlog-item","fix","fixture-owner"])).toMatchObject({name:"fixture-owner",planAcceptance:{actor:"fixture-operator",planContentHash:"fixture-plan-hash",subjectVersion:"fixture-subject"}});
 });
 it("keeps an acceptance pending, disables another click and then reflects only owner completion",async()=>{
  let release!:(value:Response)=>void;replies.set("POST /backlog/fix/fixture-owner/plan-accept",()=>new Promise(resolve=>{release=resolve}));mount();fireEvent.click(await screen.findByRole("button",{name:"Accept plan"}));const saving=await screen.findByRole("button",{name:"Saving…"});expect(saving).toBeDisabled();fireEvent.click(saving);expect(mutations()).toHaveLength(1);
  await act(async()=>{owner={...owner,plan_acceptance:acceptance};release(json({plan_acceptance:acceptance}));});await screen.findByRole("button",{name:"Un-accept"});expect(mutations()).toHaveLength(1);
 });
 it("shows canonical forbidden feedback and leaves the unaccepted item cache unchanged",async()=>{
  replies.set("POST /backlog/fix/fixture-owner/plan-accept",()=>json({error:"owner_refused",message:"Exact plan refusal"},403));const{client}=mount();fireEvent.click(await screen.findByRole("button",{name:"Accept plan"}));expect(await screen.findByText("You don't have permission to access this resource.")).toBeVisible();expect(screen.getByRole("button",{name:"Accept plan"})).toBeEnabled();expect(client.getQueryData(["backlog-item","fix","fixture-owner"])).not.toHaveProperty("planAcceptance");expect(mutations()).toHaveLength(1);
 });
 it("preserves a non-auth owner validation detail without retry or cache acceptance",async()=>{
  replies.set("POST /backlog/fix/fixture-owner/plan-accept",()=>json({error:"owner_refused",message:"Exact plan refusal"},422));const{client}=mount();fireEvent.click(await screen.findByRole("button",{name:"Accept plan"}));expect(await screen.findByText("Exact plan refusal")).toBeVisible();expect(screen.getByRole("button",{name:"Accept plan"})).toBeEnabled();expect(client.getQueryData(["backlog-item","fix","fixture-owner"])).not.toHaveProperty("planAcceptance");expect(mutations()).toEqual([{method:"POST",path:"/backlog/fix/fixture-owner/plan-accept",body:{}}]);
 });
 it("clears only the named acceptance after owner success and refreshes that item",async()=>{
  owner={...owner,plan_acceptance:acceptance};replies.set("DELETE /backlog/fix/fixture-owner/plan-accept",()=>{const{plan_acceptance:_removed,...rest}=owner;owner=rest;return new Response(null,{status:204});});const{client}=mount();fireEvent.click(await screen.findByRole("button",{name:"Un-accept"}));expect(await screen.findByText("Plan acceptance cleared.")).toBeVisible();await screen.findByRole("button",{name:"Accept plan"});expect(mutations()).toEqual([{method:"DELETE",path:"/backlog/fix/fixture-owner/plan-accept",body:undefined}]);expect(client.getQueryData(["backlog-item","fix","fixture-owner"])).not.toHaveProperty("planAcceptance");
 });
 it("retains the accepted owner state when clearing acceptance is refused",async()=>{
  owner={...owner,plan_acceptance:acceptance};replies.set("DELETE /backlog/fix/fixture-owner/plan-accept",()=>json({message:"Clear acceptance refused"},403));const{client}=mount();fireEvent.click(await screen.findByRole("button",{name:"Un-accept"}));expect(await screen.findByText("You don't have permission to access this resource.")).toBeVisible();expect(screen.getByRole("button",{name:"Un-accept"})).toBeEnabled();expect(client.getQueryData(["backlog-item","fix","fixture-owner"])).toMatchObject({planAcceptance:{planContentHash:"fixture-plan-hash"}});expect(mutations()).toHaveLength(1);
 });
 it("shows stale acceptance as ineligible and re-accepts the exact current subject only",async()=>{
  owner={...owner,plan_acceptance:acceptance};replies.set("POST /backlog/fix/fixture-owner/plan-accept",()=>json({plan_acceptance:acceptance}));mount({id:"accept_plan",compactLabel:"Re-accept",expandedLabel:"Re-accept",enabled:true,blockers:[],reason:"Scope changed"});expect(await screen.findByText("Acceptance out of date.")).toBeVisible();expect(screen.getByText("Scope changed")).toBeVisible();fireEvent.click(screen.getByRole("button",{name:"Re-accept"}));await screen.findByText("Plan accepted. Queueing will recheck this exact revision and scope.");expect(mutations()).toEqual([{method:"POST",path:"/backlog/fix/fixture-owner/plan-accept",body:{}}]);
 });
 it("opens one fixture review subject then starts only that returned session",async()=>{
  replies.set("POST /plan-workshops",()=>json({id:"fixture-session",subject:{kind:"backlog_item",ref:"fix/fixture-owner"}}));replies.set("POST /plan-workshops/fixture-session/review",()=>json({session:{id:"fixture-session"},review:{id:"fixture-review"}}));mount();fireEvent.click(await screen.findByRole("button",{name:"Run plan review"}));expect(await screen.findByText("Plan review started. Its findings and proposals will appear in Decide.")).toBeVisible();expect(mutations()).toEqual([{method:"POST",path:"/plan-workshops",body:{subject:{kind:"backlog_item",ref:"fix/fixture-owner"}}},{method:"POST",path:"/plan-workshops/fixture-session/review",body:{}}]);
 });
 it.each(["open","review"]as const)("keeps %s forbidden feedback visible without alternate session or duplicate review",async phase=>{
  replies.set("POST /plan-workshops",()=>phase==="open"?json({message:"Fixture review refused"},403):json({id:"fixture-session"}));replies.set("POST /plan-workshops/fixture-session/review",()=>json({message:"Fixture review refused"},403));mount();fireEvent.click(await screen.findByRole("button",{name:"Run plan review"}));expect(await screen.findByText("You don't have permission to access this resource.")).toBeVisible();expect(mutations()).toHaveLength(phase==="open"?1:2);expect(screen.getByRole("button",{name:"Run plan review"})).toBeEnabled();
 });
 it("holds review pending until the owner response and blocks duplicate review starts",async()=>{
  let release!:(value:Response)=>void;replies.set("POST /plan-workshops",()=>json({id:"fixture-session"}));replies.set("POST /plan-workshops/fixture-session/review",()=>new Promise(resolve=>{release=resolve}));mount();fireEvent.click(await screen.findByRole("button",{name:"Run plan review"}));const pending=await screen.findByRole("button",{name:"Starting…"});expect(pending).toBeDisabled();fireEvent.click(pending);await waitFor(()=>expect(mutations()).toHaveLength(2));await act(async()=>release(json({session:{id:"fixture-session"},review:{id:"fixture-review"}})));await screen.findByText("Plan review started. Its findings and proposals will appear in Decide.");expect(mutations()).toHaveLength(2);
 });
 it("copies exact rendered markdown and clears copied feedback after its existing timer",async()=>{
  mount();fireEvent.click(await screen.findByRole("button",{name:"Copy plan"}));await waitFor(()=>expect(copy).toHaveBeenCalledWith(markdown));expect(screen.getByRole("button",{name:"Copy plan"})).toHaveAttribute("title","Copied!");await waitFor(()=>expect(screen.getByRole("button",{name:"Copy plan"})).toHaveAttribute("title","Copy plan"),{timeout:3500});expect(mutations()).toEqual([]);
 });
 it("jumps to the exact displayed heading and closes the actual table of contents",async()=>{
  mount();fireEvent.click(await screen.findByRole("button",{name:"Table of contents"}));const nav=screen.getByTestId("toc-popover");expect(nav).toBeVisible();fireEvent.click(screen.getByRole("button",{name:"Scope"}));expect(screen.queryByTestId("toc-popover")).toBeNull();expect(scroll).toHaveBeenCalledWith({behavior:"smooth",block:"start"});expect(scroll.mock.contexts).toEqual([screen.getByRole("heading",{name:"Scope"})]);expect(mutations()).toEqual([]);
 });
 it("uses rendered formatting and duplicate heading order instead of raw markdown or global IDs",async()=>{
  replies.set("GET /backlog/fix/fixture-owner/plan-render",()=>rendered({markdown:"# **Repeated**\n\n## Repeated\n\nSetext scope\n------------\n\n```\n# Fake heading\n```"}));mount();await screen.findByRole("heading",{name:"Setext scope"});const headings=screen.getAllByRole("heading",{name:"Repeated"});fireEvent.click(await screen.findByRole("button",{name:"Table of contents"}));const nav=screen.getByTestId("toc-popover");expect(within(nav).queryByRole("button",{name:"Fake heading"})).toBeNull();fireEvent.click(within(nav).getAllByRole("button",{name:"Repeated"})[1]!);expect(scroll.mock.contexts).toEqual([headings[1]]);fireEvent.click(screen.getByRole("button",{name:"Table of contents"}));fireEvent.click(within(screen.getByTestId("toc-popover")).getByRole("button",{name:"Setext scope"}));expect(scroll.mock.contexts[1]).toBe(screen.getByRole("heading",{name:"Setext scope"}));expect(mutations()).toEqual([]);
 });
 it("keeps navigation inside this rendered panel despite an unrelated matching global ID",async()=>{
  const outside=document.createElement("h2");outside.id="scope";outside.textContent="Other panel";document.body.append(outside);try{mount();const exact=await screen.findByRole("heading",{name:"Scope"});fireEvent.click(await screen.findByRole("button",{name:"Table of contents"}));fireEvent.click(screen.getByRole("button",{name:"Scope"}));expect(scroll.mock.contexts).toEqual([exact]);expect(scroll.mock.contexts).not.toContain(outside);expect(mutations()).toEqual([]);}finally{outside.remove();}
 });
 it("rebuilds navigation for the latest rendered plan without retaining detached headings",async()=>{
  const{client}=mount();const old=await screen.findByRole("heading",{name:"Scope"});await act(async()=>{client.setQueryData(["backlog-plan-render","fix","fixture-owner"],{path:"fixture",markdown:"# New owner\n\n### Current scope",planRef:{provider:"plan-manager",planId:"fixture-plan",slug:"fixture-plan",role:"execution_spec"}});});const current=await screen.findByRole("heading",{name:"Current scope"});fireEvent.click(await screen.findByRole("button",{name:"Table of contents"}));expect(within(screen.getByTestId("toc-popover")).queryByRole("button",{name:"Scope"})).toBeNull();fireEvent.click(screen.getByRole("button",{name:"Current scope"}));expect(scroll.mock.contexts).toEqual([current]);expect(scroll.mock.contexts).not.toContain(old);expect(mutations()).toEqual([]);
 });
 it("closes table of contents through its real Escape listener without changing owner state",async()=>{
  mount();fireEvent.click(await screen.findByRole("button",{name:"Table of contents"}));expect(screen.getByTestId("toc-popover")).toBeVisible();fireEvent.keyDown(document,{key:"Escape"});await waitFor(()=>expect(screen.queryByTestId("toc-popover")).toBeNull());expect(mutations()).toEqual([]);
 });
 it("opens only the discovered exact fixture plan URL",async()=>{
  const open=vi.spyOn(window,"open").mockReturnValue(null);mount();const button=await screen.findByRole("button",{name:"Open in plan-manager"});await waitFor(()=>expect(button).toBeEnabled());fireEvent.click(button);expect(open).toHaveBeenCalledWith("https://fixture.invalid/plan-manager/plans/fixture-plan","_blank","noopener,noreferrer");expect(mutations()).toEqual([]);
 });
 it("keeps external opening disabled when the discovered service is unavailable",async()=>{
  replies.set("GET /embedded/plan-manager/external-url",()=>json({}));const open=vi.spyOn(window,"open").mockReturnValue(null);mount();const button=await screen.findByRole("button",{name:"Open in plan-manager"});expect(button).toBeDisabled();fireEvent.click(button);expect(open).not.toHaveBeenCalled();expect(mutations()).toEqual([]);
 });
});
