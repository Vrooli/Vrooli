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
    | "board"
    | "today"
    | "plan"
    | "create"
    | "review"
    | "content"
    | "learn"
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
  { key: "board", path: "/board", labelKey: strings.layout.nav.board },
  { key: "today", path: "/today", labelKey: strings.layout.nav.today },
  { key: "plan", path: "/plan", labelKey: strings.layout.nav.plan },
  { key: "create", path: "/create", labelKey: strings.layout.nav.create },
  { key: "review", path: "/review", labelKey: strings.layout.nav.review },
  { key: "content", path: "/content", labelKey: strings.layout.nav.content },
  { key: "learn", path: "/learn", labelKey: strings.layout.nav.learn },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
