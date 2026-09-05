import { describe, expect, it } from "vitest";
import App from "./App";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { renderWithProviders } from "@vrooli/api-base/testing";

describe("App accessibility", () => {
  it("has no detectable axe violations in its initial shell", async () => {
    const { container } = renderWithProviders(<App />);

    await expectNoA11yViolations(container);
    expect(container).toBeInTheDocument();
  });
});
