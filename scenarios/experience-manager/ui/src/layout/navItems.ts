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
    | "explorer"
    | "evidence"
    | "studio"
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
  { key: "fleet", path: "/", end: true, labelKey: strings.layout.nav.fleet },
  { key: "explorer", path: "/scenarios/experience-manager", labelKey: strings.layout.nav.explorer },
  {
    key: "evidence",
    path: "/scenarios/experience-manager/pages/fleet/evidence",
    labelKey: strings.layout.nav.evidence,
  },
  { key: "studio", path: "/studio", labelKey: strings.layout.nav.studio },
  { key: "findings", path: "/findings", labelKey: strings.layout.nav.findings },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
