import { describe, it, expect } from "vitest";
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
