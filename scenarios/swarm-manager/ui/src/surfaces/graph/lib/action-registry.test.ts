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

  // Operations lens.
  it("returns backlog actions for operations/backlog", () => {
    const actions = getActionsForNode("operations", "backlog");
    expect(actions.map((a) => a.id)).toEqual([
      "queue",
      "workshop",
      "view-files",
      "view-backlog-details",
    ]);
  });

  it("returns execution actions for operations/execution", () => {
    const actions = getActionsForNode("operations", "execution");
    expect(actions.map((a) => a.id)).toEqual([
      "view-execution-details",
      "view-prompt-trace",
      "follow-up",
      "retry",
      "trigger-review",
      "cancel",
    ]);
  });

  it("returns empty for operations/scenario (not in ops registry)", () => {
    expect(getActionsForNode("operations", "scenario")).toEqual([]);
  });

  it("returns empty for operations/agent-run (not in ops registry)", () => {
    expect(getActionsForNode("operations", "agent-run")).toEqual([]);
  });
});

describe("action enabled predicates", () => {
  it("cancel is disabled for terminal executions", () => {
    const cancel = getAction("operations", "execution", "cancel");
    const terminalNode = makeNode("execution/abc", "execution", "completed");
    expect(runEnabledPredicate(cancel, terminalNode)).toBe(false);
  });

  it("cancel is enabled for active executions", () => {
    const cancel = getAction("operations", "execution", "cancel");
    const activeNode = makeNode("execution/abc", "execution", "in_progress");
    expect(runEnabledPredicate(cancel, activeNode)).toBe(true);
  });

  it("retry is enabled only for failed executions", () => {
    const retry = getAction("operations", "execution", "retry");
    expect(runEnabledPredicate(retry, makeNode("execution/abc", "execution", "failed"))).toBe(true);
    expect(runEnabledPredicate(retry, makeNode("execution/abc", "execution", "completed"))).toBe(false);
    expect(runEnabledPredicate(retry, makeNode("execution/abc", "execution", "needs_fixup"))).toBe(false);
  });

  it("retry is disabled for active executions", () => {
    const retry = getAction("operations", "execution", "retry");
    const activeNode = makeNode("execution/abc", "execution", "running");
    expect(runEnabledPredicate(retry, activeNode)).toBe(false);
  });

  it("follow-up is enabled for terminal executions", () => {
    const followUp = getAction("operations", "execution", "follow-up");
    expect(runEnabledPredicate(followUp, makeNode("execution/abc", "execution", "completed"))).toBe(true);
    expect(runEnabledPredicate(followUp, makeNode("execution/abc", "execution", "failed"))).toBe(true);
    expect(runEnabledPredicate(followUp, makeNode("execution/abc", "execution", "needs_fixup"))).toBe(true);
  });

  it("trigger-review is enabled for completed, failed, and needs_fixup executions only", () => {
    const triggerReview = getAction("operations", "execution", "trigger-review");
    expect(runEnabledPredicate(triggerReview, makeNode("execution/abc", "execution", "completed"))).toBe(true);
    expect(runEnabledPredicate(triggerReview, makeNode("execution/abc", "execution", "failed"))).toBe(true);
    expect(runEnabledPredicate(triggerReview, makeNode("execution/abc", "execution", "needs_fixup"))).toBe(true);
    expect(runEnabledPredicate(triggerReview, makeNode("execution/abc", "execution", "canceled"))).toBe(false);
    expect(runEnabledPredicate(triggerReview, makeNode("execution/abc", "execution", "running"))).toBe(false);
  });

  it("queue is enabled for ready backlog items in operations", () => {
    const queue = getAction("operations", "backlog", "queue");
    const readyNode = makeNode("backlog-item/execute/my-task", "backlog", "ready", "execute");
    expect(runEnabledPredicate(queue, readyNode)).toBe(true);
  });
});

describe("action navigateTo", () => {
  it("view-backlog-details returns backlog DetailSelection", () => {
    const viewDetails = getAction("operations", "backlog", "view-backlog-details");
    const node = makeNode("execute/my-feature", "backlog", "ready", "execute");
    expect(runNavigateTo(viewDetails, node)).toEqual({ entityType: "backlog", kind: "execute", name: "my-feature" });
  });

  it("view-backlog-details returns backlog DetailSelection from operations", () => {
    const viewDetails = getAction("operations", "backlog", "view-backlog-details");
    const node = makeNode("backlog-item/execute/my-task", "backlog", "ready", "execute");
    expect(runNavigateTo(viewDetails, node)).toEqual({ entityType: "backlog", kind: "execute", name: "my-task" });
  });

  it("view-execution-details returns execution DetailSelection", () => {
    const viewDetails = getAction("operations", "execution", "view-execution-details");
    const node = makeNode("execution/abc-123", "execution", "completed");
    expect(runNavigateTo(viewDetails, node)).toEqual({ entityType: "execution", identifier: "abc-123" });
  });

  it("view-prompt-trace returns execution DetailSelection", () => {
    const viewTrace = getAction("operations", "execution", "view-prompt-trace");
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
