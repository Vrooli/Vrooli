import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import type { ScenarioSummary } from "../../api/scenarios";
import { ScenarioTable } from "./ScenarioTable";

const alpha: ScenarioSummary = {
  id: "alpha",
  displayName: "Alpha",
  description: "First scenario",
  path: "/repo/scenarios/alpha",
  flowCount: 2,
};

describe("ScenarioTable accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <ScenarioTable
        scenarios={[alpha]}
        selectedIds={new Set()}
        onToggleOne={() => {}}
        onToggleAll={() => {}}
      />,
    );
    await expectNoA11yViolations(container);
  });
});
