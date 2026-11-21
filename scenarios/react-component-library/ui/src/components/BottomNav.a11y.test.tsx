import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";
import { renderWithProviders } from "../test-utils";

import { BottomNav } from "./BottomNav";
import { expectNoA11yViolations } from "../test-utils";

describe("BottomNav accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <BottomNav
        label="Primary"
        items={[
          {
            id: "dashboard",
            href: "/",
            label: "Dashboard",
            icon: <span aria-hidden>1</span>,
            active: true,
          },
          {
            id: "components",
            href: "/components",
            label: "Components",
            icon: <span aria-hidden>2</span>,
          },
        ]}
      />,
    );

    await expectNoA11yViolations(container);
  });
});
