// Actual scenario service + protobuf validation + HTTP transport; no product lifecycle calls.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "../lib/api-client";
import { createScenariosService } from "./scenarios-service";
let calls: {path:string;method:string;body:unknown}[];let reply:()=>Response|Promise<Response>;
const target={scenarioName:"swarm-manager",providerPhase:"unit",capabilityId:"caller-identity"};
const rawTarget={scenario_name:"swarm-manager",provider_phase:"unit",capability_id:"caller-identity"};
const proposal={target:rawTarget,fingerprint:"owner-fingerprint",provenance:"owner-audit",title:"Caller identity",description:"Qualify exact caller",acceptance_criteria:["Owner refusal preserved"],acceptance_allow:["api/auth"],recommended_workflows:["caller-qualification"]};
const campaignTarget={scenarioName:"swarm-manager",maturityTarget:"ready",providerPhases:["unit","integration"]};
const rawCampaignTarget={scenario_name:"swarm-manager",maturity_target:"ready",provider_phases:["unit","integration"]};
const campaign={target:rawCampaignTarget,fingerprint:"campaign-fingerprint",title:"Owner campaign",description:"Qualify owner gates",acceptance_criteria:["Verified gates"],declared_workflow:"owner-campaign",tracker_availability:"available",tracker_ref:"goal/owner-tracker"};
function json(body:unknown,status=200){return new Response(JSON.stringify(body),{status,headers:{"content-type":"application/json"}});}
function service(){return createScenariosService(new ApiClient("http://fixture.invalid/api/v1"));}
beforeEach(()=>{calls=[];reply=()=>json({proposal});vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{const r=new Request(input,init);const u=new URL(r.url);expect(u.origin).toBe("http://fixture.invalid");calls.push({path:u.pathname+u.search,method:r.method,body:r.body?JSON.parse(await r.text()):undefined});return reply();}));});
afterEach(()=>{vi.unstubAllGlobals();vi.restoreAllMocks();});
describe("scenario remediation and campaign actual transport",()=>{
 it("previews exact remediation target and preserves existing reconciliation without applying",async()=>{
  reply=()=>json({proposal,existing:{fingerprint:"owner-fingerprint",state:"accepted",work_ref:"fix/owner-task"}});
  expect(await service().previewRemediation("swarm-manager",target)).toMatchObject({proposal:{target,fingerprint:"owner-fingerprint",title:"Caller identity",acceptanceCriteria:["Owner refusal preserved"],acceptanceAllow:["api/auth"],recommendedWorkflows:["caller-qualification"]},existing:{state:"accepted",workRef:"fix/owner-task"}});
  expect(calls).toEqual([{method:"POST",path:"/api/v1/scenarios/swarm-manager/remediation/preview",body:{target:rawTarget}}]);
 });
 it("keeps absent reconciliation absent and applies only authored fingerprint",async()=>{
  expect((await service().previewRemediation("swarm-manager",target)).existing).toBeUndefined();
  reply=()=>json({proposal,work_ref:"fix/owner-task",created:false});
  expect(await service().applyRemediation("swarm-manager",target,"owner-fingerprint")).toMatchObject({proposal:{target,fingerprint:"owner-fingerprint"},workRef:"fix/owner-task",created:false});
  expect(calls[1]).toEqual({method:"POST",path:"/api/v1/scenarios/swarm-manager/remediation/apply",body:{target:rawTarget,fingerprint:"owner-fingerprint"}});
 });
 it("previews campaign target and retains exact tracker and existing goal identity",async()=>{
  reply=()=>json({proposal:campaign,existing_goal_ref:"goal/owner-campaign"});
  expect(await service().previewMaturityCampaign("swarm-manager",campaignTarget)).toMatchObject({proposal:{target:campaignTarget,fingerprint:"campaign-fingerprint",declaredWorkflow:"owner-campaign",trackerAvailability:"available",trackerRef:"goal/owner-tracker"},existingGoalRef:"goal/owner-campaign"});
  expect(calls).toEqual([{method:"POST",path:"/api/v1/scenarios/swarm-manager/maturity-campaign/preview",body:{target:rawCampaignTarget}}]);
 });
 it("applies only the selected campaign fingerprint and preserves returned creation decision",async()=>{
  reply=()=>json({proposal:campaign,goal_ref:"goal/owner-campaign",created:true,tracker_availability:"available",tracker_ref:"goal/owner-tracker"});
  expect(await service().applyMaturityCampaign("swarm-manager",campaignTarget,"campaign-fingerprint")).toMatchObject({proposal:{target:campaignTarget,fingerprint:"campaign-fingerprint"},goalRef:"goal/owner-campaign",created:true,trackerAvailability:"available",trackerRef:"goal/owner-tracker"});
  expect(calls).toEqual([{method:"POST",path:"/api/v1/scenarios/swarm-manager/maturity-campaign/apply",body:{target:rawCampaignTarget,fingerprint:"campaign-fingerprint"}}]);
 });
 it("preserves unavailable campaign tracker without inventing a reference",async()=>{
  reply=()=>json({proposal:{...campaign,tracker_availability:"unavailable",tracker_ref:undefined}});
  const value=await service().previewMaturityCampaign("swarm-manager",campaignTarget);expect(value.existingGoalRef).toBeUndefined();expect(value.proposal.trackerAvailability).toBe("unavailable");expect(value.proposal.trackerRef).toBeUndefined();expect(calls).toHaveLength(1);
 });
 it.each(["previewRemediation","applyRemediation","previewMaturityCampaign","applyMaturityCampaign"] as const)("preserves %s owner refusal with no alternate route or retry",async action=>{
  reply=()=>json({error:"owner_refused",message:"Owner refused",details:{target:"swarm-manager"}},403);const s=service();
  const pending=action==="previewRemediation"?s.previewRemediation("swarm-manager",target):action==="applyRemediation"?s.applyRemediation("swarm-manager",target,"owner-fingerprint"):action==="previewMaturityCampaign"?s.previewMaturityCampaign("swarm-manager",campaignTarget):s.applyMaturityCampaign("swarm-manager",campaignTarget,"campaign-fingerprint");
  await expect(pending).rejects.toMatchObject({status:403,code:"owner_refused",message:"Owner refused",details:{target:"swarm-manager"},isRetryable:false});expect(calls).toHaveLength(1);
 });
 it.each([{}, {proposal:{...proposal,target:undefined}}, {proposal:{...proposal,fingerprint:""}}])("refuses absent/invalid remediation proof rather than reporting eligibility",async body=>{
  reply=()=>json(body);await expect(service().previewRemediation("swarm-manager",target)).rejects.toThrow("Invalid scenario remediation preview response");expect(calls).toHaveLength(1);
 });
 it("refuses missing campaign target and never automatically applies the invalid preview",async()=>{
  reply=()=>json({proposal:{...campaign,target:undefined}});await expect(service().previewMaturityCampaign("swarm-manager",campaignTarget)).rejects.toThrow("Invalid scenario maturity campaign preview response");expect(calls).toHaveLength(1);
 });
 it("keeps a campaign apply pending until owner commitment is returned",async()=>{
  let release!:(value:Response)=>void;reply=()=>new Promise(resolve=>{release=resolve});let settled=false;
  const pending=service().applyMaturityCampaign("swarm-manager",campaignTarget,"campaign-fingerprint").then(v=>{settled=true;return v;});await vi.waitFor(()=>expect(calls).toHaveLength(1));expect(settled).toBe(false);release(json({proposal:campaign,goal_ref:"goal/owner-campaign",created:false,tracker_availability:"available"}));expect(await pending).toMatchObject({goalRef:"goal/owner-campaign",created:false});expect(calls).toHaveLength(1);
 });
 it("normalizes legacy and current context casing while preserving goal and orphan identities",async()=>{
  reply=()=>json({scenario_name:"swarm-manager",goals:[{name:"owner-goal",scope:{in_progress:2}},{scope:{}}],orphan_items:[{kind:"fix",name:"owner-task",archived_at:"legacy"},{}],fixes:{active:[{name:"owner-task",archived_at:"old"}],archived:[{}]}});
  expect(await service().getContext("swarm-manager")).toMatchObject({scenarioName:"swarm-manager",goals:[{name:"owner-goal",rollup:{inProgress:2}},{name:"",priority:0,rollup:{total:0}}],orphanItems:[{kind:"fix",name:"owner-task",archivedAt:"legacy"},{kind:"",name:"",priority:0}],rollup:{total:0,inProgress:0},fixes:{active:[{name:"owner-task",archivedAt:"old"}],archived:[{name:"",path:""}]}});
  reply=()=>json({scenarioName:"current",orphanItems:[{archivedAt:"current"}],rollup:{inProgress:3},fixes:{active:[{archivedAt:"current"}]}});
  expect(await service().getContext("swarm-manager")).toMatchObject({scenarioName:"current",goals:[],orphanItems:[{archivedAt:"current"}],rollup:{inProgress:3},fixes:{active:[{archivedAt:"current"}],archived:[]}});
  expect(calls.every(c=>c.path==="/api/v1/scenarios/swarm-manager/context"&&c.method==="GET")).toBe(true);
 });
});
