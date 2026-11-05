import { createContext, useContext, useEffect, useState } from "react";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2.1.2";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS, iconForItem } from "./navItems";
import { useSessionExpiryBoundary } from "../features/session/SessionGate";
import { preserveRoutedTestMode } from "../features/routedTestMode";

export type QuietEditorHeader = {
  title: string;
  backLabel: string;
  onBack: () => void;
};

const QuietEditorHeaderContext = createContext<((header: QuietEditorHeader | null) => void) | null>(null);

export function useQuietEditorHeader() {
  return useContext(QuietEditorHeaderContext);
}

export function AppShell() {
  useSessionExpiryBoundary();
  const [quietEditorHeader, setQuietEditorHeader] = useState<QuietEditorHeader | null>(null);
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
  const activePathname = pathname === "/" ? "/asks" : pathname;
  const reachDestination = pathname === "/settings/reach";
  const deliveryMatch = pathname.match(/^\/delivery(?:\/([^/]+))?\/?$/);
  let deliveryNotificationId = "";
  if (deliveryMatch?.[1]) {
    try {
      deliveryNotificationId = decodeURIComponent(deliveryMatch[1]);
    } catch {
      deliveryNotificationId = deliveryMatch[1];
    }
  }
  const deliveryDestination = deliveryMatch !== null;
  const navigate = useNavigate();
  const brandMark = <img src="/public/icon-192.png" alt="" aria-hidden="true" width={32} height={32} className="h-8 w-8 shrink-0 rounded-lg" />;
  return (
    <QuietEditorHeaderContext.Provider value={setQuietEditorHeader}>
    <LibraryAppShell
      density="sidebar"
      mobileNav="menu-tabs"
      mainMode={quietEditorHeader ? "fill" : "scroll"}
      className={quietEditorHeader ? "quiet-editor-shell" : deliveryDestination ? "delivery-context-shell" : undefined}
      mainClassName={quietEditorHeader ? "quiet-editor-main" : undefined}
      header={quietEditorHeader ? (
        <div data-selected-quiet-editor-header="true" className="flex min-w-0 flex-1 items-center gap-4">
          {brandMark}
          <button type="button" onClick={quietEditorHeader.onBack} className="min-h-11 shrink-0 rounded px-1 text-sm font-medium">
            <span aria-hidden="true">{document.documentElement.dir === "rtl" ? "→" : "←"}</span> {quietEditorHeader.backLabel}
          </button>
          <span className="truncate text-sm font-semibold">{quietEditorHeader.title}</span>
        </div>
      ) : deliveryDestination ? (
        <div data-selected-delivery-header="true" className={`flex min-w-0 flex-1 items-center ${deliveryNotificationId ? "gap-2" : "gap-4"}`}>
          {brandMark}
          <Link
            to={preserveRoutedTestMode(deliveryNotificationId ? "/delivery" : "/", search)}
            className="flex min-h-11 shrink-0 items-center rounded px-1 text-sm font-medium focus-visible:outline focus-visible:outline-2"
          >
            <span aria-hidden="true">{document.documentElement.dir === "rtl" ? "→" : "←"}</span> {t(strings.pages.delivery.title)}
          </Link>
          <span className="truncate text-sm font-semibold">{deliveryNotificationId || t(strings.pages.delivery.title)}</span>
        </div>
      ) : (
        <Link
          to={preserveRoutedTestMode(reachDestination ? "/settings" : "/", search)}
          data-testid={`${selectors.layout.shell}-header-brand`}
          className="flex min-h-11 min-w-0 flex-1 items-center gap-6 text-sm font-semibold focus-visible:outline focus-visible:outline-2"
        >
          {brandMark}
          {reachDestination ? <>
            <span className="text-sm font-normal text-app-muted-foreground">{t(strings.reach.backToSettings)}</span>
            <span>{t(strings.reach.title)}</span>
          </> : <span data-testid={selectors.app.title}>{t(strings.app.title)}</span>}
        </Link>
      )}
      brand={reachDestination ? (
        <div className="flex items-center gap-6 text-base">
          {brandMark}
          <span className="text-sm text-app-muted-foreground">{t(strings.reach.backToSettings)}</span>
          <span className="text-sm font-semibold text-app-foreground">{t(strings.reach.title)}</span>
        </div>
      ) : <span data-testid={selectors.app.title} className="flex items-center gap-2">{brandMark}{t(strings.app.title)}</span>}
      brandHref={reachDestination ? "/settings" : "/"}
      items={NAV_ITEMS.map(item => ({
        id: item.key, href: preserveRoutedTestMode(item.path, search), label: t(item.labelKey), icon: iconForItem(item),
        current: item.end ? activePathname === item.path : activePathname === item.path || activePathname.startsWith(item.path + "/"),
        testId: selectors.layout.navLink({ key: item.key }),
      }))}
      menuLabel={t("navigation.app-shell.open-navigation")}
      closeLabel={t("navigation.app-shell.close-navigation")}
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
      sidebarStorageKey="notification-hub.sidebar-width"
      sidebarResize={{ defaultSize: 360 }}
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
    </QuietEditorHeaderContext.Provider>
  );
}
