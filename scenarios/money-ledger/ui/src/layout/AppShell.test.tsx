/**
 * AppShell tests — focus on the shell's structural contract (header + sidebar
 * + main + bottom nav) and compact header. Page content is exercised in the
 * per-page tests; this file only verifies the shell composes correctly.
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

const renderShell = () =>
  renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });

describe("AppShell structure (cimode)", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the title, sidebar, bottom nav, and main outlet", () => {
    renderShell();
    expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.topBar)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.sidebar)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.bottomNav)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.main)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
  });

  it("keeps locale switching out of the header chrome", () => {
    renderShell();
    expect(screen.queryByTestId(selectors.settingsPage.localeOption({ code: "en" }))).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.settingsPage.localeOption({ code: "ja" }))).not.toBeInTheDocument();
  });

  it("renders the canonical nav links in both sidebar and bottom nav", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    for (const key of [
      "dashboard",
      "journal",
      "settings",
    ] as const) {
      expect(screen.getByTestId(selectors.layout.sidebarLink({ key }))).toBeInTheDocument();
      expect(screen.getByTestId(selectors.layout.bottomNavLink({ key }))).toBeInTheDocument();
    }
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
    // Sidebar + bottom-nav both render the label, so there will be ≥1 match.
    expect((await screen.findAllByText(en.layout.nav.dashboard)).length).toBeGreaterThan(0);
    expect(document.documentElement.lang).toBe("en");
    expect(document.documentElement.dir).toBe("ltr");
  });

  it("switches to Japanese when the 日本語 toggle is clicked", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    await user.click(screen.getByTestId(selectors.settingsPage.localeOption({ code: "ja" })));

    await waitFor(() => {
      expect(screen.getAllByText(ja.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
    expect(document.documentElement.lang).toBe("ja");
  });

  it("flips <html dir> to rtl when an RTL locale (ar) is chosen", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    await user.click(screen.getByTestId(selectors.settingsPage.localeOption({ code: "ar" })));

    await waitFor(() => {
      expect(document.documentElement.dir).toBe("rtl");
      expect(screen.getAllByText(ar.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
  });
});
