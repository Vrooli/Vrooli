import { describe, expect, it } from "vitest";
import App from "./App";
import { expectNoA11yViolations } from "./test-utils/a11y";
import { renderWithProviders } from "./test-utils/renderWithProviders";

describe("App accessibility", () => {
  it("has no detectable axe violations in its initial shell", async () => {
    const { container } = renderWithProviders(<App />);

    await expectNoA11yViolations(container);
    expect(container).toBeInTheDocument();
  });
});
