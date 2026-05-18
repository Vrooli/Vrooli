import { NavLink, useLocation } from "react-router-dom";
import { Target, Wrench, FileCog, Settings as SettingsIcon } from "lucide-react";
import type { ReactNode } from "react";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ROUTES, ROUTE_PATTERNS } from "../../routes.generated";
import { cn } from "../lib/utils";

interface MobileNavItem {
  to: string;
  patternPrefix: string;
  label: string;
  testId: string;
  icon: typeof Target;
}

/** Fixed bottom navigation for mobile. Includes safe-area padding. */
export function MobileBottomNav(): ReactNode {
  const { t } = useTranslation();
  const location = useLocation();
  const items: readonly MobileNavItem[] = [
    {
      to: ROUTES.goldensIndex,
      patternPrefix: ROUTE_PATTERNS.goldensIndex,
      label: t(strings.nav.goldensLabel),
      testId: selectors.nav.mobileBottomItemGoldens,
      icon: Target,
    },
    {
      to: ROUTES.skillsIndex,
      patternPrefix: ROUTE_PATTERNS.skillsIndex,
      label: t(strings.nav.skillsLabel),
      testId: selectors.nav.mobileBottomItemSkills,
      icon: Wrench,
    },
    {
      to: ROUTES.manifestsIndex,
      patternPrefix: ROUTE_PATTERNS.manifestsIndex,
      label: t(strings.nav.manifestsLabel),
      testId: selectors.nav.mobileBottomItemManifests,
      icon: FileCog,
    },
    {
      to: ROUTES.settings,
      patternPrefix: ROUTE_PATTERNS.settings,
      label: t(strings.nav.settingsLabel),
      testId: selectors.nav.mobileBottomItemSettings,
      icon: SettingsIcon,
    },
  ];
  return (
    <nav
      data-testid={selectors.nav.mobileBottomNav}
      className="fixed inset-x-0 bottom-0 z-20 flex items-stretch border-t border-app-border bg-app-shell pb-safe backdrop-blur"
    >
      {items.map((item) => {
        const Icon = item.icon;
        const isActive =
          item.patternPrefix === "/"
            ? location.pathname === "/"
            : location.pathname.startsWith(item.patternPrefix);
        return (
          <NavLink
            key={item.to}
            to={item.to}
            data-testid={item.testId}
            aria-current={isActive ? "page" : undefined}
            className={cn(
              "flex flex-1 flex-col items-center gap-1 px-2 py-2 text-[10px] transition-colors",
              isActive ? "text-app-accent" : "text-app-muted-foreground",
            )}
          >
            <Icon className="h-5 w-5" />
            <span>{item.label}</span>
          </NavLink>
        );
      })}
    </nav>
  );
}
