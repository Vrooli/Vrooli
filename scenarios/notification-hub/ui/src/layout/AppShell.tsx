import { useEffect, useState } from "react";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { useViewportEnvironment } from "@vrooli/react-component-library/useViewportEnvironment/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { matchPath, Outlet, useLocation, useNavigate } from "react-router-dom";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useNeedsYou } from "../features/asks/useNeedsYou";
import { NAV_ITEMS, iconForItem, isCurrent } from "./navItems";
import { NavSectionContext, type NavKey } from "./navSection";
import { SidebarAccount } from "./SidebarAccount";
import { useSessionExpiryBoundary } from "../features/session/useSession";
import { preserveRoutedTestMode, livePath } from "../features/routedTestMode";
import { ROUTED_TEST_LEASE_EXPIRED_EVENT } from "../api/notifications";

/**
 * The frame every page sits in: four destinations (a tab bar on a phone, a
 * sidebar on a wide screen), a count of what needs the owner, and the main pane.
 * The shell knows nothing about individual routes; a page draws its own title
 * and back link.
 */
export function AppShell() {
  useSessionExpiryBoundary();
  const viewport = useViewportEnvironment();
  useEffect(() => {
    const height = viewport.keyboardVisible ? viewport.visibleHeight : viewport.frameHeight ?? viewport.layoutHeight;
    if (height > 0) document.documentElement.style.setProperty("--hub-viewport-height", `${height}px`);
    return () => { document.documentElement.style.removeProperty("--hub-viewport-height"); };
  }, [viewport.frameHeight, viewport.layoutHeight, viewport.visibleHeight, viewport.keyboardVisible]);
  useEffect(() => {
    const focusSkipTarget = (event: MouseEvent) => {
      const target = event.target as { closest?: (selector: string) => Element | null } | null;
      const skipLink = target?.closest?.(`[data-testid="${selectors.layout.shell}-skip"]`);
      const main = document.getElementById(`${selectors.layout.shell}-main`);
      if (!skipLink || !main) return;
      event.preventDefault();
      window.location.hash = main.id;
      main.focus({ preventScroll: true });
    };
    document.addEventListener("click", focusSkipTarget);
    return () => document.removeEventListener("click", focusSkipTarget);
  }, []);
  const { t } = useTranslation();
  const { pathname, search } = useLocation();
  const navigate = useNavigate();
  const [claimed, setClaimed] = useState<NavKey | null>(null);
  const [testLeaseExpired, setTestLeaseExpired] = useState(false);
  useEffect(() => {
    const expired = () => setTestLeaseExpired(true);
    window.addEventListener(ROUTED_TEST_LEASE_EXPIRED_EVENT, expired);
    return () => window.removeEventListener(ROUTED_TEST_LEASE_EXPIRED_EVENT, expired);
  }, []);
  const wide = useMediaQuery("(min-width: 48rem)");
  const needsYou = useNeedsYou();
  const brandMark = <img src="/public/icon-192.png" alt="" aria-hidden="true" width={32} height={32} className="h-8 w-8 shrink-0 rounded-lg" />;
  return (
    <LibraryAppShell
      className={matchPath("/asks/:askId", pathname) ? "conversation-shell" : undefined}
      density="sidebar"
      mobileNav="tabs"
      brand={<span data-testid={selectors.app.title} className="flex items-center gap-2">{brandMark}{t(strings.app.title)}</span>}
      brandHref={preserveRoutedTestMode("/", search)}
      items={NAV_ITEMS.map(item => ({
        id: item.key,
        href: preserveRoutedTestMode(item.path, search),
        label: t(item.labelKey),
        icon: iconForItem(item),
        current: isCurrent(item, pathname, claimed),
        testId: selectors.layout.navLink({ key: item.key }),
        badge: item.key === "asks" && needsYou.count > 0
          ? {
            value: needsYou.count,
            tone: "warning" as const,
            label: t(strings.layout.needsYouBadge, { count: needsYou.count }),
          }
          : undefined,
      }))}
      utility={wide ? <SidebarAccount /> : undefined}
      onNavigate={item => {
        navigate(preserveRoutedTestMode(item.href, search));
        const focusMain = () => {
          document.getElementById(`${selectors.layout.shell}-main`)?.focus({ preventScroll: true });
        };
        if (typeof window.requestAnimationFrame === "function") {
          window.requestAnimationFrame(focusMain);
        } else {
          window.setTimeout(focusMain, 0);
        }
      }}
      navigationLabel={t(strings.layout.sidebarLabel)}
      mobileNavigationLabel={t(strings.layout.bottomNavLabel)}
      sidebarStorageKey="notification-hub.nav-width"
      sidebarResize={{ defaultSize: 240, max: 320 }}
      testId={selectors.layout.shell}
    >
      <NavSectionContext.Provider value={setClaimed}>
        {testLeaseExpired && (
          <section role="alert" className="m-4 flex flex-col items-start gap-3 rounded-panel border border-app-border bg-app-surface p-4">
            <p>{t(strings.errors.routed_test_lease_expired)}</p>
            <Button type="button" variant="primary" onClick={() => window.location.assign(livePath(`${pathname}${search}`))}>
              {t(strings.common.open_live_hub)}
            </Button>
          </section>
        )}
        <Outlet />
      </NavSectionContext.Provider>
    </LibraryAppShell>
  );
}
