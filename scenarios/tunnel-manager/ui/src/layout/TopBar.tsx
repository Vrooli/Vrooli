import { useQuery } from "@tanstack/react-query";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import type { ThemeChoice } from "../theme/themeContext";
import { THEME_CHOICE_LABEL } from "../theme/themeChoiceLabels";
import { useTheme } from "../theme/useTheme";
import { fetchHealth } from "../api/health";
import { StatusBadge } from "../components/ui/StatusBadge";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

/**
 * Top app bar — title, locale switcher, theme toggle. Visible at every viewport
 * width. Replace the title with your real product surface; keep the locale and
 * theme controls (they're the canonical seams the template guarantees).
 */
export function TopBar() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const healthQuery = useQuery({
    queryKey: ["api-health"],
    queryFn: fetchHealth,
    staleTime: 15_000,
    retry: 1,
  });
  const apiStatus = healthQuery.data?.status ?? (healthQuery.isError ? "unreachable" : "checking");
  const apiTone = healthQuery.isError ? "danger" : apiStatus === "healthy" ? "success" : "warning";

  return (
    <header
      data-testid={selectors.layout.topBar}
      className="flex shrink-0 flex-wrap items-center justify-between gap-3 overflow-x-hidden border-b border-app-border bg-app-surface px-4 py-3"
    >
      <h1
        data-testid={selectors.app.title}
        className="min-w-0 text-lg font-semibold text-app-foreground"
      >
        {t(strings.app.title)}
      </h1>
      <div className="flex max-w-full flex-wrap items-center justify-end gap-2">
        <button
          type="button"
          className="rounded-pill focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
          aria-label={`${t(strings.health.statusLabel)} ${apiStatus}`}
          onClick={() => void healthQuery.refetch()}
        >
          <StatusBadge tone={apiTone}>
            <span className="mr-1.5 inline-block h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true" />
            {apiStatus}
          </StatusBadge>
        </button>
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
        <label
          data-testid={selectors.theme.switcher}
          className="flex items-center gap-2 text-xs text-app-muted-foreground"
        >
          <select
            value={choice}
            onChange={(e) => setTheme(e.target.value as ThemeChoice)}
            data-testid={selectors.theme.select}
            aria-label={t(strings.theme.switcherLabel)}
            className="rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground"
          >
            {THEME_CHOICES.map((c) => (
              <option key={c} value={c}>
                {t(THEME_CHOICE_LABEL[c])}
              </option>
            ))}
          </select>
        </label>
      </div>
    </header>
  );
}
