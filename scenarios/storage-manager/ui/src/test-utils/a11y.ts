/**
 * Shared axe-core assertion for component-level accessibility tests.
 *
 * Keeping the matcher here avoids repeating axe wiring in every feature while
 * still leaving each feature responsible for rendering and waiting for its own
 * stable state before the scan runs.
 */
import { expect } from "vitest";
import axe from "axe-core";

export async function expectNoA11yViolations(container: Element): Promise<void> {
  const results = await axe.run(container);
  expect(results.violations).toEqual([]);
}
