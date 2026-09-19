import { strings } from "../consts/strings";
import { Activity, BookOpen, Crosshair, Gauge, LayoutDashboard, Settings, Waypoints, type LucideIcon } from "lucide-react";

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
    | "substrate"
    | "coverage"
    | "condition"
    | "focus"
    | "designLanguage"
    | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey?: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
  label?: string;
  icon: LucideIcon;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard, icon: LayoutDashboard },
  // Substrate leads the projections: operating-model rule 7 orders the cascade
  // innermost-first, and the host substrate is the layer to resolve before any
  // outer projection's reading can be trusted.
  { key: "substrate", path: "/substrate", labelKey: strings.layout.nav.substrate, icon: Waypoints },
  { key: "coverage", path: "/coverage", labelKey: strings.layout.nav.coverage, icon: Gauge },
  { key: "condition", path: "/condition", labelKey: strings.layout.nav.condition, icon: Activity },
  { key: "focus", path: "/focus", labelKey: strings.layout.nav.focus, icon: Crosshair },
  { key: "designLanguage", path: "/design-language", labelKey: strings.layout.nav.designLanguage, icon: BookOpen },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: Settings },
];
