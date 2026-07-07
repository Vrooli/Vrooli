import { describe, expect, it } from "vitest";
import {
  backlogDetailPath,
  captureDetailPath,
  detailPath,
  detailPathFromNodeId,
  executionDetailPath,
  graphPath,
  initiativeDetailPath,
  isGraphLens,
  routeTargetToNodeId,
  scenarioDetailPath,
  sessionDetailPath,
} from "./route-paths";

describe("route paths", () => {
  it("builds canonical graph lens routes with query state", () => {
    expect(graphPath()).toBe("/graph");
    // Plan is a first-class top-level route, not a /graph lens.
    expect(graphPath({ lens: "plan" })).toBe("/plan");
    expect(graphPath({ lens: "plan", focus: "backlog-item/fix/x" })).toBe(
      "/plan?focus=backlog-item%2Ffix%2Fx",
    );
    expect(graphPath({ lens: "focus", focus: "backlog-item/fix/a bug", select: "node/1" })).toBe(
      "/graph?mode=focus&focus=backlog-item%2Ffix%2Fa+bug&select=node%2F1",
    );
    expect(graphPath({ lens: "graph" })).toBe("/graph");
  });

  it("builds canonical detail and command routes", () => {
    expect(backlogDetailPath("fix", "broken thing", { tab: "files" })).toBe("/backlog/fix/broken%20thing?tab=files");
    expect(scenarioDetailPath("swarm-manager")).toBe("/scenarios/swarm-manager");
    expect(executionDetailPath("exec/1")).toBe("/executions/exec%2F1");
    expect(initiativeDetailPath("route cutover")).toBe("/initiatives/route%20cutover");
    expect(captureDetailPath("cap 1")).toBe("/captures/cap%201");
    expect(sessionDetailPath("sess 1")).toBe("/sessions/sess%201");
    expect(sessionDetailPath("sess-1", { tab: "proposals" })).toBe("/sessions/sess-1?tab=proposals");
  });

  it("converts detail targets and node IDs into canonical routes", () => {
    expect(detailPath({ entityType: "backlog", kind: "execute", name: "ship" })).toBe("/backlog/execute/ship");
    expect(detailPath({ entityType: "execution", identifier: "exec-1" })).toBe("/executions/exec-1");
    expect(detailPath({ entityType: "session", identifier: "sess-1" })).toBe("/sessions/sess-1");
    expect(routeTargetToNodeId({ entityType: "session", identifier: "sess-1" })).toBeNull();
    expect(detailPathFromNodeId("capture/cap-1")).toBe("/captures/cap-1");
    expect(detailPathFromNodeId("scenario/swarm-manager")).toBe("/scenarios/swarm-manager");
  });

  it("builds graph node IDs from route targets", () => {
    expect(routeTargetToNodeId({ entityType: "backlog", kind: "fix", name: "bug" })).toBe("backlog-item/fix/bug");
    expect(routeTargetToNodeId({ entityType: "initiative", name: "routing" })).toBe("initiative/routing");
    expect(routeTargetToNodeId({ entityType: "capture", identifier: "cap-1" })).toBe("capture/cap-1");
  });

  it("validates graph lenses", () => {
    expect(isGraphLens("plan")).toBe(true);
    expect(isGraphLens("graph")).toBe(true);
    expect(isGraphLens("focus")).toBe(false);
    expect(isGraphLens("topology")).toBe(false);
    expect(isGraphLens("operations")).toBe(false);
    expect(isGraphLens("bad")).toBe(false);
  });
});
