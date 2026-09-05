import { strings } from "../consts/strings";

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
    | "brands"
    | "assignments"
    | "assets"
    | "generation"
    | "apply"
    | "discovery"
    | "design"
    | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard },
  { key: "brands", path: "/brands", labelKey: strings.layout.nav.brands },
  { key: "assignments", path: "/assignments", labelKey: strings.layout.nav.assignments },
  { key: "assets", path: "/assets", labelKey: strings.layout.nav.assets },
  { key: "generation", path: "/generation", labelKey: strings.layout.nav.generation },
  { key: "apply", path: "/apply", labelKey: strings.layout.nav.apply },
  { key: "discovery", path: "/discovery", labelKey: strings.layout.nav.discovery },
  { key: "design", path: "/design", labelKey: strings.layout.nav.design },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
