import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { BacklogDetailsPage } from "./BacklogDetailsPage";
import { defaultApiClient } from "../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import { useAgentActivitiesStore, useBacklogDetailUIStore, useBacklogStore } from "../stores";
import type { BacklogItem } from "../types";

// Outside-only draft: real page/hooks/services/widgets/stores and routed views.
// Only upstream client/fetch boundaries are substituted. Synthetic StartTransition
// success qualifies page translation, never actual run admission or caller proof.
const owner = "/backlog/idea/fixture-owner";
const clients: QueryClient[] = [];
const unmounts: Array<() => void> = [];
const unexpected: string[] = [];
const offers: Array<{ path: string; method: string; body: unknown }> = [];
let expectedPatchCalls: unknown[][];
let expectedTransportWrites: Array<{ path: string; method: string; body: unknown }>;
const planWrite = { path: "/vrooli.swarm_manager.v1.api.TransitionService/StartTransition", method: "POST", body: { transitionKey: "plan.author", subjectRef: { subject: "backlog-item", value: "idea/fixture-owner" } } };
const steeringWrite = { path: "/vrooli.swarm_manager.v1.api.BacklogService/DecideAttempt", method: "POST", body: { subjectKind: "backlog-item", subjectRef: "idea/fixture-owner", roundNum: 3, decision: "followup", actor: "fixture-operator", rationale: "Fixture correction before continuing", followUp: { steering: "Fixture correction before continuing", disposition: "replan" } } };
let history: Array<Record<string, unknown>>;
let current: BacklogItem;
let files: Array<{ name: string; path: string; type: string; size: string }>;
let startRefused: boolean;
let startDeferred: boolean;
let releaseStart: (() => void) | undefined;
let decisionRefused: boolean;
let operatorBefore: string | null;
const review = { round: 3, status: "complete", generated_at: "2026-01-01T00:00:00Z", agent_assessment: "Fixture owner feedback needs correction", classification: "inconclusive", evidence: [], disposition: { kind: "attention", rationale: "Fixture owner needs steering", confidence: "medium" } };
function Location() { const location = useLocation(); return <output data-testid="backlog-owner-location">{location.pathname + location.search}</output>; }
function mount(path = `${owner}?keep=fixture`) {
  const client = createTestQueryClient(); clients.push(client);
  const view = renderWithProviders(<><Routes><Route path="/backlog/:kind/:name" element={<BacklogDetailsPage />} /><Route path="/graph" element={<h1>Exact graph owner destination</h1>} /><Route path="/plan" element={<h1>Exact plan owner destination</h1>} /></Routes><Location /></>, { queryClient: client, initialEntries: [path] });
  unmounts.push(view.unmount); return { ...view, client };
}
async function ready() { return screen.findByText("Fixture owner"); }
function noAlternateWrites() {
  expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(defaultApiClient.put).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
  expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual(expectedPatchCalls);
  expect(offers.filter(offer => !offer.path.endsWith("/ListTransitions"))).toEqual(expectedTransportWrites);
}
function countOwnerReads() { return vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === owner).length; }
function fillSteering() {
  const card = screen.getByRole("region", { name: "Review decision" });
  fireEvent.change(within(card).getByLabelText("Operator identity"), { target: { value: " fixture-operator " } });
  fireEvent.change(within(card).getByLabelText("Decision rationale"), { target: { value: " Fixture correction before continuing " } });
  fireEvent.change(within(card).getByLabelText("Follow-up disposition"), { target: { value: "replan" } });
  return card;
}
beforeEach(() => {
  current = { kind: "idea", name: "fixture-owner", title: "Fixture owner", description: "Fixture specification", status: "backlog", priority: 1, tags: [], suggestedSkills: [], created: "2026-01-01T00:00:00Z", updated: "2026-01-01T00:00:00Z" };
  history = [];
  expectedPatchCalls = []; expectedTransportWrites = [];
  files = []; startRefused = false; startDeferred = false; releaseStart = undefined; decisionRefused = false; unexpected.length = 0; offers.length = 0;
  operatorBefore = localStorage.getItem("swarm-manager:operator-identity:v1"); localStorage.removeItem("swarm-manager:operator-identity:v1");
  useBacklogStore.getState().setItems([]); useBacklogDetailUIStore.getState().reset(); useAgentActivitiesStore.setState({ activities: [], isRefreshing: false });
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/settings") return { settings: {} };
    if (path === "/goals") return { items: [] };
    if (path === owner) return { item: current };
    if (path === `${owner}/files`) return { files };
    if (path === `${owner}/plan-render`) return { markdown: "", planRef: null };
    if (path === `${owner}/next-action`) return { action: { id: "none", compactLabel: "No action", expandedLabel: "No action", enabled: false, blockers: [] } };
    if (path === `${owner}/archive/targets`) return { targets: [], requirements: [], has_archive: false };
    if (path === `${owner}/review`) return { rounds: current.status === "review_pending" ? [review] : [] };
    if (path === `${owner}/work-feed`) return { items: [] };
    if (path === "/execution?backlog_kind=idea&backlog_name=fixture-owner") return { items: history };
    if (path.startsWith("/backlog?")) return { items: [], blocking: {} };
    if (path.startsWith("/agent-activities?")) return { items: [] };
    if (path === "/proposal-sessions?target_type=backlog_item&target_ref=idea%2Ffixture-owner") return { sessions: [] };
    if (path === `${owner}/files/spec.json`) return { content: '{"title":"Fixture owner"}' };
    if (path === `${owner}/files/notes.md`) return { content: "# Fixture notes" };
    unexpected.push(path); throw new Error(`Unexpected backlog owner read ${path}`);
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected backlog owner ${method}`));
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? new Request(input, init) : new Request(String(input), init);
    const path = new URL(request.url).pathname; const body: unknown = JSON.parse(await request.text()); offers.push({ path, method: request.method, body });
    if (request.method !== "POST") { unexpected.push(path); throw new Error("Unexpected transport method"); }
    if (path === "/vrooli.swarm_manager.v1.api.TransitionService/ListTransitions") { expect(body).toEqual({}); return new Response(JSON.stringify({ transitions: [{ key: "plan.author", subject: "backlog-item" }] }), { status: 200, headers: { "Content-Type": "application/json" } }); }
    if (path === "/vrooli.swarm_manager.v1.api.TransitionService/StartTransition") {
      expect(body).toEqual({ transitionKey: "plan.author", subjectRef: { subject: "backlog-item", value: "idea/fixture-owner" } });
      if (startDeferred) await new Promise<void>(resolve => { releaseStart = resolve; });
      return new Response(JSON.stringify(startRefused ? { code: "permission_denied", message: "Fixture plan author refused" } : { executionId: "fixture-plan-execution" }), { status: startRefused ? 403 : 200, headers: { "Content-Type": "application/json" } });
    }
    if (path === "/vrooli.swarm_manager.v1.api.BacklogService/DecideAttempt") {
      expect(body).toEqual({ subjectKind: "backlog-item", subjectRef: "idea/fixture-owner", roundNum: 3, decision: "followup", actor: "fixture-operator", rationale: "Fixture correction before continuing", followUp: { steering: "Fixture correction before continuing", disposition: "replan" } });
      return new Response(JSON.stringify(decisionRefused ? { code: "permission_denied", message: "Fixture review owner refused steering" } : {}), { status: decisionRefused ? 403 : 200, headers: { "Content-Type": "application/json" } });
    }
    unexpected.push(path); throw new Error(`Unexpected backlog owner transport ${path}`);
  }));
});
afterEach(() => {
  unmounts.splice(0).forEach(unmount => unmount()); cleanup(); getSpatialNav()?.dispose();
  try { expect(unexpected).toEqual([]); noAlternateWrites(); } finally {
    clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); vi.unstubAllGlobals(); useBacklogStore.getState().setItems([]); useBacklogDetailUIStore.getState().reset(); useAgentActivitiesStore.setState({ activities: [], isRefreshing: false });
    if (operatorBefore === null) localStorage.removeItem("swarm-manager:operator-identity:v1"); else localStorage.setItem("swarm-manager:operator-identity:v1", operatorBefore);
  }
});

describe("Backlog owner plan and status actions", () => {
  it("holds exact plan authoring pending and routes to Activity only after its synthetic owner response", async () => {
    expectedTransportWrites = [planWrite];
    startDeferred = true; mount(`${owner}?keep=fixture&tab=prompt`); const author = await screen.findByTestId("backlog-plan-author-cta"); fireEvent.click(author);
    await waitFor(() => expect(releaseStart).toBeDefined()); expect(author).toBeDisabled(); fireEvent.click(author);
    expect(offers.filter(offer => offer.path.endsWith("/StartTransition"))).toHaveLength(1);
    expect(screen.getByTestId("backlog-owner-location")).toHaveTextContent("tab=prompt");
    await act(async () => releaseStart?.()); await waitFor(() => expect(screen.getByTestId("backlog-owner-location")).toHaveTextContent("tab=activity"));
    expect(screen.getByTestId("backlog-owner-location")).toHaveTextContent("keep=fixture"); expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
  it("keeps refused authoring in Plan with its owner reason and no alternate create-run request", async () => {
    expectedTransportWrites = [planWrite];
    startRefused = true; mount(`${owner}?tab=prompt&keep=fixture`); fireEvent.click(await screen.findByTestId("backlog-plan-author-cta"));
    expect(await screen.findByRole("alert")).toHaveTextContent("Fixture plan author refused"); expect(screen.getByTestId("backlog-owner-location")).toHaveTextContent("tab=prompt");
    expect(offers.filter(offer => offer.path.endsWith("/StartTransition"))).toHaveLength(1); expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
  it("changes only the exact owner status and keeps the status chooser inert while pending", async () => {
    expectedPatchCalls = [[owner, { status: "ready" }]];
    let release!: (value: unknown) => void; vi.mocked(defaultApiClient.patch).mockImplementation((path, body) => { expect([path, body]).toEqual([owner, { status: "ready" }]); return new Promise(resolve => { release = resolve; }); });
    const view = mount(); await ready(); fireEvent.click(screen.getByTestId("status-badge")); fireEvent.click(screen.getByRole("button", { name: "Ready" }));
    await waitFor(() => expect(defaultApiClient.patch).toHaveBeenCalledOnce()); fireEvent.click(screen.getByTestId("status-badge")); expect(screen.queryByRole("button", { name: "Ready" })).toBeNull();
    await act(async () => { current = { ...current, status: "ready" }; release({ item: current }); });
    await waitFor(() => expect(screen.getByTestId("status-badge")).toHaveTextContent("ready")); expect(view.client.getQueryData(["backlog", "idea", "fixture-owner"])).toEqual(current);
    expect(useBacklogStore.getState().items.find(item => item.name === "fixture-owner")?.status).toBe("ready"); expect(defaultApiClient.patch).toHaveBeenCalledOnce();
  });
  it("retains exact owner cache and status when a status patch is refused", async () => {
    expectedPatchCalls = [[owner, { status: "ready" }]];
    vi.mocked(defaultApiClient.patch).mockImplementation(async (path, body) => { expect([path, body]).toEqual([owner, { status: "ready" }]); throw new Error("Fixture status owner refused"); });
    const view = mount(); await ready(); const before = view.client.getQueryData(["backlog", "idea", "fixture-owner"]);
    fireEvent.click(screen.getByTestId("status-badge")); fireEvent.click(screen.getByRole("button", { name: "Ready" })); expect(await screen.findByText("Fixture status owner refused")).toBeVisible();
    expect(screen.getByTestId("status-badge")).toHaveTextContent("backlog"); expect(view.client.getQueryData(["backlog", "idea", "fixture-owner"])).toBe(before); expect(defaultApiClient.patch).toHaveBeenCalledOnce();
  });
  it("does not offer a status patch when the currently selected status is chosen again", async () => {
    mount(); await ready(); fireEvent.click(screen.getByTestId("status-badge")); fireEvent.click(screen.getByRole("button", { name: "Backlog" })); expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
});

describe("Backlog owner review steering and file URL reconciliation", () => {
  it("requires operator rationale before any synthetic Send back disposition request", async () => {
    current = { ...current, status: "review_pending" }; mount(`${owner}?tab=activity`); const card = await screen.findByRole("region", { name: "Review decision" });
    fireEvent.click(within(card).getByRole("button", { name: "Send back" })); expect(await within(card).findByText("Operator identity and a rationale (or explicit agreement) are required.")).toBeVisible();
    expect(offers.filter(offer => offer.path.endsWith("/DecideAttempt"))).toHaveLength(0); expect(useBacklogDetailUIStore.getState().followUpTarget).toBeNull(); expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
  it("requires a follow-up disposition even with operator identity and steering", async () => {
    current = { ...current, status: "review_pending" }; mount(`${owner}?tab=activity`); const card = await screen.findByRole("region", { name: "Review decision" });
    fireEvent.change(within(card).getByLabelText("Operator identity"), { target: { value: "fixture-operator" } }); fireEvent.change(within(card).getByLabelText("Decision rationale"), { target: { value: "Fixture correction before continuing" } });
    fireEvent.click(within(card).getByRole("button", { name: "Send back" })); expect(await within(card).findByText("Send back requires steering text and a follow-up disposition.")).toBeVisible(); expect(offers.filter(offer => offer.path.endsWith("/DecideAttempt"))).toHaveLength(0);
  });
  it("stores exact review feedback locally only after synthetic Send back succeeds and refreshes owner reads", async () => {
    expectedTransportWrites = [steeringWrite];
    history = [{ executionId: "fixture-historical-execution", backlogKind: "idea", backlogName: "fixture-owner", status: "completed", mode: "yolo", createdAt: "2026-01-01T00:00:00Z", updatedAt: "2026-01-01T00:00:00Z", runId: "" }];
    current = { ...current, status: "review_pending" }; mount(`${owner}?tab=activity`); await screen.findByRole("region", { name: "Review decision" }); const before = countOwnerReads();
    fireEvent.click(within(fillSteering()).getByRole("button", { name: "Send back" }));
    await waitFor(() => expect(useBacklogDetailUIStore.getState().followUpContext).toBe("Address the review feedback before continuing.\n\nReview note: Fixture owner feedback needs correction\n\nDisposition: attention — Fixture owner needs steering"));
    await waitFor(() => expect(countOwnerReads()).toBeGreaterThan(before));
    const followUp = await screen.findByTestId("review-follow-up-sheet");
    expect(within(followUp).getByRole("textbox")).toHaveValue("Address the review feedback before continuing.\n\nReview note: Fixture owner feedback needs correction\n\nDisposition: attention — Fixture owner needs steering");
    expect(useBacklogDetailUIStore.getState().followUpTarget?.executionId).toBe("fixture-historical-execution");
    fireEvent.click(within(followUp).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(useBacklogDetailUIStore.getState().followUpTarget).toBeNull());
    expect(useBacklogDetailUIStore.getState().followUpContext).toBeUndefined(); expect(offers.filter(offer => offer.path.endsWith("/DecideAttempt"))).toHaveLength(1);
    expect(offers.filter(offer => offer.path.endsWith("/StartTransition"))).toHaveLength(0); expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
  it("retains operator steering and original review data when the owner refuses Send back", async () => {
    expectedTransportWrites = [steeringWrite];
    current = { ...current, status: "review_pending" }; decisionRefused = true; const view = mount(`${owner}?tab=activity`); await screen.findByRole("region", { name: "Review decision" }); const before = view.client.getQueryData(["backlog", "idea", "fixture-owner"]);
    const card = fillSteering(); fireEvent.click(within(card).getByRole("button", { name: "Send back" })); expect(await within(card).findByText(/Fixture review owner refused steering/)).toBeVisible();
    expect(within(card).getByLabelText("Decision rationale")).toHaveValue(" Fixture correction before continuing "); expect(useBacklogDetailUIStore.getState().followUpTarget).toBeNull();
    expect(view.client.getQueryData(["backlog", "idea", "fixture-owner"])).toBe(before); expect(offers.filter(offer => offer.path.endsWith("/DecideAttempt"))).toHaveLength(1); expect(offers.filter(offer => offer.path.endsWith("/StartTransition"))).toHaveLength(0);
  });
  it("repairs a missing file deep link to the canonical spec while preserving unrelated URL state", async () => {
    files = [{ name: "spec.json", path: "spec.json", type: "file", size: "25" }]; mount(`${owner}?file=missing.md&keep=fixture`); await ready();
    await waitFor(() => expect(screen.getByTestId("backlog-owner-location")).toHaveTextContent("file=spec.json")); expect(screen.getByTestId("backlog-owner-location")).toHaveTextContent("keep=fixture"); expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
  it("adds the canonical preview file when absent without changing the selected tab or offering a write", async () => {
    files = [{ name: "spec.json", path: "spec.json", type: "file", size: "25" }]; mount(`${owner}?tab=activity&keep=fixture`); await ready();
    await waitFor(() => expect(screen.getByTestId("backlog-owner-location")).toHaveTextContent("file=spec.json")); expect(screen.getByTestId("backlog-owner-location")).toHaveTextContent("tab=activity"); expect(screen.getByTestId("backlog-owner-location")).toHaveTextContent("keep=fixture"); expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
  it("retains an explicit missing file reference when no canonical fallback exists rather than guessing one", async () => {
    files = [{ name: "notes.md", path: "notes.md", type: "file", size: "15" }]; mount(`${owner}?file=missing.md&keep=fixture`); await ready();
    expect(screen.getByTestId("backlog-owner-location")).toHaveTextContent("file=missing.md&keep=fixture"); expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
});
