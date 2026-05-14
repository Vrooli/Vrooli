import { afterEach, describe, it } from "vitest";
import { cleanup, render } from "@testing-library/react";

import { SidebarShell } from "../../../library/components/SidebarShell/versions/1.0.0/SidebarShell";
import { expectNoA11yViolations } from "../test-utils";

describe("SidebarShell accessibility", () => {
  afterEach(() => cleanup());

  it("renders the open mobile dialog without axe violations", async () => {
    const { container } = render(
      <SidebarShell
        mobileOpen
        onMobileClose={() => {}}
        mobileLabel="Navigation drawer"
        closeLabel="Close navigation"
        mobileHeader={<span>Component Library</span>}
      >
        <nav aria-label="Primary navigation">
          <a href="/">Dashboard</a>
        </nav>
      </SidebarShell>,
    );

    await expectNoA11yViolations(container);
  });
});
