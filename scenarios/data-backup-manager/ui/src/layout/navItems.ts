import {
  Archive,
  CalendarClock,
  HardDrive,
  History,
  LayoutDashboard,
  RotateCcw,
  Settings,
  type LucideIcon,
} from "lucide-react";

import { strings } from "../consts/strings";

/**
 * Canonical nav-item list shared by `Sidebar` and `BottomNav` so the two
 * surfaces never drift. `key` doubles as the selector parameter so tests can
 * target a specific link without binding to the translated label. The order
 * here is the order shown in both nav surfaces.
 */
export interface NavItem {
  /** Selector parameter; stable across locales. Mirror in `selectors.ts`. */
  key: "overview" | "targets" | "destinations" | "plans" | "runs" | "restores" | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
  /** Lucide icon rendered in both nav surfaces. */
  icon: LucideIcon;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "overview", path: "/", end: true, labelKey: strings.layout.nav.overview, icon: LayoutDashboard },
  { key: "targets", path: "/targets", labelKey: strings.layout.nav.targets, icon: Archive },
  { key: "destinations", path: "/destinations", labelKey: strings.layout.nav.destinations, icon: HardDrive },
  { key: "plans", path: "/plans", labelKey: strings.layout.nav.plans, icon: CalendarClock },
  { key: "runs", path: "/runs", labelKey: strings.layout.nav.runs, icon: History },
  { key: "restores", path: "/restores", labelKey: strings.layout.nav.restores, icon: RotateCcw },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: Settings },
];
