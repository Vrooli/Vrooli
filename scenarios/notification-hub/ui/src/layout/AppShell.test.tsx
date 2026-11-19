/**
 * AppShell tests — focus on the shell's structural contract (header + sidebar
 * + main + persistent mobile destinations). Page content is exercised in the
 * per-page tests; this file only verifies the shell composes correctly.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { readFileSync } from "node:fs";
import { act, cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import { en, ja, ar } from "../i18n/catalog";
import { TestAppRouter } from "../app/routes";
import { AppShell } from "./AppShell";
import { conversationsClient, identityClient, recipientsClient, ROUTED_TEST_LEASE_EXPIRED_EVENT } from "../api/notifications";
import { makeAsk } from "../features/packet/mocks/askFactory";
import { clearAskDrafts, readAskDraft } from "../features/packet/model/draftStore";

const viewportState = vi.hoisted(() => ({ frameHeight: 844, layoutHeight: 785, visibleHeight: 500, keyboardVisible: false }));
vi.mock("@vrooli/react-component-library/useViewportEnvironment/1", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@vrooli/react-component-library/useViewportEnvironment/1")>()),
  useViewportEnvironment: () => viewportState,
}));

vi.mock("../api/notifications", () => ({
  ROUTED_TEST_LEASE_EXPIRED_EVENT: "notification-hub:routed-test-lease-expired",
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  recipientsClient: { listDevices: vi.fn(() => Promise.resolve({ devices: [], vapidPublicKey: "" })) },
  registerBrowserPushSubscription: vi.fn(),
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(() => Promise.resolve({ asks: [], totalAwaiting: 0 })), listNativeConsents: vi.fn(() => Promise.resolve({ consents: [] })) },
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
    Object.assign(viewportState, { frameHeight: 844, layoutHeight: 785, visibleHeight: 500, keyboardVisible: false });
  });

  it("renders the sidebar and main outlet with the persistent mobile tab bar and no header chrome", () => {
    renderShell();
    expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.shell)).toHaveAttribute("data-mobile-nav", "tabs");
    expect(screen.getByRole("main")).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.app.title).length).toBeGreaterThan(0);
    expect(screen.getByTestId(`${selectors.layout.navLink({ key: "dashboard" })}-tab`)).toHaveAttribute("aria-current", "page");
    expect(screen.queryByTestId(`${selectors.layout.shell}-menu`)).not.toBeInTheDocument();
    expect(screen.queryByTestId(`${selectors.layout.shell}-header`)).not.toBeInTheDocument();
  });

  it.each([["/asks/decision-1", true], ["/asks", false], ["/history", false]])(
    "only hides phone tabs on a single conversation at %s", (path, conversation) => {
      renderWithProviders(<AppShellWithRouterLocation initialEntry={path} />, { withoutRouter: true });
      expect(screen.getByTestId(selectors.layout.shell).classList.contains("conversation-shell")).toBe(conversation);
    },
  );

  it("consumes the same top safe-area inset at the shell boundary while retaining viewport height", () => {
    const styles = readFileSync("src/styles.css", "utf8");
    const shellRule = styles.match(/\[data-rcl-app-shell\]\s*\{([^}]*)\}/)?.[1] ?? "";

    expect(shellRule).toContain("--rcl-app-shell-block-size: calc(var(--hub-viewport-height, 100dvh) - var(--safe-area-inset-top))");
    expect(shellRule).toContain("block-size: var(--rcl-app-shell-block-size)");
    expect(shellRule).toContain("margin-block-start: var(--safe-area-inset-top)");
    expect(styles).toContain("padding-bottom: var(--safe-area-inset-bottom)");
  });

  it("sizes the shell from the measured frame and the visible viewport when the keyboard opens", () => {
    const view = renderWithProviders(<AppShellWithRouterLocation initialEntry="/" />, { withoutRouter: true });
    expect(document.documentElement.style.getPropertyValue("--hub-viewport-height")).toBe("844px");
    viewportState.keyboardVisible = true;
    view.rerender(<AppShellWithRouterLocation initialEntry="/" />);
    expect(document.documentElement.style.getPropertyValue("--hub-viewport-height")).toBe("500px");
    view.unmount();
    expect(document.documentElement.style.getPropertyValue("--hub-viewport-height")).toBe("");
  });

  it("keeps locale switching out of the header chrome", () => {
    renderShell();
    expect(screen.queryByTestId(selectors.settingsPage.localeOption({ code: "en" }))).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.settingsPage.localeOption({ code: "ja" }))).not.toBeInTheDocument();
  });

  it("renders the four canonical destinations in desktop navigation, with Delivery inside Updates", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    const desktopNavigation = screen.getByTestId(`${selectors.layout.shell}-navigation`);
    const orderedKeys = ["asks", "dashboard", "history", "settings"] as const;
    expect(Array.from(desktopNavigation.querySelectorAll("a")).map(link => link.getAttribute("data-testid"))).toEqual(
      orderedKeys.map(key => selectors.layout.navLink({ key })),
    );
  });

  it.each(["/delivery", "/delivery/DEMO-N043"])("keeps Updates current while %s is open", (path) => {
    renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });
    const desktopNavigation = screen.getByTestId(`${selectors.layout.shell}-navigation`);
    expect(within(desktopNavigation).getByTestId(selectors.layout.navLink({ key: "dashboard" }))).toHaveAttribute("aria-current", "page");
    expect(within(desktopNavigation).getByTestId(selectors.layout.navLink({ key: "asks" }))).not.toHaveAttribute("aria-current");
  });

  it("counts what needs the owner on the Needs you tab and says so to assistive technology", async () => {
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({ asks: [makeAsk({ id: "a" })], nextCursor: "", totalAwaiting: 2 } as never);
    vi.mocked(conversationsClient.listNativeConsents).mockResolvedValue({ consents: [{ authorizationId: "c", state: "pending" }, { authorizationId: "d", state: "unavailable" }] } as never);
    renderShell();
    const badge = await screen.findByRole("img", { name: "3 waiting", hidden: true });
    expect(within(screen.getByTestId(selectors.layout.navLink({ key: "asks" }))).getByRole("img", { name: "3 waiting", hidden: true })).toBe(badge);
    expect(conversationsClient.listAsks).toHaveBeenCalledWith({ limit: 1, stateFilter: "awaiting" }, expect.anything());
  });

  it("shows no count when nothing waits, and shows the Hub's exact total however long the queue is", async () => {
    await setLocale("en");
    renderShell();
    await screen.findByTestId(selectors.pages.dashboard);
    expect(screen.queryByRole("img", { name: /waiting/, hidden: true })).not.toBeInTheDocument();
    cleanup();
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({ asks: [makeAsk({ id: "ask-1" })], nextCursor: "more", totalAwaiting: 120 } as never);
    renderShell();
    expect((await screen.findAllByRole("img", { name: "120 waiting", hidden: true })).length).toBeGreaterThan(0);
    expect((await screen.findAllByText("120")).length).toBeGreaterThan(0);
  });

  it("keeps the read-only Updates feed available when the owner session expires", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.dashboard)).toBeInTheDocument();
    window.dispatchEvent(new Event("notification-hub:session-expired"));
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("opens Updates signed out without querying owner-only decision or permission queues", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValueOnce({ signedIn: false } as never);
    vi.mocked(conversationsClient.listAsks).mockClear();
    vi.mocked(conversationsClient.listNativeConsents).mockClear();

    renderShell();

    expect(await screen.findByTestId(selectors.pages.dashboard)).toBeInTheDocument();
    expect(conversationsClient.listAsks).not.toHaveBeenCalled();
    expect(conversationsClient.listNativeConsents).not.toHaveBeenCalled();
  });

  it("keeps the visually hidden skip link as the first keyboard navigation target", async () => {
    const user = userEvent.setup();
    renderShell();

    await screen.findByTestId(selectors.pages.dashboard);
    expect(screen.getByTestId(selectors.layout.main).tagName).toBe("MAIN");
    const skipLink = screen.getByTestId(`${selectors.layout.shell}-skip`);
    expect(skipLink).not.toHaveFocus();
    expect(getComputedStyle(skipLink).transform).toBe("translateY(-200%)");
    await user.tab();

    expect(skipLink).toHaveFocus();
    // jsdom does not implement :focus-visible, so the reveal is pinned in the stylesheet that browsers apply.
    expect(readFileSync("src/styles.css", "utf8")).toMatch(/\[data-rcl-app-shell-skip\]:focus-visible\s*\{\s*transform:\s*none;/);
    expect(skipLink).toHaveAttribute("href", `#${selectors.layout.shell}-main`);
    const main = screen.getByTestId(`${selectors.layout.shell}-main`);
    expect(main).toHaveAttribute("tabindex", "-1");
    await user.keyboard("{Enter}");
    expect(window.location.hash).toBe(`#${selectors.layout.shell}-main`);
    await waitFor(() => expect(main).toHaveFocus());
  });

  it("keeps the four phone destinations in the bottom bar and moves focus to the page on selection", async () => {
    setViewportWidth(390);
    await setLocale("en");
    const user = userEvent.setup();
    renderWithProviders(
      <AppShellWithRouterLocation initialEntry="/history?vrooli_test_mode=1&q=mobile" />,
      { withoutRouter: true },
    );

    expect(screen.getByTestId(selectors.layout.shell)).toHaveAttribute("data-mobile-nav", "tabs");
    const bottomBar = screen.getByTestId(`${selectors.layout.shell}-tabs`);
    expect(bottomBar.querySelectorAll("a")).toHaveLength(4);
    expect(within(bottomBar).getByTestId(`${selectors.layout.navLink({ key: "history" })}-tab`)).toHaveAttribute("aria-current", "page");

    const destinations = [
      ["dashboard", "/updates"],
      ["asks", "/asks"],
      ["history", "/history"],
      ["settings", "/settings"],
    ] as const;
    for (const [key, path] of destinations) {
      await user.click(within(bottomBar).getByTestId(`${selectors.layout.navLink({ key })}-tab`));
      await waitFor(() => expect(screen.getByTestId("router-location")).toHaveTextContent(`${path}?vrooli_test_mode=1`));
      await waitFor(() => expect(screen.getByTestId(`${selectors.layout.shell}-main`)).toHaveFocus());
      expect(within(bottomBar).getByTestId(`${selectors.layout.navLink({ key })}-tab`)).toHaveAttribute("aria-current", "page");
    }
  });

  it("keeps the unsent ask draft stored when the owner leaves its route", async () => {
    setViewportWidth(390);
    await setLocale("en");
    clearAskDrafts();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({
        question: "Which option should we take?",
        options: [{ key: "re-aim", label: "Re-aim" }, { key: "close", label: "Close" }],
      }),
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
    expect(readAskDraft("owner-1", "ask-1")).toMatchObject({ answer: "close", note: "keep this unsent note" });

    const bottomBar = screen.getByTestId(`${selectors.layout.shell}-tabs`);
    await user.click(within(bottomBar).getByTestId(`${selectors.layout.navLink({ key: "settings" })}-tab`));
    expect(await screen.findByTestId(selectors.pages.settings)).toBeInTheDocument();
    expect(readAskDraft("owner-1", "ask-1")).toMatchObject({ answer: "close", note: "keep this unsent note" });

    expect(readAskDraft("owner-1", "ask-1")).toMatchObject({ answer: "close", note: "keep this unsent note" });
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

    await user.click(screen.getByTestId(selectors.layout.navLink({ key: "dashboard" })));

    await waitFor(() => {
      expect(screen.getByTestId("router-location")).toHaveTextContent("/updates?vrooli_test_mode=1");
      expect(screen.getByTestId(`${selectors.layout.shell}-main`)).toHaveFocus();
    });
  });

  it("offers a live Hub recovery when a routed test lease has expired", async () => {
    renderWithProviders(
      <AppShellWithRouterLocation initialEntry="/settings/reach?vrooli_test_mode=1&e4_fixture=old" />,
      { withoutRouter: true },
    );
    await act(() => Promise.resolve(window.dispatchEvent(new Event(ROUTED_TEST_LEASE_EXPIRED_EVENT))));
    expect(await screen.findByRole("alert")).toHaveTextContent("errors.routed_test_lease_expired");
    expect(screen.getByRole("button", { name: "common.open_live_hub" })).toBeInTheDocument();
  });

  it("keeps desktop navigation on the shared column with no open compact panel", async () => {
    setViewportWidth(1280);
    await setLocale("en");
    renderWithProviders(<TestAppRouter initialEntries={["/history"]} />, { withoutRouter: true });

    const desktopNavigation = screen.getByTestId(`${selectors.layout.shell}-navigation`);
    for (const key of ["dashboard", "asks", "history", "settings"] as const) {
      expect(within(desktopNavigation).getByTestId(selectors.layout.navLink({ key }))).toBeInTheDocument();
    }
    expect(screen.queryByTestId(`${selectors.layout.shell}-menu-panel`)).not.toBeInTheDocument();
    expect(screen.queryByTestId(`${selectors.layout.shell}-menu`)).not.toBeInTheDocument();
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

  it("re-renders the navigation in Japanese when the language changes", async () => {
    renderShell();
    await act(() => setLocale("ja"));

    await waitFor(() => {
      expect(screen.getAllByText(ja.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
    expect(document.documentElement.lang).toBe("ja");
  });

  it("flips <html dir> to rtl and mirrors the navigation when an RTL locale (ar) is chosen", async () => {
    renderShell();
    await act(() => setLocale("ar"));

    await waitFor(() => {
      expect(document.documentElement.dir).toBe("rtl");
      expect(screen.getAllByText(ar.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
  });
});
