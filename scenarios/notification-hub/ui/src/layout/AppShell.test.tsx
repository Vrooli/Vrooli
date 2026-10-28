/**
 * AppShell tests — focus on the shell's structural contract (header + sidebar
 * + main + persistent mobile destinations). Page content is exercised in the
 * per-page tests; this file only verifies the shell composes correctly.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { readFileSync } from "node:fs";
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
import { conversationsClient, recipientsClient } from "../api/notifications";
import { clearAskDrafts, readAskDraft } from "../features/asks/askDrafts";

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
    vi.mocked(conversationsClient.getAsk).mockReset();
    vi.mocked(recipientsClient.listDevices).mockReset().mockResolvedValue({ devices: [], vapidPublicKey: "" } as never);
    clearAskDrafts();
    vi.restoreAllMocks();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalInnerWidth });
  });

  it("renders the title, sidebar, and main outlet with the persistent mobile navigation", () => {
    renderShell();
    expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.shell)).toHaveAttribute("data-mobile-nav", "menu-tabs");
    expect(screen.getByRole("main")).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.app.title).length).toBeGreaterThan(0);
    expect(screen.getByTestId(`${selectors.layout.navLink({ key: "asks" })}-tab`)).toHaveAttribute("aria-current", "page");
  });

  it("consumes the same top safe-area inset at the shell boundary while retaining viewport height", () => {
    const styles = readFileSync("src/styles.css", "utf8");
    const shellRule = styles.match(/\[data-rcl-app-shell\]\s*\{([^}]*)\}/)?.[1] ?? "";

    expect(shellRule).toContain("--rcl-app-shell-block-size: calc(100dvh - var(--safe-area-inset-top))");
    expect(shellRule).toContain("block-size: var(--rcl-app-shell-block-size)");
    expect(shellRule).toContain("margin-block-start: var(--safe-area-inset-top)");
    expect(styles).toContain("padding-bottom: var(--safe-area-inset-bottom)");
  });

  it("keeps locale switching out of the header chrome", () => {
    renderShell();
    expect(screen.queryByTestId(selectors.settingsPage.localeOption({ code: "en" }))).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.settingsPage.localeOption({ code: "ja" }))).not.toBeInTheDocument();
  });

  it("renders all five canonical destinations in desktop navigation", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    const desktopNavigation = screen.getByTestId(`${selectors.layout.shell}-navigation`);
    const orderedKeys = [
      "asks",
      "dashboard",
      "history",
      "delivery",
      "settings",
    ] as const;
    for (const key of orderedKeys) {
      expect(within(desktopNavigation).getByTestId(selectors.layout.navLink({ key }))).toBeInTheDocument();
    }
    expect(Array.from(desktopNavigation.querySelectorAll("a")).map(link => link.getAttribute("data-testid"))).toEqual(
      orderedKeys.map(key => selectors.layout.navLink({ key })),
    );
  });

  it("shows the nested Settings and Reach header while keeping primary navigation", async () => {
    setViewportWidth(390);
    await setLocale("en");
    renderWithProviders(
      <AppShellWithRouterLocation initialEntry="/settings/reach" />,
      { withoutRouter: true },
    );
    const header = screen.getByTestId(`${selectors.layout.shell}-header`);
    expect(within(header).getByTestId(`${selectors.layout.shell}-header-brand`)).toHaveAttribute("href", "/settings");
    expect(within(header).getByTestId(`${selectors.layout.shell}-menu`)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.shell)).toHaveAttribute("data-mobile-nav", "menu-tabs");
    await userEvent.click(within(header).getByTestId(`${selectors.layout.shell}-header-brand`));
    expect(screen.getByTestId("router-location")).toHaveTextContent("/settings");
  });

  it("uses the selected device context header for removal review and returns to the owner list", async () => {
    setViewportWidth(390);
    await setLocale("en");
    vi.mocked(recipientsClient.listDevices).mockResolvedValue({
      devices: [{ id: "device-demo-work", name: "Work browser", machineId: "demo-workstation", channels: ["web_push"] }],
      vapidPublicKey: "AQID",
    } as never);
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/settings/reach"]} />, { withoutRouter: true });
    await user.click(await screen.findByRole("button", { name: "Remove device" }));

    const header = screen.getByTestId(`${selectors.layout.shell}-header`);
    const contextHeader = header.querySelector<HTMLElement>("[data-selected-quiet-editor-header]");
    expect(contextHeader).not.toBeNull();
    if (!contextHeader) throw new Error("selected device context header is missing");
    expect(within(contextHeader).getByText("Devices")).toBeInTheDocument();
    expect(within(contextHeader).getByText("Work browser")).toBeInTheDocument();
    expect(within(header).getByTestId(`${selectors.layout.shell}-menu`)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.shell)).toHaveAttribute("data-mobile-nav", "menu-tabs");

    await user.click(within(contextHeader).getByRole("button", { name: /Devices/ }));
    expect(screen.queryByTestId("reach-remove-review")).not.toBeInTheDocument();
    await waitFor(() => expect(document.activeElement).toHaveAttribute("id", "reach-remove-trigger-device-demo-work"));
  });

  it("uses one selected Delivery context header and returns to the Delivery list", async () => {
    setViewportWidth(390);
    await setLocale("en");
    const user = userEvent.setup();
    renderWithProviders(
      <AppShellWithRouterLocation initialEntry="/delivery/DEMO-N043?vrooli_test_mode=1" />,
      { withoutRouter: true },
    );

    const header = screen.getByTestId(`${selectors.layout.shell}-header`);
    expect(within(header).getByTestId(`${selectors.layout.shell}-menu`)).toBeInTheDocument();
    const selectedHeader = header.querySelector<HTMLElement>("[data-selected-delivery-header]");
    expect(selectedHeader).not.toBeNull();
    if (!selectedHeader) throw new Error("selected Delivery header is missing");
    expect(within(selectedHeader).getByText("DEMO-N043")).toBeInTheDocument();
    expect(within(selectedHeader).getByRole("link", { name: "Delivery" })).toHaveAttribute("href", "/delivery?vrooli_test_mode=1");
    const styles = readFileSync("src/styles.css", "utf8");
    expect(styles).toContain(".delivery-context-shell [data-rcl-app-shell-header-brand] {\n    display: none !important;");

    const backLink = within(selectedHeader).getByRole("link", { name: "Delivery" });
    expect(backLink.tabIndex).toBe(0);

    await user.click(backLink);
    expect(screen.getByTestId("router-location")).toHaveTextContent("/delivery?vrooli_test_mode=1");
  });

  it("uses the selected Delivery context on the analytics route with a distinct back target", async () => {
    setViewportWidth(390);
    await setLocale("en");
    renderWithProviders(
      <AppShellWithRouterLocation initialEntry="/delivery?vrooli_test_mode=1" />,
      { withoutRouter: true },
    );

    const header = screen.getByTestId(`${selectors.layout.shell}-header`);
    const selectedHeader = header.querySelector<HTMLElement>("[data-selected-delivery-header]");
    expect(selectedHeader).not.toBeNull();
    if (!selectedHeader) throw new Error("selected Delivery analytics header is missing");
    expect(within(selectedHeader).getAllByText("Delivery")).toHaveLength(2);
    expect(within(selectedHeader).getByRole("link", { name: "Delivery" })).toHaveAttribute("href", "/?vrooli_test_mode=1");
    expect(selectedHeader.className).toContain("gap-4");
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

  it("keeps all five phone destinations in the bottom bar and restores compact-menu focus", async () => {
    setViewportWidth(390);
    await setLocale("en");
    const user = userEvent.setup();
    renderWithProviders(
      <AppShellWithRouterLocation initialEntry="/history?vrooli_test_mode=1&q=mobile" />,
      { withoutRouter: true },
    );

    const menuButton = screen.getByTestId(`${selectors.layout.shell}-menu`);
    expect(menuButton).toHaveAccessibleName("Open navigation");
    expect(menuButton.querySelector("svg")).toBeInTheDocument();
    expect(menuButton).not.toHaveTextContent("Menu");
    expect(screen.getByTestId(selectors.layout.shell)).toHaveAttribute("data-mobile-nav", "menu-tabs");
    const compactHeader = screen.getByTestId(`${selectors.layout.shell}-header`);
    expect(within(compactHeader).getByTestId(selectors.app.title)).toHaveTextContent(en.app.title);
    expect(menuButton).toHaveAttribute("data-rcl-app-shell-menu");
    await user.click(menuButton);

    const menuPanel = await screen.findByTestId(`${selectors.layout.shell}-menu-panel`);
    expect(menuPanel.querySelectorAll("a")).toHaveLength(5);
    expect(within(menuPanel).getByTestId(selectors.layout.navLink({ key: "history" }))).toHaveAttribute("aria-current", "page");

    await user.keyboard("{Escape}");
    await waitFor(() => expect(menuButton).toHaveFocus());
    expect(screen.queryByTestId(`${selectors.layout.shell}-menu-panel`)).not.toBeInTheDocument();

    const destinations = [
      ["dashboard", "/updates"],
      ["asks", "/asks"],
      ["history", "/history"],
      ["delivery", "/delivery"],
      ["settings", "/settings"],
    ] as const;
    for (const [key, path] of destinations) {
      const bottomBar = screen.getByTestId(`${selectors.layout.shell}-tabs`);
      await user.click(within(bottomBar).getByTestId(`${selectors.layout.navLink({ key })}-tab`));
      await waitFor(() => expect(screen.getByTestId("router-location")).toHaveTextContent(`${path}?vrooli_test_mode=1`));
      await waitFor(() => expect(screen.getByTestId(`${selectors.layout.shell}-main`)).toHaveFocus());
      expect(within(bottomBar).getByTestId(`${selectors.layout.navLink({ key })}-tab`)).toHaveAttribute("aria-current", "page");
      expect(screen.queryByTestId(`${selectors.layout.shell}-menu-panel`)).not.toBeInTheDocument();
    }
  });

  it("keeps the unsent ask draft stored when the owner leaves its route", async () => {
    setViewportWidth(390);
    await setLocale("en");
    clearAskDrafts();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: {
        id: "ask-1",
        question: "Which option should we take?",
        options: [{ key: "re-aim", label: "Re-aim" }, { key: "close", label: "Close" }],
        recommended: "re-aim",
        recommendationReason: "Quality gaps remain.",
        defaultAnswer: "re-aim",
        reversible: true,
        state: "pending",
        contextUrl: "",
        bigPicture: "",
        technicalDetails: "",
      },
    } as never);
    const user = userEvent.setup();
    renderWithProviders(
      <TestAppRouter initialEntries={["/asks/ask-1?vrooli_test_mode=1"]} />,
      { withoutRouter: true },
    );

    const question = await screen.findByTestId(selectors.asks.question);
    expect(question).toBeInTheDocument();
    const closeOption = (await screen.findAllByTestId(selectors.asks.option)).find(
      option => option.getAttribute("data-option-key") === "close",
    );
    expect(closeOption).toBeDefined();
    await user.click(closeOption!);
    await user.type(screen.getByTestId(selectors.asks.note), "keep this unsent note");
    expect(readAskDraft("owner-1", "ask-1")).toEqual({ answer: "close", note: "keep this unsent note" });

    const bottomBar = screen.getByTestId(`${selectors.layout.shell}-tabs`);
    await user.click(within(bottomBar).getByTestId(`${selectors.layout.navLink({ key: "settings" })}-tab`));
    expect(await screen.findByTestId(selectors.pages.settings)).toBeInTheDocument();
    expect(readAskDraft("owner-1", "ask-1")).toEqual({ answer: "close", note: "keep this unsent note" });

    expect(readAskDraft("owner-1", "ask-1")).toEqual({ answer: "close", note: "keep this unsent note" });
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
      expect(screen.getByTestId(`${selectors.layout.shell}-main`)).toHaveFocus();
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
    expect(screen.getByTestId(`${selectors.layout.shell}-menu`)).toBeInTheDocument();
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
    // The visible desktop sidebar carries the canonical destination label.
    expect((await screen.findAllByText(en.layout.nav.dashboard)).length).toBeGreaterThan(0);
    expect(document.documentElement.lang).toBe("en");
    expect(document.documentElement.dir).toBe("ltr");
  });

  it("switches to Japanese when the 日本語 toggle is clicked", async () => {
    const user = userEvent.setup();
    const { queryClient } = renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    await waitFor(() => {
      expect(queryClient.getQueryData(["owner-session"])).toEqual(expect.objectContaining({ signedIn: true }));
    });
    await user.click(await screen.findByTestId(selectors.settingsPage.localeOption({ code: "ja" })));

    await waitFor(() => {
      expect(screen.getAllByText(ja.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
    expect(document.documentElement.lang).toBe("ja");
  });

  it("flips <html dir> to rtl when an RTL locale (ar) is chosen", async () => {
    const user = userEvent.setup();
    const { queryClient } = renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    await waitFor(() => {
      expect(queryClient.getQueryData(["owner-session"])).toEqual(expect.objectContaining({ signedIn: true }));
    });
    await user.click(await screen.findByTestId(selectors.settingsPage.localeOption({ code: "ar" })));

    await waitFor(() => {
      expect(document.documentElement.dir).toBe("rtl");
      expect(screen.getAllByText(ar.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
  });
});
