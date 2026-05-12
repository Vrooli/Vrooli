import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { MobileNav } from "./MobileNav";

describe("MobileNav accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(<MobileNav />);
    await expectNoA11yViolations(container);
  });
});
