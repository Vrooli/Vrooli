import { afterEach, describe, it } from "vitest";
import { cleanup, render } from "@testing-library/react";

import { BottomNav } from "../../../library/components/BottomNav/versions/1.0.0/BottomNav";
import { expectNoA11yViolations } from "../test-utils";

describe("BottomNav accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = render(
      <BottomNav
        label="Primary"
        items={[
          { id: "dashboard", href: "/", label: "Dashboard", icon: <span aria-hidden>1</span>, active: true },
          { id: "components", href: "/components", label: "Components", icon: <span aria-hidden>2</span> },
        ]}
      />,
    );

    await expectNoA11yViolations(container);
  });
});
