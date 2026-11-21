import { Moon, Settings, Sun } from "lucide-react";
import { NavLink, Outlet, useLocation } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useTheme } from "../theme/ThemeProvider";
import { NAV_ITEMS, isNavItemActive } from "./navItems";

const primaryNavigation = NAV_ITEMS.filter((item) => item.key !== "settings");
const phoneNavigation = primaryNavigation;

export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const { resolved, setTheme } = useTheme();
  const evening = resolved === "dark";

  return (
    <div className="app-shell" data-testid={selectors.layout.shell}>
      <a className="app-skip" href={`#${selectors.layout.main}`} data-testid={selectors.layout.skip}>{t(strings.layout.skipToContent)}</a>
      <header className="app-header">
        <NavLink className="app-wordmark" to="/" data-testid={selectors.layout.brand}>Nooch</NavLink>
        <nav className="app-primary-nav" aria-label={t(strings.layout.navigationLabel)} data-testid={selectors.layout.navigation}>
          {primaryNavigation.map((item) => (
            <NavLink key={item.key} to={item.path} end={item.end} className="app-nav-link" aria-current={isNavItemActive(item, pathname) ? "page" : undefined} data-testid={selectors.layout.navLink({ key: item.key })}>
              <span>{t(item.labelKey)}</span>
            </NavLink>
          ))}
        </nav>
        <div className="app-header-tools">
          <button className="app-icon-button" type="button" aria-label={`Switch to ${evening ? "Light" : "Evening"} appearance`} onClick={() => setTheme(evening ? "light" : "dark")}>
            {evening ? <Sun aria-hidden="true" /> : <Moon aria-hidden="true" />}
          </button>
          <NavLink className="app-icon-button" aria-label={t(strings.layout.nav.settings)} to="/settings" aria-current={isNavItemActive(NAV_ITEMS.find((item) => item.key === "settings")!, pathname) ? "page" : undefined} data-testid={selectors.layout.navLink({ key: "settings" })}>
            <Settings aria-hidden="true" />
          </NavLink>
        </div>
      </header>
      <main id={selectors.layout.main} className="app-main" tabIndex={-1} data-testid={selectors.layout.main}>
        <div className="app-page-frame"><Outlet /></div>
      </main>
      <nav className="app-phone-nav" aria-label={t(strings.layout.mobileNavigationLabel)} data-testid={selectors.layout.tabs}>
        {phoneNavigation.map((item) => (
          <NavLink key={item.key} to={item.path} end={item.end} className="app-phone-tab" aria-current={isNavItemActive(item, pathname) ? "page" : undefined} data-testid={selectors.layout.navTab({ key: item.key })}>
            <span className="app-phone-tab-icon" aria-hidden="true">{item.icon}</span><span>{t(item.labelKey)}</span>
          </NavLink>
        ))}
      </nav>
    </div>
  );
}
