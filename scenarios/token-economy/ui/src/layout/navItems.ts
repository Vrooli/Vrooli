import { LayoutDashboard, Coins, Network, BookOpen, ShieldCheck, ScrollText, Settings, type LucideIcon } from "lucide-react";
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
    | "tokens"
    | "holders"
    | "earning"
    | "grants"
    | "catalog"
    | "approvals"
    | "journal"
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
  { key: "tokens", path: "/tokens", labelKey: strings.layout.nav.tokens, icon: Coins },
  { key: "holders", path: "/holders", labelKey: strings.layout.nav.holders, icon: Network },
  { key: "earning", path: "/earning", labelKey: strings.layout.nav.earning, icon: Coins },
  { key: "grants", path: "/grants", labelKey: strings.layout.nav.grants, icon: Coins },
  { key: "catalog", path: "/catalog", labelKey: strings.layout.nav.catalog, icon: BookOpen },
  { key: "approvals", path: "/approvals", labelKey: strings.layout.nav.approvals, icon: ShieldCheck },
  { key: "journal", path: "/journal", labelKey: strings.layout.nav.journal, icon: ScrollText },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: Settings },
];
