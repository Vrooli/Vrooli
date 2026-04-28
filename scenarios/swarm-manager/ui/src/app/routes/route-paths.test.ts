import { describe, expect, it } from "vitest";
import {
  backlogDetailPath,
  captureDetailPath,
  commandPostPath,
  decisionStreamPath,
  detailPath,
  detailPathFromNodeId,
  executionDetailPath,
  graphPath,
  initiativeDetailPath,
  isGraphLens,
  routeTargetToNodeId,
  scenarioDetailPath,
} from "./route-paths";

describe("route paths", () => {
  it("builds canonical graph lens routes with query state", () => {
    expect(graphPath()).toBe("/graph");
    expect(graphPath({ lens: "operations", focus: "backlog-item/fix/a bug", select: "node/1" })).toBe(
      "/graph/operations?focus=backlog-item%2Ffix%2Fa+bug&select=node%2F1",
    );
  });

  it("builds canonical detail and command routes", () => {
    expect(backlogDetailPath("fix", "broken thing", { tab: "files" })).toBe("/backlog/fix/broken%20thing?tab=files");
    expect(scenarioDetailPath("swarm-manager")).toBe("/scenarios/swarm-manager");
    expect(executionDetailPath("exec/1")).toBe("/executions/exec%2F1");
    expect(initiativeDetailPath("route cutover")).toBe("/initiatives/route%20cutover");
    expect(captureDetailPath("cap 1")).toBe("/captures/cap%201");
    expect(commandPostPath()).toBe("/command-post");
    expect(decisionStreamPath()).toBe("/command-post/decisions");
  });

  it("converts detail targets and node IDs into canonical routes", () => {
    expect(detailPath({ entityType: "backlog", kind: "execute", name: "ship" })).toBe("/backlog/execute/ship");
    expect(detailPath({ entityType: "execution", identifier: "exec-1" })).toBe("/executions/exec-1");
    expect(detailPathFromNodeId("capture/cap-1")).toBe("/captures/cap-1");
    expect(detailPathFromNodeId("scenario/swarm-manager")).toBe("/scenarios/swarm-manager");
  });

  it("builds graph node IDs from route targets", () => {
    expect(routeTargetToNodeId({ entityType: "backlog", kind: "fix", name: "bug" })).toBe("backlog-item/fix/bug");
    expect(routeTargetToNodeId({ entityType: "initiative", name: "routing" })).toBe("initiative/routing");
    expect(routeTargetToNodeId({ entityType: "capture", identifier: "cap-1" })).toBe("capture/cap-1");
  });

  it("validates graph lenses", () => {
    expect(isGraphLens("focus")).toBe(true);
    expect(isGraphLens("topology")).toBe(true);
    expect(isGraphLens("operations")).toBe(true);
    expect(isGraphLens("bad")).toBe(false);
  });
});
