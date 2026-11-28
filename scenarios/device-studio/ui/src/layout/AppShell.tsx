import { Link, NavLink, Outlet } from "react-router-dom";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { BrandMark } from "./BrandMark";
import { NAV_ITEMS } from "./navItems";

/** Scenario-owned navigation shell: the approved top-nav pattern is not available in RCL AppShell/2. */
export function AppShell() {
  const { t } = useTranslation();
  return <div className="studio-shell" data-testid={selectors.layout.shell}>
    <a className="studio-skip" href={`#${selectors.layout.main}`} data-testid={selectors.layout.skip}>{t(strings.layout.skipToContent)}</a>
    <header className="studio-global-header">
      <Link className="studio-global-brand" data-testid={selectors.layout.brand} to="/" aria-label={t(strings.app.title)}>
        <span className="studio-global-mark"><BrandMark /></span><span data-testid={selectors.app.title}>{t(strings.app.title)}</span>
      </Link>
      <nav aria-label={t(strings.layout.navigationLabel)} data-testid={selectors.layout.navigation} className="studio-global-nav">
        {NAV_ITEMS.map((item) => <NavLink key={item.key} to={item.path} end={item.end} data-testid={selectors.layout.navLink({ key: item.key })} className={({ isActive }) => `studio-global-nav__link${isActive ? " is-active" : ""}`}>{t(item.labelKey)}</NavLink>)}
      </nav>
    </header>
    <main className="studio-shell__main" id={selectors.layout.main} data-testid={selectors.layout.main} tabIndex={-1}>
      <Outlet />
    </main>
  </div>;
}
