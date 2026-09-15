import axe from "axe-core";
import { expect } from "vitest";

/** Assert that a rendered container has no axe-core accessibility violations. */
export async function expectNoA11yViolations(container: Element): Promise<void> {
  const results = await axe.run(container, {
    rules: { "color-contrast": { enabled: false } },
  });
  expect(results.violations).toEqual([]);
}
