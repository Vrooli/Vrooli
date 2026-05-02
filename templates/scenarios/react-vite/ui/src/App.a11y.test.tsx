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
 * Add new a11y test files per high-traffic surface as the scenario grows.
 * One axe run per surface is enough; full-page scans are expensive.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import axe from "axe-core";

vi.mock("./lib/api", () => ({
  fetchHealth: vi.fn().mockResolvedValue({
    status: "ok",
    service: "test-service",
    timestamp: "2026-05-01T00:00:00Z",
  }),
}));

import App from "./App";
import { selectors } from "./consts/selectors";
import { setLocale } from "./i18n";

const renderApp = () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <App />
    </QueryClientProvider>,
  );
};

describe("App accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders without axe violations in English", async () => {
    const { container } = renderApp();
    // Wait for React Query to resolve so we scan the post-loading DOM.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.health.statusValue)).toBeInTheDocument();
    });
    const results = await axe.run(container);
    expect(results.violations).toEqual([]);
  });
});
