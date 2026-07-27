import { expect } from 'vitest';
import axe from 'axe-core';

/**
 * Run the canonical accessibility audit against a rendered UI container.
 * Tests should assert on violations explicitly so product regressions are not
 * hidden behind a test-only matcher.
 */
export async function expectNoA11yViolations(container: Element): Promise<void> {
  // jsdom has no canvas implementation, which Axe needs only for its
  // color-contrast rule. Production visual checks cover contrast against real
  // rendered pixels; keeping the rule off here avoids a misleading test error.
  const results = await axe.run(container, {
    rules: {
      'color-contrast': { enabled: false },
    },
  });
  expect(results.violations).toEqual([]);
}
