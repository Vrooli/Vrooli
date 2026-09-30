/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the actual structural composition
 * (header + landmark nav + main + bottom landmark nav). Feature cards keep
 * their own a11y tests.
 */
import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { TestAppRouter } from "../app/routes";

describe("AppShell accessibility", () => {
  it("preserves phone utility placement and navigation markup", async () => {
    await setLocale("en");
    const original = window.matchMedia;
    const media = vi.spyOn(window, "matchMedia").mockImplementation(query => ({ ...original(query), matches: false }));
    try {
      renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
      // jsdom does not evaluate viewport CSS; browser validation owns visibility.
      expect(document.querySelector('nav[aria-label="Mobile navigation"]')).not.toBeNull();
      expect(document.querySelector("select")?.closest("[data-rcl-app-shell-header]")).not.toBeNull();
    } finally { media.mockRestore(); }
  });
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the shell without axe violations in English", async () => {
    const { container } = renderWithProviders(
      <TestAppRouter initialEntries={["/"]} />,
      { withoutRouter: true },
    );
    await expectNoA11yViolations(container);
  });

  it("exposes exactly one primary navigation landmark", () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/"]} />,
      { withoutRouter: true },
    );

    expect(screen.getAllByRole("navigation", { name: "Primary navigation" })).toHaveLength(1);
    expect(screen.queryByRole("navigation", { name: "Mobile navigation" })).not.toBeInTheDocument();
  });
});
