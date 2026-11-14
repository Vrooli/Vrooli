// Real controlled form and HTTP client; reviewed field edits remain local drafts.
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { BacklogFormDetailsSection, type BacklogFormDetailsSectionProps } from "./backlog-form-details-section";
type Fields=Pick<BacklogFormDetailsSectionProps,"description"|"status"|"priority"|"tagsInput"|"milestone"|"dependsOn"|"effort"|"acceptanceAllow"|"acceptanceDeny"|"executionMode"|"executionLimits"|"continuation"|"scopePolicy">;
const initial:Fields={description:"Owner draft",status:"backlog",priority:4,tagsInput:"owner",milestone:undefined,dependsOn:undefined,effort:undefined,acceptanceAllow:undefined,acceptanceDeny:undefined,executionMode:undefined,executionLimits:undefined,continuation:undefined,scopePolicy:undefined};
let changes:ReturnType<typeof vi.fn>;let clear:ReturnType<typeof vi.fn>;let calls:{method:string;path:string}[];let reply:()=>Response|Promise<Response>;
function json(value:unknown,status=200){return new Response(JSON.stringify(value),{status,headers:{"content-type":"application/json"}});}
function Harness({values={},disabled=false,edit=true}:{values?:Partial<Fields>;disabled?:boolean;edit?:boolean}){
 const[fields,setFields]=useState<Fields>({...initial,...values});
 return <><BacklogFormDetailsSection {...fields} isEditMode={edit} isSubmitting={disabled} onFieldChange={(field,value)=>{changes(field,value);setFields(previous=>({...previous,[field]:value}));}} onTagsInputChange={value=>{changes("tagsInput",value);setFields(previous=>({...previous,tagsInput:value}));}} onClearError={clear}/><output data-testid="draft">{JSON.stringify(fields)}</output></>;
}
function draft(){return JSON.parse(screen.getByTestId("draft").textContent??"{}");}
function change(label:string,value:string){fireEvent.change(screen.getByLabelText(label),{target:{value}});}
async function mount(ui=<Harness/>){const result=renderWithProviders(ui);await screen.findByRole("option",{name:"Goal session"});return result;}
beforeEach(()=>{changes=vi.fn();clear=vi.fn();calls=[];reply=()=>json({items:[{id:"sliced",display_name:"Phased drain"},{id:"goal",display_name:"Goal session"}]});vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{const r=new Request(input,init);const path=new URL(r.url).pathname;calls.push({method:r.method,path});expect(path.endsWith("/execution/strategies")).toBe(true);expect(r.method).toBe("GET");return reply();}));});
afterEach(()=>{expect(calls.every(c=>c.method==="GET"&&c.path.endsWith("/execution/strategies"))).toBe(true);vi.unstubAllGlobals();vi.restoreAllMocks();});
describe("backlog reviewed form workflows",()=>{
 it("edits exact description, priority, status and tags locally without a mutation",async()=>{
  await mount();change("Description","Exact revision");change("Priority (1-10)","7");change("Status","ready");change("Tags","owner, scoped");
  expect(draft()).toMatchObject({description:"Exact revision",priority:7,status:"ready",tagsInput:"owner, scoped"});expect(changes.mock.calls).toEqual([["description","Exact revision"],["priority",7],["status","ready"],["tagsInput","owner, scoped"]]);expect(clear).toHaveBeenCalledTimes(4);expect(calls).toHaveLength(1);
 });
 it("uses the existing empty-priority fallback and keeps create status noneditable",async()=>{
  await mount(<Harness edit={false}/>);expect(screen.queryByRole("combobox",{name:"Status"})).toBeNull();expect(screen.getByText("Backlog",{selector:"span"})).toBeVisible();change("Priority (1-10)","");expect(draft().priority).toBe(1);expect(changes).toHaveBeenCalledWith("priority",1);expect(clear).toHaveBeenCalledTimes(1);
 });
 it("normalizes authored references and patterns while retaining selected milestone and effort",async()=>{
  await mount();change("Milestone","owner-milestone");change("Dependencies"," fix/owner-task, , idea/next-step , ");change("Effort","M");change("Acceptance Allow"," src/**, , docs/** ");change("Acceptance Deny"," *.lock, node_modules/**, ");
  expect(draft()).toMatchObject({milestone:"owner-milestone",dependsOn:["fix/owner-task","idea/next-step"],effort:"M",acceptanceAllow:["src/**","docs/**"],acceptanceDeny:["*.lock","node_modules/**"]});expect(clear).toHaveBeenCalledTimes(5);expect(changes).toHaveBeenCalledTimes(5);expect(calls).toHaveLength(1);
 });
 it("clears optional references without retaining empty pattern entries",async()=>{
  await mount(<Harness values={{milestone:"old",dependsOn:["fix/old"],effort:"M",acceptanceAllow:["src/**"],acceptanceDeny:["old/**"]}}/>);for(const label of["Milestone","Dependencies","Acceptance Allow","Acceptance Deny","Effort"])change(label,"");expect(draft()).toMatchObject({milestone:"",dependsOn:[],effort:"",acceptanceAllow:[],acceptanceDeny:[]});expect(clear).toHaveBeenCalledTimes(5);
 });
 it("selects an owner-returned strategy and keeps continuation and scope choices as draft state",async()=>{
  await mount();expect(screen.getByLabelText("Execution strategy")).toHaveValue("sliced");change("Execution strategy","goal");fireEvent.click(screen.getByRole("radio",{name:/Until allowance is spent/}));fireEvent.click(screen.getByRole("radio",{name:/Extend with record/}));expect(draft()).toMatchObject({executionMode:"goal",continuation:"until-allowance",scopePolicy:"extend-with-record"});
  fireEvent.click(screen.getByRole("radio",{name:/Manual/}));fireEvent.click(screen.getByRole("radio",{name:/Fixed/}));expect(draft()).toMatchObject({continuation:"manual",scopePolicy:"fixed"});expect(changes).toHaveBeenCalledWith("executionMode","goal");expect(changes).toHaveBeenCalledWith("scopePolicy","fixed");expect(calls).toHaveLength(1);
 });
 it("edits each reviewed limit while preserving other values and explicit zero",async()=>{
  await mount(<Harness values={{executionLimits:{maxSlices:7,maxTokens:1000,maxWallSeconds:60,maxTurns:30,maxChargeMicroUsd:5,maxChildren:2,maxNodeAttempts:3,maxRetries:4}}}/>);
  const fields=[["Maximum slices","maxSlices",9],["Maximum tokens","maxTokens",2000],["Maximum wall seconds","maxWallSeconds",90],["Maximum turns","maxTurns",40],["Maximum charge (micro-USD)","maxChargeMicroUsd",0],["Maximum child runs","maxChildren",6],["Maximum node attempts","maxNodeAttempts",8],["Maximum retries","maxRetries",0]]as const;
  for(const[label,key,value]of fields){change(label,String(value));expect(draft().executionLimits[key]).toBe(value);}
  expect(draft().executionLimits).toEqual({maxSlices:9,maxTokens:2000,maxWallSeconds:90,maxTurns:40,maxChargeMicroUsd:0,maxChildren:6,maxNodeAttempts:8,maxRetries:0});expect(clear).toHaveBeenCalledTimes(8);expect(changes).toHaveBeenCalledTimes(8);expect(calls).toHaveLength(1);
 });
 it("uses reviewed defaults when initializing absent limits on the first edit",async()=>{
  await mount();change("Maximum retries","0");expect(draft().executionLimits).toEqual({maxSlices:128,maxTokens:2000000,maxWallSeconds:604800,maxTurns:2400,maxChargeMicroUsd:250000000,maxChildren:512,maxNodeAttempts:512,maxRetries:0});
 });
 it("disables every field during parent submission and emits no edits",async()=>{
  const user=userEvent.setup();await mount(<Harness disabled/>);for(const control of[...screen.getAllByRole("textbox"),...screen.getAllByRole("spinbutton"),...screen.getAllByRole("combobox"),...screen.getAllByRole("radio")])expect(control).toBeDisabled();await user.type(screen.getByLabelText("Description"),"No edit");await user.click(screen.getByRole("radio",{name:/Until allowance is spent/}));expect(draft().description).toBe("Owner draft");expect(changes).not.toHaveBeenCalled();expect(clear).not.toHaveBeenCalled();expect(calls).toHaveLength(1);
 });
 it.each(["missing","refused"]as const)("keeps the selected strategy fallback on %s catalog with no invented choices or mutation",async mode=>{
  let release!:(value:Response)=>void;reply=()=>new Promise(resolve=>{release=resolve});renderWithProviders(<Harness values={{executionMode:"owner-selected"}}/>);await waitFor(()=>expect(calls).toHaveLength(1));
  const response=mode==="refused"?json({message:"Owner refused"},403):json({});let consumed!:()=>void;const consumption=new Promise<void>(resolve=>{consumed=resolve});
  // Observe only transport-body consumption; leave the actual parser/client intact.
  if(mode==="refused"){const text=response.text.bind(response);vi.spyOn(response,"text").mockImplementation(async()=>{const result=await text();consumed();return result;});}
  else {const read=response.json.bind(response);vi.spyOn(response,"json").mockImplementation(async()=>{const result=await read();consumed();return result;});}
  await act(async()=>{release(response);await consumption;});expect(screen.getByRole("option",{name:"owner-selected"})).toBeVisible();expect(screen.getByLabelText("Execution strategy").querySelectorAll("option")).toHaveLength(1);expect(changes).not.toHaveBeenCalled();expect(calls).toHaveLength(1);
 });
 it("does not let an obsolete late lookup replace the next form's strategy choices",async()=>{
  let release!:(value:Response)=>void;reply=()=>new Promise(resolve=>{release=resolve});const first=renderWithProviders(<Harness/>);await waitFor(()=>expect(calls).toHaveLength(1));first.unmount();reply=()=>json({items:[{id:"current",display_name:"Current owner strategy"},{id:"fallback-name",display_name:""}]});renderWithProviders(<Harness values={{executionMode:"current"}}/>);await screen.findByRole("option",{name:"Current owner strategy"});expect(screen.getByRole("option",{name:"fallback-name"})).toBeVisible();await act(async()=>release(json({items:[{id:"obsolete",display_name:"Obsolete strategy"}]})));expect(screen.queryByRole("option",{name:"Obsolete strategy"})).toBeNull();expect(screen.getByLabelText("Execution strategy")).toHaveValue("current");expect(calls).toHaveLength(2);expect(changes).not.toHaveBeenCalled();
 });
});
