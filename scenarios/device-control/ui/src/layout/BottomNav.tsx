import { FileCheck2, Home, ListChecks, Settings } from "lucide-react";
import { useLocation, useNavigate } from "react-router-dom";

import { BottomNav as CanonicalBottomNav, type BottomNavItem } from "@vrooli/react-component-library/BottomNav/1.2.0";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS, type NavItem } from "./navItems";

/**
 * Mobile bottom nav. Visible below the `md` breakpoint; on desktop, see
 * `Sidebar`. Same nav targets as `NAV_ITEMS`, rendered as a flex row pinned to
 * the viewport bottom.
 */
export function BottomNav() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const items = NAV_ITEMS.map((item): BottomNavItem => ({
    id: item.key,
    label: t(item.labelKey),
    icon: iconForItem(item),
    active: item.end ? location.pathname === item.path : location.pathname.startsWith(item.path),
    testId: selectors.layout.bottomNavLink({ key: item.key }),
  }));

  return (
    <CanonicalBottomNav
      items={items}
      label={t(strings.layout.bottomNavLabel)}
      testId={selectors.layout.bottomNav}
      onItemSelect={(item) => {
        const navItem = NAV_ITEMS.find((entry) => entry.key === item.id);
        if (navItem) {
          navigate(navItem.path);
        }
      }}
    />
  );
}

function iconForItem(item: NavItem) {
  const iconClass = "h-5 w-5";
  switch (item.key) {
    case "flows":
      return <ListChecks aria-hidden className={iconClass} />;
    case "evidence":
      return <FileCheck2 aria-hidden className={iconClass} />;
    case "settings":
      return <Settings aria-hidden className={iconClass} />;
    case "dashboard":
      return <Home aria-hidden className={iconClass} />;
  }
}
