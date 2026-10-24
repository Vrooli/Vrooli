import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { ProposalSessionsPanel } from "./ProposalSessionsPanel";
import { defaultApiClient } from "../../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../../test-utils";
import type { ProposalSession } from "../../services/proposal-session-service";

// The component, services, generated Connect client, providers and query cache
// are real. Only REST methods and the Connect fetch boundary are controlled.
function session(id = "session-one", proposalId = "proposal-one", status = "ready"): ProposalSession {
  return { id, title: `Owner ${id}`, status: "proposal_ready", skill_id: "fixture-skill", proposal_target: { type: "backlog_item", ref: "fix/owner-item", name: "Owner item" }, proposals: [{ id: proposalId, kind: "mutation_list", status, summary: `Change ${proposalId}`, payload_json: JSON.stringify({ mutations: [{ id: `${proposalId}-mutation`, op: "reset_artifacts", target: "fix/owner-item", rationale: "Reconcile retained evidence" }] }), created_at: "2026-01-28T00:00:00Z", updated_at: "2026-01-28T00:00:00Z" }] };
}
const clients: QueryClient[] = [];
const unexpected: string[] = [];
const decisions: Array<{ path: string; body: unknown }> = [];
let sessions: ProposalSession[];
let decisionResponse: (body: unknown) => Promise<Response>;
function Location() { const l = useLocation(); return <output data-testid="proposal-location">{l.pathname + l.search}</output>; }
function mount() {
  const client = createTestQueryClient(); clients.push(client);
  const view = renderWithProviders(<><ProposalSessionsPanel target={{ type: "backlog_item", ref: "fix/owner-item", name: "Owner item" }} /><Location /></>, { queryClient: client, initialEntries: ["/plan"] });
  return { ...view, client };
}
function card(summary = "Change proposal-one") { const h = screen.getByRole("heading", { name: summary }); const article = h.closest("article"); if (!article) throw new Error("Proposal card absent"); return article; }
async function loaded() { return screen.findByRole("heading", { name: "Change proposal-one" }); }
function ok() { return new Response("{}", { status: 200, headers: { "Content-Type": "application/json" } }); }
beforeEach(() => {
  sessions = [session()]; decisions.length = 0; unexpected.length = 0; decisionResponse = async () => ok();
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/proposal-sessions?target_type=backlog_item&target_ref=fix%2Fowner-item") return { sessions };
    if (path.startsWith("/agent-sessions/")) { const s = sessions.find(s => path === `/agent-sessions/${s.id}`); if (s) return s; }
    unexpected.push(path); throw new Error(`Unexpected proposal read ${path}`);
  });
  vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected proposal POST"));
  vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new Error("Unexpected proposal PATCH"));
  vi.spyOn(defaultApiClient, "delete").mockRejectedValue(new Error("Unexpected proposal DELETE"));
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    const r = new Request(input, init); const path = new URL(r.url).pathname; const body: unknown = JSON.parse(await r.text());
    if (!path.endsWith("/vrooli.swarm_manager.v1.api.BacklogService/DecideAttempt")) throw new Error(`Unexpected proposal RPC ${path}`);
    decisions.push({ path, body }); return decisionResponse(body);
  });
});
afterEach(() => { clients.splice(0).forEach(c => c.clear()); expect(unexpected).toEqual([]); expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled(); vi.restoreAllMocks(); });
function decision(proposal = "proposal-one", ownerSession = "session-one", kind = "accept", note = "") {
  return { subjectKind: "agent-session-proposal", subjectRef: `${ownerSession}/${proposal}`, roundNum: 1, decision: kind, actor: "operator-ui", rationale: note, ...(kind === "accept" ? { acceptedProposalIds: [`${proposal}-mutation`] } : {}) };
}

