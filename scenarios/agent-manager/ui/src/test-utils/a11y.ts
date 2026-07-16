import { expect } from "vitest";
import axe from "axe-core";

export async function expectNoA11yViolations(container: Element = document.body): Promise<void> {
  const results = await axe.run(container, {
    rules: {
      region: { enabled: false },
      "color-contrast": { enabled: false },
    },
  });
  expect(results.violations).toEqual([]);
}
