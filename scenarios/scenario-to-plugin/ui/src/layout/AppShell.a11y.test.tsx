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
import { selectors } from "../consts/selectors";

vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  const { makeApiMocks } = await import("../test-utils");
  return { ...actual, ...makeApiMocks() };
});

describe("AppShell accessibility", () => {
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
    await screen.findByText(/Readiness is unavailable|No governed scenarios were returned/);
    await screen.findByTestId(selectors.health.statusValue);
    await expectNoA11yViolations(container);
  });

  it("exposes exactly one primary navigation landmark", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/"]} />,
      { withoutRouter: true },
    );

    expect(screen.getAllByRole("navigation", { name: "Primary navigation" })).toHaveLength(1);
    expect(document.querySelector('nav[aria-label="Mobile navigation"]')).not.toBeNull();
    await screen.findByText(/Readiness is unavailable|No governed scenarios were returned/);
    await screen.findByTestId(selectors.health.statusValue);
  });
});
