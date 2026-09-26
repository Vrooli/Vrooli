import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { selectors } from "../../consts/selectors";
import { GraphCanvas } from "./GraphCanvas";
import type { GraphLayout } from "./lib/graphAdapter";

const layout: GraphLayout = {
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
      id: "file:graph/b.go",
      label: "b.go",
      path: "graph/b.go",
      domain: "graph",
      layer: 1,
      index: 0,
      x: 220,
      y: 0,
      conflictSeverity: "critical",
    },
  ],
  edges: [{ from: "file:graph/a.go", to: "file:graph/b.go" }],
  domains: ["graph"],
};

afterEach(() => {
  cleanup();
});

describe("GraphCanvas", () => {
  it("renders one element per node and labels them for assistive tech", () => {
    renderWithProviders(<GraphCanvas layout={layout} scenario="demo" />);
    expect(
      screen.getByTestId(selectors.features.graph.canvas.node({ id: "file:graph/a.go" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.graph.canvas.node({ id: "file:graph/b.go" })),
    ).toBeInTheDocument();
    // Conflict node renders with the focusable-button shape so screen readers
    // can land on it and announce its aria-label. (Test runs in i18next cimode
    // so the label is the key, not the translated string — the interpolation
    // is exercised in the page test where i18next is initialized.)
    expect(
      screen.getByTestId(selectors.features.graph.canvas.node({ id: "file:graph/b.go" })),
    ).toHaveAttribute("role", "button");
  });

  it("renders the node-count + edge-count summary", () => {
    renderWithProviders(<GraphCanvas layout={layout} scenario="demo" />);
    const summary = screen.getByTestId(selectors.features.graph.canvas.summary);
    expect(summary).toBeInTheDocument();
  });

  it("has no axe-core violations", async () => {
    const { container } = renderWithProviders(<GraphCanvas layout={layout} scenario="demo" />);
    await expectNoA11yViolations(container);
  });
});
