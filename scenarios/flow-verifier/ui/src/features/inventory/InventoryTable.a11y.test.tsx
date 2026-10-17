import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { InventoryTable } from "./InventoryTable";

describe("InventoryTable accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <InventoryTable
        flows={[
          { flowId: "alpha.flow", contractPath: "a/flow.json", language: "ts", schemaVersion: 1, kind: "temporal" },
        ]}
        latestByFlow={new Map()}
        onVerifyOne={() => {}}
        anyPending={false}
      />,
    );
    await expectNoA11yViolations(container);
  });
});
