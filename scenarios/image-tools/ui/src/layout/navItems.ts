import { strings } from "../consts/strings";
import { Activity, Images, LayoutDashboard, ListChecks, Settings, Sparkles, Wand2, Workflow, type LucideIcon } from "lucide-react";

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
  icon: LucideIcon;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "home", path: "/", end: true, labelKey: strings.layout.nav.home, icon: LayoutDashboard },
  { key: "workspace", path: "/workspace", labelKey: strings.layout.nav.workspace, icon: Wand2 },
  { key: "library", path: "/library", labelKey: strings.layout.nav.library, icon: Images },
  { key: "select", path: "/select", labelKey: strings.layout.nav.select, icon: Sparkles },
  { key: "compare", path: "/compare", labelKey: strings.layout.nav.compare, icon: ListChecks },
  { key: "activity", path: "/activity", labelKey: strings.layout.nav.activity, icon: Activity },
  { key: "models", path: "/models", labelKey: strings.layout.nav.models, icon: Workflow },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: Settings },
];

export const iconForItem = (item: NavItem) => item.icon;
