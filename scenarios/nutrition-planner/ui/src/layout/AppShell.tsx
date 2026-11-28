import { Moon, Settings, Sun } from "lucide-react";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useTheme } from "../theme/ThemeProvider";
import { NAV_ITEMS } from "./navItems";

const primaryNavigation = NAV_ITEMS.filter((item) => item.key !== "settings");

/** Shared RCL shell for the six-route planner. */
export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const { resolved, setTheme } = useTheme();
  const evening = resolved === "dark";

  return (
    <LibraryAppShell
      density="sidebar"
      mobileNav="tabs"
      mainMode="scroll"
      brand={<span>Nooch</span>}
      brandHref="/"
      items={primaryNavigation.map((item) => ({
        id: item.key,
        href: item.path,
        label: t(item.labelKey),
        icon: item.icon,
        current: item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(`${item.path}/`),
        mobile: true,
        testId: selectors.layout.navLink({ key: item.key }),
      }))}
      utility={(
        <div className="flex items-center gap-2">
          <button
            className="app-icon-button"
            type="button"
            aria-label={`Switch to ${evening ? "Light" : "Evening"} appearance`}
            onClick={() => setTheme(evening ? "light" : "dark")}
          >
            {evening ? <Sun aria-hidden="true" /> : <Moon aria-hidden="true" />}
          </button>
          <NavLink
            className="app-icon-button"
            aria-label={t(strings.layout.nav.settings)}
            to="/settings"
            aria-current={pathname === "/settings" ? "page" : undefined}
            data-testid={selectors.layout.navLink({ key: "settings" })}
          >
            <Settings aria-hidden="true" />
          </NavLink>
        </div>
      )}
      renderLink={(item, { href, children, ...props }) => (
        <NavLink to={href} end={item.id === "today"} {...props}>{children}</NavLink>
      )}
      onNavigate={(item) => navigate(item.href)}
      navigationLabel={t(strings.layout.navigationLabel)}
      mobileNavigationLabel={t(strings.layout.mobileNavigationLabel)}
      testId={selectors.layout.shell}
    >
      <div className="app-page-frame"><Outlet /></div>
    </LibraryAppShell>
  );
}
