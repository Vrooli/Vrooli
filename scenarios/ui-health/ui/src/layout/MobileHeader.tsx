/**
 * MobileHeader — sticky top bar shown below the `md` breakpoint. Hosts the
 * drawer trigger, brand, and the health pill / theme toggle.
 */
import { Menu } from "lucide-react";
import { Link } from "react-router-dom";

import { HealthPill } from "../components/HealthPill";
import { ThemeToggle } from "../components/ThemeToggle";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ROUTES } from "../routes.generated";

interface Props {
  onOpenDrawer: () => void;
}

export function MobileHeader({ onOpenDrawer }: Props) {
  const { t } = useTranslation();
  return (
    <header
      data-testid={selectors.layout.mobileHeader}
      aria-label={t(strings.layout.mobileHeaderLabel)}
      className="pt-safe sticky top-0 z-30 flex h-14 items-center gap-2 border-b border-app-border bg-app-surface px-3 md:hidden"
    >
      <button
        type="button"
        onClick={onOpenDrawer}
        aria-label={t(strings.layout.openDrawer)}
        data-testid={selectors.layout.drawerOpen}
        className="inline-flex h-touch w-touch items-center justify-center rounded-control text-app-foreground hover:bg-app-surface-muted"
      >
        <Menu aria-hidden className="h-5 w-5" />
      </button>
      <Link
        to={ROUTES.dashboard}
        data-testid={selectors.app.brand}
        className="flex items-center gap-2 text-app-foreground"
      >
        <span
          aria-hidden
          className="inline-flex h-7 w-7 items-center justify-center rounded-control bg-app-primary text-sm font-semibold text-app-primary-foreground"
        >
          {t(strings.app.brandInitials)}
        </span>
        <span className="text-sm font-semibold tracking-tight">{t(strings.app.brand)}</span>
      </Link>
      <div className="ms-auto flex items-center gap-1">
        <HealthPill />
        <ThemeToggle />
      </div>
    </header>
  );
}
