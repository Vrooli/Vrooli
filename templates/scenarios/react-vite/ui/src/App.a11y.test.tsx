/**
 * Accessibility regression test using axe-core directly.
 *
 * Renders the App in real English (so axe checks against the actual user-
 * facing copy, not the cimode keys) and asserts no axe violations. Catches:
 *
 *   - Missing labels (forms, icon-only buttons)
 *   - Insufficient color contrast detectable from CSS
 *   - Misused ARIA attributes
 *   - Heading hierarchy violations
 *   - Missing landmark roles
 *
 * We use `axe-core` directly rather than a matcher library so there's no
 * matcher-signature mismatch between jest-axe and vitest. The assertion is
 * `expect(results.violations).toEqual([])` — readable failure output
 * because vitest prints the array contents on failure.
 *
 * Render and mock plumbing comes from `@/test-utils` for parity with the
 * canonical `App.test.tsx`. Diverging patterns between sibling test files
 * is the failure mode the test-utils package exists to prevent — when a
 * second a11y test is added, it inherits exactly the shape of this one.
 *
 * Add new a11y test files per high-traffic surface as the scenario grows.
 * One axe run per surface is enough; full-page scans are expensive.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import axe from "axe-core";

import { makeHealthResponse, makeListNotesResponse, renderWithProviders } from "./test-utils";

vi.mock("./lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./lib/api")>();
  return {
    ...actual,
    fetchHealth: vi.fn().mockResolvedValue(makeHealthResponse()),
  };
});

vi.mock("./lib/notes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./lib/notes")>();
  return {
    ...actual,
    listNotes: vi.fn().mockResolvedValue(makeListNotesResponse()),
  };
});

import App from "./App";
import { selectors } from "./consts/selectors";
import { setLocale } from "./i18n";

describe("App accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders without axe violations in English", async () => {
    const { container } = renderWithProviders(<App />);
    // Wait for React Query to resolve so we scan the post-loading DOM
    // for both the health and notes panes.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.health.statusValue)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.notes.empty)).toBeInTheDocument();
    });
    const results = await axe.run(container);
    expect(results.violations).toEqual([]);
  });
});
