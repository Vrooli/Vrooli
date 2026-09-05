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

describe("GraphCanvas", () => {
  afterEach(cleanup);

  it("renders the canvas root and a labelled svg", () => {
    renderWithProviders(<GraphCanvas layout={buildGraphLayout(graph)} target="m" />);
    expect(screen.getByTestId(selectors.features.explorer.canvas.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.explorer.canvas.summary)).toBeInTheDocument();
  });

  it("renders one focusable node group per module node", () => {
    renderWithProviders(<GraphCanvas layout={buildGraphLayout(graph)} target="m" />);
    expect(
      screen.getByTestId(selectors.features.explorer.canvas.node({ id: "ts_module:src/a.ts" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.explorer.canvas.node({ id: "ts_module:src/b.ts" })),
    ).toBeInTheDocument();
  });
});
