import { describe, it, expect } from "vitest";
import type { Node } from "@xyflow/react";
import { getActionsForNode, actionRegistry } from "./action-registry";
import type { GraphLens, EntityType } from "../stores/graph-data-store";

const makeNode = (id: string, entityType: EntityType, status?: string, kind?: string): Node => ({
  id,
  type: entityType,
  position: { x: 0, y: 0 },
  data: { label: id, entityType, status, kind },
});

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
    const actions = getActionsForNode("flow", "execution");
    const cancel = actions.find((a) => a.id === "cancel")!;
    const terminalNode = makeNode("execution/abc", "execution", "completed");
    expect(cancel.enabled!(terminalNode)).toBe(false);
  });

  it("cancel is enabled for active executions", () => {
    const actions = getActionsForNode("flow", "execution");
    const cancel = actions.find((a) => a.id === "cancel")!;
    const activeNode = makeNode("execution/abc", "execution", "in_progress");
    expect(cancel.enabled!(activeNode)).toBe(true);
  });

  it("retry is enabled for terminal executions", () => {
    const actions = getActionsForNode("flow", "execution");
    const retry = actions.find((a) => a.id === "retry")!;
    const terminalNode = makeNode("execution/abc", "execution", "failed");
    expect(retry.enabled!(terminalNode)).toBe(true);
  });

  it("retry is disabled for active executions", () => {
    const actions = getActionsForNode("flow", "execution");
    const retry = actions.find((a) => a.id === "retry")!;
    const activeNode = makeNode("execution/abc", "execution", "running");
    expect(retry.enabled!(activeNode)).toBe(false);
  });

  it("start is enabled for stopped scenarios", () => {
    const actions = getActionsForNode("operations", "scenario");
    const start = actions.find((a) => a.id === "start")!;
    const stoppedNode = makeNode("scenario/my-scenario", "scenario", "stopped");
    expect(start.enabled!(stoppedNode)).toBe(true);
  });

  it("start is disabled for running scenarios", () => {
    const actions = getActionsForNode("operations", "scenario");
    const start = actions.find((a) => a.id === "start")!;
    const runningNode = makeNode("scenario/my-scenario", "scenario", "running");
    expect(start.enabled!(runningNode)).toBe(false);
  });

  it("stop is enabled for running scenarios", () => {
    const actions = getActionsForNode("operations", "scenario");
    const stop = actions.find((a) => a.id === "stop")!;
    const runningNode = makeNode("scenario/my-scenario", "scenario", "running");
    expect(stop.enabled!(runningNode)).toBe(true);
  });

  it("stop is disabled for stopped scenarios", () => {
    const actions = getActionsForNode("operations", "scenario");
    const stop = actions.find((a) => a.id === "stop")!;
    const stoppedNode = makeNode("scenario/my-scenario", "scenario", "stopped");
    expect(stop.enabled!(stoppedNode)).toBe(false);
  });
});

describe("action navigateTo", () => {
  it("view-backlog-details builds correct path", () => {
    const actions = getActionsForNode("flow", "backlog");
    const viewDetails = actions.find((a) => a.id === "view-backlog-details")!;
    const node = makeNode("execute/my-feature", "backlog", "ready", "execute");
    expect(viewDetails.navigateTo!(node)).toBe("/details/backlog/execute/my-feature");
  });

  it("view-scenario-details builds correct path", () => {
    const actions = getActionsForNode("operations", "scenario");
    const viewDetails = actions.find((a) => a.id === "view-scenario-details")!;
    const node = makeNode("scenario/swarm-manager", "scenario", "running");
    expect(viewDetails.navigateTo!(node)).toBe("/details/scenario/swarm-manager");
  });

  it("view-execution-details builds correct path", () => {
    const actions = getActionsForNode("flow", "execution");
    const viewDetails = actions.find((a) => a.id === "view-execution-details")!;
    const node = makeNode("execution/abc-123", "execution", "completed");
    expect(viewDetails.navigateTo!(node)).toBe("/details/execution/abc-123");
  });

  it("view-prompt-trace builds correct path", () => {
    const actions = getActionsForNode("flow", "execution");
    const viewTrace = actions.find((a) => a.id === "view-prompt-trace")!;
    const node = makeNode("execution/abc-123", "execution", "completed");
    expect(viewTrace.navigateTo!(node)).toBe("/details/execution/abc-123/prompt-trace");
  });

  // Topology navigation.
  it("edit-backlog navigates to backlog detail page", () => {
    const actions = getActionsForNode("topology", "backlog");
    const edit = actions.find((a) => a.id === "edit-backlog")!;
    const node = makeNode("backlog-item/execute/my-task", "backlog", "ready", "execute");
    expect(edit.navigateTo!(node)).toBe("/details/backlog/execute/my-task");
  });

  it("edit-initiative navigates to initiative detail page", () => {
    const actions = getActionsForNode("topology", "initiative");
    const edit = actions.find((a) => a.id === "edit-initiative")!;
    const node = makeNode("initiative/my-init", "initiative", "active");
    expect(edit.navigateTo!(node)).toBe("/details/initiative/my-init");
  });

  it("view-scenario-files navigates to scenario files tab", () => {
    const actions = getActionsForNode("topology", "scenario");
    const viewFiles = actions.find((a) => a.id === "view-scenario-files")!;
    const node = makeNode("scenario/my-app", "scenario", "running");
    expect(viewFiles.navigateTo!(node)).toBe("/details/scenario/my-app?tab=files");
  });
});
