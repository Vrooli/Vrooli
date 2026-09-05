import { strings, type Strings } from "../consts/strings";

type NavLabelKey = Strings["layout"]["nav"][keyof Strings["layout"]["nav"]];

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
    | "runs"
    | "settings"
    | "sessions"
    | "rollouts"
    | "trust"
    | "setup";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: NavLabelKey;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard },
  { key: "runs", path: "/runs", labelKey: strings.layout.nav.runs },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
  { key: "sessions", path: "/sessions", labelKey: strings.layout.nav.sessions },
  { key: "rollouts", path: "/rollouts", labelKey: strings.layout.nav.rollouts },
  { key: "trust", path: "/trust", labelKey: strings.layout.nav.trust },
  { key: "setup", path: "/setup", labelKey: strings.layout.nav.setup },
];
