import { describe, it, expect } from "vitest";
import { getActionsForNode } from "./action-registry";
import type { EntityType } from "../stores/graph-data-store";
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
  // Topology lens: actions for capture, backlog, initiative, scenario.
  it("returns capture actions for topology/capture", () => {
    const actions = getActionsForNode("topology", "capture");
    expect(actions.map((a) => a.id)).toEqual(["classify", "create-item", "delete-capture"]);
  });

  it("returns backlog actions for topology/backlog", () => {
    const actions = getActionsForNode("topology", "backlog");
    expect(actions.map((a) => a.id)).toEqual([
      "edit-backlog", "queue", "workshop", "add-dependency", "assign-initiative", "view-files",
    ]);
  });

  it("returns initiative actions for topology/initiative", () => {
    const actions = getActionsForNode("topology", "initiative");
    expect(actions.map((a) => a.id)).toEqual(["edit-initiative", "manage-members", "archive-initiative"]);
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

  // Flow lens.
  it("returns queue and view-details for flow/backlog", () => {
    const actions = getActionsForNode("flow", "backlog");
    expect(actions.map((a) => a.id)).toEqual(["queue", "view-backlog-details"]);
  });

  it("returns execution actions for flow/execution", () => {
    const actions = getActionsForNode("flow", "execution");
    expect(actions.map((a) => a.id)).toEqual([
      "view-execution-details",
      "view-prompt-trace",
      "follow-up",
      "retry",
      "trigger-review",
      "cancel",
    ]);
  });

  it("returns empty for flow/scenario (not in flow registry)", () => {
    expect(getActionsForNode("flow", "scenario")).toEqual([]);
  });

  // Operations lens.
  it("returns scenario actions for operations/scenario", () => {
    const actions = getActionsForNode("operations", "scenario");
    expect(actions.map((a) => a.id)).toEqual([
      "start",
      "stop",
      "restart",
      "view-scenario-details",
    ]);
  });

  it("returns execution actions for operations/execution", () => {
    const actions = getActionsForNode("operations", "execution");
    expect(actions.map((a) => a.id)).toEqual([
      "view-execution-details",
      "view-prompt-trace",
      "cancel",
    ]);
  });

  it("returns stop action for operations/agent-run", () => {
    const actions = getActionsForNode("operations", "agent-run");
    expect(actions.map((a) => a.id)).toEqual(["stop-run"]);
  });
});

describe("action enabled predicates", () => {
  it("cancel is disabled for terminal executions", () => {
    const cancel = getAction("flow", "execution", "cancel");
    const terminalNode = makeNode("execution/abc", "execution", "completed");
    expect(runEnabledPredicate(cancel, terminalNode)).toBe(false);
  });

  it("cancel is enabled for active executions", () => {
    const cancel = getAction("flow", "execution", "cancel");
    const activeNode = makeNode("execution/abc", "execution", "in_progress");
    expect(runEnabledPredicate(cancel, activeNode)).toBe(true);
  });

  it("retry is enabled for terminal executions", () => {
    const retry = getAction("flow", "execution", "retry");
    const terminalNode = makeNode("execution/abc", "execution", "failed");
    expect(runEnabledPredicate(retry, terminalNode)).toBe(true);
  });

  it("retry is disabled for active executions", () => {
    const retry = getAction("flow", "execution", "retry");
    const activeNode = makeNode("execution/abc", "execution", "running");
    expect(runEnabledPredicate(retry, activeNode)).toBe(false);
  });

  it("start is enabled for stopped scenarios", () => {
    const start = getAction("operations", "scenario", "start");
    const stoppedNode = makeNode("scenario/my-scenario", "scenario", "stopped");
    expect(runEnabledPredicate(start, stoppedNode)).toBe(true);
  });

  it("start is disabled for running scenarios", () => {
    const start = getAction("operations", "scenario", "start");
    const runningNode = makeNode("scenario/my-scenario", "scenario", "running");
    expect(runEnabledPredicate(start, runningNode)).toBe(false);
  });

  it("stop is enabled for running scenarios", () => {
    const stop = getAction("operations", "scenario", "stop");
    const runningNode = makeNode("scenario/my-scenario", "scenario", "running");
    expect(runEnabledPredicate(stop, runningNode)).toBe(true);
  });

  it("stop is disabled for stopped scenarios", () => {
    const stop = getAction("operations", "scenario", "stop");
    const stoppedNode = makeNode("scenario/my-scenario", "scenario", "stopped");
    expect(runEnabledPredicate(stop, stoppedNode)).toBe(false);
  });
});

describe("action navigateTo", () => {
  it("view-backlog-details returns backlog DetailSelection", () => {
    const viewDetails = getAction("flow", "backlog", "view-backlog-details");
    const node = makeNode("execute/my-feature", "backlog", "ready", "execute");
    expect(runNavigateTo(viewDetails, node)).toEqual({ entityType: "backlog", kind: "execute", name: "my-feature" });
  });

  it("view-scenario-details returns scenario DetailSelection", () => {
    const viewDetails = getAction("operations", "scenario", "view-scenario-details");
    const node = makeNode("scenario/swarm-manager", "scenario", "running");
    expect(runNavigateTo(viewDetails, node)).toEqual({ entityType: "scenario", name: "swarm-manager" });
  });

  it("view-execution-details returns execution DetailSelection", () => {
    const viewDetails = getAction("flow", "execution", "view-execution-details");
    const node = makeNode("execution/abc-123", "execution", "completed");
    expect(runNavigateTo(viewDetails, node)).toEqual({ entityType: "execution", identifier: "abc-123" });
  });

  it("view-prompt-trace returns execution DetailSelection", () => {
    const viewTrace = getAction("flow", "execution", "view-prompt-trace");
    const node = makeNode("execution/abc-123", "execution", "completed");
    expect(runNavigateTo(viewTrace, node)).toEqual({ entityType: "execution", identifier: "abc-123" });
  });

  // Topology navigation.
  it("edit-backlog returns backlog DetailSelection", () => {
    const edit = getAction("topology", "backlog", "edit-backlog");
    const node = makeNode("backlog-item/execute/my-task", "backlog", "ready", "execute");
    expect(runNavigateTo(edit, node)).toEqual({ entityType: "backlog", kind: "execute", name: "my-task" });
  });

  it("edit-initiative returns initiative DetailSelection", () => {
    const edit = getAction("topology", "initiative", "edit-initiative");
    const node = makeNode("initiative/my-init", "initiative", "active");
    expect(runNavigateTo(edit, node)).toEqual({ entityType: "initiative", name: "my-init" });
  });

  it("view-scenario-files returns scenario DetailSelection with tab", () => {
    const viewFiles = getAction("topology", "scenario", "view-scenario-files");
    const node = makeNode("scenario/my-app", "scenario", "running");
    expect(runNavigateTo(viewFiles, node)).toEqual({ entityType: "scenario", name: "my-app", tab: "files" });
  });
});
