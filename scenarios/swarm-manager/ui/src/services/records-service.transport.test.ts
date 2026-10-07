// Real records adapter and HTTP client; disposable fetch is the sole service boundary.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "../lib/api-client";
import { createRecordsService } from "./records-service";

type Call = { method: string; path: string; body: unknown };
let calls: Call[]; let reply: () => Response | Promise<Response>;
const record = {id:"record-owner",kind:"fix",scenario:"swarm-manager",backlog_ref:"fix/owner-task",milestone_id:"milestone-owner",supersedes:"record-old",superseded_by:"record-next",trigger:"Owner trigger",approach:"Owner approach",ruled_out:["Guessing"],evidence:"Owner receipt",commit:"owner-commit",files_changed:["src/owner.ts"],outcome:"shipped",stub:false,draft:false,created_at:"2026-10-05T00:00:00Z",created_by:"operator",narrative_at:"2026-10-05T01:00:00Z"};
function json(body: unknown, status=200) { return new Response(JSON.stringify(body),{status,headers:{"content-type":"application/json"}}); }
function service() { return createRecordsService(new ApiClient("http://fixture.invalid/api/v1")); }
beforeEach(()=>{calls=[];reply=()=>json({record});vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{const request=new Request(input,init);const url=new URL(request.url);expect(url.origin).toBe("http://fixture.invalid");expect(request.cache).toBe("no-store");calls.push({method:request.method,path:url.pathname+url.search,body:request.body?JSON.parse(await request.text()):undefined});return reply();}));});
afterEach(()=>{vi.unstubAllGlobals();vi.restoreAllMocks();});
describe("records actual service transport",()=>{
  it("maps full record identity and authored narrative through exact read",async()=>{
    expect(await service().get("record-owner")).toEqual({id:"record-owner",kind:"fix",scenario:"swarm-manager",backlogRef:"fix/owner-task",milestoneId:"milestone-owner",supersedes:"record-old",supersededBy:"record-next",trigger:"Owner trigger",approach:"Owner approach",ruledOut:["Guessing"],evidence:"Owner receipt",commit:"owner-commit",filesChanged:["src/owner.ts"],outcome:"shipped",stub:false,draft:false,capture:undefined,createdAt:"2026-10-05T00:00:00Z",createdBy:"operator",narrativeAt:"2026-10-05T01:00:00Z"});
    expect(calls).toEqual([{method:"GET",path:"/api/v1/records/record-owner",body:undefined}]);
  });
  it("lists exact selected filters and does not add false or nonpositive filters",async()=>{
    reply=()=>json({records:[record]});expect(await service().list({scenario:"owner scenario",kind:"fix",backlogRef:"fix/owner-task",includeStubs:true,limit:2,offset:1})).toMatchObject([{id:"record-owner"}]);
    await service().list({includeStubs:false,limit:0,offset:-1});await service().list();
    expect(calls.map(c=>c.path)).toEqual(["/api/v1/records?scenario=owner+scenario&kind=fix&backlog_ref=fix%2Fowner-task&include_stubs=true&limit=2&offset=1","/api/v1/records","/api/v1/records"]);
  });
  it("returns empty list and existing absent-field display defaults",async()=>{
    reply=()=>json({});expect(await service().list()).toEqual([]);
    reply=()=>json({record:{}});expect(await service().get("record-owner")).toMatchObject({id:"",kind:"fix",scenario:"",trigger:"",approach:"",ruledOut:[],filesChanged:[],outcome:"shipped",stub:false,draft:false,createdAt:""});
  });
  it("creates exact narrative and attribution without inventing optional references",async()=>{
    await service().create({kind:"fix",scenario:"swarm-manager",trigger:"Owner trigger",approach:"Owner approach",ruledOut:["Guessing"],outcome:"shipped"});
    expect(calls).toEqual([{method:"POST",path:"/api/v1/records",body:{kind:"fix",scenario:"swarm-manager",backlog_ref:"",milestone_id:"",supersedes:"",trigger:"Owner trigger",approach:"Owner approach",ruled_out:["Guessing"],commit:"",files_changed:[],outcome:"shipped",created_by:""}}]);
  });
  it("retains explicit create linkage, commit, files and operator",async()=>{
    await service().create({kind:"fix",scenario:"swarm-manager",backlogRef:"fix/owner-task",milestoneId:"milestone-owner",supersedes:"record-old",trigger:"Owner trigger",approach:"Owner approach",ruledOut:[],commit:"owner-commit",filesChanged:["src/owner.ts"],outcome:"shipped",createdBy:"operator"});
    expect(calls[0]?.body).toEqual({kind:"fix",scenario:"swarm-manager",backlog_ref:"fix/owner-task",milestone_id:"milestone-owner",supersedes:"record-old",trigger:"Owner trigger",approach:"Owner approach",ruled_out:[],commit:"owner-commit",files_changed:["src/owner.ts"],outcome:"shipped",created_by:"operator"});
  });
  it("retains draft capture needs, invalid fields and exact repair identity",async()=>{
    const metadata={raw:{outcome:""},accepted:{kind:"fix"},needs:["outcome"],invalid:[{field:"outcome",value:"unknown",message:"Choose an outcome"}],warnings:["Owner review"]};
    reply=()=>json({disposition:"draft",record:{...record,draft:true,capture:metadata},...metadata,next_action:["records","edit","record-owner"]});
    const input={kind:"fix" as const,scenario:"swarm-manager",trigger:"Owner trigger",approach:"Owner approach",ruledOut:[],outcome:"",evidence:"Owner receipt",createdBy:"operator",idempotencyKey:"owner-key"};
    expect(await service().capture(input)).toMatchObject({disposition:"draft",record:{id:"record-owner",draft:true,capture:metadata},accepted:{kind:"fix"},needs:["outcome"],invalid:metadata.invalid,warnings:["Owner review"],nextAction:["records","edit","record-owner"]});
    reply=()=>json({disposition:"published",record});expect(await service().repairCapture("record-owner",{...input,outcome:"shipped"})).toMatchObject({disposition:"published",record:{id:"record-owner",draft:false},accepted:{},needs:[],invalid:[],warnings:[],nextAction:[]});
    expect(calls).toEqual([{method:"POST",path:"/api/v1/records/capture",body:{kind:"fix",scenario:"swarm-manager",trigger:"Owner trigger",approach:"Owner approach",evidence:"Owner receipt",ruled_out:[],outcome:"",created_by:"operator",idempotency_key:"owner-key"}},{method:"PATCH",path:"/api/v1/records/record-owner/capture",body:{kind:"fix",scenario:"swarm-manager",trigger:"Owner trigger",approach:"Owner approach",evidence:"Owner receipt",ruled_out:[],outcome:"shipped",created_by:"operator",idempotency_key:"owner-key"}}]);
  });
  it("keeps unknown capture disposition a draft and absent metadata empty",async()=>{
    reply=()=>json({disposition:"unknown",record:{capture:{}}});
    expect(await service().capture({kind:"fix",scenario:"swarm-manager",trigger:"",approach:"",ruledOut:[],outcome:""})).toMatchObject({disposition:"draft",record:{capture:{needs:[],invalid:[],warnings:[]}},accepted:{},needs:[],invalid:[],warnings:[],nextAction:[]});
    expect(calls[0]?.body).toMatchObject({evidence:"",created_by:"",idempotency_key:""});
  });
  it("fills only the named narrative and supersedes only with explicit successor",async()=>{
    await service().fillNarrative("record-owner",{trigger:"Why",approach:"How",ruledOut:["Alternative"],outcome:"shipped"});
    await service().supersede("record-owner","record-next","Owner reason");await service().supersede("record-owner","record-next");
    expect(calls).toEqual([{method:"PATCH",path:"/api/v1/records/record-owner/narrative",body:{trigger:"Why",approach:"How",ruled_out:["Alternative"],commit:"",files_changed:[],outcome:"shipped"}},{method:"POST",path:"/api/v1/records/record-owner/supersede",body:{successor_id:"record-next",reason:"Owner reason"}},{method:"POST",path:"/api/v1/records/record-owner/supersede",body:{successor_id:"record-next",reason:""}}]);
  });
  it("returns scored search identities with exact filters and conservative defaults",async()=>{
    reply=()=>json({hits:[{record,score:0.75}]});expect(await service().search("Owner query",{kind:"fix",scenario:"swarm-manager",limit:0})).toMatchObject([{record:{id:"record-owner"},score:0.75}]);
    reply=()=>json({});expect(await service().search("Other query")).toEqual([]);
    expect(calls.map(c=>c.body)).toEqual([{query:"Owner query",kind:"fix",scenario:"swarm-manager",limit:0},{query:"Other query",kind:"",scenario:"",limit:10}]);
  });
  it("keeps a capture pending until its owner response and sends only one request",async()=>{
    let release!:(value:Response)=>void;reply=()=>new Promise(resolve=>{release=resolve});let settled=false;
    const pending=service().capture({kind:"fix",scenario:"swarm-manager",trigger:"",approach:"",ruledOut:[],outcome:""}).then(result=>{settled=true;return result;});
    await vi.waitFor(()=>expect(calls).toHaveLength(1));expect(settled).toBe(false);release(json({disposition:"draft",record}));expect(await pending).toMatchObject({record:{id:"record-owner"},disposition:"draft"});expect(calls).toHaveLength(1);
  });
  it.each(["create","repairCapture","supersede"] as const)("preserves %s refusal without retries or alternate mutation",async action=>{
    reply=()=>json({error:"owner_refused",message:"Exact owner refusal",details:{record:"record-owner"}},403);const s=service();
    const pending=action==="create"?s.create({kind:"fix",scenario:"swarm-manager",trigger:"",approach:"",ruledOut:[],outcome:"shipped"}):action==="repairCapture"?s.repairCapture("record-owner",{kind:"fix",scenario:"swarm-manager",trigger:"",approach:"",ruledOut:[],outcome:""}):s.supersede("record-owner","record-next");
    await expect(pending).rejects.toMatchObject({name:"ApiError",status:403,code:"owner_refused",message:"Exact owner refusal",details:{record:"record-owner"},isRetryable:false});expect(calls).toHaveLength(1);
  });
});
