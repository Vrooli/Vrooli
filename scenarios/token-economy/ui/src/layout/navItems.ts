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
    | "tokens"
    | "holders"
    | "earning"
    | "grants"
    | "catalog"
    | "approvals"
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
  { key: "tokens", path: "/tokens", labelKey: strings.layout.nav.tokens },
  { key: "holders", path: "/holders", labelKey: strings.layout.nav.holders },
  { key: "earning", path: "/earning", labelKey: strings.layout.nav.earning },
  { key: "grants", path: "/grants", labelKey: strings.layout.nav.grants },
  { key: "catalog", path: "/catalog", labelKey: strings.layout.nav.catalog },
  { key: "approvals", path: "/approvals", labelKey: strings.layout.nav.approvals },
  { key: "journal", path: "/journal", labelKey: strings.layout.nav.journal },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
