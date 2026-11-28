import { Link } from "react-router-dom";
import { Images, Settings } from "lucide-react";
import { ROUTES } from "../routes.generated";
import { HealthPill } from "./HealthPill";
import { ThemeToggle } from "./ThemeToggle";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";

export function ShellUtility() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();

  return (
    <div className="flex flex-wrap items-center gap-2">
      <div
        role="group"
        aria-label={t(strings.locale.switcherLabel)}
        data-testid={selectors.locale.switcher}
        className="flex items-center gap-1 rounded-control border border-app-border bg-app-surface-muted p-1 text-xs"
      >
        {SUPPORTED_LOCALES.map((lng) => (
          <button
            key={lng}
            type="button"
            data-testid={selectors.locale.toggle({ code: lng })}
            onClick={() => void setLocale(lng)}
            aria-pressed={currentLocale === lng}
            className={
              currentLocale === lng
                ? "rounded-control bg-app-primary px-2 py-1 font-medium text-app-primary-foreground"
                : "rounded-control px-2 py-1 text-app-muted-foreground hover:text-app-foreground"
            }
          >
            {getLocaleConfig(lng).nativeLabel}
          </button>
        ))}
      </div>
      <HealthPill />
      <ThemeToggle />
      <Link to={ROUTES.captures} className="inline-flex min-h-touch min-w-touch items-center justify-center md:hidden" aria-label={t(strings.layout.nav.captures)}><Images aria-hidden className="h-5 w-5" /></Link>
      <Link to={ROUTES.settings} className="inline-flex min-h-touch min-w-touch items-center justify-center md:hidden" aria-label={t(strings.layout.nav.settings)}><Settings aria-hidden className="h-5 w-5" /></Link>
    </div>
  );
}
