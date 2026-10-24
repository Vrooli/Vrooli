/**
 * AppShell tests — focus on the shell's structural contract (header + sidebar
 * + main + bottom nav) and compact header. Page content is exercised in the
 * per-page tests; this file only verifies the shell composes correctly.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import en from "../i18n/locales/en.json";
import ja from "../i18n/locales/ja.json";
import ar from "../i18n/locales/ar.json";
import { TestAppRouter } from "../app/routes";
import { AppShell } from "./AppShell";

vi.mock("../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  recipientsClient: { listDevices: vi.fn(() => Promise.resolve({ devices: [], vapidPublicKey: "" })) },
  registerBrowserPushSubscription: vi.fn(),
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(() => Promise.resolve({ asks: [] })) },
  deliveryClient: { getTimeline: vi.fn(() => Promise.resolve({ notifications: [] })) },
}));

const originalInnerWidth = window.innerWidth;

const setViewportWidth = (width: number) => {
  Object.defineProperty(window, "innerWidth", { configurable: true, value: width });
  vi.spyOn(window, "matchMedia").mockImplementation((media) => {
    const match = /min-width:\s*([\d.]+)(rem|px)/i.exec(media);
    const minimum = match?.[1] && match[2] ? Number(match[1]) * (match[2].toLowerCase() === "rem" ? 16 : 1) : Infinity;
    return {
      media,
      matches: width >= minimum,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: () => false,
    } as MediaQueryList;
  });
};

const renderShell = () =>
  renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });

function RouterLocation() {
  const location = useLocation();
  return <output data-testid="router-location">{`${location.pathname}${location.search}`}</output>;
}

function AppShellWithRouterLocation({ initialEntry }: { initialEntry: string }) {
  return (
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="*" element={<><AppShell /><RouterLocation /></>} />
      </Routes>
    </MemoryRouter>
  );
}

describe("AppShell structure (cimode)", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalInnerWidth });
  });

  it("renders the title, sidebar, bottom nav, and main outlet", () => {
    renderShell();
    expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
    expect(screen.getByRole("main")).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.app.title).length).toBeGreaterThan(0);
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
      "asks",
      "history",
      "delivery",
      "settings",
    ] as const) {
      expect(screen.getByTestId(selectors.layout.navLink({ key }))).toBeInTheDocument();
      expect(screen.getByTestId(selectors.layout.navTab({ key }))).toBeInTheDocument();
    }
  });

  it("keeps ordinary recipient reads available when the owner session expires", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.asks)).toBeInTheDocument();
    window.dispatchEvent(new Event("notification-hub:session-expired"));
    expect(screen.getByTestId(selectors.pages.asks)).toBeInTheDocument();
  });

  it("keeps the visually hidden skip link as the first keyboard navigation target", async () => {
    const user = userEvent.setup();
    renderShell();

    await screen.findByTestId(selectors.pages.asks);
    expect(screen.getByTestId(selectors.layout.main).tagName).toBe("MAIN");
    const skipLink = screen.getByTestId(`${selectors.layout.shell}-skip`);
    expect(skipLink).not.toHaveFocus();
    expect(getComputedStyle(skipLink).transform).toBe("translateY(-200%)");
    await user.tab();

    expect(skipLink).toHaveFocus();
    expect(skipLink.matches(":focus-visible")).toBe(true);
    expect(skipLink).toHaveAttribute("href", `#${selectors.layout.shell}-main`);
    const main = screen.getByTestId(`${selectors.layout.shell}-main`);
    expect(main).toHaveAttribute("tabindex", "-1");
    await user.keyboard("{Enter}");
    expect(window.location.hash).toBe(`#${selectors.layout.shell}-main`);
    await waitFor(() => expect(main).toHaveFocus());
  });

  it("reaches every canonical destination through the shared phone navigation", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });

    for (const key of ["dashboard", "asks", "history", "delivery", "settings"] as const) {
      const tab = screen.getByTestId(selectors.layout.navTab({ key }));
      await user.click(tab);
      await waitFor(() => expect(tab).toHaveAttribute("aria-current", "page"));
    }
  });

  it("opens and dismisses the shared phone menu accessibly while all five tabs persist", async () => {
    setViewportWidth(390);
    await setLocale("en");
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/history?q=mobile"]} />, { withoutRouter: true });

    const tabs = screen.getByTestId(`${selectors.layout.shell}-tabs`);
    for (const key of ["dashboard", "asks", "history", "delivery", "settings"] as const) {
      expect(within(tabs).getByTestId(selectors.layout.navTab({ key }))).toBeInTheDocument();
    }
    const menuButton = screen.getByRole("button", { name: "Open navigation" });
    const compactHeader = screen.getByTestId(`${selectors.layout.shell}-header`);
    expect(within(compactHeader).getByTestId(selectors.app.title)).toHaveClass("text-xs");
    expect(menuButton).toHaveAttribute("data-rcl-app-shell-menu");
    expect(menuButton).toHaveAttribute("aria-expanded", "false");
    await user.click(menuButton);

    const menuPanel = screen.getByTestId(`${selectors.layout.shell}-menu-panel`);
    expect(menuButton).toHaveAttribute("aria-expanded", "true");
    expect(menuButton).toHaveAccessibleName("Close navigation");
    expect(within(menuPanel).getAllByRole("link")).toHaveLength(5);
    expect(within(menuPanel).getByRole("link", { name: "History" })).toHaveAttribute("aria-current", "page");
    for (const key of ["dashboard", "asks", "history", "delivery", "settings"] as const) {
      expect(within(tabs).getByTestId(selectors.layout.navTab({ key }))).toBeInTheDocument();
    }

    await user.keyboard("{Escape}");
    expect(screen.queryByTestId(`${selectors.layout.shell}-menu-panel`)).not.toBeInTheDocument();
    expect(menuButton).toHaveAttribute("aria-expanded", "false");
    expect(menuButton).toHaveFocus();

    await user.click(menuButton);
    const reopenedPanel = screen.getByTestId(`${selectors.layout.shell}-menu-panel`);
    await user.click(within(reopenedPanel).getByRole("link", { name: "Delivery" }));
    await waitFor(() => expect(within(tabs).getByTestId(selectors.layout.navTab({ key: "delivery" }))).toHaveAttribute("aria-current", "page"));
    expect(screen.queryByTestId(`${selectors.layout.shell}-menu-panel`)).not.toBeInTheDocument();
  });

  it("uses shared default links for router navigation and retains routed test mode", async () => {
    await setLocale("en");
    const user = userEvent.setup();
    renderWithProviders(
      <AppShellWithRouterLocation initialEntry="/history?vrooli_test_mode=1&q=mobile" />,
      { withoutRouter: true },
    );

    const historyLink = screen.getByTestId(selectors.layout.navLink({ key: "history" }));
    expect(historyLink).toHaveAttribute("href", "/history?vrooli_test_mode=1");
    expect(screen.getByTestId("router-location")).toHaveTextContent("/history?vrooli_test_mode=1&q=mobile");

    await user.click(screen.getByTestId(selectors.layout.navLink({ key: "delivery" })));

    await waitFor(() => {
      expect(screen.getByTestId("router-location")).toHaveTextContent("/delivery?vrooli_test_mode=1");
      expect(screen.getByTestId(selectors.layout.navTab({ key: "delivery" }))).toHaveAttribute("aria-current", "page");
    });
  });

  it("keeps desktop navigation on the shared column with no open compact panel", async () => {
    setViewportWidth(1280);
    await setLocale("en");
    renderWithProviders(<TestAppRouter initialEntries={["/history"]} />, { withoutRouter: true });

    const desktopNavigation = screen.getByTestId(`${selectors.layout.shell}-navigation`);
    for (const key of ["dashboard", "asks", "history", "delivery", "settings"] as const) {
      expect(within(desktopNavigation).getByTestId(selectors.layout.navLink({ key }))).toBeInTheDocument();
    }
    expect(screen.queryByTestId(`${selectors.layout.shell}-menu-panel`)).not.toBeInTheDocument();
    expect(screen.getByTestId(`${selectors.layout.shell}-menu`)).toHaveAttribute("aria-expanded", "false");
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
    await user.click(await screen.findByTestId(selectors.settingsPage.localeOption({ code: "ja" })));

    await waitFor(() => {
      expect(screen.getAllByText(ja.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
    expect(document.documentElement.lang).toBe("ja");
  });

  it("flips <html dir> to rtl when an RTL locale (ar) is chosen", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    await user.click(await screen.findByTestId(selectors.settingsPage.localeOption({ code: "ar" })));

    await waitFor(() => {
      expect(document.documentElement.dir).toBe("rtl");
      expect(screen.getAllByText(ar.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
  });
});
