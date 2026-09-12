import { strings } from "../consts/strings";
import { BookOpen, Gauge, GitBranch, LayoutDashboard, ListChecks, Play, Settings, Siren, Users, type LucideIcon } from "lucide-react";

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
    | "plans"
    | "authoring"
    | "execution"
    | "families"
    | "validation"
    | "triage"
    | "velocity"
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
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard, icon: LayoutDashboard },
  { key: "plans", path: "/plans", labelKey: strings.layout.nav.plans, icon: BookOpen },
  { key: "authoring", path: "/authoring", labelKey: strings.layout.nav.authoring, icon: GitBranch },
  { key: "execution", path: "/execution", labelKey: strings.layout.nav.execution, icon: Play },
  { key: "families", path: "/families", labelKey: strings.layout.nav.families, icon: Users },
  { key: "validation", path: "/validation", labelKey: strings.layout.nav.validation, icon: ListChecks },
  { key: "triage", path: "/triage", labelKey: strings.layout.nav.triage, icon: Siren },
  { key: "velocity", path: "/velocity", labelKey: strings.layout.nav.velocity, icon: Gauge },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: Settings },
];
