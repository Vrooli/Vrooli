import { useEffect, useState } from "react";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2.1.4";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useNeedsYou } from "../features/asks/useNeedsYou";
import { NAV_ITEMS, iconForItem, isCurrent } from "./navItems";
import { NavSectionContext, type NavKey } from "./navSection";
import { SidebarAccount } from "./SidebarAccount";
import { useSessionExpiryBoundary } from "../features/session/useSession";
import { preserveRoutedTestMode } from "../features/routedTestMode";

/**
 * The frame every page sits in: four destinations (a tab bar on a phone, a
 * sidebar on a wide screen), a count of what needs the owner, and the main pane.
 * The shell knows nothing about individual routes; a page draws its own title
 * and back link.
 */
export function AppShell() {
  useSessionExpiryBoundary();
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
  const wide = useMediaQuery("(min-width: 48rem)");
  const needsYou = useNeedsYou();
  const brandMark = <img src="/public/icon-192.png" alt="" aria-hidden="true" width={32} height={32} className="h-8 w-8 shrink-0 rounded-lg" />;
  return (
    <LibraryAppShell
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
        <Outlet />
      </NavSectionContext.Provider>
    </LibraryAppShell>
  );
}
