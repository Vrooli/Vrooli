import { Archive, Bell, Inbox, Settings } from "lucide-react";
import { strings } from "../consts/strings";

/**
 * The destinations of the Hub, in the order they appear in the phone tab bar and
 * the desktop sidebar. `key` doubles as the selector parameter so tests can
 * target a link without binding to the translated label. Delivery is not a
 * destination of its own: it is the second tab of Updates, so the Updates item
 * stays current on `/delivery` and `/delivery/:id` as well.
 */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key: "dashboard" | "asks" | "history" | "settings";
  /** Router path the item opens. */
  path: string;
  /** Every path prefix that keeps this item current. */
  within: readonly string[];
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "asks", path: "/asks", within: ["/asks"], labelKey: strings.layout.nav.asks },
  { key: "dashboard", path: "/updates", within: ["/updates", "/delivery"], labelKey: strings.layout.nav.dashboard },
  { key: "history", path: "/history", within: ["/history"], labelKey: strings.layout.nav.history },
  { key: "settings", path: "/settings", within: ["/settings"], labelKey: strings.layout.nav.settings },
];

/**
 * Whether the item is the current destination. A page may claim a destination
 * (see `useClaimNavSection`), which wins; otherwise `pathname` decides, with the
 * index route belonging to Updates.
 */
export function isCurrent(item: NavItem, pathname: string, claimed: NavItem["key"] | null = null): boolean {
  if (claimed) return item.key === claimed;
  const path = pathname === "/" ? "/updates" : pathname;
  return item.within.some((prefix) => path === prefix || path.startsWith(`${prefix}/`));
}

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
  }
}
