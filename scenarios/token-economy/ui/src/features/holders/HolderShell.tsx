import { Gift, History, Home, Sparkles } from "lucide-react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";

import { BottomNav, type BottomNavItem } from "@vrooli/react-component-library/BottomNav/1";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export function HolderShell() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const links = [
    { to: "/me", end: true, icon: Home, key: "home", label: t(strings.holderView.nav.home) },
    { to: "/me/history", icon: History, key: "history", label: t(strings.holderView.nav.history) },
    { to: "/me/rewards", icon: Gift, key: "rewards", label: t(strings.holderView.nav.rewards) },
  ];
  const items = links.map(({ to, end, icon: Icon, key, label }): BottomNavItem => ({
    id: to,
    label,
    icon: <Icon className="h-5 w-5" />,
    active: end ? location.pathname === to : location.pathname.startsWith(to),
    testId: `holder-nav-${key}`,
  }));

  return (
    <div className="min-h-dvh bg-app-background text-app-foreground" data-testid="holder-shell">
      <header className="border-b border-app-border bg-app-surface px-4 pb-4 pt-4">
        <div className="mx-auto flex max-w-3xl items-center gap-3">
          <span className="grid h-12 w-12 place-items-center rounded-panel bg-app-primary text-app-primary-foreground">
            <Sparkles aria-hidden className="h-7 w-7" />
          </span>
          <div>
            <p className="flex items-center gap-2 text-lg font-semibold">
              <Sparkles aria-hidden className="h-5 w-5 text-app-primary" />
              {t(strings.holderView.appTitle)}
            </p>
            <p className="text-sm text-app-muted-foreground">{t(strings.holderView.appDescription)}</p>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-3xl px-4 py-6 pb-28" aria-label={t(strings.holderView.mainLabel)}>
        <Outlet />
      </main>
      <BottomNav
        items={items}
        label={t(strings.holderView.navigationLabel)}
        testId="holder-bottom-nav"
        onItemSelect={(item) => navigate(item.id)}
        className="px-2 pt-2 md:flex"
        itemClassName="min-h-14 gap-1 rounded-control px-3 text-sm font-semibold focus-visible:ring-2 focus-visible:ring-app-primary/50"
        activeItemClassName="bg-app-primary text-app-primary-foreground"
        inactiveItemClassName="text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
      />
    </div>
  );
}
