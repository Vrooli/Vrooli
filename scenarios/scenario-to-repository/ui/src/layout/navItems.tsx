import {
  Home,
  Settings,
} from "lucide-react";
import type { ReactNode } from "react";

import { strings } from "../consts/strings";

/**
 * The scenario's navigation, as data. `AppShell` (the library's) renders this
 * list as the desktop column and the phone tab bar, so the two can never drift.
 *
 * Add a destination by appending an entry here and a route in
 * `app/routes.tsx`. `key` doubles as the selector parameter so tests can target
 * a link without binding to its translated label.
 */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key:
    | "dashboard"
    | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (exact match for the active state). */
  end?: boolean;
  /** Translation key path for the full label. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
  /** Icon rendered by the shell at the size its density calls for. */
  icon: ReactNode;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard, icon: <Home aria-hidden="true" /> },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, icon: <Settings aria-hidden="true" /> },
];

/** True when `pathname` is inside this item's route (exact for the index route). */
export function isNavItemActive(item: NavItem, pathname: string): boolean {
  return item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(`${item.path}/`);
}
