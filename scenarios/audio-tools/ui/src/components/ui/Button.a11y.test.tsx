import { describe, expect, it } from "vitest";

import { Button } from "./button";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";

describe("Button accessibility", () => {
  it("has no axe violations for its standard interactive state", async () => {
    const { container, getByRole } = renderWithProviders(<Button>Save settings</Button>);
    expect(getByRole("button", { name: "Save settings" })).toBeEnabled();
    await expectNoA11yViolations(container);
  });
});
