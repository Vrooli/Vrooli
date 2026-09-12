import { LayoutDashboard, GitBranch, Network, Waypoints, Settings, type LucideIcon } from "lucide-react";
import { strings } from "../consts/strings";

/**
 * Canonical nav-item list shared by `Sidebar` and `BottomNav` so the two
 * surfaces never drift. Replace these entries when this scenario's routes
 * change. `key` doubles as the selector parameter so tests can target a
 * specific link without binding to the translated label.
 */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key: "dashboard" | "graph" | "ontology" | "planning" | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
  icon: LucideIcon;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard, icon: LayoutDashboard },
  { key: "graph", path: "/graph", labelKey: strings.layout.nav.graph, icon: GitBranch },
  { key: "ontology", path: "/ontology", labelKey: strings.layout.nav.ontology, icon: Network },
  { key: "planning", path: "/planning", labelKey: strings.layout.nav.planning, icon: Waypoints },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: Settings },
];
