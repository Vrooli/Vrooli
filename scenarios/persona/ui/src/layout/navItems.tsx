import {
  Home,
  ClipboardCheck,
  FileClock,
  Fingerprint,
  Settings,
} from "lucide-react";
import { strings } from "../consts/strings";

/**
 * Canonical destinations and icons configured into the library AppShell. Replace these entries when this scenario's routes
 * change. `key` doubles as the selector parameter so tests can target a
 * specific link without binding to the translated label.
 */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key:
    | "dashboard"
    | "personas"
    | "handoffs"
    | "journal"
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
  { key: "personas", path: "/personas", labelKey: strings.layout.nav.personas },
  { key: "handoffs", path: "/handoffs", labelKey: strings.layout.nav.handoffs },
  { key: "journal", path: "/journal", labelKey: strings.layout.nav.journal },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];

export function iconForItem(item: NavItem) {
  const iconClass = "h-5 w-5";
  switch (item.key) {
    case "personas":
      return <Fingerprint aria-hidden className={iconClass} />;
    case "handoffs":
      return <ClipboardCheck aria-hidden className={iconClass} />;
    case "journal":
      return <FileClock aria-hidden className={iconClass} />;
    case "settings":
      return <Settings aria-hidden className={iconClass} />;
    case "dashboard":
      return <Home aria-hidden className={iconClass} />;
  }
}
