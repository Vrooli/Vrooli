import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { DecisionDrawer } from "./DecisionDrawer";
import { defaultApiClient } from "../../../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../../../test-utils";
import { useBacklogStore } from "../../../stores";
import { useSnoozeStore } from "../../../stores/snooze-store";
import type { NextActionFeedEntry } from "../../../services/next-action-service";

// Real drawer/cards, stores, services and query/router/provider composition.
// Only network methods are spied; unexpected reads are independently rejected.
const clients: QueryClient[] = [];
const unexpectedReads: string[] = [];
function entry(name: string, id = "accept_plan"): NextActionFeedEntry {
  return { entity_kind: "backlog_item", entity_ref: `idea/${name}`, entity_title: `Item ${name}`, tier: 1, action: { id, compact_label: id === "author_followup" ? "Author follow-up" : "Accept plan", expanded_label: `Review ${name}`, enabled: true, reason: "Owner decision required", effect: "state_change" } };
}
function network(entries: NextActionFeedEntry[]) {
  return vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/next-actions/feed") return { entries };
    if (path === "/backlog/summary") return { pending_questions: { items: [] } };
    if (path === "/proposal-sessions" || path.startsWith("/proposal-sessions?target_type=backlog_item&target_ref=")) return { sessions: [] };
    unexpectedReads.push(path); throw new Error(`Unexpected fixture read ${path}`);
  });
}
function Location() { const location = useLocation(); return <output data-testid="drawer-location">{location.pathname + location.search}</output>; }
function mount(entries = [entry("first"), entry("second")], options: { scope?: string | null; position?: string; open?: boolean } = {}) {
  const get = network(entries); const client = createTestQueryClient(); clients.push(client);
  const completed = vi.fn(); const close = vi.fn();
  const view = renderWithProviders(<><DecisionDrawer isOpen={options.open ?? true} onClose={close} scopeItemKey={options.scope ?? null} currentQuestionId={null} onCurrentQuestionChange={vi.fn()} onCompleted={completed} /><Location /></>, { queryClient: client, initialEntries: [`/plan?decisionPosition=${options.position ?? "0"}`] });
  return { ...view, client, get, completed, close };
}
beforeEach(() => {
  unexpectedReads.length = 0; useBacklogStore.getState().setItems([]); useSnoozeStore.setState({ entries: new Map() }); localStorage.removeItem("swarm-manager.snooze.v1");
  vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new Error("Unexpected fixture patch"));
  vi.spyOn(defaultApiClient, "delete").mockRejectedValue(new Error("Unexpected fixture delete"));
});
afterEach(() => { expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled(); clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); useBacklogStore.getState().setItems([]); useSnoozeStore.setState({ entries: new Map() }); localStorage.removeItem("swarm-manager.snooze.v1"); expect(unexpectedReads).toEqual([]); });

async function loaded() { return screen.findByTestId("decision-queue-counter"); }
function writesAbsent(post: unknown) { expect(post).not.toHaveBeenCalled(); expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled(); }

