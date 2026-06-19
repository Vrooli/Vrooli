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
    | "overview"
    | "exposure"
    | "recovery"
    | "metrics"
    | "audit"
    | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "overview", path: "/", end: true, labelKey: strings.layout.nav.overview },
  { key: "exposure", path: "/exposure", labelKey: strings.layout.nav.exposure },
  { key: "recovery", path: "/recovery", labelKey: strings.layout.nav.recovery },
  { key: "metrics", path: "/metrics", labelKey: strings.layout.nav.metrics },
  { key: "audit", path: "/audit", labelKey: strings.layout.nav.audit },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
