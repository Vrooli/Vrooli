/**
 * Accessibility regression for the shell and the Now page, rendered through
 * the real route table once the page has loaded (empty API).
 */
import { cleanup, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { TestAppRouter } from "../app/routes";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import en from "../i18n/locales/en.json";
import { expectNoA11yViolations, renderWithProviders } from "../test-utils";

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the loaded Now page without axe violations", async () => {
    const { container } = renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    await screen.findByText(en.now.nothingOpen);
    await screen.findByText(en.now.nothingNeedsYou);
    await expectNoA11yViolations(container);
  });

  it("exposes one primary navigation landmark, a main region, and a skip link", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    await screen.findByText(en.now.nothingOpen);
    expect(screen.getAllByRole("navigation", { name: en.layout.navigationLabel })).toHaveLength(1);
    expect(screen.getByRole("main")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: en.layout.skipToContent })).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.tabs)).toBeInTheDocument();
  });
});
