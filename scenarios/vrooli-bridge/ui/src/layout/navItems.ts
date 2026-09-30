import { LayoutDashboard, Activity, Settings, Waypoints, ShieldCheck, type LucideIcon } from "lucide-react";
import { strings, type Strings } from "../consts/strings";

type NavLabelKey = Strings["layout"]["nav"][keyof Strings["layout"]["nav"]];

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
    | "runs"
    | "settings"
    | "sessions"
    | "rollouts"
    | "trust"
    | "setup";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: NavLabelKey;
  icon: LucideIcon;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard, icon: LayoutDashboard },
  { key: "runs", path: "/runs", labelKey: strings.layout.nav.runs, icon: Activity },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: Settings },
  { key: "sessions", path: "/sessions", labelKey: strings.layout.nav.sessions, icon: Activity },
  { key: "rollouts", path: "/rollouts", labelKey: strings.layout.nav.rollouts, icon: Waypoints },
  { key: "trust", path: "/trust", labelKey: strings.layout.nav.trust, icon: ShieldCheck },
  { key: "setup", path: "/setup", labelKey: strings.layout.nav.setup, icon: Settings },
];
