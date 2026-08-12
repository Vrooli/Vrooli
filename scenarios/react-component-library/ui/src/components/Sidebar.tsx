/** @vrooliComponentSource navigation.sidebar */
import { type ReactNode } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";

import { useTranslation } from "../i18n";
import { BarChart3, Blocks, FolderTree, PanelLeftClose, Sparkles } from "lucide-react";
import { BrandMark } from "./BrandMark";
import { AppNavigation } from "./ui/AppNavigation/versions/1.0.0/AppNavigation";
import { NavigationTree } from "./ui/NavigationTree/versions/1.0.0/NavigationTree";

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
      "group flex min-h-11 items-center gap-space-2xs rounded-control px-space-xs py-space-2xs text-sm transition-colors",
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
          <BrandMark className="h-7 w-7 shrink-0 text-app-primary" />
          <span className="text-sm font-semibold tracking-tight">
            {t("app.brand", { defaultValue: "Component Library" })}
          </span>
        </Link>
        <div className="ms-auto flex items-center gap-space-3xs">
          {headerSlot}
          {onCollapse ? (
            <button
              type="button"
              onClick={onCollapse}
              aria-label={t("nav.closeDrawer", { defaultValue: "Close navigation" })}
              data-testid="sidebar-collapse"
              className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
            >
              <PanelLeftClose aria-hidden className="h-4 w-4" />
            </button>
          ) : null}
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto px-space-2xs py-space-xs">
        <AppNavigation brand={t("app.brand", { defaultValue: "Component Library" })}>
          <ul data-rcl-app-navigation-list>
            <li>
              <Link
                to="/catalog"
                onClick={onNavigate}
                aria-current={isCatalogActive ? "page" : undefined}
                className={navClass({ isActive: isCatalogActive })}
              >
                <FolderTree aria-hidden className="h-4 w-4 shrink-0 text-app-primary" />
                <span className="truncate">
                  {t("nav.browseAssets", { defaultValue: "Browse assets" })}
                </span>
              </Link>
            </li>
            <li>
              <NavLink to="/coverage" onClick={onNavigate} className={navClass}>
                <BarChart3 aria-hidden className="h-4 w-4 shrink-0" />
                <span className="truncate">Catalog coverage</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/capabilities" onClick={onNavigate} className={navClass}>
                <Sparkles aria-hidden className="h-4 w-4 shrink-0" />
                <span className="truncate">Capability readiness</span>
              </NavLink>
            </li>
          </ul>
        </AppNavigation>
        <div className="mb-space-xs flex items-center gap-space-2xs border-b border-app-border px-space-xs pb-space-2xs text-[11px] font-semibold uppercase tracking-wide text-app-muted-foreground">
          <Blocks aria-hidden className="h-3.5 w-3.5" />
          <span>Library inventory</span>
        </div>
        <NavigationTree title="Library inventory" items={[]}>
          <div data-rcl-navigation-tree-list>{inventorySlot}</div>
        </NavigationTree>
      </div>
    </div>
  );
}
