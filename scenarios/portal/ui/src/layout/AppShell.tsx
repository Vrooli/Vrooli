import { ChatAccountProvider } from "../features/chat/ChatAccount";
import { ContextRetentionProvider, ContextRetentionActions } from "../features/companion/ContextRetention";
import { DesktopRecoveryPanel } from "../features/surfaces/DesktopSessionPanel";
import { DesktopSessionsProvider, useDesktopSession } from "../features/surfaces/useDesktopSession";
import { AgentTasksProvider } from "../features/chat/useAgentTask";
import { CompanionPresentation, CompanionToolbar } from "../features/companion/CompanionPresentation";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { ModeIndicator } from "../features/integrations/ModeIndicator";
import { NAV_ITEMS } from "./navItems";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

function PortalUtility() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  return <div className="flex flex-wrap items-center gap-3">
    <ModeIndicator />
    <div role="group" aria-label={t(strings.locale.switcherLabel)} data-testid={selectors.locale.switcher} className="flex items-center gap-1">
      {SUPPORTED_LOCALES.map((code) => <button key={code} type="button" data-testid={selectors.locale.toggle({ code })} onClick={() => void setLocale(code)} aria-pressed={currentLocale === code}>{getLocaleConfig(code).nativeLabel}</button>)}
    </div>
    <label data-testid={selectors.theme.switcher} className="flex items-center gap-2 text-xs text-app-muted-foreground">
      <span className="sr-only">{t(strings.theme.switcherLabel)}</span>
      <select value={choice} onChange={(event) => setTheme(event.target.value as ThemeChoice)} data-testid={selectors.theme.select} aria-label={t(strings.theme.switcherLabel)}>
        {THEME_CHOICES.map((theme) => <option key={theme} value={theme}>{t(strings.theme.choice[theme])}</option>)}
      </select>
    </label>
  </div>;
}

/**
 * Responsive app shell. CSS grid with a header row and a (sidebar | content)
 * main row; mobile collapses the sidebar to a pinned bottom nav.
 *
 * This is the real shell — replaces the centered single-card placeholder.
 * Page content renders into the `<Outlet />`; routes are configured in
 * `app/routes.tsx`.
 */
export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();

  return (
    <CompanionPresentation><DesktopSessionsProvider><RetainedContextShell>
      <LibraryAppShell density="sidebar" mobileNav="tabs" mainMode="scroll"
        brand={<span data-testid={selectors.app.title}>{t(strings.app.title)}</span>} brandHref="/"
        items={NAV_ITEMS.map((item) => ({ id: item.key, href: item.path, label: t(item.labelKey), icon: <item.icon size={16} aria-hidden />, current: item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(`${item.path}/`), testId: selectors.layout.navLink({ key: item.key }) }))}
        utility={<div className="flex items-center gap-3"><CompanionToolbar /><PortalUtility /></div>}
        renderLink={(item, { href, children, ...props }) => <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find((entry) => entry.key === item.id)?.end} {...props}>{children}</NavLink>}
        onNavigate={(item) => navigate(item.href)} navigationLabel={t(strings.layout.sidebarLabel)} mobileNavigationLabel={t(strings.layout.bottomNavLabel)} sidebarStorageKey="portal.sidebar-width" testId={selectors.layout.shell}>
        <div className="companion-content"><DesktopRecoveryPanel /><ContextRetentionActions /><Outlet /></div>
      </LibraryAppShell>
    </RetainedContextShell></DesktopSessionsProvider></CompanionPresentation>
  );
}

function RetainedContextShell({children}:{children:React.ReactNode}){const {account}=useDesktopSession();return <ChatAccountProvider account={account}><AgentTasksProvider><ContextRetentionProvider account={account}>{children}</ContextRetentionProvider></AgentTasksProvider></ChatAccountProvider>;}
