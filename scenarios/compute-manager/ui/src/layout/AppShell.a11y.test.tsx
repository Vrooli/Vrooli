/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the composition the library shell
 * produces (skip link, navigation landmarks, main). Feature cards keep their
 * own a11y tests.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import { TestAppRouter } from "../app/routes";

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the shell without axe violations in English", async () => {
    const { container } = renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(container.querySelector("main")).toBeTruthy();
    await expectNoA11yViolations(container);
  });

  it("exposes one primary navigation landmark, a main region, and a skip link", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });

    expect(screen.getAllByRole("navigation", { name: "Primary navigation" })).toHaveLength(1);
    expect(screen.getByRole("main")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Skip to content" })).toBeInTheDocument();
    // The phone tab bar is a second navigation landmark that the shell hides
    // above the md breakpoint. jsdom has no viewport, so it is present in the
    // DOM but not in the accessibility tree here; its role is asserted by the
    // library's own story contract at phone width.
    expect(screen.getByTestId(selectors.layout.tabs)).toBeInTheDocument();
  });
});
