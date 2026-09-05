import { strings } from "../consts/strings";

/**
 * Canonical nav-item list shared by `Sidebar` and `BottomNav` so the two
 * surfaces never drift. `key` doubles as the selector parameter so tests can
 * target a specific link without binding to the translated label.
 *
 * Dual-mode ordering: the Studio surfaces (Home, Workspace, Library) come
 * first, then the Console surfaces (Activity, Models, Settings).
 */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key: "home" | "workspace" | "library" | "select" | "compare" | "activity" | "models" | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "home", path: "/", end: true, labelKey: strings.layout.nav.home },
  { key: "workspace", path: "/workspace", labelKey: strings.layout.nav.workspace },
  { key: "library", path: "/library", labelKey: strings.layout.nav.library },
  { key: "select", path: "/select", labelKey: strings.layout.nav.select },
  { key: "compare", path: "/compare", labelKey: strings.layout.nav.compare },
  { key: "activity", path: "/activity", labelKey: strings.layout.nav.activity },
  { key: "models", path: "/models", labelKey: strings.layout.nav.models },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
