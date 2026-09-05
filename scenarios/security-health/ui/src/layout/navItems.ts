import { strings } from "../consts/strings";

/**
 * Canonical nav-item list shared by `Sidebar` and `BottomNav` so the two
 * surfaces never drift. Replace these entries when this scenario's routes
 * change. `key` doubles as the selector parameter so tests can target a
 * specific link without binding to the translated label.
 */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key: "posture" | "dependencies" | "secrets" | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "posture", path: "/", end: true, labelKey: strings.layout.nav.posture },
  { key: "dependencies", path: "/dependencies", labelKey: strings.layout.nav.dependencies },
  { key: "secrets", path: "/secrets", labelKey: strings.layout.nav.secrets },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
