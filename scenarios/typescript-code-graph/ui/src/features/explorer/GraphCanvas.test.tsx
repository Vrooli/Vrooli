import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { CodeGraphSchema, NodeKind, EdgeKind } from "@vrooli/proto-types/common/v1/code_graph_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { GraphCanvas } from "./GraphCanvas";
import { buildGraphLayout } from "./lib/graphAdapter";

const graph = create(CodeGraphSchema, {
  nodes: [
    { id: "ts_module:src/a.ts", kind: NodeKind.MODULE, name: "a.ts", path: "src/a.ts", attributes: { kind: "TS_NODE_KIND_MODULE" } },
    { id: "ts_module:src/b.ts", kind: NodeKind.MODULE, name: "b.ts", path: "src/b.ts", attributes: { kind: "TS_NODE_KIND_MODULE" } },
  ],
  edges: [{ id: "e", kind: EdgeKind.IMPORT, fromNodeId: "ts_module:src/a.ts", toNodeId: "ts_module:src/b.ts" }],
});
const cycleGraph = create(CodeGraphSchema, {
  nodes: graph.nodes,
  edges: [
    ...graph.edges,
    { id: "reverse", kind: EdgeKind.IMPORT, fromNodeId: "ts_module:src/b.ts", toNodeId: "ts_module:src/a.ts" },
  ],
});

describe("GraphCanvas", () => {
  afterEach(cleanup);

  it("renders the canvas root and a labelled svg", () => {
    renderWithProviders(<GraphCanvas layout={buildGraphLayout(graph)} target="m" />);
    expect(screen.getByTestId(selectors.features.explorer.canvas.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.explorer.canvas.summary)).toBeInTheDocument();
  });

  it("renders one labelled, focusable static node group per module node", () => {
    renderWithProviders(<GraphCanvas layout={buildGraphLayout(graph)} target="m" />);
    expect(
      screen.getByTestId(selectors.features.explorer.canvas.node({ id: "ts_module:src/a.ts" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.explorer.canvas.node({ id: "ts_module:src/b.ts" })),
    ).toBeInTheDocument();
    const nodeA = screen.getByTestId(selectors.features.explorer.canvas.node({ id: "ts_module:src/a.ts" }));
    expect(nodeA).toHaveAttribute("role", "group");
    expect(nodeA).toHaveAttribute("aria-label");
    expect(nodeA).toHaveAttribute("tabindex", "0");
  });

  it("marks cycle nodes and import edges with the domain danger token", () => {
    renderWithProviders(<GraphCanvas layout={buildGraphLayout(cycleGraph)} target="m" />);

    const node = screen.getByTestId(selectors.features.explorer.canvas.node({ id: "ts_module:src/a.ts" }));
    expect(node.querySelector("rect")).toHaveClass("stroke-app-danger");
    const edges = screen.getByTestId(selectors.features.explorer.canvas.root).querySelectorAll("line");
    expect(edges).toHaveLength(2);
    expect(Array.from(edges).every((edge) => edge.classList.contains("stroke-app-danger"))).toBe(true);
  });
});
