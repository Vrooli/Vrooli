/**
 * Shared axe-core assertion for Test Genie's UI accessibility regressions.
 *
 * Feature tests render their own stable state, then delegate the actual audit
 * here so every scan uses the same axe configuration and assertion shape.
 */
import axe from "axe-core";
import { expect } from "vitest";

export async function expectNoA11yViolations(container: Element): Promise<void> {
  const results = await axe.run(container);
  expect(results.violations).toEqual([]);
}
