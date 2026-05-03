/**
 * Accessibility regression test using axe-core directly.
 *
 * Renders the App in real English (so axe checks against the actual
 * user-facing copy, not the cimode keys) and asserts no axe
 * violations. Catches:
 *
 *   - Missing labels (forms, icon-only buttons)
 *   - Insufficient color contrast detectable from CSS
 *   - Misused ARIA attributes
 *   - Heading hierarchy violations
 *   - Missing landmark roles
 *
 * Scope is the **shell** — heading + locale switcher + the
 * card-stack region. Per-feature a11y scans live in
 * features/<name>/<Name>Card.a11y.test.tsx (add when a feature ships
 * its first interactive widget). Splitting this way keeps the smoke
 * resilient when REPLACING-NOTES.md is followed: deleting a feature
 * does not break this file.
 *
 * We use `axe-core` directly rather than a matcher library so there's
 * no signature mismatch between jest-axe and vitest. The assertion is
 * `expect(results.violations).toEqual([])` — readable failure output
 * because vitest prints the array contents on failure.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import axe from "axe-core";

import { renderWithProviders } from "./test-utils";

import App from "./App";
import { selectors } from "./consts/selectors";
import { setLocale } from "./i18n";

describe("App accessibility (shell scope)", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders without axe violations in English", async () => {
    const { container } = renderWithProviders(<App />);
    // Shell selectors are present immediately; waiting on them ensures
    // the AppShell + locale switcher are in the DOM before axe scans.
    // Per-feature waits belong in per-feature a11y tests.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.locale.switcher)).toBeInTheDocument();
    });
    const results = await axe.run(container);
    expect(results.violations).toEqual([]);
  });
});
