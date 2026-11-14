import { strings } from "../consts/strings";
import { Activity, Gauge, Laptop, LayoutDashboard, Settings, SlidersHorizontal, type LucideIcon } from "lucide-react";

/**
 * Canonical nav-item list shared by `Sidebar` and `BottomNav` so the two
 * surfaces never drift. Replace these entries when this scenario's routes
 * change. `key` doubles as the selector parameter so tests can target a
 * specific link without binding to the translated label.
 */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key:
    | "dashboard"
    | "snapshots"
    | "resolver"
    | "devices"
    | "optimization"
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
  { key: "snapshots", path: "/snapshots", labelKey: strings.layout.nav.snapshots, icon: Activity },
  { key: "resolver", path: "/resolver", labelKey: strings.layout.nav.resolver, icon: Gauge },
  { key: "devices", path: "/devices", labelKey: strings.layout.nav.devices, icon: Laptop },
  { key: "optimization", path: "/optimization", labelKey: strings.layout.nav.optimization, icon: SlidersHorizontal },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: Settings },
];
