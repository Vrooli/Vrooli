/**
 * Shared axe-core assertion for component-level accessibility tests.
 *
 * Keeping the matcher here avoids repeating axe wiring in every feature while
 * still leaving each feature responsible for rendering and waiting for its own
 * stable state before the scan runs.
 */
import { expect } from "vitest";
import axe from "axe-core";

// jsdom does not implement getComputedStyle for pseudo-elements, which
// makes axe-core's color-contrast rule both unreliable and noisy (it
// triggers "not implemented" console.error from jsdom that the test
// harness escalates into a failure). Skip the rule here and rely on the
// design-tokens contract + Lighthouse pass in Phase 8 for contrast.
export async function expectNoA11yViolations(container: Element): Promise<void> {
  const results = await axe.run(container, {
    rules: {
      "color-contrast": { enabled: false },
    },
  });
  expect(results.violations).toEqual([]);
}
