import axe from "axe-core";
import { expect } from "vitest";

export async function expectNoA11yViolations(container) {
  const result = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  expect(result.violations).toEqual([]);
}
