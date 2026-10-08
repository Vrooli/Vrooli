import { afterEach, beforeEach, describe, it, expect, vi } from "vitest";
import { createElement } from "react";
import { fireEvent, screen, waitFor, act } from "@testing-library/react";
import { useLocation } from "react-router-dom";
import { defaultApiClient } from "../../../lib/api-client";
import { DetailActionButtons } from "../../../components/detail/DetailActionButtons";
import { renderWithProviders, createTestQueryClient } from "../../../test-utils";
import { useGraphDataStore, graphDataInitialState } from "../stores/graph-data-store";
import { useGraphUIStore, graphUIInitialState } from "../stores/graph-ui-store";
import { getActionsForNode } from "./action-registry";
import type { EntityType } from "../stores/graph-settings-store";
import type { GraphNode } from "../types";
import { makeGraphNode } from "../test-helpers";

const makeNode = (id: string, entityType: EntityType, status?: string, kind?: string): GraphNode =>
  makeGraphNode(id, entityType, { label: id, status, kind });

function expectDefined<T>(value: T | undefined, message: string): T {
  expect(value).toBeDefined();
  if (value === undefined) {
    throw new Error(message);
  }
  return value;
}

function getAction(
  lens: Parameters<typeof getActionsForNode>[0],
  entityType: Parameters<typeof getActionsForNode>[1],
  actionId: string,
) {
  return expectDefined(
    getActionsForNode(lens, entityType).find((action) => action.id === actionId),
    `Expected action "${actionId}" for ${lens}/${entityType}`,
  );
}

function runEnabledPredicate(action: ReturnType<typeof getAction>, node: GraphNode): boolean {
  const enabled = expectDefined(action.enabled, `Expected enabled predicate for action "${action.id}"`);
  return enabled(node);
}

function runNavigateTo(action: ReturnType<typeof getAction>, node: GraphNode) {
  const navigateTo = expectDefined(action.navigateTo, `Expected navigateTo for action "${action.id}"`);
  return navigateTo(node);
}

describe("getActionsForNode", () => {
  // Full graph mode: actions for capture, backlog, goal, scenario.
  it("returns capture actions for topology/capture", () => {
    const actions = getActionsForNode("topology", "capture");
    expect(actions.map((a) => a.id)).toEqual(["classify", "delete-capture"]);
  });

  it("returns backlog actions for topology/backlog", () => {
    const actions = getActionsForNode("topology", "backlog");
    expect(actions.map((a) => a.id)).toEqual([
      "edit-backlog", "queue", "add-dependency", "assign-goal", "view-files",
    ]);
  });

  it("returns goal actions for topology/goal", () => {
    const actions = getActionsForNode("topology", "goal");
    expect(actions.map((a) => a.id)).toEqual(["edit-goal", "manage-members", "archive-goal"]);
  });

  it("returns scenario actions for topology/scenario", () => {
    const actions = getActionsForNode("topology", "scenario");
    expect(actions.map((a) => a.id)).toEqual(["view-scenario-files", "edit-scenario"]);
  });

  it("returns empty array for topology/execution", () => {
    expect(getActionsForNode("topology", "execution")).toEqual([]);
  });

  it("returns empty array for topology/agent-run", () => {
    expect(getActionsForNode("topology", "agent-run")).toEqual([]);
  });
});

describe("action enabled predicates", () => {
  it("queue is enabled for ready backlog items", () => {
    const queue = getAction("topology", "backlog", "queue");
    const node = makeNode("backlog-item/execute/x", "backlog", "ready", "execute");
    expect(runEnabledPredicate(queue, node)).toBe(true);
    const queued = makeNode("backlog-item/execute/y", "backlog", "queued", "execute");
    expect(runEnabledPredicate(queue, queued)).toBe(false);
  });
});

