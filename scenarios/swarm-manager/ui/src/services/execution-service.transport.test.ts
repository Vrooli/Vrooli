// Real service, protobuf validation/mapping and HTTP client; disposable fetch boundary only.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient, ApiError } from "../lib/api-client";
import { createExecutionService } from "./execution-service";

type Call = { method: string; path: string; body: unknown };
let calls: Call[];
let reply: () => Response | Promise<Response>;
const record = { execution_id: "exec-owner", backlog_kind: "fix", backlog_name: "owner-task", mode: "manual", status: "running", created_at: "2026-10-05T00:00:00Z", updated_at: "2026-10-05T00:00:00Z", run_id: "run-owner" };
function json(body: unknown, status = 200) { return new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } }); }
function service() { return createExecutionService(new ApiClient("http://fixture.invalid/api/v1")); }
beforeEach(() => {
  calls = []; reply = () => json({ execution: record });
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const request = new Request(input, init); const url = new URL(request.url);
    expect(url.origin).toBe("http://fixture.invalid"); expect(request.cache).toBe("no-store");
    calls.push({ method: request.method, path: url.pathname + url.search, body: request.body ? JSON.parse(await request.text()) : undefined });
    return reply();
  }));
});
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });
describe("execution actual service transport", () => {
  it("retains exact caller filters and maps the returned identity", async () => {
    reply = () => json({ items: [record] });
    const result = await service().list({status:"running",mode:"manual",backlogKind:"fix",backlogName:"owner task",startedBy:"verified operator",createdFrom:"from",createdTo:"to"});
    expect(result).toMatchObject([{executionId:"exec-owner",backlogKind:"fix",backlogName:"owner-task",runId:"run-owner",status:"running",mode:"manual"}]);
    expect(calls).toEqual([{method:"GET",path:"/api/v1/execution?status=running&mode=manual&backlog_kind=fix&backlog_name=owner+task&started_by=verified+operator&created_from=from&created_to=to",body:undefined}]);
  });
  it("lists without invented filters and reads the exact execution", async () => {
    reply = () => json({items:[]}); expect(await service().list()).toEqual([]);
    reply = () => json({execution:record}); expect(await service().get("exec-owner")).toMatchObject({executionId:"exec-owner"});
    expect(calls).toEqual([{method:"GET",path:"/api/v1/execution",body:undefined},{method:"GET",path:"/api/v1/execution/exec-owner",body:undefined}]);
  });
  it("creates with explicit attribution and complete execution preferences", async () => {
    expect(await service().create({backlogKind:"fix",backlogName:"owner-task",mode:"manual",startedBy:"operator",operation:"improver",preferredRunner:"codex",model:"owner-model",effort:"high"})).toMatchObject({executionId:"exec-owner"});
    expect(calls).toEqual([{method:"POST",path:"/api/v1/execution",body:{backlog_kind:"fix",backlog_name:"owner-task",mode:"manual",started_by:"operator",operation:"improver",execution_preferences:{preferred_runner:"codex",model:"owner-model",effort:"high"}}}]);
  });
  it("omits absent attribution/preferences and preserves partially specified preferences", async () => {
    await service().create({backlogKind:"fix",backlogName:"owner-task",mode:"manual"});
    await service().create({backlogKind:"fix",backlogName:"owner-task",mode:"manual",model:"owner-model"});
    expect(calls.map(c=>c.body)).toEqual([{backlog_kind:"fix",backlog_name:"owner-task",mode:"manual"},{backlog_kind:"fix",backlog_name:"owner-task",mode:"manual",execution_preferences:{preferred_runner:"",model:"owner-model",effort:""}}]);
  });
  it.each(["start","cancel","triggerReview"] as const)("%s uses the exact target once and maps the result", async action => {
    expect(await service()[action]("exec-owner")).toMatchObject({executionId:"exec-owner",runId:"run-owner"});
    expect(calls).toEqual([{method:"POST",path:`/api/v1/execution/exec-owner/${action === "triggerReview" ? "trigger-review" : action}`,body:{}}]);
  });
  it("retries only on explicit invocation, retaining authored note", async () => {
    await service().retry("exec-owner","Owner follow-up"); await service().retry("exec-owner");
    expect(calls.map(c=>[c.path,c.body])).toEqual([["/api/v1/execution/exec-owner/retry",{note:"Owner follow-up"}],["/api/v1/execution/exec-owner/retry",{}]]);
  });
  it("retains follow-up type, execution identity, context and chosen run mode", async () => {
    await service().followUp("exec-owner",{followUpType:"custom",context:"Exact owner context",runMode:"continue"});
    await service().followUp("exec-owner",{followUpType:"fixup",runMode:"new"});
    expect(calls.map(c=>[c.path,c.body])).toEqual([["/api/v1/execution/exec-owner/follow-up",{execution_id:"exec-owner",follow_up_type:"custom",context:"Exact owner context",run_mode:"continue"}],["/api/v1/execution/exec-owner/follow-up",{execution_id:"exec-owner",follow_up_type:"fixup",run_mode:"new"}]]);
  });
  it("halts and resumes only the selected continuation with optional reason", async () => {
    reply = () => new Response(null,{status:204});
    await service().haltContinuation("fix/owner-task","Owner hold"); await service().haltContinuation("fix/owner-task"); await service().resumeContinuation("fix/owner-task");
    expect(calls).toEqual([{method:"POST",path:"/api/v1/execution/continuation/halt",body:{item:"fix/owner-task",reason:"Owner hold"}},{method:"POST",path:"/api/v1/execution/continuation/halt",body:{item:"fix/owner-task"}},{method:"POST",path:"/api/v1/execution/continuation/resume",body:{item:"fix/owner-task"}}]);
  });
  it("does not resolve a pending creation before its owner response", async () => {
    let release!: (value: Response) => void; reply = () => new Promise(resolve=>{release=resolve});
    let settled = false; const pending = service().create({backlogKind:"fix",backlogName:"owner-task",mode:"manual"}).then(value=>{settled=true;return value;});
    await vi.waitFor(()=>expect(calls).toHaveLength(1)); expect(settled).toBe(false);
    release(json({execution:record})); expect(await pending).toMatchObject({executionId:"exec-owner"}); expect(calls).toHaveLength(1);
  });
  it.each(["create","cancel","followUp"] as const)("preserves %s refusal without automatic retries or alternate effects", async action => {
    reply = () => json({error:"caller_identity_required",message:"Owner refused",details:{channel:"verified"}},403);
    const s=service(); const pending=action==="create"?s.create({backlogKind:"fix",backlogName:"owner-task",mode:"manual"}):action==="followUp"?s.followUp("exec-owner",{followUpType:"fixup",runMode:"new"}):s.cancel("exec-owner");
    await expect(pending).rejects.toMatchObject({name:"ApiError",status:403,code:"caller_identity_required",message:"Owner refused",details:{channel:"verified"},isRetryable:false}); expect(calls).toHaveLength(1);
  });
  it.each([{}, {execution:{...record,status:7}}, {execution:"wrong shape"}])("refuses incomplete or malformed response rather than inventing success", async body => {
    reply=()=>json(body); await expect(service().get("exec-owner")).rejects.toThrow("Invalid execution response"); expect(calls).toHaveLength(1);
  });
  it("preserves malformed JSON as an HTTP parse failure with one request", async () => {
    reply=()=>new Response("{broken",{headers:{"content-type":"application/json"}}); await expect(service().get("exec-owner")).rejects.toBeInstanceOf(ApiError); expect(calls).toHaveLength(1);
  });
});
