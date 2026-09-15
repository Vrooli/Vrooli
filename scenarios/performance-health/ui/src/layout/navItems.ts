import { strings } from "../consts/strings";
import { Activity, ChartNoAxesCombined, Gauge, LayoutDashboard, ListChecks, Settings, ShieldCheck, Waypoints, type LucideIcon } from "lucide-react";

/**
 * Canonical nav-item list shared by `Sidebar` and `BottomNav` so the two
 * surfaces never drift. `key` doubles as the selector parameter so tests can
 * target a specific link without binding to the translated label.
 */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key:
    | "dashboard"
    | "audit"
    | "trends"
    | "fleet"
    | "trace"
    | "readiness"
    | "budgets"
    | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
  icon: LucideIcon;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard, icon: LayoutDashboard },
  { key: "audit", path: "/audit", labelKey: strings.layout.nav.audit, icon: ListChecks },
  { key: "trends", path: "/trends", labelKey: strings.layout.nav.trends, icon: ChartNoAxesCombined },
  { key: "fleet", path: "/fleet", labelKey: strings.layout.nav.fleet, icon: Waypoints },
  { key: "trace", path: "/trace", labelKey: strings.layout.nav.trace, icon: Activity },
  { key: "readiness", path: "/readiness", labelKey: strings.layout.nav.readiness, icon: ShieldCheck },
  { key: "budgets", path: "/budgets", labelKey: strings.layout.nav.budgets, icon: Gauge },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: Settings },
];
