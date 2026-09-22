/**
 * AppShell tests — the shell is the component library's; this file verifies
 * the configuration this scenario feeds it (nav items, router adapter, labels)
 * and that the landmarks a page relies on are present. Page content is
 * exercised in the per-page tests.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, screen, waitFor } from "@testing-library/react";
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
    window.localStorage.removeItem("personal-planner.sidebar-collapsed");
    window.localStorage.removeItem("vrooli.theme");
    vi.unstubAllGlobals();
  });

  it("renders the shell landmarks, the brand, and the main outlet", () => {
    renderShell();
    expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.navigation)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.tabs)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.main)).toBeInTheDocument();
    expect(document.querySelector(".planner-route-scroll")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.brand)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.skip)).toHaveAttribute("href", `#${selectors.layout.main}`);
  });

  it("keeps preferences out of the shell chrome", () => {
    renderShell();
    expect(screen.queryByTestId(selectors.settingsPage.localeSelect)).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.settingsPage.themeSelect)).not.toBeInTheDocument();
  });

  it("uses the Observatory day/night chrome vocabulary on every route", async () => {
    window.localStorage.setItem("vrooli.theme", "dark");
    renderShell("/settings");
    await waitFor(() => {
      expect(screen.getByTestId(selectors.layout.shell)).toHaveClass("appearance-night");
    });
    expect(screen.getByTestId(selectors.layout.shell)).not.toHaveClass("appearance-dark");
  });

  it("paints the browser chrome and safe-area strip from the resolved Night appearance", async () => {
    window.localStorage.setItem("vrooli.theme", "night");
    renderShell("/settings");
    await waitFor(() => expect(screen.getByTestId(selectors.layout.shell)).toHaveClass("appearance-night"));
    expect(screen.getByTestId("planner-status-fill")).toBeInTheDocument();
    expect(document.documentElement.style.getPropertyValue("--rcl-status-fill")).toBe("rgb(23 22 52)");
    expect(document.querySelector('meta[name="theme-color"]')?.getAttribute("content")).toBe("rgb(23 22 52)");
  });

  it("starts the rendered shell in the URL-requested Night appearance", async () => {
    window.history.replaceState({}, "", "/settings?appearance=night");
    renderShell("/settings?appearance=night");
    await waitFor(() => expect(screen.getByTestId(selectors.layout.shell)).toHaveClass("appearance-night"));
    expect(document.querySelector('[data-testid="layout-shell-sidebar"]')).toHaveStyle({ "--rcl-panel-size": "144px" });
    expect(document.documentElement.getAttribute("data-theme")).toBe("night");
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

  it("restores the main scroll position when navigation changes the route", async () => {
    const user = userEvent.setup();
    const originalScrollTo = HTMLElement.prototype.scrollTo;
    const scrollTo = vi.fn();
    Object.defineProperty(HTMLElement.prototype, "scrollTo", { configurable: true, value: scrollTo });
    const view = renderShell();
    const sidebarContent = document.querySelector<HTMLElement>(".rcl-sidebar-shell__content");
    if (sidebarContent) sidebarContent.scrollTop = 42;
    scrollTo.mockClear();
    try {
      await user.click(screen.getByTestId(selectors.layout.navTab({ key: "settings" })));
      await waitFor(() => expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument());
      expect(scrollTo).toHaveBeenCalledWith({ top: 0, left: 0, behavior: "auto" });
      expect(sidebarContent?.scrollTop ?? 0).toBe(0);
    } finally {
      view.unmount();
      Object.defineProperty(HTMLElement.prototype, "scrollTo", { configurable: true, value: originalScrollTo });
    }
  });

  it("restores nested sidebar scroll containers when navigation changes the route", async () => {
    const user = userEvent.setup();
    const originalScrollTo = HTMLElement.prototype.scrollTo;
    const scrollTo = vi.fn();
    Object.defineProperty(HTMLElement.prototype, "scrollTo", { configurable: true, value: scrollTo });
    const view = renderShell();
    const sidebar = screen.getByTestId("layout-shell-sidebar");
    const nestedScrollContainer = document.createElement("div");
    nestedScrollContainer.scrollTop = 36;
    sidebar.querySelector(".rcl-sidebar-shell__content")?.appendChild(nestedScrollContainer);
    try {
      await user.click(screen.getByTestId(selectors.layout.navTab({ key: "review" })));
      await waitFor(() => expect(screen.getByTestId(selectors.pages.review)).toBeInTheDocument());
      expect(nestedScrollContainer.scrollTop).toBe(0);
      expect(scrollTo).toHaveBeenCalledWith({ top: 0, left: 0, behavior: "auto" });
    } finally {
      view.unmount();
      Object.defineProperty(HTMLElement.prototype, "scrollTo", { configurable: true, value: originalScrollTo });
    }
  });

  it("does not overwrite a user scroll after the route has rendered", () => {
    vi.useFakeTimers();
    try {
      renderShell("/goals");
      const routeScroller = document.querySelector<HTMLElement>(".planner-route-scroll");
      expect(routeScroller).not.toBeNull();
      routeScroller!.scrollTop = 96;
      act(() => vi.advanceTimersByTime(1_100));

      expect(routeScroller!.scrollTop).toBe(96);
    } finally {
      vi.useRealTimers();
    }
  });

  it("supports an icon-only collapse state and tracks the resizable shell", async () => {
    const user = userEvent.setup();
    let observed = false;
    let disconnected = false;
    vi.stubGlobal("ResizeObserver", class {
      constructor(private readonly callback: ResizeObserverCallback) {}
      observe() { observed = true; this.callback([], this as unknown as ResizeObserver); }
      disconnect() { disconnected = true; }
      unobserve() {}
    });
    renderShell();
    const toggle = screen.getByRole("button", { name: "Collapse sidebar" });
    expect(observed).toBe(true);
    await user.click(toggle);
    expect(screen.getByRole("button", { name: "Expand sidebar" })).toHaveAttribute("aria-pressed", "true");
    await user.click(screen.getByRole("button", { name: "Expand sidebar" }));
    expect(screen.getByRole("button", { name: "Collapse sidebar" })).toHaveAttribute("aria-pressed", "false");
    cleanup();
    expect(disconnected).toBe(true);
  });

  it("exposes a useful Observatory sidebar resize range", () => {
    renderShell();
    const separator = screen.getByRole("separator", { name: /Resize layout\.navigationLabel/i });
    expect(separator).toHaveAttribute("aria-valuemin", "128");
    expect(separator).toHaveAttribute("aria-valuemax", "280");
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
    await user.click(screen.getByTestId(selectors.settingsPage.localeSelect));
    await user.click(await screen.findByRole("option", { name: "日本語" }));

    await waitFor(() => {
      expect(screen.getAllByText(ja.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
    expect(document.documentElement.lang).toBe("ja");
  });

  it("flips <html dir> to rtl when an RTL locale (ar) is chosen", async () => {
    const user = userEvent.setup();
    renderShell("/settings");
    await user.click(screen.getByTestId(selectors.settingsPage.localeSelect));
    await user.click(await screen.findByRole("option", { name: "العربية" }));

    await waitFor(() => {
      expect(document.documentElement.dir).toBe("rtl");
      expect(screen.getAllByText(ar.layout.nav.dashboard).length).toBeGreaterThan(0);
    });
  });
});
