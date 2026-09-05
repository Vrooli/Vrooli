import { strings } from "../consts/strings";

/**
 * Canonical nav-item list shared by `Sidebar` and `BottomNav` so the two
 * surfaces never drift. Replace these entries when this scenario's routes
 * change. `key` doubles as the selector parameter so tests can target a
 * specific link without binding to the translated label.
 */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key: "dashboard" | "targets" | "runs" | "readiness" | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard },
  { key: "targets", path: "/targets", labelKey: strings.layout.nav.targets },
  { key: "runs", path: "/runs", labelKey: strings.layout.nav.runs },
  { key: "readiness", path: "/readiness", labelKey: strings.layout.nav.readiness },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
