import { renderWithQueryClient } from "./test-utils";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import App from "./App";

describe("App accessibility", () => {
  it("has no axe violations in the initial operator flow", async () => {
    const { container } = renderWithQueryClient(<App />);
    await expectNoA11yViolations(container);
    expect(container).toBeTruthy();
  });
});
