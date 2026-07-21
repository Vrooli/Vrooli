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

export function SidebarContent({ onNavigate, headerSlot, inventorySlot, onCollapse }: SidebarContentProps) {
  const { t } = useTranslation();

  return (
    <div data-testid="app-sidebar-content" className="flex min-h-0 flex-1 flex-col">
      <div className="hidden items-center gap-2 border-b border-app-border px-4 py-4 md:flex">
        <Link
          to="/"
          onClick={onNavigate}
          className="flex items-center gap-2 text-app-foreground"
          data-testid="app-brand"
        >
          <BrandMark className="h-7 w-7 shrink-0 text-app-primary" />
          <span className="text-sm font-semibold tracking-tight">
            {t("app.brand", { defaultValue: "Component Library" })}
          </span>
        </Link>
        <div className="ms-auto flex items-center gap-1">{headerSlot}{onCollapse ? <button type="button" onClick={onCollapse} aria-label={t("nav.closeDrawer", { defaultValue: "Close navigation" })} data-testid="sidebar-collapse" className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"><PanelLeftClose aria-hidden className="h-4 w-4" /></button> : null}</div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto px-2 py-3">
        <Link to="/catalog" onClick={onNavigate} className="mb-2 flex rounded-control px-2 py-2 text-sm text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground">{t("nav.browseAssets", { defaultValue: "Browse assets" })}</Link>
        {inventorySlot}
      </div>
    </div>
  );
}