describe("action navigateTo", () => {
  // Topology navigation.
  it("edit-backlog returns backlog DetailSelection", () => {
    const edit = getAction("topology", "backlog", "edit-backlog");
    const node = makeNode("backlog-item/execute/my-task", "backlog", "ready", "execute");
    expect(runNavigateTo(edit, node)).toEqual({ entityType: "backlog", kind: "execute", name: "my-task" });
  });

  it("edit-goal returns goal DetailSelection", () => {
    const edit = getAction("topology", "goal", "edit-goal");
    const node = makeNode("goal/my-init", "goal", "active");
    expect(runNavigateTo(edit, node)).toEqual({ entityType: "goal", name: "my-init" });
  });

  it("view-scenario-files returns scenario DetailSelection with tab", () => {
    const viewFiles = getAction("topology", "scenario", "view-scenario-files");
    const node = makeNode("scenario/my-app", "scenario", "running");
    expect(runNavigateTo(viewFiles, node)).toEqual({ entityType: "scenario", name: "my-app", tab: "files" });
  });
});

// Render the actual registry consumer against real graph state and services.
// Only API network methods (and Connect fetch for classify) are controlled.
describe("registry actions through detail controls", () => {
  const clients: ReturnType<typeof createTestQueryClient>[] = [];
  function RouteObservation() { const loc = useLocation(); return createElement("output", { "data-testid": "actual-route" }, loc.pathname + loc.search); }
  function mount(entityType: EntityType, node: GraphNode | null) {
    const client = createTestQueryClient(); clients.push(client);
    return renderWithProviders(createElement("div", {}, createElement(DetailActionButtons, { entityType, node }), createElement(RouteObservation)), { queryClient: client, initialEntries: ["/graph?keep=owner"] });
  }
  function noNetwork() { for (const method of ["get", "post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); }
  beforeEach(() => {
    useGraphDataStore.setState({ ...graphDataInitialState, lens: "topology" }); useGraphUIStore.setState({ ...graphUIInitialState });
    for (const method of ["get", "post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected registry ${method}`));
  });
  afterEach(() => { clients.splice(0).forEach(c => c.clear()); vi.restoreAllMocks(); useGraphDataStore.setState({ ...graphDataInitialState }); useGraphUIStore.setState({ ...graphUIInitialState }); });
  it.each(["queued", "in_progress", "completed"])("keeps owner nonqueueable status %s disabled with zero requests", async status => {
    mount("backlog", makeNode("backlog-item/execute/exact-task", "backlog", status, "execute")); const button = screen.getByTestId("detail-action-queue"); expect(button).toBeDisabled(); fireEvent.click(button); await act(async () => {}); noNetwork(); expect(screen.getByTestId("actual-route")).toHaveTextContent("/graph?keep=owner");
  });
  it("queues only the exact node once while pending and does not silently navigate", async () => {
    let resolve!: (value: unknown) => void; const post = vi.mocked(defaultApiClient.post).mockImplementation(() => new Promise(r => { resolve = r; }));
    mount("backlog", makeNode("backlog-item/execute/exact-task", "backlog", "ready", "execute")); const button = screen.getByTestId("detail-action-queue"); fireEvent.click(button);
    await waitFor(() => expect(post.mock.calls).toEqual([["/backlog/execute/exact-task/queue", {}]])); expect(button).toBeDisabled(); fireEvent.click(button); expect(post).toHaveBeenCalledOnce();
    expect(screen.getByTestId("actual-route")).toHaveTextContent("/graph?keep=owner"); await act(async () => resolve({})); await waitFor(() => expect(button).toBeEnabled()); expect(screen.queryByTestId("detail-action-error-queue")).toBeNull();
    expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
  it("shows the owner queue refusal inline without alternate writes or selection effects", async () => {
    vi.mocked(defaultApiClient.post).mockRejectedValue(new Error("Queue admission refused")); const node = makeNode("backlog-item/execute/exact-task", "backlog", "ready", "execute");
    useGraphUIStore.setState({ selectedNodeId: node.id }); mount("backlog", node); fireEvent.click(screen.getByTestId("detail-action-queue"));
    expect(await screen.findByTestId("detail-action-error-queue")).toHaveTextContent("Queue admission refused"); expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual([["/backlog/execute/exact-task/queue", {}]]);
    expect(useGraphUIStore.getState().selectedNodeId).toBe(node.id); expect(screen.getByTestId("actual-route")).toHaveTextContent("/graph?keep=owner"); expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
  it("refuses malformed backlog identity before transport and invalid navigation has no effects", async () => {
    mount("backlog", makeNode("not-a-node", "backlog", "ready", "execute")); fireEvent.click(screen.getByTestId("detail-action-queue"));
    expect(await screen.findByTestId("detail-action-error-queue")).toHaveTextContent("Cannot determine backlog item identity");
    fireEvent.click(screen.getByTestId("detail-action-add-dependency")); fireEvent.click(screen.getByTestId("detail-action-assign-goal")); fireEvent.click(screen.getByTestId("detail-action-view-files")); noNetwork(); expect(screen.getByTestId("actual-route")).toHaveTextContent("/graph?keep=owner");
  });
  it("does not reuse a pending queue obligation after the controls are removed", async () => {
    let resolve!: (value: unknown) => void; const post = vi.mocked(defaultApiClient.post).mockImplementation(() => new Promise(r => { resolve = r; }));
    const view = mount("backlog", makeNode("backlog-item/execute/exact-task", "backlog", "ready", "execute")); fireEvent.click(screen.getByTestId("detail-action-queue")); await waitFor(() => expect(post).toHaveBeenCalledOnce());
    view.unmount(); await act(async () => resolve({})); expect(post.mock.calls).toEqual([["/backlog/execute/exact-task/queue", {}]]); expect(useGraphUIStore.getState().selectedNodeId).toBeNull(); expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
  it.each([
    { entity: "backlog" as const, id: "backlog-item/execute/exact-task", action: "add-dependency", expected: "/backlog/execute/exact-task?tab=dependencies" },
    { entity: "backlog" as const, id: "backlog-item/execute/exact-task", action: "assign-goal", expected: "/backlog/execute/exact-task?tab=goal" },
    { entity: "backlog" as const, id: "backlog-item/execute/exact-task", action: "view-files", expected: "/backlog/execute/exact-task?tab=files" },
    { entity: "goal" as const, id: "goal/exact-goal", action: "edit-goal", expected: "/goals/exact-goal" },
    { entity: "goal" as const, id: "goal/exact-goal", action: "manage-members", expected: "/goals/exact-goal?tab=items" },
    { entity: "scenario" as const, id: "scenario/exact-scenario", action: "view-scenario-files", expected: "/scenarios/exact-scenario?tab=files" },
  ])("navigates $action to the exact declared scope without a write", async c => {
    mount(c.entity, makeNode(c.id, c.entity, "ready", "execute")); fireEvent.click(screen.getByTestId(`detail-action-${c.action}`)); await waitFor(() => expect(screen.getByTestId("actual-route")).toHaveTextContent(c.expected)); noNetwork();
  });
  it("archives only the exact goal and surfaces refusal without queue/delete fallback", async () => {
    vi.mocked(defaultApiClient.patch).mockRejectedValue(new Error("Archive owner refused")); mount("goal", makeNode("goal/exact-goal", "goal", "active")); fireEvent.click(screen.getByTestId("detail-action-archive-goal"));
    expect(await screen.findByTestId("detail-action-error-archive-goal")).toHaveTextContent("Archive owner refused"); expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/goals/exact-goal/archive-item", {}]]); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
  });
  it("deletes the exact capture with no unrelated graph store changes", async () => {
    vi.mocked(defaultApiClient.delete).mockResolvedValue({}); const node = makeNode("capture/exact-capture", "capture", "captured"); useGraphDataStore.setState({ nodes: [node] }); mount("capture", node); fireEvent.click(screen.getByTestId("detail-action-delete-capture"));
    await waitFor(() => expect(vi.mocked(defaultApiClient.delete).mock.calls).toEqual([["/captures/exact-capture"]])); expect(useGraphDataStore.getState().nodes).toEqual([node]); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
  it("refuses malformed goal and capture identifiers before transport", async () => {
    const first = mount("goal", makeNode("goal/", "goal", "active")); fireEvent.click(screen.getByTestId("detail-action-archive-goal")); expect(await screen.findByTestId("detail-action-error-archive-goal")).toHaveTextContent("Cannot determine goal name"); first.unmount();
    mount("capture", makeNode("capture/", "capture", "captured")); fireEvent.click(screen.getByTestId("detail-action-delete-capture")); expect(await screen.findByTestId("detail-action-error-delete-capture")).toHaveTextContent("Cannot determine capture identity"); noNetwork();
  });
  it("renders no undeclared actions in plan lens and null nodes have zero effects", async () => {
    useGraphDataStore.setState({ lens: "plan" }); const first = mount("backlog", makeNode("backlog-item/execute/exact-task", "backlog", "ready", "execute")); expect(screen.queryByTestId("detail-action-buttons")).toBeNull(); first.unmount();
    useGraphDataStore.setState({ lens: "topology" }); mount("backlog", null); fireEvent.click(screen.getByTestId("detail-action-queue")); fireEvent.click(screen.getByTestId("detail-action-add-dependency")); await act(async () => {}); noNetwork(); expect(screen.getByTestId("actual-route")).toHaveTextContent("/graph?keep=owner");
  });
});

describe("declared capture classification transport", () => {
  it("uses the actual declared transition subject and surfaces a subsequent owner refusal without fallback", async () => {
    const calls: Array<{ path: string; body: unknown }> = [];
    let refused = false;
    const fetch = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const request = new Request(input, init); const path = new URL(request.url).pathname; const body: unknown = JSON.parse(await request.text()); calls.push({ path, body });
      if (path.endsWith("/ListTransitions")) return new Response(JSON.stringify({ transitions: [{ key: "capture.classify", subject: "capture", kind: "TRANSITION_KIND_WORKFLOW" }] }), { status: 200, headers: { "Content-Type": "application/json" } });
      if (path.endsWith("/StartTransition")) return new Response(JSON.stringify(refused ? { code: "permission_denied", message: "Classification owner refused" } : { executionId: "transport-only-fixture" }), { status: refused ? 403 : 200, headers: { "Content-Type": "application/json" } });
      throw new Error(`Unexpected Connect fixture route ${path}`);
    });
    const client = createTestQueryClient(); useGraphDataStore.setState({ ...graphDataInitialState, lens: "topology" });
    const view = renderWithProviders(createElement(DetailActionButtons, { entityType: "capture", node: makeNode("capture/exact-capture", "capture", "captured") }), { queryClient: client });
    try {
      fireEvent.click(screen.getByTestId("detail-action-classify")); await waitFor(() => expect(calls).toHaveLength(2)); await waitFor(() => expect(screen.getByTestId("detail-action-classify")).toBeEnabled());
      const listCall = expectDefined(calls[0], "ListTransitions was not observed"); const startCall = expectDefined(calls[1], "StartTransition was not observed"); expect(listCall.path).toBe("/vrooli.swarm_manager.v1.api.TransitionService/ListTransitions"); expect(listCall.body).toEqual({});
      expect(startCall.path).toBe("/vrooli.swarm_manager.v1.api.TransitionService/StartTransition"); expect(startCall.body).toEqual({ transitionKey: "capture.classify", subjectRef: { subject: "capture", value: "exact-capture" } });
      refused = true; fireEvent.click(screen.getByTestId("detail-action-classify")); expect(await screen.findByTestId("detail-action-error-classify")).toHaveTextContent("Classification owner refused");
      expect(calls).toHaveLength(3); expect(calls[2]).toEqual(calls[1]);
    } finally { view.unmount(); client.clear(); fetch.mockRestore(); useGraphDataStore.setState({ ...graphDataInitialState }); }
  });
});
