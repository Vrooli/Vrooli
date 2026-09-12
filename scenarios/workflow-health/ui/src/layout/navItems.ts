import { LayoutDashboard, Boxes, Search, Activity, FileSearch, Workflow, Settings, type LucideIcon } from "lucide-react";
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
    | "inventory"
    | "search"
    | "runs"
    | "findings"
    | "fixes"
    | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
  icon: LucideIcon;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "overview", path: "/", end: true, labelKey: strings.layout.nav.overview, icon: LayoutDashboard },
  { key: "inventory", path: "/inventory", labelKey: strings.layout.nav.inventory, icon: Boxes },
  { key: "search", path: "/search", labelKey: strings.layout.nav.search, icon: Search },
  { key: "runs", path: "/runs", labelKey: strings.layout.nav.runs, icon: Activity },
  { key: "findings", path: "/findings", labelKey: strings.layout.nav.findings, icon: FileSearch },
  { key: "fixes", path: "/fixes", labelKey: strings.layout.nav.fixes, icon: Workflow },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: Settings },
];
