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
    | "personas"
    | "handoffs"
    | "journal"
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
  { key: "personas", path: "/personas", labelKey: strings.layout.nav.personas },
  { key: "handoffs", path: "/handoffs", labelKey: strings.layout.nav.handoffs },
  { key: "journal", path: "/journal", labelKey: strings.layout.nav.journal },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
