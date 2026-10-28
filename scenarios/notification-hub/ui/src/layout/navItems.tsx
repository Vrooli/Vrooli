import {
  Archive,
  Bell,
  ChartNoAxesCombined,
  Inbox,
  Settings,
} from "lucide-react";
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
    | "asks"
    | "history"
    | "delivery"
    | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "asks", path: "/asks", labelKey: strings.layout.nav.asks },
  { key: "dashboard", path: "/updates", end: true, labelKey: strings.layout.nav.dashboard },
  { key: "history", path: "/history", labelKey: strings.layout.nav.history },
  { key: "delivery", path: "/delivery", labelKey: strings.layout.nav.delivery },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];

export function iconForItem(item: NavItem) {
  const iconClass = "h-5 w-5";
  switch (item.key) {
    case "settings":
      return <Settings aria-hidden className={iconClass} />;
    case "dashboard":
      return <Bell aria-hidden className={iconClass} />;
    case "asks":
      return <Inbox aria-hidden className={iconClass} />;
    case "history":
      return <Archive aria-hidden className={iconClass} />;
    case "delivery":
      return <ChartNoAxesCombined aria-hidden className={iconClass} />;
  }
}
