import { Link } from "react-router-dom";
import { Menu, Settings as SettingsIcon } from "lucide-react";

import { useTranslation } from "../i18n";
import { BrandMark } from "./BrandMark";

interface Props {
  onOpenDrawer: () => void;
}

export function MobileHeader({ onOpenDrawer }: Props) {
  const { t } = useTranslation();
  return (
    <header
      data-testid="mobile-header"
      className="pt-safe sticky top-0 z-30 flex h-14 items-center gap-2 border-b border-app-border bg-app-surface px-3 md:hidden"
    >
      <button
        type="button"
        onClick={onOpenDrawer}
        aria-label={t("nav.openDrawer", { defaultValue: "Open navigation" })}
        data-testid="mobile-header-drawer"
        className="touch-target inline-flex items-center justify-center rounded-control text-app-foreground hover:bg-app-surface-muted"
      >
        <Menu aria-hidden className="h-5 w-5" />
      </button>
      <Link to="/" data-testid="mobile-header-brand" className="flex items-center gap-2 text-app-foreground">
        <BrandMark className="h-7 w-7 shrink-0 text-app-primary" />
        <span className="text-sm font-semibold tracking-tight">
          {t("app.brand", { defaultValue: "Component Library" })}
        </span>
      </Link>
      <Link
        to="/settings"
        data-testid="mobile-header-settings"
        aria-label={t("nav.settings", { defaultValue: "Settings" })}
        className="touch-target ms-auto inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
      >
        <SettingsIcon aria-hidden className="h-5 w-5" />
      </Link>
    </header>
  );
}
