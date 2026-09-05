import { strings } from "../consts/strings";

/**
 * Canonical nav-item list shared by `Sidebar` and `BottomNav` so the two
 * surfaces never drift. `key` doubles as the selector parameter so tests can
 * target a specific link without binding to the translated label.
 *
 * `mobile: false` keeps an item out of the bottom nav; a bottom nav holds five
 * targets comfortably and Settings is reached from the top bar on mobile.
 */
export interface NavItem {
  key: "dashboard" | "conversations" | "agents" | "contacts" | "channels" | "settings";
  path: string;
  end?: boolean;
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
  /** Shorter label for the bottom nav, where width is scarce. */
  shortLabelKey?: (typeof strings.layout.navShort)[keyof typeof strings.layout.navShort];
  mobile: boolean;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard, shortLabelKey: strings.layout.navShort.dashboard, mobile: true },
  { key: "conversations", path: "/conversations", labelKey: strings.layout.nav.conversations, shortLabelKey: strings.layout.navShort.conversations, mobile: true },
  { key: "agents", path: "/agents", labelKey: strings.layout.nav.agents, mobile: true },
  { key: "contacts", path: "/contacts", labelKey: strings.layout.nav.contacts, mobile: true },
  { key: "channels", path: "/channels", labelKey: strings.layout.nav.channels, mobile: true },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings, mobile: false },
];

export function isNavItemActive(item: NavItem, pathname: string): boolean {
  return item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(`${item.path}/`);
}
