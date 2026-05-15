import { useLocation, useNavigate } from "react-router-dom";
import { GaugeCircle, GitBranch, Library, Settings as SettingsIcon } from "lucide-react";
import type { MouseEvent, ReactNode } from "react";

import {
  type BottomNavItem,
  BottomNav,
} from "../../../library/components/BottomNav/versions/1.0.0/BottomNav";
import { useTranslation } from "../i18n";

interface Item {
  to: string;
  label: string;
  end?: boolean;
  icon: ReactNode;
  testid: string;
}

export function MobileNav() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const items: Item[] = [
    {
      to: "/",
      end: true,
      label: t("nav.dashboard", { defaultValue: "Dashboard" }),
      icon: <GaugeCircle aria-hidden className="h-5 w-5" />,
      testid: "mobile-nav-dashboard",
    },
    {
      to: "/components",
      label: t("nav.components", { defaultValue: "Components" }),
      icon: <Library aria-hidden className="h-5 w-5" />,
      testid: "mobile-nav-components",
    },
    {
      to: "/adoptions",
      label: t("nav.adoptions", { defaultValue: "Adoptions" }),
      icon: <GitBranch aria-hidden className="h-5 w-5" />,
      testid: "mobile-nav-adoptions",
    },
    {
      to: "/settings",
      label: t("nav.settings", { defaultValue: "Settings" }),
      icon: <SettingsIcon aria-hidden className="h-5 w-5" />,
      testid: "mobile-nav-settings",
    },
  ];
  const isActiveRoute = (item: Item) =>
    item.end
      ? location.pathname === item.to
      : location.pathname === item.to || location.pathname.startsWith(`${item.to}/`);
  const bottomNavItems: BottomNavItem[] = items.map((item) => ({
    id: item.to,
    href: item.to,
    label: item.label,
    icon: item.icon,
    active: isActiveRoute(item),
    testId: item.testid,
  }));
  const handleSelect = (item: BottomNavItem, event: MouseEvent<HTMLAnchorElement | HTMLButtonElement>) => {
    if (!item.href) return;
    event.preventDefault();
    void navigate(item.href);
  };

  return (
    <BottomNav
      items={bottomNavItems}
      label={t("nav.bottomLabel", { defaultValue: "Primary" })}
      testId="mobile-nav"
      onItemSelect={handleSelect}
      className="mobile-nav"
    />
  );
}
