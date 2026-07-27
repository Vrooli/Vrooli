/** Shared axe-core assertion for component-level accessibility tests. */
import axe from "axe-core";
import { expect } from "vitest";

export async function expectNoA11yViolations(
  container: Element,
): Promise<void> {
  const results = await axe.run(container);
  expect(results.violations).toEqual([]);
}
