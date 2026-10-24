import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { ScenarioFilters } from "./ScenarioFilters";

describe("ScenarioFilters accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <ScenarioFilters
        value={{ search: "", flows: "any", errors: "any", sort: { key: "name", dir: "asc" } }}
        onChange={() => {}}
        onReload={() => {}}
        onGenerateAll={() => {}}
        onClearAll={() => {}}
        scenariosCount={3}
        selectedCount={0}
      />,
    );
    await expectNoA11yViolations(container);
  });
});
