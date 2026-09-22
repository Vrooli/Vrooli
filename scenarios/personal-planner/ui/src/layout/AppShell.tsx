import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { PanelLeftClose, PanelLeftOpen } from "lucide-react";

import { AppShell as LibraryAppShell, type AppShellNavItem } from "@vrooli/react-component-library/AppShell/2";
import { StatusBarFill } from "@vrooli/react-component-library/ChromeTheme/1";
import { Button } from "@vrooli/react-component-library/Button/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { BrandMark } from "./BrandMark";
import { NAV_ITEMS, isNavItemActive } from "./navItems";
import { useTheme } from "../theme/ThemeProvider";
import { GlobalCapture } from "../components/GlobalCapture";

/**
 * The shell is the component library's. This file configures it and plugs in
 * the router; it does not draw chrome.
 *
 * Decide these three settings in Gate 5 of `docs/START-HERE.md` and change
 * them here. Nothing else in the tree needs to know.
 *
 * - `density`: `"sidebar"` (icon + label, resizable) for a tool with several
 *   peer surfaces; `"rail"` (icon over a short label, narrow) when one surface
 *   needs the width.
 * - `mobileNav`: `"tabs"` for three to five destinations; `"drawer"` for more.
 * - `mainMode`: `"scroll"` pads and scrolls pages for you; `"fill"` hands a
 *   page the whole pane so it can pin its own header and composer. The
 *   planner owns page padding so the Observatory can reach the viewport edge
 *   on mobile.
 *
 * If the shell cannot do what your primary surface needs, do not fork it:
 * record the gap in `docs/reference/component-library-gaps.md` and eject with
 * `react-component-library adoptions eject --reason`.
 */
const SHELL = {
  density: "sidebar",
  mobileNav: "tabs",
  mainMode: "fill",
} as const;

export function AppShell() {
  const { t } = useTranslation();
  const { resolved } = useTheme();
  const { pathname, search } = useLocation();
  const navigate = useNavigate();
  const routeScrollerRef = useRef<HTMLDivElement>(null);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    if (typeof window === "undefined") return false;
    return window.localStorage.getItem("personal-planner.sidebar-collapsed") === "true";
  });
  const [sidebarWidth, setSidebarWidth] = useState(144);

  useEffect(() => {
    window.localStorage.setItem("personal-planner.sidebar-collapsed", String(sidebarCollapsed));
  }, [sidebarCollapsed]);

  useLayoutEffect(() => {
    if (typeof window !== "undefined" && "scrollRestoration" in window.history) {
      window.history.scrollRestoration = "manual";
    }
    const sidebar = document.querySelector<HTMLElement>("[data-rcl-sidebar-shell]");
    // Route entry gets one synchronous reset. Delayed retries used to win a
    // race against the person's first swipe and visibly snap the page upward.
    const routeScroller = routeScrollerRef.current;
    if (routeScroller) {
      routeScroller.scrollTop = 0;
      routeScroller.scrollLeft = 0;
      routeScroller.scrollTo?.({ top: 0, left: 0, behavior: "auto" });
    }
    if (sidebar) {
      // A library upgrade may add a nested sidebar scroller. Reset these once
      // at route commit; unlike page content, the user is not interacting with
      // the desktop navigation during this layout effect.
      for (const scrollContainer of [sidebar, ...Array.from(sidebar.querySelectorAll<HTMLElement>("*"))]) {
        scrollContainer.scrollTop = 0;
        scrollContainer.scrollLeft = 0;
        scrollContainer.scrollTo?.({ top: 0, left: 0, behavior: "auto" });
      }
    }
  }, [pathname, search]);

  useEffect(() => {
    const sidebar = document.querySelector<HTMLElement>(`[data-testid="${selectors.layout.shell}-sidebar"]`);
    if (!sidebar || typeof ResizeObserver === "undefined") return;
    const update = () => setSidebarWidth(sidebar.getBoundingClientRect().width);
    update();
    const observer = new ResizeObserver(update);
    observer.observe(sidebar);
    return () => observer.disconnect();
  }, []);

  const items: AppShellNavItem[] = NAV_ITEMS.map((item) => ({
    id: item.key,
    label: t(item.labelKey),
    href: item.path,
    icon: item.icon,
    current: isNavItemActive(item, pathname),
    testId: selectors.layout.navLink({ key: item.key }),
  }));

  return (
    <>
      <StatusBarFill className="planner-status-fill" testId="planner-status-fill" />
      <LibraryAppShell
        className={`planner-shell appearance-${resolved === "dark" ? "night" : "day"}${sidebarCollapsed ? " sidebar-collapsed" : ""}`}
        brand={<span className="planner-brand" data-testid={selectors.app.title}><BrandMark /><strong>{t(strings.app.title)}</strong></span>}
        brandHref="/"
        items={items}
        density={SHELL.density}
        mobileNav={SHELL.mobileNav}
        mainMode={SHELL.mainMode}
        mainClassName="planner-app-main"
        renderLink={(item, { href, children, onClick, ...rest }) => (
          <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find((entry) => entry.key === item.id)?.end === true} onClick={onClick} {...rest}>
            {children}
          </NavLink>
        )}
        onNavigate={(item) => navigate(item.href)}
        navigationLabel={t(strings.layout.navigationLabel)}
        mobileNavigationLabel={t(strings.layout.mobileNavigationLabel)}
        skipLabel={t(strings.layout.skipToContent)}
        menuLabel={t(strings.layout.openNavigation)}
        closeLabel={t(strings.layout.closeNavigation)}
        // The v2 key intentionally establishes the tighter Observatory baseline;
        // subsequent user resizing remains persisted normally.
        sidebarStorageKey="personal-planner.sidebar-width-observatory-v2"
        // Keep the approved 144px Observatory baseline, but give the shared
        // resize handle a meaningful working range instead of the old 56px
        // window that made dragging appear stuck.
        sidebarResize={{ adjacentMin: 320, min: 128, max: 280, defaultSize: 144, step: 8, coarseStep: 40 }}
        testId={selectors.layout.shell}
      >
        <div className="planner-route-scroll" ref={routeScrollerRef}>
          <Outlet />
        </div>
      </LibraryAppShell>
      <GlobalCapture />
      <Button
        type="button"
        className="planner-sidebar-toggle"
        variant="ghost"
        size="icon"
        shape="pill"
        style={{ insetInlineStart: `${Math.max(12, (sidebarCollapsed ? 72 : sidebarWidth) - 44)}px` }}
        aria-label={sidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
        aria-pressed={sidebarCollapsed}
        title={sidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
        onClick={() => setSidebarCollapsed((collapsed) => !collapsed)}
      >
        {sidebarCollapsed ? <PanelLeftOpen size={17} aria-hidden="true" /> : <PanelLeftClose size={17} aria-hidden="true" />}
      </Button>
    </>
  );
}
