/**
 * Shared accessibility assertion for component-level UI tests.
 *
 * Features render a settled state and delegate the scan here so the suite has
 * one consistent axe configuration and one obvious place to evolve it.
 */
import axe from "axe-core";
import { expect } from "vitest";

export async function expectNoA11yViolations(container: Element): Promise<void> {
  const results = await axe.run(container, {
    // jsdom cannot compute rendered contrast; runtime/browser checks own that
    // pixel-dependent rule.
    rules: { "color-contrast": { enabled: false } },
  });
  expect(results.violations).toEqual([]);
}
