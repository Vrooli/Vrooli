import {
  Boxes,
  GaugeCircle,
  RefreshCw,
  Search,
  Settings as SettingsIcon,
  ShieldCheck,
  type LucideIcon,
} from "lucide-react";

import { strings } from "../consts/strings";
import { ROUTES } from "../routes.generated";

/**
 * Canonical nav-item list shared by `Sidebar`, `BottomNav`, and the mobile
 * drawer so the three surfaces never drift. `key` doubles as the selector
 * parameter so tests can target a specific link without binding to the
 * translated label.
 */
export type NavKey =
  | "dashboard"
  | "validation"
  | "search"
  | "inventory"
  | "reindex"
  | "settings";

export interface NavItem {
  key: NavKey;
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
  icon: LucideIcon;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: ROUTES.dashboard, end: true, labelKey: strings.layout.nav.dashboard, icon: GaugeCircle },
  { key: "validation", path: ROUTES.validation, labelKey: strings.layout.nav.validation, icon: ShieldCheck },
  { key: "search", path: ROUTES.search, labelKey: strings.layout.nav.search, icon: Search },
  { key: "inventory", path: ROUTES.inventory, labelKey: strings.layout.nav.inventory, icon: Boxes },
  { key: "reindex", path: ROUTES.reindex, labelKey: strings.layout.nav.reindex, icon: RefreshCw },
  { key: "settings", path: ROUTES.settings, labelKey: strings.layout.nav.settings, icon: SettingsIcon },
];
