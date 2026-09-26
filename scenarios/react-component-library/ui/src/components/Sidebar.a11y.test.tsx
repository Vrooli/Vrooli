import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { SidebarContent } from "./Sidebar";

describe("SidebarContent accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(<SidebarContent />);
    await expectNoA11yViolations(container);
  });
});
