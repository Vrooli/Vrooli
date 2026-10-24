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

vi.mock("../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { listAsks: vi.fn(() => Promise.resolve({ asks: [] })) },
  deliveryClient: { getTimeline: vi.fn(() => Promise.resolve({ notifications: [] })) },
}));

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
    await screen.findByText(/Nothing needs an answer/);
    await expectNoA11yViolations(container);
  });

  it("exposes exactly one primary navigation landmark", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/"]} />,
      { withoutRouter: true },
    );

    await screen.findByText(/Nothing needs an answer/);

    expect(screen.getAllByRole("navigation", { name: "Primary navigation" })).toHaveLength(1);
    expect(document.querySelector('nav[aria-label="Mobile navigation"]')).not.toBeNull();
  });
});
