import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { QueryClient } from "@tanstack/react-query";
import { PromptsPage } from "./PromptsPage";
import { defaultApiClient } from "../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import type { PromptSkillSummary, PromptSkillVersions } from "../types";

// Real page, services, hooks, providers and cache. Only network methods are
// controlled. The existing global Monaco browser-input affordance is retained;
// these cases do not qualify a real installed Monaco worker or live publication.
const skill: PromptSkillSummary = { id:"owner-skill",name:"Owner skill",description:"Scoped authored instructions",draft:false,usage_type:"direct_runtime",trigger_count:1,impact_summary:"One existing workflow",groups:["backlog"],current_content:"Current\nKeep" };
let current: PromptSkillSummary;
let versions: PromptSkillVersions;
let empty: boolean;
const clients: QueryClient[]=[];const unexpected:string[]=[];const permitted=new Set<string>();
function mount(){const client=createTestQueryClient();clients.push(client);return {...renderWithProviders(<PromptsPage/>,{queryClient:client}),client};}
async function viewer(){await userEvent.click(await screen.findByRole("tab",{name:"Skills Viewer"}));const editor=await screen.findByTestId(selectors.prompts.contentInput);await waitFor(()=>expect(editor).toHaveValue(current.current_content));return editor;}
beforeEach(()=>{
 current=structuredClone(skill);empty=false;unexpected.length=0;permitted.clear();versions={skillId:skill.id,current:2,versions:[{version:1,content:"Original\r\nKeep",name:skill.name,updatedAt:"2026-10-01T00:00:00Z"},{version:2,content:skill.current_content ?? "",name:skill.name,updatedAt:"2026-10-05T00:00:00Z"}]};
 vi.spyOn(defaultApiClient,"get").mockImplementation(async path=>{
  if(path === "/prompts/catalog")return {items:[{id:"owner-entry",title:"Owner workflow",group:"backlog",usage_type:"direct_runtime",source_type:"skill",skill_id:skill.id,trigger:"Existing user request",purpose:"Preserve scoped instructions"}]};
  if(path === "/prompts/skills")return {items:empty ? []:[current]};
  if(path === "/prompts/skills/owner-skill")return {item:current};
  if(path === "/prompts/skills/owner-skill/versions")return versions;
  unexpected.push(path);throw new Error(`Unexpected prompt read ${path}`);
 });
 vi.spyOn(defaultApiClient,"put").mockRejectedValue(new Error("Unexpected prompt PUT"));vi.spyOn(defaultApiClient,"post").mockRejectedValue(new Error("Unexpected prompt POST"));vi.spyOn(defaultApiClient,"patch").mockRejectedValue(new Error("Unexpected prompt PATCH"));vi.spyOn(defaultApiClient,"delete").mockRejectedValue(new Error("Unexpected prompt DELETE"));
});
afterEach(()=>{expect(unexpected).toEqual([]);if(!permitted.has("put"))expect(defaultApiClient.put).not.toHaveBeenCalled();if(!permitted.has("post"))expect(defaultApiClient.post).not.toHaveBeenCalled();expect(defaultApiClient.patch).not.toHaveBeenCalled();expect(defaultApiClient.delete).not.toHaveBeenCalled();clients.splice(0).forEach(c=>c.clear());vi.restoreAllMocks();});

