import { Gift, History, Home } from "lucide-react";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export function HolderShell() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const items = [
    { id: "home", href: "/me", label: t(strings.holderView.nav.home), icon: <Home size={16} aria-hidden />, current: location.pathname === "/me" },
    { id: "history", href: "/me/history", label: t(strings.holderView.nav.history), icon: <History size={16} aria-hidden />, current: location.pathname.startsWith("/me/history") },
    { id: "rewards", href: "/me/rewards", label: t(strings.holderView.nav.rewards), icon: <Gift size={16} aria-hidden />, current: location.pathname.startsWith("/me/rewards") },
  ];

  return (
    <LibraryAppShell
      density="sidebar"
      mobileNav="tabs"
      mainMode="scroll"
      brand={<span data-testid={selectors.holder.shell}>{t(strings.holderView.appTitle)}</span>}
      brandHref="/me"
      items={items}
      renderLink={(_item, { href, children, ...props }) => <NavLink to={href} {...props}>{children}</NavLink>}
      onNavigate={(item) => navigate(item.href)}
      navigationLabel={t(strings.holderView.navigationLabel)}
      mobileNavigationLabel={t(strings.holderView.navigationLabel)}
      testId={selectors.holder.shell}
    >
        <Outlet />
    </LibraryAppShell>
  );
}
