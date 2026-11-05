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
    | "matrix"
    | "fleet"
    | "wizard"
    | "findings"
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
  { key: "matrix", path: "/matrix", labelKey: strings.layout.nav.matrix },
  { key: "fleet", path: "/fleet", labelKey: strings.layout.nav.fleet },
  { key: "wizard", path: "/wizard", labelKey: strings.layout.nav.wizard },
  { key: "findings", path: "/findings", labelKey: strings.layout.nav.findings },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
