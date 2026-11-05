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
import { Providers } from "../app/providers";
import { selectors } from "../consts/selectors";
import { ThemeProvider } from "../theme/ThemeProvider";

vi.mock("../api/channelManager", () => ({
  overview: vi.fn().mockResolvedValue({ identities: {}, actions: {} }),
}));

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  // [REQ:CHANMGR-P0-019] The operator console is rendered through its real
  // route composition and checked for detectable accessibility violations.
  it("renders the shell without axe violations in English", async () => {
    const { container } = renderWithProviders(
      <Providers><TestAppRouter initialEntries={["/"]} /></Providers>,
      { withoutRouter: true },
    );
    await screen.findByText(/No identities yet/i);
    await expectNoA11yViolations(container);
  });

  // [REQ:CHANMGR-P0-019] Theme selection must not remove the console's
  // semantic labels, landmarks, or focusable controls.
  it("renders the shell without axe violations in the dark theme", async () => {
    const { container } = renderWithProviders(
      <Providers><TestAppRouter initialEntries={["/"]} /></Providers>,
      {
        withoutRouter: true,
        extraProviders: (children) => <ThemeProvider initialChoice="dark">{children}</ThemeProvider>,
      },
    );
    await screen.findByText(/No identities yet/i);
    await expectNoA11yViolations(container);
  });

  it("exposes exactly one primary navigation landmark", async () => {
    renderWithProviders(
      <Providers><TestAppRouter initialEntries={["/"]} /></Providers>,
      { withoutRouter: true },
    );

    await screen.findByText(/No identities yet/i);
    expect(screen.getByRole("navigation", { name: "Primary navigation" })).toBeInTheDocument();
    expect(screen.getByTestId(`${selectors.layout.shell}-tabs`)).toBeInTheDocument();
  });
});
