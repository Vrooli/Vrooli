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
    | "substrate"
    | "coverage"
    | "condition"
    | "focus"
    | "designLanguage"
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
  // Substrate leads the projections: operating-model rule 7 orders the cascade
  // innermost-first, and the host substrate is the layer to resolve before any
  // outer projection's reading can be trusted.
  { key: "substrate", path: "/substrate", labelKey: strings.layout.nav.substrate },
  { key: "coverage", path: "/coverage", labelKey: strings.layout.nav.coverage },
  { key: "condition", path: "/condition", labelKey: strings.layout.nav.condition },
  { key: "focus", path: "/focus", labelKey: strings.layout.nav.focus },
  { key: "designLanguage", path: "/design-language", labelKey: strings.layout.nav.designLanguage },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
