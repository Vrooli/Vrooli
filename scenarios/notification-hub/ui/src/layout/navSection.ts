import { createContext, useContext, useEffect } from "react";

import type { NavItem } from "./navItems";

export type NavKey = NavItem["key"];

/** How a page tells the shell which destination it belongs to. The shell provides it. */
export const NavSectionContext = createContext<(key: NavKey | null) => void>(() => undefined);

/**
 * Claim a destination for as long as the page is mounted. A page whose URL
 * sits in one section but whose content belongs to another (a decision opened
 * from History lives at `/asks/:id`) uses this so the navigation agrees with
 * where the operator came from. `null` leaves the choice to the URL.
 */
export function useClaimNavSection(key: NavKey | null): void {
  const claim = useContext(NavSectionContext);
  useEffect(() => {
    claim(key);
    return () => claim(null);
  }, [claim, key]);
}
