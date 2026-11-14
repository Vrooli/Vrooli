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
    { id: "package:m/a", kind: NodeKind.PACKAGE, name: "a", path: "m/a" },
    { id: "package:m/b", kind: NodeKind.PACKAGE, name: "b", path: "m/b" },
  ],
  edges: [
    { id: "e1", kind: EdgeKind.IMPORT, fromNodeId: "package:m/a", toNodeId: "package:m/b" },
    { id: "e2", kind: EdgeKind.IMPORT, fromNodeId: "package:m/b", toNodeId: "package:m/a" },
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
