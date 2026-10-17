import {
  AppShell as LibraryAppShell,
  type AppShellLinkProps,
  type AppShellNavItem,
} from "@vrooli/react-component-library/AppShell/2";
import { Settings, Waves } from "lucide-react";
import { useState } from "react";
import { NavLink, Outlet, useLocation } from "react-router-dom";

import { useKeyboardShortcuts } from "../hooks/useKeyboardShortcuts";
import { useTranslation } from "../i18n";
import { strings } from "../consts/strings";
import { NAV_ITEMS } from "./shell/nav-items";
import { ServiceStatusPill } from "./shell/ServiceStatusPill";
import { SettingsDrawer } from "./shell/SettingsDrawer";

const SIDEBAR_STORAGE = "audio-tools.sidebar.width.v1";

/**
 * Application shell. Full-bleed layout (no centered card) with a sticky top
 * bar, persistent sidebar on desktop, and a bottom tab nav on mobile. Pages
 * render through React Router's `<Outlet />`.
 */
export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const [settingsOpen, setSettingsOpen] = useState(false);

  useKeyboardShortcuts([
    {
      key: ",",
      ctrlOrMeta: true,
      handler: () => setSettingsOpen((prev) => !prev),
      allowInInputs: true,
    },
    {
      key: "Escape",
      handler: () => setSettingsOpen(false),
      allowInInputs: true,
    },
  ]);

  const items: AppShellNavItem[] = NAV_ITEMS.map((item) => ({
    id: item.to === "/" ? "overview" : item.to.slice(1).replace(/\//g, "-"),
    href: item.to,
    label: t(item.labelKey),
    icon: <item.icon aria-hidden className="h-4 w-4" />,
    current: item.to === "/" ? pathname === "/" : pathname.startsWith(item.to),
  }));

  const renderLink = (item: AppShellNavItem, props: AppShellLinkProps) => (
    <NavLink
      to={item.href}
      end={item.href === "/"}
      className={props.className}
      aria-current={props["aria-current"]}
      aria-disabled={props["aria-disabled"]}
      data-testid={props["data-testid"]}
      onClick={props.onClick}
    >
      {props.children}
    </NavLink>
  );

  return (
    <>
      <LibraryAppShell
        brand={t(strings.app.title)}
        brandMark={<Waves aria-hidden className="h-4 w-4" />}
        brandHref="/"
        items={items}
        renderLink={renderLink}
        density="sidebar"
        mobileNav="drawer"
        mainMode="scroll"
        header={
          <div className="flex items-center gap-2">
            <ServiceStatusPill />
            <button
              type="button"
              className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
              aria-label={t(strings.shell.openSettings)}
              aria-keyshortcuts="Control+,"
              onClick={() => setSettingsOpen(true)}
            >
              <Settings className="h-5 w-5" aria-hidden="true" />
            </button>
          </div>
        }
        sidebarStorageKey={SIDEBAR_STORAGE}
        navigationLabel={t(strings.shell.primaryNav)}
        mobileNavigationLabel={t(strings.shell.primaryNav)}
        skipLabel={t("a11y.skipToMain", { defaultValue: "Skip to main content" })}
        menuLabel={t(strings.shell.openMenu)}
        closeLabel={t("shell.closeMenu", { defaultValue: "Close navigation" })}
        testId="audio-tools-app-shell"
      >
        <main id="main" aria-label={t(strings.shell.mainContent)}>
          <Outlet />
        </main>
      </LibraryAppShell>
      <SettingsDrawer open={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </>
  );
}

export default AppShell;