describe("ProposalSessionsPanel actual decision workflows", () => {
  it("applies the exact reviewed mutation IDs and note only once while pending, then refreshes", async () => {
    let resolve!: (r: Response) => void; decisionResponse = () => new Promise(r => { resolve = r; }); const { client } = mount(); await loaded(); const before = client.getQueryData(["proposal-sessions", "backlog_item", "fix/owner-item"]);
    fireEvent.change(within(card()).getByPlaceholderText("Decision note (optional)"), { target: { value: "Owner reviewed exact evidence" } }); fireEvent.click(within(card()).getByRole("button", { name: "Apply proposal" }));
    await waitFor(() => expect(decisions.map(d => d.body)).toEqual([decision("proposal-one", "session-one", "accept", "Owner reviewed exact evidence")])); expect(within(card()).getByRole("button", { name: "Apply proposal" })).toBeDisabled(); fireEvent.click(within(card()).getByRole("button", { name: "Apply proposal" })); expect(decisions).toHaveLength(1); expect(client.getQueryData(["proposal-sessions", "backlog_item", "fix/owner-item"])).toBe(before);
    await act(async () => resolve(ok())); await waitFor(() => expect(within(card()).getByRole("button", { name: "Apply proposal" })).toBeEnabled()); expect(defaultApiClient.post).not.toHaveBeenCalled();
    await waitFor(() => expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([p]) => p.startsWith("/proposal-sessions?")).length).toBeGreaterThan(1));
  });
  it("rejects without accepting any mutation and carries the exact operator note", async () => {
    mount(); await loaded(); fireEvent.change(within(card()).getByPlaceholderText("Decision note (optional)"), { target: { value: "Keep existing owner data" } }); fireEvent.click(within(card()).getByRole("button", { name: "Reject" })); await waitFor(() => expect(decisions.map(d => d.body)).toEqual([decision("proposal-one", "session-one", "drop", "Keep existing owner data")])); expect(defaultApiClient.post).not.toHaveBeenCalled();
  });
  it("preserves an owner-refused decision and editable note without fallback or cache invalidation", async () => {
    decisionResponse = async () => new Response(JSON.stringify({ code: "permission_denied", message: "Disposition owner refused" }), { status: 403, headers: { "Content-Type": "application/json" } }); const { client } = mount(); await loaded(); const before = client.getQueryData(["proposal-sessions", "backlog_item", "fix/owner-item"]); const readsBefore = vi.mocked(defaultApiClient.get).mock.calls.length;
    fireEvent.change(within(card()).getByPlaceholderText("Decision note (optional)"), { target: { value: "Retain exact note" } }); fireEvent.click(within(card()).getByRole("button", { name: "Apply proposal" })); expect(await screen.findByText(/Disposition owner refused/)).toBeVisible(); expect(within(card()).getByPlaceholderText("Decision note (optional)")).toHaveValue("Retain exact note"); expect(client.getQueryData(["proposal-sessions", "backlog_item", "fix/owner-item"])).toBe(before); expect(vi.mocked(defaultApiClient.get).mock.calls).toHaveLength(readsBefore); expect(decisions).toHaveLength(1); expect(defaultApiClient.post).not.toHaveBeenCalled();
  });
  function keep() { const s = session(); const p = s.proposals?.[0]; if (!p) throw new Error("Missing fixture proposal"); p.payload_json = JSON.stringify({ form: "mutation_list", rationale: "Owner evidence supports the current item", mutations: [] }); return s; }
  it("accepts a no-change recommendation through the exact keep route with note and no mutation decision", async () => {
    sessions = [keep()]; vi.mocked(defaultApiClient.post).mockResolvedValue(sessions[0]); mount(); await screen.findByRole("heading", { name: "No changes recommended" }); const article = card("No changes recommended"); fireEvent.change(within(article).getByPlaceholderText("Decision note (optional)"), { target: { value: "Keep reviewed state" } }); fireEvent.click(within(article).getByRole("button", { name: "Accept keep recommendation" }));
    await waitFor(() => expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/agent-sessions/session-one/proposals/proposal-one/accept-keep", { note: "Keep reviewed state" }]])); expect(decisions).toEqual([]);
  });
  it("requests revision for the exact proposal and navigates only after the owner returns the session", async () => {
    let resolve!: (v: unknown) => void; vi.mocked(defaultApiClient.post).mockImplementation(() => new Promise(r => { resolve = r; })); mount(); await loaded(); fireEvent.change(within(card()).getByPlaceholderText("Decision note (optional)"), { target: { value: "Recheck the receipt" } }); fireEvent.click(within(card()).getByRole("button", { name: "Request revision" }));
    await waitFor(() => expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/agent-sessions/session-one/proposals/proposal-one/revise", { note: "Recheck the receipt" }]])); expect(screen.getByTestId("proposal-location")).toHaveTextContent("/plan"); expect(within(card()).getByRole("button", { name: "Request revision" })).toBeDisabled(); await act(async () => resolve(sessions[0])); await waitFor(() => expect(screen.getByTestId("proposal-location")).toHaveTextContent("/sessions/session-one")); expect(decisions).toEqual([]);
  });
  it("retains revision refusal without navigating, decision fallback or lost note", async () => {
    vi.mocked(defaultApiClient.post).mockRejectedValue(new Error("Revision owner refused")); mount(); await loaded(); fireEvent.change(within(card()).getByPlaceholderText("Decision note (optional)"), { target: { value: "Retain guidance" } }); fireEvent.click(within(card()).getByRole("button", { name: "Request revision" })); expect(await screen.findByText("Revision owner refused")).toBeVisible(); expect(within(card()).getByPlaceholderText("Decision note (optional)")).toHaveValue("Retain guidance"); expect(screen.getByTestId("proposal-location")).toHaveTextContent("/plan"); expect(decisions).toEqual([]); expect(defaultApiClient.post).toHaveBeenCalledOnce();
  });
  it("opens the source session without writing a decision or changing its note", async () => {
    mount(); await loaded(); fireEvent.click(within(card()).getByRole("button", { name: "From session: Owner session-one" })); await waitFor(() => expect(screen.getByTestId("proposal-location")).toHaveTextContent("/sessions/session-one")); expect(decisions).toEqual([]); expect(defaultApiClient.post).not.toHaveBeenCalled();
  });
  async function select() { await loaded(); fireEvent.click(screen.getByRole("button", { name: "Select proposals" })); }
  it("toggles selection and clears only transient selection/note without a write", async () => {
    sessions = [session(), session("session-two", "proposal-two")]; mount(); await select(); fireEvent.click(screen.getByRole("checkbox", { name: "Select Change proposal-one" })); fireEvent.change(screen.getByPlaceholderText("Shared decision note (optional)"), { target: { value: "Unsaved shared note" } }); fireEvent.click(screen.getByRole("button", { name: "Done selecting" })); expect(screen.queryByRole("checkbox")).toBeNull(); expect(screen.queryByPlaceholderText("Shared decision note (optional)")).toBeNull(); fireEvent.click(screen.getByRole("button", { name: "Select proposals" })); expect(screen.getByRole("checkbox", { name: "Select Change proposal-one" })).not.toBeChecked(); expect(decisions).toEqual([]); expect(defaultApiClient.post).not.toHaveBeenCalled();
  });
  it("unchecks a selection without a batch decision or owner effect", async () => {
    sessions = [session(), session("session-two", "proposal-two")]; mount(); await select(); const box = screen.getByRole("checkbox", { name: "Select Change proposal-one" }); fireEvent.click(box); fireEvent.click(box); expect(screen.queryByRole("button", { name: "Apply selected" })).toBeNull(); expect(decisions).toEqual([]); expect(defaultApiClient.post).not.toHaveBeenCalled();
  });
  it.each(["apply", "reject"])("%s batch includes only selected ready proposals and clears transient state on success", async action => {
    sessions = [session(), session("session-two", "proposal-two"), session("session-stale", "proposal-stale", "applied")]; mount(); await select(); fireEvent.click(screen.getByRole("checkbox", { name: "Select Change proposal-one" })); fireEvent.click(screen.getByRole("checkbox", { name: "Select Change proposal-stale" })); fireEvent.change(screen.getByPlaceholderText("Shared decision note (optional)"), { target: { value: "Exact selected decision" } }); fireEvent.click(screen.getByRole("button", { name: action === "apply" ? "Apply selected" : "Reject selected" }));
    await waitFor(() => expect(decisions.map(d => d.body)).toEqual([decision("proposal-one", "session-one", action === "apply" ? "accept" : "drop", "Exact selected decision")])); await waitFor(() => expect(screen.queryByPlaceholderText("Shared decision note (optional)")).toBeNull()); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(screen.getByRole("checkbox", { name: "Select Change proposal-stale" })).not.toBeChecked();
  });
  it("keeps an entirely ineligible batch selected and emits no owner request", async () => {
    sessions = [session("session-one", "proposal-one", "applied"), session("session-two", "proposal-two", "superseded")]; mount(); await select(); fireEvent.click(screen.getByRole("checkbox", { name: "Select Change proposal-one" })); fireEvent.click(screen.getByRole("button", { name: "Apply selected" })); await act(async () => {}); expect(decisions).toEqual([]); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(screen.getByRole("checkbox", { name: "Select Change proposal-one" })).toBeChecked();
  });
  it("requests revisions only for selected revisable proposals and retains unselected cards", async () => {
    sessions = [session(), session("session-two", "proposal-two", "needs_revision"), session("session-stale", "proposal-stale", "applied")]; vi.mocked(defaultApiClient.post).mockResolvedValue(sessions[0]); mount(); await select(); fireEvent.click(screen.getByRole("checkbox", { name: "Select Change proposal-two" })); fireEvent.click(screen.getByRole("checkbox", { name: "Select Change proposal-stale" })); fireEvent.change(screen.getByPlaceholderText("Shared decision note (optional)"), { target: { value: "Bounded revision" } }); fireEvent.click(screen.getByRole("button", { name: "Request revisions" }));
    await waitFor(() => expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/agent-sessions/session-two/proposals/proposal-two/revise", { note: "Bounded revision" }]])); await waitFor(() => expect(screen.queryByPlaceholderText("Shared decision note (optional)")).toBeNull()); expect(decisions).toEqual([]); expect(screen.getByTestId("proposal-location")).toHaveTextContent("/plan");
  });
});

describe("ProposalSessionsPanel bounded batch reconciliation", () => {
  async function chooseTwo() {
    sessions = [session(), session("session-two", "proposal-two")]; const mounted = mount(); await loaded(); fireEvent.click(screen.getByRole("button", { name: "Select proposals" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "Select Change proposal-one" })); fireEvent.click(screen.getByRole("checkbox", { name: "Select Change proposal-two" })); fireEvent.change(screen.getByPlaceholderText("Shared decision note (optional)"), { target: { value: "Retain original reviewed note" } }); return mounted;
  }
  function refused() { return new Response(JSON.stringify({code:"permission_denied",message:"Batch owner refused"}),{status:403,headers:{"Content-Type":"application/json"}}); }
  it.each(["apply", "reject", "revise"])("handles %s batch refusal, retains selection and note, refreshes without retry or alternate operation", async action => {
    decisionResponse = async () => refused(); vi.mocked(defaultApiClient.post).mockRejectedValue(new Error("Revision owner refused")); await chooseTwo();
    fireEvent.click(screen.getByRole("button",{name:action === "apply" ? "Apply selected" : action === "reject" ? "Reject selected" : "Request revisions"}));
    expect(await screen.findByRole("alert")).toHaveTextContent("0 of 2 proposal requests returned successfully. 2 did not return successfully; some changes may already have been saved.");
    await waitFor(() => expect(screen.getByRole("button",{name:"Done selecting"})).toBeEnabled()); expect(screen.getByPlaceholderText("Shared decision note (optional)")).toHaveValue("Retain original reviewed note"); expect(screen.getAllByRole("checkbox").every(box => (box as HTMLInputElement).checked)).toBe(true);
    expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([p])=>p.startsWith("/proposal-sessions?")).length).toBeGreaterThan(1);
    if(action === "revise") { expect(decisions).toEqual([]); expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/agent-sessions/session-one/proposals/proposal-one/revise",{note:"Retain original reviewed note"}],["/agent-sessions/session-two/proposals/proposal-two/revise",{note:"Retain original reviewed note"}]]); }
    else { expect(decisions.map(d=>d.body)).toEqual([decision("proposal-one","session-one",action === "apply" ? "accept" : "drop","Retain original reviewed note"),decision("proposal-two","session-two",action === "apply" ? "accept" : "drop","Retain original reviewed note")]); expect(defaultApiClient.post).not.toHaveBeenCalled(); }
    expect(screen.getByTestId("proposal-location")).toHaveTextContent("/plan");
  });
  it("drains a late successful sibling after early refusal, blocks overlapping input, and clears only its fulfilled selection", async () => {
    let finish!: (r:Response)=>void;
    decisionResponse=async body => { if(typeof body === "object" && body !== null && "subjectRef" in body && body.subjectRef === "session-one/proposal-one") return refused(); return new Promise(r=>{finish=r;}); };
    const {client}=await chooseTwo(); const before=client.getQueryData(["proposal-sessions","backlog_item","fix/owner-item"]); fireEvent.click(screen.getByRole("button",{name:"Apply selected"}));
    await waitFor(()=>expect(decisions).toHaveLength(2)); expect(screen.queryByRole("alert")).toBeNull(); expect(screen.getByRole("button",{name:"Done selecting"})).toBeDisabled(); expect(screen.getByRole("button",{name:"Reject selected"})).toBeDisabled(); expect(screen.getByRole("button",{name:"Request revisions"})).toBeDisabled(); expect(screen.getByPlaceholderText("Shared decision note (optional)")).toBeDisabled(); expect(screen.getAllByRole("checkbox").every(box => (box as HTMLInputElement).disabled)).toBe(true);
    fireEvent.click(screen.getByRole("button",{name:"Apply selected"})); expect(decisions).toHaveLength(2); expect(client.getQueryData(["proposal-sessions","backlog_item","fix/owner-item"])).toBe(before);
    const sibling=sessions[1]?.proposals?.[0]; if(!sibling) throw new Error("Missing sibling"); sibling.status="applied";
    await act(async()=>finish(ok())); expect(await screen.findByRole("alert")).toHaveTextContent("1 of 2 proposal requests returned successfully. 1 did not return successfully; some changes may already have been saved."); await waitFor(()=>expect(screen.getByRole("button",{name:"Done selecting"})).toBeEnabled());
    expect(screen.getByRole("checkbox",{name:"Select Change proposal-one"})).toBeChecked(); expect(screen.getByRole("checkbox",{name:"Select Change proposal-two"})).not.toBeChecked(); expect(screen.getByPlaceholderText("Shared decision note (optional)")).toHaveValue("Retain original reviewed note"); expect(decisions).toHaveLength(2); expect(defaultApiClient.post).not.toHaveBeenCalled();
  });
  it("treats post-decision read failure as uncertain committed effects and refuses stale applied replay after refresh", async () => {
    const originalGet=vi.mocked(defaultApiClient.get).getMockImplementation(); if(!originalGet) throw new Error("Missing read transport");
    vi.mocked(defaultApiClient.get).mockImplementation(async path=>{if(path === "/agent-sessions/session-one") throw new Error("Receipt read unavailable"); return originalGet(path);});
    decisionResponse=async()=>{const proposal=sessions[0]?.proposals?.[0]; if(!proposal) throw new Error("Missing committed proposal"); proposal.status="applied"; return ok();};
    sessions=[session(),session("session-two","proposal-two")]; mount(); await loaded(); fireEvent.click(screen.getByRole("button",{name:"Select proposals"})); fireEvent.click(screen.getByRole("checkbox",{name:"Select Change proposal-one"})); fireEvent.change(screen.getByPlaceholderText("Shared decision note (optional)"),{target:{value:"Exact original note"}}); fireEvent.click(screen.getByRole("button",{name:"Apply selected"}));
    expect(await screen.findByRole("alert")).toHaveTextContent("0 of 1 proposal requests returned successfully. 1 did not return successfully; some changes may already have been saved."); await waitFor(()=>expect(within(card()).getByText("Applied")).toBeVisible()); expect(screen.getByRole("checkbox",{name:"Select Change proposal-one"})).toBeChecked(); expect(screen.getByRole("button",{name:"Apply selected"})).toBeDisabled(); expect(screen.getByRole("button",{name:"Reject selected"})).toBeDisabled(); expect(screen.getByRole("button",{name:"Request revisions"})).toBeDisabled(); fireEvent.click(screen.getByRole("button",{name:"Apply selected"})); expect(decisions).toHaveLength(1); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(screen.getByPlaceholderText("Shared decision note (optional)")).toHaveValue("Exact original note");
  });
  it("reports failed reconciliation without claiming success or automatically repeating writes", async () => {
    const originalGet=vi.mocked(defaultApiClient.get).getMockImplementation(); if(!originalGet) throw new Error("Missing read transport"); let refuseRefresh=false;
    vi.mocked(defaultApiClient.get).mockImplementation(async path=>{if(refuseRefresh && path.startsWith("/proposal-sessions?")) throw new Error("Owner list unavailable"); return originalGet(path);});
    await chooseTwo(); refuseRefresh=true; fireEvent.click(screen.getByRole("button",{name:"Apply selected"}));
    expect(await screen.findByText(/Couldn't refresh the proposals\. Current effects remain uncertain/)).toBeVisible(); await waitFor(()=>expect(screen.getByRole("button",{name:"Done selecting"})).toBeEnabled()); expect(decisions).toHaveLength(2); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(screen.queryByPlaceholderText("Shared decision note (optional)")).toBeNull();
  });
});
