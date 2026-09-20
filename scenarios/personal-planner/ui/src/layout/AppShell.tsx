import { useEffect, useLayoutEffect, useState } from "react";
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
    const main = document.querySelector<HTMLElement>(`[data-testid="${selectors.layout.shell}-main"]`);
    const sidebar = document.querySelector<HTMLElement>(`[data-testid="${selectors.layout.shell}-sidebar"]`);
    const reset = () => {
      // The shell owns the scroll container. Keep route entry deterministic and
      // prevent browser focus restoration from leaving content controls clipped
      // at the top of a mobile or direct-route capture.
      if (main) {
        main.scrollTop = 0;
        main.scrollLeft = 0;
        if (typeof main.scrollTo === "function") main.scrollTo({ top: 0, left: 0, behavior: "auto" });
      }
      if (sidebar) {
        // The library owns the sidebar structure, and a library upgrade may
        // introduce an additional scroll container around the navigation
        // list. Reset the shell and every descendant so route entry cannot
        // reopen with the top of the navigation hidden.
        const scrollContainers = [sidebar, ...Array.from(sidebar.querySelectorAll<HTMLElement>("*"))];
        for (const scrollContainer of scrollContainers) {
          scrollContainer.scrollTop = 0;
          scrollContainer.scrollLeft = 0;
          if (typeof scrollContainer.scrollTo === "function") {
            scrollContainer.scrollTo({ top: 0, left: 0, behavior: "auto" });
          }
        }
      }
      window.scrollTo?.({ top: 0, left: 0, behavior: "auto" });
    };
    reset();
    const frame = window.requestAnimationFrame(reset);
    const deferred = window.setTimeout(reset, 100);
    return () => {
      window.cancelAnimationFrame(frame);
      window.clearTimeout(deferred);
    };
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
        <Outlet />
      </LibraryAppShell>
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
