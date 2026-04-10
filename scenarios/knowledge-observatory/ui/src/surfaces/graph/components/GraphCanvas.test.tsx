import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphCanvas } from "./GraphCanvas";
import {
  buildCanvasRenderModel,
  toCanvasGraph,
  type CanvasViewport,
} from "../graphCanvasModel";
import type { GraphResponse } from "../../../shared/services/api";

function createModel() {
  const response: GraphResponse = {
    center: "semantic drift",
    took_ms: 12,
    nodes: [
      { id: "center", label: "semantic drift", score: 1, metadata: { type: "center" } },
      { id: "n1", label: "Node 1", score: 0.9, metadata: {} },
      { id: "n2", label: "Node 2", score: 0.7, metadata: {} },
    ],
    edges: [
      { source: "center", target: "n1", weight: 0.8, relationship: "semantic_similarity" },
      { source: "n1", target: "n2", weight: 0.6, relationship: "semantic_similarity" },
    ],
  };

  const graph = toCanvasGraph(response);
  return buildCanvasRenderModel({
    graph,
    layoutMode: "radial",
    minWeight: 0,
    maxNodes: 30,
    selectedNodeID: "n1",
    highlightNeighbors: true,
  });
}

describe("GraphCanvas", () => {
  it("renders viewport controls and selected node details", () => {
    const onViewportChange = vi.fn();
    const viewport: CanvasViewport = { scale: 1, offsetX: 0, offsetY: 0 };

    render(
      <GraphCanvas
        model={createModel()}
        viewport={viewport}
        onViewportChange={onViewportChange}
        selectedNodeID="n1"
        onSelectNode={vi.fn()}
        onFit={vi.fn()}
        onReset={vi.fn()}
        canExpand
        isExpanding={false}
        onExpand={vi.fn()}
      />
    );

    expect(screen.getByTestId("ko-graph-canvas")).toBeDefined();
    expect(screen.getByTestId("ko-graph-fit")).toBeDefined();
    expect(screen.getByTestId("ko-graph-details")).toBeDefined();
  });

  it("calls selection and control callbacks", async () => {
    const user = userEvent.setup();
    const onSelectNode = vi.fn();
    const onFit = vi.fn();
    const onReset = vi.fn();

    render(
      <GraphCanvas
        model={createModel()}
        viewport={{ scale: 1, offsetX: 0, offsetY: 0 }}
        onViewportChange={vi.fn()}
        selectedNodeID="n1"
        onSelectNode={onSelectNode}
        onFit={onFit}
        onReset={onReset}
        canExpand
        isExpanding={false}
        onExpand={vi.fn()}
      />
    );

    await user.click(screen.getByTestId("ko-graph-node-n2"));
    expect(onSelectNode).toHaveBeenCalledWith("n2");

    await user.click(screen.getByTestId("ko-graph-fit"));
    expect(onFit).toHaveBeenCalledTimes(1);

    await user.click(screen.getByTestId("ko-graph-reset-viewport"));
    expect(onReset).toHaveBeenCalledTimes(1);
  });
});
