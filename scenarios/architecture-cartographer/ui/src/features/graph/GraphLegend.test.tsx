import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { selectors } from "../../consts/selectors";
import { GraphLegend } from "./GraphLegend";

afterEach(() => {
  cleanup();
});

describe("GraphLegend", () => {
  it("renders one row per severity level + a no-conflict row", () => {
    renderWithProviders(<GraphLegend />);
    expect(screen.getByTestId(selectors.features.graph.legend.root)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.graph.legend.noConflict),
    ).toBeInTheDocument();
    for (const level of ["info", "low", "medium", "high", "critical"] as const) {
      expect(
        screen.getByTestId(selectors.features.graph.legend.severity({ level })),
      ).toBeInTheDocument();
    }
  });

  it("has no axe-core violations", async () => {
    const { container } = renderWithProviders(<GraphLegend />);
    await expectNoA11yViolations(container);
  });
});
