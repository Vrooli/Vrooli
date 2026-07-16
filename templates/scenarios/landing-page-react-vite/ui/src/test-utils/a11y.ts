/**
 * Shared axe-core assertion for component-level accessibility tests.
 *
 * Keeping the matcher here avoids repeating axe wiring in every surface while
 * still leaving each test responsible for rendering and waiting for its own
 * stable state before the scan runs. UI Health treats this helper plus at
 * least one *.a11y.test.tsx as the generated harness contract; add route and
 * interaction a11y tests where a single component scan is insufficient.
 */
import { expect } from 'vitest';
import axe from 'axe-core';

export async function expectNoA11yViolations(container: Element): Promise<void> {
  const results = await axe.run(container);
  expect(results.violations).toEqual([]);
}
