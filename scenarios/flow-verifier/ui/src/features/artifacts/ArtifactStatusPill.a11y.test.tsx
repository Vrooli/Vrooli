import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { ArtifactStatusPill } from "./ArtifactStatusPill";

describe("ArtifactStatusPill accessibility", () => {
  afterEach(() => cleanup());

  it("renders all three statuses without axe violations", async () => {
    for (const status of ["fresh", "missing", "needs_generate"] as const) {
      const { container, unmount } = renderWithProviders(<ArtifactStatusPill status={status} />);
      await expectNoA11yViolations(container);
      unmount();
    }
  });
});
