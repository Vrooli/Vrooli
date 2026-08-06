/** @vrooliComponentSource navigation.sidebar */
import { type ReactNode } from "react";
import { Link } from "react-router-dom";

import { useTranslation } from "../i18n";
import { PanelLeftClose } from "lucide-react";
import { BrandMark } from "./BrandMark";

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

  return (
    <div data-testid="app-sidebar-content" className="flex min-h-0 flex-1 flex-col">
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
        <Link
          to="/catalog"
          onClick={onNavigate}
          className="mb-space-2xs flex rounded-control px-space-2xs py-space-2xs text-sm text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
        >
          {t("nav.browseAssets", { defaultValue: "Browse assets" })}
        </Link>
        <nav
          aria-label="Operator views"
          className="mb-space-xs grid gap-space-3xs border-b border-app-border pb-space-xs"
        >
          <Link
            to="/coverage"
            onClick={onNavigate}
            className="flex rounded-control px-space-2xs py-space-2xs text-sm text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
          >
            Catalog coverage
          </Link>
          <Link
            to="/capabilities"
            onClick={onNavigate}
            className="flex rounded-control px-space-2xs py-space-2xs text-sm text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
          >
            Capability readiness
          </Link>
        </nav>
        {inventorySlot}
      </div>
    </div>
  );
}
