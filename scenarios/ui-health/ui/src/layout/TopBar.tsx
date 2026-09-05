/**
 * TopBar — desktop top strip shown above the main content. The sidebar
 * already carries brand + primary nav, so this strip hosts the locale
 * switcher, health pill, and theme toggle.
 */
import { HealthPill } from "../components/HealthPill";
import { ThemeToggle } from "../components/ThemeToggle";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";

export function TopBar() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();

  return (
    <header
      data-testid={selectors.layout.topBar}
      className="hidden h-14 shrink-0 items-center justify-end gap-3 border-b border-app-border bg-app-surface px-6 md:flex"
    >
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
    </header>
  );
}
