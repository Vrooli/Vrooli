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
    | "coverage"
    | "condition"
    | "focus"
    | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey?: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
  label?: string;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard },
  { key: "coverage", path: "/coverage", label: "Coverage" },
  { key: "condition", path: "/condition", label: "Condition" },
  { key: "focus", path: "/focus", label: "Focus" },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
