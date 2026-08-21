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
  const apiStatusLabel = apiStatus === "healthy"
    ? strings.health.statusHealthy
    : apiStatus === "unreachable"
      ? strings.health.statusUnreachable
      : strings.health.statusChecking;

  return (
    <header
      data-testid={selectors.layout.topBar}
      className="flex shrink-0 flex-wrap items-center justify-between gap-3 overflow-x-hidden border-b border-slate-700/70 bg-slate-950 px-4 py-3 text-slate-100 shadow-[0_1px_0_rgba(255,255,255,0.03)] sm:px-6"
    >
      <div className="flex min-w-0 items-center gap-3">
        <span className="relative flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-cyan-400/30 bg-cyan-400/10" aria-hidden="true">
          <span className="absolute h-5 w-5 rounded-full border-2 border-cyan-300/70" />
          <span className="h-2 w-2 rounded-full bg-cyan-300 shadow-[0_0_12px_rgba(103,232,249,0.9)]" />
        </span>
        <div className="min-w-0">
          <h1 data-testid={selectors.app.title} className="truncate text-base font-semibold tracking-tight text-white sm:text-lg">
            {t(strings.app.title)}
          </h1>
          <p className="break-words text-[10px] font-medium uppercase tracking-[0.16em] text-slate-400">
            {t(strings.app.tagline)}
          </p>
        </div>
      </div>
      <div className="flex max-w-full flex-wrap items-center justify-end gap-1.5 sm:gap-2">
        <button
          type="button"
          className="rounded-pill focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
          aria-label={`${t(strings.health.statusLabel)} ${t(apiStatusLabel)}`}
          onClick={() => void healthQuery.refetch()}
        >
            <StatusBadge tone={apiTone} className="min-h-11 items-center px-3" data-testid={selectors.health.statusBadge}>
            <span className="mr-1.5 inline-block h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true" />
            {t(apiStatusLabel)}
          </StatusBadge>
        </button>
        <div
          role="group"
          aria-label={t(strings.locale.switcherLabel)}
          data-testid={selectors.locale.switcher}
          className="flex items-center gap-0.5 rounded-control border border-white/10 bg-white/5 p-0.5 text-xs sm:gap-1 sm:p-1"
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
                  ? "min-h-11 min-w-11 rounded-control bg-app-primary px-2 py-1 font-medium text-app-primary-foreground"
                : "min-h-11 min-w-11 rounded-control px-1.5 py-1 text-slate-400 hover:text-white sm:px-2"
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
            className="min-h-11 max-w-[7rem] rounded-control border border-white/10 bg-white/5 px-2 py-1 text-slate-100 sm:max-w-none"
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
