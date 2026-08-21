/** @vrooliComponentSource navigation.app-shell */
import { Link } from "react-router-dom";
import { Menu, Settings as SettingsIcon } from "lucide-react";

import { useTranslation } from "../i18n";
import { BrandMark } from "./BrandMark";
import { IconButton } from "./IconButton";

interface Props {
  onOpenDrawer: () => void;
}

export function MobileHeader({ onOpenDrawer }: Props) {
  const { t } = useTranslation();
  return (
    <header
      data-testid="mobile-header"
      className="pt-safe sticky top-0 z-30 flex h-control-2xl items-center gap-space-2xs border-b border-app-border bg-app-surface px-space-xs md:hidden"
    >
      {/* IconButton (not a hand-rolled <button>) so the drawer toggle inherits the
          shared control treatment: tap-target sizing, hover/active/:focus-visible,
          disabled opacity, and the token-backed motion curve with its
          prefers-reduced-motion opt-out. */}
      <IconButton
        onClick={onOpenDrawer}
        aria-label={t("nav.openDrawer", { defaultValue: "Open navigation" })}
        data-testid="mobile-header-drawer"
      >
        <Menu aria-hidden className="h-icon-md w-icon-md" />
      </IconButton>
      <Link
        to="/"
        data-testid="mobile-header-brand"
        className="flex items-center gap-space-2xs text-app-foreground"
      >
        <BrandMark className="h-control-compact w-control-compact shrink-0 text-app-primary" />
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
        <SettingsIcon aria-hidden className="h-icon-md w-icon-md" />
      </Link>
    </header>
  );
}
