import { strings } from "../consts/strings";

/**
 * Canonical nav-item list shared by `Sidebar` and `BottomNav` so the two
 * surfaces never drift. `key` doubles as the selector parameter so tests can
 * target a specific link without binding to the translated label.
 *
 * Shell-level nav is intentionally narrow: it points at the cross-target
 * surfaces (Overview, Targets, Settings). Per-target sections (Graph,
 * Manifest, Conflicts, Apply, Analytics) appear as a sub-nav inside the
 * target workspace, not here.
 */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key: "overview" | "targets" | "history" | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "overview", path: "/", end: true, labelKey: strings.layout.nav.overview },
  { key: "targets", path: "/targets/new", labelKey: strings.layout.nav.targets },
  { key: "history", path: "/history", labelKey: strings.layout.nav.history },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
