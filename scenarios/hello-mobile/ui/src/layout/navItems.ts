import { strings } from "../consts/strings";
import { BarChart3, Home, Settings, type LucideIcon } from "lucide-react";

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
    | "notes" // EXAMPLE-DOMAIN:notes
    | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
  /** Library shell icon. */
  icon: LucideIcon;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard, icon: Home },
  { key: "notes", path: "/notes", labelKey: strings.layout.nav.notes, icon: BarChart3 }, // EXAMPLE-DOMAIN:notes
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: Settings },
];
