import { NavLink } from "react-router-dom";
import { Boxes, GaugeCircle, Layers, Settings as SettingsIcon } from "lucide-react";
import type { ReactNode } from "react";

import { useTranslation } from "../i18n";
import { ROUTES } from "../routes.generated";

interface Item {
  to: string;
  label: string;
  end?: boolean;
  icon: ReactNode;
  testid: string;
}

export function MobileNav() {
  const { t } = useTranslation();
  const items: Item[] = [
    {
      to: ROUTES.dashboard,
      end: true,
      label: t("nav.dashboard", { defaultValue: "Dashboard" }),
      icon: <GaugeCircle aria-hidden className="h-5 w-5" />,
      testid: "mobile-nav-dashboard",
    },
    {
      to: ROUTES.scenarios,
      label: t("nav.scenarios", { defaultValue: "Scenarios" }),
      icon: <Boxes aria-hidden className="h-5 w-5" />,
      testid: "mobile-nav-scenarios",
    },
    {
      to: ROUTES.flowsInventory,
      label: t("nav.flows", { defaultValue: "All flows" }),
      icon: <Layers aria-hidden className="h-5 w-5" />,
      testid: "mobile-nav-flows",
    },
    {
      to: ROUTES.settings,
      label: t("nav.settings", { defaultValue: "Settings" }),
      icon: <SettingsIcon aria-hidden className="h-5 w-5" />,
      testid: "mobile-nav-settings",
    },
  ];

  return (
    <nav
      data-testid="mobile-nav"
      aria-label={t("nav.bottomLabel", { defaultValue: "Primary" })}
      className="pb-safe fixed inset-x-0 bottom-0 z-30 flex border-t border-app-border bg-app-surface md:hidden"
    >
      {items.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          end={item.end}
          data-testid={item.testid}
          className={({ isActive }) =>
            [
              "touch-target flex flex-1 flex-col items-center justify-center gap-0.5 py-2 text-xs",
              isActive
                ? "text-app-primary"
                : "text-app-muted-foreground hover:text-app-foreground",
            ].join(" ")
          }
        >
          {item.icon}
          <span>{item.label}</span>
        </NavLink>
      ))}
    </nav>
  );
}