describe("DecisionDrawer actual queue and recovery workflows", () => {
  it("does not acquire a feed or mutate state while closed", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    const { get, completed } = mount(undefined, { open: false }); await act(async () => {});
    expect(get).not.toHaveBeenCalled(); expect(screen.queryByTestId("plan-decision-drawer")).toBeNull(); writesAbsent(post); expect(completed).not.toHaveBeenCalled();
  });
  it("moves through queue positions and persists the deep link without submitting decisions", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    const { completed } = mount(); expect(await loaded()).toHaveTextContent("1 of 2");
    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Skip" }));
    expect(screen.getByTestId("decision-queue-title")).toHaveTextContent("Item second"); expect(screen.getByTestId("drawer-location")).toHaveTextContent("decisionPosition=1");
    expect(screen.getByRole("button", { name: "Skip" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Previous" })); expect(screen.getByTestId("drawer-location")).toHaveTextContent("decisionPosition=0"); writesAbsent(post); expect(completed).not.toHaveBeenCalled();
  });
  it.each(["-20", "999", "not-a-number"])("bounds an untrusted URL queue position %s", async position => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    mount(undefined, { position }); expect(await loaded()).toHaveTextContent(position === "999" ? "2 of 2" : "1 of 2"); writesAbsent(post);
  });
  it("filters by title, reference and action, preserves original queue indices, and reports no matches", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    mount([entry("first"), { ...entry("second"), entity_title: "Special title", action: { ...entry("second").action, expanded_label: "Unique outcome" } }]);
    fireEvent.click(await loaded()); const filter = screen.getByRole("textbox", { name: "Filter decisions" });
    for (const value of [" SPECIAL ", "idea/second", "unique OUTCOME"]) {
      fireEvent.change(filter, { target: { value } }); expect(screen.getAllByTestId("decision-queue-navigator-row")).toHaveLength(1); expect(screen.getByTestId("decision-queue-navigator-row")).toHaveTextContent("Special title");
    }
    fireEvent.change(filter, { target: { value: "missing" } }); expect(screen.getByText("No decisions match that filter.")).toBeVisible(); expect(screen.queryByTestId("decision-queue-navigator-row")).toBeNull();
    fireEvent.change(filter, { target: { value: "second" } }); fireEvent.click(screen.getByTestId("decision-queue-navigator-row"));
    expect(screen.queryByTestId("decision-queue-navigator")).toBeNull(); expect(screen.getByTestId("decision-queue-counter")).toHaveTextContent("2 of 2"); expect(screen.getByTestId("drawer-location")).toHaveTextContent("decisionPosition=1"); writesAbsent(post);
  });
  it("limits a large jump list and lets filtering reach entries beyond its first hundred", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    mount(Array.from({ length: 101 }, (_, i) => entry(`item-${i}`))); fireEvent.click(await loaded());
    expect(screen.getAllByTestId("decision-queue-navigator-row")).toHaveLength(100); expect(screen.getByText(/Showing 100 of 101 matches/)).toBeVisible();
    fireEvent.change(screen.getByRole("textbox", { name: "Filter decisions" }), { target: { value: "idea/item-100" } }); fireEvent.click(screen.getByTestId("decision-queue-navigator-row"));
    expect(screen.getByTestId("decision-queue-counter")).toHaveTextContent("101 of 101"); writesAbsent(post);
  });
  it("scopes to direct or chained exact item references and leaves unrelated decisions out", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    const chained = { ...entry("goal-action"), entity_kind: "goal" as const, entity_ref: "goal-a", chained_ref: "idea/first" };
    const { get } = mount([entry("first"), entry("second"), chained], { scope: "idea/first" }); expect(await loaded()).toHaveTextContent("1 of 2");
    fireEvent.click(screen.getByRole("button", { name: "Skip" })); expect(screen.getByTestId("decision-queue-title")).toHaveTextContent("Item goal-action");
    expect(get.mock.calls.map(([path]) => path)).toContain("/proposal-sessions?target_type=backlog_item&target_ref=idea%2Ffirst"); writesAbsent(post);
  });
  it("snoozes only the current entity for one hour without a mutation request", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write"));
    mount(); await loaded(); const before = Date.now(); fireEvent.click(screen.getByRole("button", { name: "Snooze item" })); const after = Date.now();
    const snoozed = useSnoozeStore.getState().entries.get("backlog_item:idea/first"); expect(snoozed?.expiresAt).toBeGreaterThanOrEqual(before + 3_600_000); expect(snoozed?.expiresAt).toBeLessThanOrEqual(after + 3_600_000); expect(useSnoozeStore.getState().entries.size).toBe(1); expect(localStorage.getItem("swarm-manager.snooze.v1")).toContain("backlog_item:idea/first"); writesAbsent(post);
  });
  it("accepts only the selected owner plan and refreshes the decision feed after success", async () => {
    let resolve!: (value: unknown) => void; const post = vi.spyOn(defaultApiClient, "post").mockImplementation(() => new Promise(r => { resolve = r; }));
    const { completed, get } = mount(); await loaded(); fireEvent.click(screen.getByTestId("next-action-primary"));
    await waitFor(() => expect(post.mock.calls).toEqual([["/backlog/idea/first/plan-accept", {}]])); expect(completed).not.toHaveBeenCalled(); expect(screen.getByTestId("next-action-primary")).toBeDisabled();
    fireEvent.click(screen.getByTestId("next-action-primary")); expect(post.mock.calls).toHaveLength(1); await act(async () => resolve({}));
    await waitFor(() => expect(completed).toHaveBeenCalledTimes(1)); await waitFor(() => expect(get.mock.calls.filter(([path]) => path === "/next-actions/feed").length).toBeGreaterThan(1));
  });
  it("keeps a refused plan decision inline and does not complete, navigate or use another write channel", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Owner revision refused")); const { completed } = mount(); await loaded(); fireEvent.click(screen.getByTestId("next-action-primary"));
    expect(await screen.findByRole("alert")).toHaveTextContent("Owner revision refused"); expect(completed).not.toHaveBeenCalled(); expect(screen.getByTestId("drawer-location")).toHaveTextContent("/plan?decisionPosition=0"); expect(post.mock.calls).toEqual([["/backlog/idea/first/plan-accept", {}]]); expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
  it("validates steering and both child fields before submitting exact recovery direction", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockResolvedValue({}); const { completed } = mount([entry("first", "author_followup")]); await loaded();
    const save = screen.getByRole("button", { name: "Save follow-up" }); expect(save).toBeDisabled();
    fireEvent.change(screen.getByPlaceholderText("Describe the work needed to recover this item…"), { target: { value: "  " } }); expect(save).toBeDisabled();
    fireEvent.change(screen.getByPlaceholderText("Describe the work needed to recover this item…"), { target: { value: "Recover exact evidence" } }); fireEvent.change(screen.getByRole("combobox", { name: "Recovery disposition" }), { target: { value: "new_items" } }); expect(save).toBeDisabled();
    fireEvent.change(screen.getByRole("textbox", { name: "Machine name" }), { target: { value: " child-evidence " } }); expect(save).toBeDisabled(); writesAbsent(post);
    fireEvent.change(screen.getByRole("textbox", { name: "Title" }), { target: { value: " Restore evidence " } }); fireEvent.change(screen.getByRole("combobox", { name: "Kind" }), { target: { value: "research" } }); fireEvent.click(save);
    await waitFor(() => expect(post.mock.calls).toEqual([["/backlog/idea/first/follow-up/author", { follow_up: { steering: "Recover exact evidence", disposition: "new_items", items: [{ kind: "research", name: "child-evidence", title: "Restore evidence" }] } }]])); await waitFor(() => expect(completed).toHaveBeenCalledTimes(1));
  });
  it("retains authored recovery input after an owner refusal without claiming completion", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Recovery binding refused")); const { completed } = mount([entry("first", "author_followup")]); await loaded();
    const steering = screen.getByPlaceholderText("Describe the work needed to recover this item…"); fireEvent.change(steering, { target: { value: "Reconcile retained receipt" } }); fireEvent.click(screen.getByRole("button", { name: "Save follow-up" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Recovery binding refused"); expect(steering).toHaveValue("Reconcile retained receipt"); expect(completed).not.toHaveBeenCalled(); expect(post.mock.calls).toEqual([["/backlog/idea/first/follow-up/author", { follow_up: { steering: "Reconcile retained receipt", disposition: "replan" } }]]);
  });
  it("saves follow-up-run direction once while pending and completes only after owner success", async () => {
    let resolve!: (v: unknown) => void; const post = vi.spyOn(defaultApiClient, "post").mockImplementation(() => new Promise(r => { resolve = r; }));
    const { completed, client } = mount([entry("first", "author_followup")]); await loaded();
    const before = client.getQueryData(["next-actions-feed"]);
    fireEvent.change(screen.getByPlaceholderText("Describe the work needed to recover this item…"), { target: { value: "Retain exact retry boundary" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Recovery disposition" }), { target: { value: "follow_up_run" } });
    fireEvent.click(screen.getByRole("button", { name: "Save follow-up" }));
    await waitFor(() => expect(post.mock.calls).toEqual([["/backlog/idea/first/follow-up/author", { follow_up: { steering: "Retain exact retry boundary", disposition: "follow_up_run" } }]]));
    expect(screen.getByRole("button", { name: "Saving…" })).toBeDisabled(); fireEvent.click(screen.getByRole("button", { name: "Saving…" }));
    expect(post).toHaveBeenCalledTimes(1); expect(completed).not.toHaveBeenCalled(); expect(client.getQueryData(["next-actions-feed"])).toBe(before);
    await act(async () => resolve({})); await waitFor(() => expect(completed).toHaveBeenCalledTimes(1));
  });
  it("retains refused child kind, trimmed payload and untrimmed editable fields without a second write", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Child obligation refused")); const { completed } = mount([entry("first", "author_followup")]); await loaded();
    fireEvent.change(screen.getByPlaceholderText("Describe the work needed to recover this item…"), { target: { value: "Investigate evidence" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Recovery disposition" }), { target: { value: "new_items" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Kind" }), { target: { value: "fix" } });
    fireEvent.change(screen.getByRole("textbox", { name: "Machine name" }), { target: { value: " child-one " } });
    fireEvent.change(screen.getByRole("textbox", { name: "Title" }), { target: { value: " Repair receipt " } }); fireEvent.click(screen.getByRole("button", { name: "Save follow-up" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Child obligation refused"); expect(completed).not.toHaveBeenCalled();
    expect(screen.getByRole("combobox", { name: "Kind" })).toHaveValue("fix"); expect(screen.getByRole("textbox", { name: "Machine name" })).toHaveValue(" child-one "); expect(screen.getByRole("textbox", { name: "Title" })).toHaveValue(" Repair receipt ");
    expect(post.mock.calls).toEqual([["/backlog/idea/first/follow-up/author", { follow_up: { steering: "Investigate evidence", disposition: "new_items", items: [{ kind: "fix", name: "child-one", title: "Repair receipt" }] } }]]);
  });
  it("opens an authored follow-up target without saving its incomplete form or claiming completion", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write")); const { completed, close } = mount([entry("first", "author_followup")]); await loaded();
    fireEvent.change(screen.getByPlaceholderText("Describe the work needed to recover this item…"), { target: { value: "Unsaved steering" } }); fireEvent.click(screen.getByRole("button", { name: "Open" }));
    expect(screen.getByTestId("drawer-location")).toHaveTextContent("/backlog/idea/first"); expect(close).not.toHaveBeenCalled(); expect(completed).not.toHaveBeenCalled(); writesAbsent(post);
  });
  it("performs retry for exactly the current item and completes only after the owner response", async () => {
    let resolve!: (v: unknown) => void; const post = vi.spyOn(defaultApiClient, "post").mockImplementation(() => new Promise(r => { resolve = r; }));
    const { completed } = mount([{ ...entry("first", "retry"), action: { ...entry("first", "retry").action, compact_label: "Retry" } }]); await loaded(); fireEvent.click(screen.getByTestId("next-action-primary"));
    await waitFor(() => expect(post.mock.calls).toEqual([["/backlog/idea/first/retry", { note: "Retried from decision stream" }]]));
    expect(completed).not.toHaveBeenCalled(); expect(screen.getByTestId("next-action-primary")).toBeDisabled(); await act(async () => resolve({ new_execution_id: "retry-receipt", parent_execution_id: "old-receipt", status: "queued" })); await waitFor(() => expect(completed).toHaveBeenCalledTimes(1));
  });
  it("preserves retry refusal without invalidating feed or reporting completion", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Retry authority expired")); const { completed, client, get } = mount([entry("first", "retry")]); await loaded();
    const before = client.getQueryData(["next-actions-feed"]); const readsBefore = get.mock.calls.filter(([p]) => p === "/next-actions/feed").length; fireEvent.click(screen.getByTestId("next-action-primary"));
    expect(await screen.findByRole("alert")).toHaveTextContent("Retry authority expired"); expect(completed).not.toHaveBeenCalled(); expect(client.getQueryData(["next-actions-feed"])).toBe(before);
    expect(get.mock.calls.filter(([p]) => p === "/next-actions/feed")).toHaveLength(readsBefore); expect(post.mock.calls).toEqual([["/backlog/idea/first/retry", { note: "Retried from decision stream" }]]);
  });
  it("requires destructive confirmation and allows cancellation without changing the feed", async () => {
    const post = vi.spyOn(defaultApiClient, "post").mockRejectedValue(new Error("Unexpected fixture write")); const { completed, client } = mount([{ ...entry("first", "archive"), action: { ...entry("first", "archive").action, compact_label: "Archive" } }]); await loaded();
    const before = client.getQueryData(["next-actions-feed"]); fireEvent.click(screen.getByTestId("next-action-primary")); const dialog = await screen.findByRole("alertdialog");
    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" })); await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
    expect(completed).not.toHaveBeenCalled(); expect(client.getQueryData(["next-actions-feed"])).toBe(before); writesAbsent(post);
  });
});
