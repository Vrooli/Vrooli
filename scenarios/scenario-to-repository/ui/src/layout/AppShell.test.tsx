/**
 * AppShell tests — the shell is the component library's; this file verifies
 * the configuration this scenario feeds it (nav items, router adapter, labels)
 * and that the landmarks a page relies on are present. Page content is
 * exercised in the per-page tests.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import en from "../i18n/locales/en.json";
import ja from "../i18n/locales/ja.json";
import ar from "../i18n/locales/ar.json";
import { TestAppRouter } from "../app/routes";

const renderShell = (path = "/") =>
  renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });

describe("AppShell structure (cimode)", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the shell landmarks, the brand, and the main outlet", () => {
    renderShell();
    expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.navigation)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.tabs)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.main)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.brand)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.skip)).toHaveAttribute("href", `#${selectors.layout.main}`);
  });

  it("keeps preferences out of the shell chrome", () => {
    renderShell();
    expect(screen.queryByTestId(selectors.settingsPage.localeSelect)).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.settingsPage.themeSelect)).not.toBeInTheDocument();
  });

  it("renders every nav item as a desktop link and a phone tab", () => {
    renderShell("/settings");
    for (const key of [
      "dashboard",
      "settings",
    ] as const) {
      expect(screen.getByTestId(selectors.layout.navLink({ key }))).toBeInTheDocument();
      expect(screen.getByTestId(selectors.layout.navTab({ key }))).toBeInTheDocument();
    }
  });

  it("marks the current route on the desktop link and the phone tab", () => {
    renderShell("/settings");
    expect(screen.getByTestId(selectors.layout.navLink({ key: "settings" }))).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId(selectors.layout.navTab({ key: "settings" }))).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId(selectors.layout.navLink({ key: "dashboard" }))).not.toHaveAttribute("aria-current");
  });

  it("navigates through the router when a phone tab is selected", async () => {
    const user = userEvent.setup();
    renderShell("/");
    await user.click(screen.getByTestId(selectors.layout.navTab({ key: "settings" })));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
    });
  });
});

describe("Locale switching through the shell (real locales)", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders English copy by default and reflects it on <html>", async () => {
    renderShell();
    // The desktop link and the phone tab both render the label, so there will be ≥1 match.
    expect((await screen.findAllByText(en.layout.nav.dashboard)).length).toBeGreaterThan(0);
    expect(document.documentElement.lang).toBe("en");
    expect(document.documentElement.dir).toBe("ltr");
  });

  it("switches to Japanese when 日本語 is selected", async () => {
    const user = userEvent.setup();
    renderShell("/settings");
    await user.selectOptions(screen.getByTestId(selectors.settingsPage.localeSelect), "ja");

    await waitFor(() => {
      expect(screen.getAllByText(ja.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
    expect(document.documentElement.lang).toBe("ja");
  });

  it("flips <html dir> to rtl when an RTL locale (ar) is chosen", async () => {
    const user = userEvent.setup();
    renderShell("/settings");
    await user.selectOptions(screen.getByTestId(selectors.settingsPage.localeSelect), "ar");

    await waitFor(() => {
      expect(document.documentElement.dir).toBe("rtl");
      expect(screen.getAllByText(ar.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
  });
});
