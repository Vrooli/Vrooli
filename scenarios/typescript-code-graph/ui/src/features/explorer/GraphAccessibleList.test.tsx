import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { CodeGraphSchema, NodeKind, EdgeKind } from "@vrooli/proto-types/common/v1/code_graph_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { GraphAccessibleList } from "./GraphAccessibleList";
import { buildGraphLayout } from "./lib/graphAdapter";

const cyclicGraph = create(CodeGraphSchema, {
  nodes: [
    { id: "ts_module:src/a.ts", kind: NodeKind.MODULE, name: "a.ts", path: "src/a.ts", attributes: { kind: "TS_NODE_KIND_MODULE" } },
    { id: "ts_module:src/b.ts", kind: NodeKind.MODULE, name: "b.ts", path: "src/b.ts", attributes: { kind: "TS_NODE_KIND_MODULE" } },
  ],
  edges: [
    { id: "e1", kind: EdgeKind.IMPORT, fromNodeId: "ts_module:src/a.ts", toNodeId: "ts_module:src/b.ts" },
    { id: "e2", kind: EdgeKind.IMPORT, fromNodeId: "ts_module:src/b.ts", toNodeId: "ts_module:src/a.ts" },
  ],
});

describe("GraphAccessibleList", () => {
  afterEach(cleanup);

  it("renders one list row per layout node (text-alternative parity)", () => {
    const layout = buildGraphLayout(cyclicGraph);
    renderWithProviders(<GraphAccessibleList layout={layout} importLabels={new Map()} />);
    for (const node of layout.nodes) {
      expect(
        screen.getByTestId(selectors.features.explorer.accessibleList.item({ id: node.id })),
      ).toBeInTheDocument();
    }
  });

  it("marks cycle members with a severity badge (label, not color-only)", () => {
    const layout = buildGraphLayout(cyclicGraph);
    renderWithProviders(<GraphAccessibleList layout={layout} importLabels={new Map()} />);
    // Both packages are in the cycle → two high-severity badges.
    expect(screen.getAllByTestId(selectors.shared.severityBadge.root({ level: "high" }))).toHaveLength(2);
  });

  it("shows the empty state when there are no nodes", () => {
    renderWithProviders(
      <GraphAccessibleList layout={buildGraphLayout(undefined)} importLabels={new Map()} />,
    );
    expect(screen.getByTestId(selectors.features.explorer.accessibleList.empty)).toBeInTheDocument();
  });
});
