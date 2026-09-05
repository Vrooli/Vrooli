/** @vrooliComponentSource navigation.sidebar */
import { type ReactNode } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";

import { useTranslation } from "../i18n";
import { BarChart3, FolderTree, PanelLeftClose, Sparkles } from "lucide-react";
import { BrandMark } from "./BrandMark";
import { IconButton } from "@vrooli/react-component-library/IconButton/2";
import { AppNavigation } from "@vrooli/react-component-library/AppNavigation/1";
import { NavigationTree } from "@vrooli/react-component-library/NavigationTree/1";

interface SidebarContentProps {
  onNavigate?: () => void;
  headerSlot?: ReactNode;
  inventorySlot?: ReactNode;
  onCollapse?: () => void;
}

export function SidebarContent({
  onNavigate,
  headerSlot,
  inventorySlot,
  onCollapse,
}: SidebarContentProps) {
  const { t } = useTranslation();
  const location = useLocation();
  const navClass = ({ isActive }: { isActive: boolean }) =>
    [
      "group flex min-h-touch items-center gap-space-2xs rounded-control px-space-xs py-space-2xs text-sm transition-colors",
      isActive
        ? "bg-app-surface-muted font-semibold text-app-foreground shadow-sm"
        : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground",
    ].join(" ");
  const isCatalogActive =
    location.pathname === "/" ||
    location.pathname === "/catalog" ||
    location.pathname.startsWith("/assets/");

  return (
    <div data-testid="app-sidebar-content" className="flex min-h-0 min-w-0 flex-1 flex-col">
      <div className="hidden items-center gap-space-2xs border-b border-app-border px-space-sm py-space-sm md:flex">
        <Link
          to="/"
          onClick={onNavigate}
          className="flex items-center gap-space-2xs text-app-foreground"
          data-testid="app-brand"
        >
          <BrandMark className="h-control-compact w-control-compact shrink-0 text-app-primary" />
          <span className="text-sm font-semibold tracking-tight">
            {t("app.brand", { defaultValue: "Component Library" })}
          </span>
        </Link>
        <div className="ms-auto flex items-center gap-space-3xs">
          {headerSlot}
          {onCollapse ? (
            /* IconButton rather than a hand-rolled <button>: the collapse control
               gets the shared tap-target sizing, hover/active/:focus-visible and
               disabled treatment, plus the token-backed transition and its
               prefers-reduced-motion opt-out, instead of a local class list that
               only covered hover. */
            <IconButton
              onClick={onCollapse}
              aria-label={t("nav.closeDrawer", { defaultValue: "Close navigation" })}
              data-testid="sidebar-collapse"
              density="compact"
            >
              <PanelLeftClose aria-hidden className="h-icon-sm w-icon-sm" />
            </IconButton>
          ) : null}
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto px-space-2xs py-space-xs">
        {/* No brand: the sidebar header above already renders it alongside the
            collapse and settings controls. AppNavigation renders its own brand
            when given one, so passing it here produced the product name twice. */}
        <AppNavigation>
          <ul data-rcl-app-navigation-list>
            <li>
              <Link
                to="/catalog"
                onClick={onNavigate}
                aria-current={isCatalogActive ? "page" : undefined}
                className={navClass({ isActive: isCatalogActive })}
              >
                <FolderTree aria-hidden className="h-icon-sm w-icon-sm shrink-0 text-app-primary" />
                <span className="truncate">
                  {t("nav.browseAssets", { defaultValue: "Browse assets" })}
                </span>
              </Link>
            </li>
            <li>
              <NavLink to="/coverage" onClick={onNavigate} className={navClass}>
                <BarChart3 aria-hidden className="h-icon-sm w-icon-sm shrink-0" />
                <span className="truncate">Catalog coverage</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/capabilities" onClick={onNavigate} className={navClass}>
                <Sparkles aria-hidden className="h-icon-sm w-icon-sm shrink-0" />
                <span className="truncate">Capability readiness</span>
              </NavLink>
            </li>
          </ul>
        </AppNavigation>
        {/* NavigationTree renders its own title; a hand-rolled header here
            produced "Library inventory" twice. The heading is the component's
            to own — pass it through the prop rather than drawing one beside it. */}
        <NavigationTree title="Library inventory" items={[]}>
          <div data-rcl-navigation-tree-list>{inventorySlot}</div>
        </NavigationTree>
      </div>
    </div>
  );
}
