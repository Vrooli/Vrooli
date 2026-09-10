import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { ThemeToggle } from "./ThemeToggle";
import { ThemeProvider } from "./theme/ThemeProvider";

describe("ThemeToggle accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <ThemeProvider><ThemeToggle /></ThemeProvider>,
    );
    await expectNoA11yViolations(container);
  });
});
