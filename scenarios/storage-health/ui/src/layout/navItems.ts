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
    | "fleet"
    | "validate"
    | "advisor"
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
  { key: "fleet", path: "/fleet", labelKey: strings.layout.nav.fleet },
  { key: "validate", path: "/validate", labelKey: strings.layout.nav.validate },
  { key: "advisor", path: "/advisor", labelKey: strings.layout.nav.advisor },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
