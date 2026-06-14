import axe from "axe-core";
import { expect } from "vitest";

export async function expectNoA11yViolations(root: HTMLElement = document.body): Promise<void> {
  const results = await axe.run(root);
  expect(results.violations).toEqual([]);
}
