import axe from 'axe-core';

/** Asserts that a rendered UI container has no axe accessibility violations. */
export async function expectNoA11yViolations(container: Element): Promise<void> {
  const results = await axe.run(container);
  expect(results.violations).toEqual([]);
}
