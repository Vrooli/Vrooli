import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { MobileHeader } from "./MobileHeader";

describe("MobileHeader accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(<MobileHeader onOpenDrawer={() => {}} />);
    await expectNoA11yViolations(container);
  });
});
