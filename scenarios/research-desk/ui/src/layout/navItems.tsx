import { Activity, BookOpenCheck, FolderKanban, Inbox, Rocket, Server, Shapes } from "lucide-react";
import type { ReactNode } from "react";

import { strings } from "../consts/strings";

/**
 * The desk's navigation, as data. `AppShell` (the library's) renders this
 * list as the desktop column and the phone tab bar, so the two can never
 * drift. `mobile: false` keeps a destination in the phone menu instead of
 * the tab bar, which holds the four the operator checks most; Launch is the
 * first button on Now and stays in the menu.
 */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key: "now" | "needsYou" | "topics" | "commissions" | "launch" | "reviews" | "desk";
  path: string;
  /** True when this is the index route (exact match for the active state). */
  end?: boolean;
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
  icon: ReactNode;
  /** False keeps the item out of the phone tab bar (it stays in the menu). */
  mobile?: boolean;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "now", path: "/", end: true, labelKey: strings.layout.nav.now, icon: <Activity aria-hidden="true" /> },
  { key: "needsYou", path: "/needs-you", labelKey: strings.layout.nav.needsYou, icon: <Inbox aria-hidden="true" /> },
  { key: "topics", path: "/topics", labelKey: strings.layout.nav.topics, icon: <Shapes aria-hidden="true" /> },
  { key: "commissions", path: "/commissions", labelKey: strings.layout.nav.commissions, icon: <FolderKanban aria-hidden="true" /> },
  { key: "launch", path: "/launch", labelKey: strings.layout.nav.launch, icon: <Rocket aria-hidden="true" />, mobile: false },
  { key: "reviews", path: "/reviews", labelKey: strings.layout.nav.reviews, icon: <BookOpenCheck aria-hidden="true" />, mobile: false },
  { key: "desk", path: "/desk", labelKey: strings.layout.nav.desk, icon: <Server aria-hidden="true" />, mobile: false },
];

/** True when `pathname` is inside this item's route (exact for the index route). */
export function isNavItemActive(item: NavItem, pathname: string): boolean {
  if (item.end) return pathname === item.path;
  if (item.key === "commissions" && pathname.startsWith("/files/")) return false;
  return pathname === item.path || pathname.startsWith(`${item.path}/`);
}
