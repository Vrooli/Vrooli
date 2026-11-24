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
    { id: "package:m/a", kind: NodeKind.PACKAGE, name: "a", path: "m/a" },
    { id: "package:m/b", kind: NodeKind.PACKAGE, name: "b", path: "m/b" },
  ],
  edges: [{ id: "e", kind: EdgeKind.IMPORT, fromNodeId: "package:m/a", toNodeId: "package:m/b" }],
});
const cycleGraph = create(CodeGraphSchema, {
  nodes: graph.nodes,
  edges: [
    ...graph.edges,
    { id: "reverse", kind: EdgeKind.IMPORT, fromNodeId: "package:m/b", toNodeId: "package:m/a" },
  ],
});

describe("GraphCanvas", () => {
  afterEach(cleanup);

  it("renders the canvas root and a labelled svg", () => {
    renderWithProviders(<GraphCanvas layout={buildGraphLayout(graph)} target="m" />);
    expect(screen.getByTestId(selectors.features.explorer.canvas.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.explorer.canvas.summary)).toBeInTheDocument();
  });

  it("renders one labelled, focusable static node group per package node", () => {
    renderWithProviders(<GraphCanvas layout={buildGraphLayout(graph)} target="m" />);
    expect(
      screen.getByTestId(selectors.features.explorer.canvas.node({ id: "package:m/a" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.explorer.canvas.node({ id: "package:m/b" })),
    ).toBeInTheDocument();
    const nodeA = screen.getByTestId(selectors.features.explorer.canvas.node({ id: "package:m/a" }));
    expect(nodeA).toHaveAttribute("role", "group");
    expect(nodeA).toHaveAttribute("aria-label");
    expect(nodeA).toHaveAttribute("tabindex", "0");
  });

  it("marks cycle nodes and import edges with the domain danger token", () => {
    renderWithProviders(<GraphCanvas layout={buildGraphLayout(cycleGraph)} target="m" />);

    const node = screen.getByTestId(selectors.features.explorer.canvas.node({ id: "package:m/a" }));
    expect(node.querySelector("rect")).toHaveClass("stroke-app-danger");
    const edges = screen.getByTestId(selectors.features.explorer.canvas.root).querySelectorAll("line");
    expect(edges).toHaveLength(2);
    expect(Array.from(edges).every((edge) => edge.classList.contains("stroke-app-danger"))).toBe(true);
  });
});
