import { strings } from "../consts/strings";

/** Route destinations configured into the library-owned shell. */
export interface NavItem {
  /** Selector parameter; stable across locales. */
  key: "dashboard" | "settings";
  /** Router path. */
  path: string;
  /** True when this is the index route (used for `<NavLink end>`). */
  end?: boolean;
  /** Translation key path. */
  labelKey: (typeof strings.layout.nav)[keyof typeof strings.layout.nav];
}

export const NAV_ITEMS: readonly NavItem[] = [
  { key: "dashboard", path: "/", end: true, labelKey: strings.layout.nav.dashboard },
  { key: "settings", path: "/settings", labelKey: strings.layout.nav.settings },
];
