import { Menu, Settings, Waves } from "lucide-react";
import { Button } from "../ui/button";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { ServiceStatusPill } from "./ServiceStatusPill";

interface Props {
  onOpenSettings: () => void;
  onToggleMobileMenu?: () => void;
  showMobileMenuButton?: boolean;
}

export function TopBar({ onOpenSettings, onToggleMobileMenu, showMobileMenuButton }: Props) {
  const { t } = useTranslation();
  return (
    <header className="sticky top-0 z-sticky flex h-topbar items-center gap-3 border-b border-app-border bg-app-surface px-3 md:px-4">
      {showMobileMenuButton ? (
        <Button
          variant="ghost"
          size="icon"
          aria-label={t(strings.shell.openMenu)}
          onClick={onToggleMobileMenu}
        >
          <Menu className="h-5 w-5" aria-hidden="true" />
        </Button>
      ) : null}

      <div className="flex min-w-0 items-center gap-2">
        <span
          aria-hidden="true"
          className="flex h-7 w-7 items-center justify-center rounded-control bg-app-primary text-app-primary-foreground"
        >
          <Waves className="h-4 w-4" />
        </span>
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold leading-none text-app-foreground">
            {t(strings.app.title)}
          </p>
          <p className="mt-0.5 truncate text-[10px] uppercase tracking-wide text-app-muted-foreground">
            {t(strings.app.eyebrow)}
          </p>
        </div>
      </div>

      <div className="ms-auto flex items-center gap-2">
        <ServiceStatusPill />
        <Button
          variant="ghost"
          size="icon"
          aria-label={t(strings.shell.openSettings)}
          aria-keyshortcuts="Control+,"
          onClick={onOpenSettings}
        >
          <Settings className="h-5 w-5" aria-hidden="true" />
        </Button>
      </div>
    </header>
  );
}
