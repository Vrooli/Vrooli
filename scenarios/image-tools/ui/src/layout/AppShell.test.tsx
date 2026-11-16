/**
 * AppShell tests — focus on the shell's structural contract and utility seam. Page content is exercised
 * in the per-page tests; this file only verifies the shell composes correctly.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
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
    vi.restoreAllMocks();
  });

  it("renders the title, navigation, theme utility, and main outlet", async () => {
    renderShell();
    expect(await screen.findByTestId(selectors.layout.shell)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.shell)).toHaveAttribute("data-mobile-nav", "drawer");
    expect(screen.getByTestId(`${selectors.layout.shell}-menu`)).toBeInTheDocument();
    expect(screen.getByTestId(`${selectors.layout.shell}-sidebar`)).toHaveAttribute("data-mode", "responsive");
    expect(screen.getByRole("main")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.theme.select)).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.app.title).length).toBeGreaterThan(0);
  });

  it("keeps all eight destinations reachable from the phone drawer", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "matchMedia").mockImplementation((media) => ({
      media,
      matches: false,
      onchange: null,
      addListener() {},
      removeListener() {},
      addEventListener() {},
      removeEventListener() {},
      dispatchEvent: () => false,
    }));
    renderShell();
    await user.click(screen.getByTestId(`${selectors.layout.shell}-menu`));

    const drawer = await screen.findByTestId(`${selectors.layout.shell}-sidebar`);
    expect(drawer).toHaveAttribute("data-open", "true");
    for (const key of ["home", "workspace", "library", "select", "compare", "activity", "models", "settings"] as const) {
      expect(within(drawer).getByTestId(selectors.layout.navLink({ key }))).toBeInTheDocument();
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
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    // Sidebar + bottom-nav both render the label, so there will be ≥1 match.
    expect((await screen.findAllByText(en.layout.nav.home)).length).toBeGreaterThan(0);
    expect(document.documentElement.lang).toBe("en");
    expect(document.documentElement.dir).toBe("ltr");
  });

  it("switches to Japanese when the 日本語 toggle is clicked", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    await user.click(screen.getByTestId(selectors.settingsPage.localeOption({ code: "ja" })));

    await waitFor(() => {
      expect(screen.getAllByText(ja.layout.nav.home).length).toBeGreaterThan(0);
    });
    expect(document.documentElement.lang).toBe("ja");
  });

  it("flips <html dir> to rtl when an RTL locale (ar) is chosen", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    await user.click(screen.getByTestId(selectors.settingsPage.localeOption({ code: "ar" })));

    await waitFor(() => {
      expect(document.documentElement.dir).toBe("rtl");
      expect(screen.getAllByText(ar.layout.nav.home).length).toBeGreaterThan(0);
    });
  });
});
