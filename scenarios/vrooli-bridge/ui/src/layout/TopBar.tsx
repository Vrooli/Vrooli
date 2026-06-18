import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

// Static label map so each theme key has a visible callsite (the registry's
// no-unused-keys gate can't see dynamic `strings.theme.choice[c]` indexing).
const THEME_CHOICE_LABEL = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
} as const satisfies Record<ThemeChoice, string>;

/**
 * Top app bar — title, locale switcher, theme toggle. Visible at every viewport
 * width. Replace the title with your real product surface; keep the locale and
 * theme controls (they're the canonical seams the template guarantees).
 */
export function TopBar() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();

  return (
    <header
      data-testid={selectors.layout.topBar}
      className="flex shrink-0 items-center justify-between gap-4 border-b border-app-border bg-app-surface px-4 py-3"
    >
      <h1
        data-testid={selectors.app.title}
        className="text-lg font-semibold text-app-foreground"
      >
        {t(strings.app.title)}
      </h1>
      <div className="flex items-center gap-3">
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
          <span className="sr-only">{t(strings.theme.switcherLabel)}</span>
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
