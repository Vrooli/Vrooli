import { describe, expect, it } from "vitest";
import { computeVisualFocus, clearVisualFocus } from "./visual-focus";
import { makeBacklogNode, makeCaptureNode, makeGraphEdge, makeScenarioNode } from "../test-helpers";

describe("computeVisualFocus", () => {
  const nodeA = makeBacklogNode("backlog-item/execute/task-a", { label: "A", status: "queued" });
  const nodeB = makeCaptureNode("capture/c-1", { label: "C1", status: "classified" });
  const nodeC = makeScenarioNode("scenario/app", { label: "App", status: "running" });
  const nodes = [nodeA, nodeB, nodeC];
  const edges = [
    makeGraphEdge("e1", nodeA.id, nodeB.id, "classified_as"),
  ];

  it("returns selectedNodeId and dim highlight for an existing node", () => {
    const result = computeVisualFocus(nodeA.id, nodes, edges);
    expect(result).not.toBeNull();
    expect(result!.selectedNodeId).toBe(nodeA.id);
    expect(result!.highlightState.mode).toBe("dim");
    // BFS neighborhood should include the start node and its neighbor
    expect(result!.highlightState.highlighted).toContain(nodeA.id);
    expect(result!.highlightState.highlighted).toContain(nodeB.id);
    // Non-neighbor should not be highlighted
    expect(result!.highlightState.highlighted).not.toContain(nodeC.id);
  });

  it("returns null when the node does not exist in the graph", () => {
    const result = computeVisualFocus("nonexistent/node", nodes, edges);
    expect(result).toBeNull();
  });

  it("includes only the start node when it has no edges", () => {
    const result = computeVisualFocus(nodeC.id, nodes, edges);
    expect(result).not.toBeNull();
    expect(result!.highlightState.highlighted.size).toBe(1);
    expect(result!.highlightState.highlighted).toContain(nodeC.id);
  });
});

describe("clearVisualFocus", () => {
  it("returns null selection and normal highlight mode", () => {
    const result = clearVisualFocus();
    expect(result.selectedNodeId).toBeNull();
    expect(result.highlightState.mode).toBe("normal");
    expect(result.highlightState.highlighted.size).toBe(0);
  });
});
