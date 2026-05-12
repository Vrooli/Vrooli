import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { MobileDrawer } from "./MobileDrawer";

describe("MobileDrawer accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations when open", async () => {
    const { container } = renderWithProviders(
      <MobileDrawer open onClose={() => {}}>
        <nav aria-label="drawer-content">
          <a href="/">home</a>
        </nav>
      </MobileDrawer>,
    );
    await expectNoA11yViolations(container);
  });
});
