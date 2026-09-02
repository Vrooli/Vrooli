import { useLocation, useNavigate } from "react-router-dom";

import { BottomNav as CanonicalBottomNav, type BottomNavItem } from "@vrooli/react-component-library/BottomNav/1.5.3";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useAttention } from "./useAttention";
import { navIcon } from "./navIcons";
import { NAV_ITEMS, isNavItemActive } from "./navItems";

/**
 * Mobile bottom nav. The RCL component is always `position: fixed`, so the
 * breakpoint lives on this wrapper: hidden from `md` up, where the sidebar
 * takes over.
 */
export function BottomNav() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const attention = useAttention();
  const items = NAV_ITEMS.filter((item) => item.mobile).map((item): BottomNavItem => ({
    id: item.key,
    label: t(item.shortLabelKey ?? item.labelKey),
    icon: navIcon(item.key),
    active: isNavItemActive(item, location.pathname),
    testId: selectors.layout.bottomNavLink({ key: item.key }),
    badge:
      item.key === "dashboard" && attention.pending > 0
        ? { value: attention.pending, tone: "warning", label: t(strings.console.attention.pendingCount, { count: attention.pending }) }
        : undefined,
  }));

  return (
    <div className="md:hidden">
      <CanonicalBottomNav
        items={items}
        label={t(strings.layout.bottomNavLabel)}
        testId={selectors.layout.bottomNav}
        safeArea="floor"
        onItemSelect={(item) => {
          const navItem = NAV_ITEMS.find((entry) => entry.key === item.id);
          if (navItem) navigate(navItem.path);
        }}
      />
    </div>
  );
}