describe("PromptsPage exact owner skill workflows",()=>{
 it("keeps local content edits and raw/rendered preview changes free of owner writes or cache mutations",async()=>{
  const {client}=mount();const editor=await viewer();const before=client.getQueryData(["prompts","skill",skill.id]);fireEvent.change(editor,{target:{value:"# Local draft\nUnpublished instructions"}});fireEvent.click(screen.getByRole("button",{name:"Show rendered markdown"}));expect(await screen.findByRole("heading",{name:"Local draft"})).toBeVisible();fireEvent.click(screen.getByRole("button",{name:"Show raw markdown"}));expect(screen.getByTestId(selectors.prompts.contentInput)).toHaveValue("# Local draft\nUnpublished instructions");expect(client.getQueryData(["prompts","skill",skill.id])).toBe(before);
 });
 it.each([["Save Draft",true,"Draft saved"],["Publish",false,"Skill published"]] as const)("%s sends only exact current skill/content and waits for owner completion",async(label,draft,confirmation)=>{
  permitted.add("put");let complete!: (r:unknown)=>void;vi.mocked(defaultApiClient.put).mockImplementation(()=>new Promise(r=>{complete=r;}));const {client}=mount();const editor=await viewer();const before=client.getQueryData(["prompts","skill",skill.id]);fireEvent.change(editor,{target:{value:"Exact owner revision"}});fireEvent.click(screen.getByRole("button",{name:label}));await waitFor(()=>expect(vi.mocked(defaultApiClient.put).mock.calls).toEqual([["/prompts/skills/owner-skill",{content:"Exact owner revision",draft}]]));expect(screen.getByRole("button",{name:"Save Draft"})).toBeDisabled();expect(screen.getByRole("button",{name:"Publish"})).toBeDisabled();fireEvent.click(screen.getByRole("button",{name:label}));expect(defaultApiClient.put).toHaveBeenCalledOnce();expect(client.getQueryData(["prompts","skill",skill.id])).toBe(before);
  current={...current,current_content:"Owner canonical revision",draft};await act(async()=>complete({item:current}));expect(await screen.findByText(confirmation)).toBeVisible();await waitFor(()=>expect(screen.getByTestId(selectors.prompts.contentInput)).toHaveValue("Owner canonical revision"));expect(client.getQueryData(["prompts","skill",skill.id])).toEqual(current);
 });
 it("retains a refused draft and exact owner cache without automatic retry or publication fallback",async()=>{
  permitted.add("put");vi.mocked(defaultApiClient.put).mockRejectedValue(new Error("Skill owner refused"));const {client}=mount();const editor=await viewer();const before=client.getQueryData(["prompts","skill",skill.id]);fireEvent.change(editor,{target:{value:"Retain local authored revision"}});fireEvent.click(screen.getByRole("button",{name:"Save Draft"}));expect(await screen.findByText("Skill owner refused")).toBeVisible();expect(screen.getByTestId(selectors.prompts.contentInput)).toHaveValue("Retain local authored revision");expect(client.getQueryData(["prompts","skill",skill.id])).toBe(before);expect(vi.mocked(defaultApiClient.put).mock.calls).toEqual([["/prompts/skills/owner-skill",{content:"Retain local authored revision",draft:true}]]);expect(screen.queryByText("Skill published")).toBeNull();
 });
 it("does not treat a missing owner update record as successful publication",async()=>{
  permitted.add("put");vi.mocked(defaultApiClient.put).mockResolvedValue({});const {client}=mount();await viewer();const before=client.getQueryData(["prompts","skill",skill.id]);fireEvent.click(screen.getByRole("button",{name:"Publish"}));expect(await screen.findByText("Prompt skill update failed")).toBeVisible();expect(screen.queryByText("Skill published")).toBeNull();expect(client.getQueryData(["prompts","skill",skill.id])).toBe(before);expect(defaultApiClient.put).toHaveBeenCalledOnce();
 });
 it("compares normalized line endings with exact added/removed/unchanged lines without writes",async()=>{
  mount();await viewer();const history=screen.getByTestId(selectors.prompts.versions);fireEvent.click(within(history).getAllByRole("button",{name:"Compare"})[0]!);expect(await screen.findByRole("heading",{name:"Diff vs v1"})).toBeVisible();expect(screen.getByText("- Original\n+ Current\n  Keep",{selector:"pre",normalizer:value=>value})).toBeVisible();
 });
 it("rolls back the exact selected version only after owner completion and clears its comparison",async()=>{
  permitted.add("post");let complete!: (r:unknown)=>void;vi.mocked(defaultApiClient.post).mockImplementation(()=>new Promise(r=>{complete=r;}));const {client}=mount();await viewer();const before=client.getQueryData(["prompts","skill",skill.id]);const history=screen.getByTestId(selectors.prompts.versions);fireEvent.click(within(history).getAllByRole("button",{name:"Compare"})[0]!);fireEvent.click(within(history).getAllByRole("button",{name:"Rollback"})[0]!);await waitFor(()=>expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/prompts/skills/owner-skill/revert/1",{}]]));expect(within(history).getAllByRole("button",{name:"Rollback"}).every(b=>(b as HTMLButtonElement).disabled)).toBe(true);expect(client.getQueryData(["prompts","skill",skill.id])).toBe(before);current={...current,current_content:"Owner restored version"};await act(async()=>complete({item:current}));expect(await screen.findByText("Reverted to version 1")).toBeVisible();await waitFor(()=>expect(screen.queryByRole("heading",{name:"Diff vs v1"})).toBeNull());expect(screen.getByTestId(selectors.prompts.contentInput)).toHaveValue("Owner restored version");
 });
 it("retains a refused rollback comparison and cache without fallback write",async()=>{
  permitted.add("post");vi.mocked(defaultApiClient.post).mockRejectedValue(new Error("Version owner refused"));const {client}=mount();await viewer();const before=client.getQueryData(["prompts","skill",skill.id]);const history=screen.getByTestId(selectors.prompts.versions);fireEvent.click(within(history).getAllByRole("button",{name:"Compare"})[0]!);fireEvent.click(within(history).getAllByRole("button",{name:"Rollback"})[0]!);expect(await screen.findByText("Version owner refused")).toBeVisible();expect(screen.getByRole("heading",{name:"Diff vs v1"})).toBeVisible();expect(client.getQueryData(["prompts","skill",skill.id])).toBe(before);expect(defaultApiClient.post).toHaveBeenCalledOnce();
 });
 it("does not load a missing selected owner record or enable write controls for an empty skill list",async()=>{
  empty=true;mount();await userEvent.click(await screen.findByRole("tab",{name:"Skills Viewer"}));expect(await screen.findByText("Select a prompt skill to inspect and edit.")).toBeVisible();expect(screen.queryByRole("button",{name:"Publish"})).toBeNull();expect(vi.mocked(defaultApiClient.get).mock.calls.some(([p])=>p === "/prompts/skills/owner-skill" || p.endsWith("/versions"))).toBe(false);
 });
});
