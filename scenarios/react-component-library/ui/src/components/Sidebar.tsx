import { type ReactNode } from "react";
import { Link } from "react-router-dom";

import { useTranslation } from "../i18n";
import { BrandMark } from "./BrandMark";

interface SidebarContentProps {
  onNavigate?: () => void;
  headerSlot?: ReactNode;
  inventorySlot?: ReactNode;
}

export function SidebarContent({ onNavigate, headerSlot, inventorySlot }: SidebarContentProps) {
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
        <div className="ms-auto">{headerSlot}</div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto px-2 py-3">
        {inventorySlot}
      </div>
    </div>
  );
}
