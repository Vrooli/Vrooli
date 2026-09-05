import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { selectors } from "../../consts/selectors";
import { GraphAccessibleList } from "./GraphAccessibleList";
import type { GraphLayout } from "./lib/graphAdapter";

const baseLayout: GraphLayout = {
  nodes: [
    {
      id: "file:graph/a.go",
      label: "a.go",
      path: "graph/a.go",
      domain: "graph",
      layer: 0,
      index: 0,
      x: 0,
      y: 0,
    },
    {
      id: "file:conflicts/c.go",
      label: "c.go",
      path: "conflicts/c.go",
      domain: "conflicts",
      layer: 1,
      index: 0,
      x: 220,
      y: 0,
      conflictSeverity: "high",
    },
  ],
  edges: [],
  domains: ["conflicts", "graph"],
};

afterEach(() => {
  cleanup();
});

describe("GraphAccessibleList", () => {
  it("renders one item per node with its path", () => {
    renderWithProviders(<GraphAccessibleList layout={baseLayout} />);
    expect(
      screen.getByTestId(
        selectors.features.graph.accessibleList.item({ id: "file:graph/a.go" }),
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(
        selectors.features.graph.accessibleList.item({ id: "file:conflicts/c.go" }),
      ),
    ).toBeInTheDocument();
  });

  it("renders an empty-state when the layout has no nodes", () => {
    renderWithProviders(
      <GraphAccessibleList
        layout={{ nodes: [], edges: [], domains: [] }}
      />,
    );
    expect(
      screen.getByTestId(selectors.features.graph.accessibleList.empty),
    ).toBeInTheDocument();
  });

  it("has no axe-core violations", async () => {
    const { container } = renderWithProviders(<GraphAccessibleList layout={baseLayout} />);
    await expectNoA11yViolations(container);
  });
});
