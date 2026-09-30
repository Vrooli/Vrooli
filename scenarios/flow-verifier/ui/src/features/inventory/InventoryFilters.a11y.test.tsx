import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { InventoryFilters } from "./InventoryFilters";

describe("InventoryFilters accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <InventoryFilters
        value={{
          scenarioId: "",
          search: "",
          language: "all",
          status: [],
          sort: { key: "flowId", dir: "asc" },
        }}
        scenarios={[]}
        onChange={() => {}}
        onReload={() => {}}
        onVerifyAll={() => {}}
        flowsCount={3}
      />,
    );
    await expectNoA11yViolations(container);
  });
});
