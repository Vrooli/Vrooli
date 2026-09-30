import {
  BarChart3, // EXAMPLE-DOMAIN:notes
  Home,
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
    | "notes" // EXAMPLE-DOMAIN:notes
    | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard },
  { key: "notes", path: "/notes", labelKey: strings.layout.nav.notes }, // EXAMPLE-DOMAIN:notes
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];

export function iconForItem(item: NavItem) {
  const iconClass = "h-5 w-5";
  switch (item.key) {
    // EXAMPLE-DOMAIN:notes START
    case "notes":
      return <BarChart3 aria-hidden className={iconClass} />;
    // EXAMPLE-DOMAIN:notes END
    case "settings":
      return <Settings aria-hidden className={iconClass} />;
    case "dashboard":
      return <Home aria-hidden className={iconClass} />;
  }
}
