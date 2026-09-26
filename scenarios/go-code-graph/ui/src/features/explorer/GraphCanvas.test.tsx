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

describe("GraphCanvas", () => {
  afterEach(cleanup);

  it("renders the canvas root and a labelled svg", () => {
    renderWithProviders(<GraphCanvas layout={buildGraphLayout(graph)} target="m" />);
    expect(screen.getByTestId(selectors.features.explorer.canvas.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.explorer.canvas.summary)).toBeInTheDocument();
  });

  it("renders one focusable node group per package node", () => {
    renderWithProviders(<GraphCanvas layout={buildGraphLayout(graph)} target="m" />);
    expect(
      screen.getByTestId(selectors.features.explorer.canvas.node({ id: "package:m/a" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.explorer.canvas.node({ id: "package:m/b" })),
    ).toBeInTheDocument();
  });
});
