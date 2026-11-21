import { expect } from "vitest";
import axe from "axe-core";

export async function expectNoA11yViolations(container: Element): Promise<void> {
  const results = await axe.run(container);
  expect(results.violations).toEqual([]);
}
