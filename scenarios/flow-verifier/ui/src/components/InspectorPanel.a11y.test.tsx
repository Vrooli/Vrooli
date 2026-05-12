import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { InspectorPanel } from "./InspectorPanel";

describe("InspectorPanel accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations when open", async () => {
    const { container } = renderWithProviders(
      <InspectorPanel open onClose={() => {}} title="Selected run">
        <p>Selected run summary</p>
      </InspectorPanel>,
    );
    await expectNoA11yViolations(container);
  });
});
