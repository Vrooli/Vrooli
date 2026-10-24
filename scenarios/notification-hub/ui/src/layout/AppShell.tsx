import { useEffect } from "react";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2.1.2";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { ThemeControl } from "../components/ThemeControl";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS, iconForItem } from "./navItems";
import { useSessionExpiryBoundary } from "../features/session/SessionGate";
import { preserveRoutedTestMode } from "../features/routedTestMode";

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
  return (
    <LibraryAppShell
      density="sidebar"
      mobileNav="menu-tabs"
      mainMode="scroll"
      brand={<span data-testid={selectors.app.title}>{t(strings.app.title)}</span>}
      brandHref="/"
      items={NAV_ITEMS.map(item => ({
        id: item.key, href: preserveRoutedTestMode(item.path, search), label: t(item.labelKey), icon: iconForItem(item),
        current: item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(item.path + "/"),
        testId: selectors.layout.navLink({ key: item.key }),
      }))}
      utility={<ThemeControl />}
      header={(
        <div className="nh-mobile-header-content">
          <span className="font-semibold text-xs" data-testid={selectors.app.title}>
            {t(strings.app.title)}
          </span>
        </div>
      )}
      menuLabel={t("navigation.app-shell.open-navigation")}
      closeLabel={t("navigation.app-shell.close-navigation")}
      onNavigate={item => navigate(preserveRoutedTestMode(item.href, search))}
      navigationLabel={t(strings.layout.sidebarLabel)}
      mobileNavigationLabel={t(strings.layout.bottomNavLabel)}
      sidebarStorageKey="notification-hub.sidebar-width"
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
  );
}
