import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { type ThemeChoice } from "../theme/ThemeProvider";
import { useTheme } from "../theme/useTheme";

type ThemeLabelKey = typeof strings.theme.choice[keyof typeof strings.theme.choice];

const THEME_CHOICES = [
  { choice: "light", labelKey: strings.theme.choice.light },
  { choice: "dark", labelKey: strings.theme.choice.dark },
  { choice: "system", labelKey: strings.theme.choice.system },
] as const satisfies ReadonlyArray<{ choice: ThemeChoice; labelKey: ThemeLabelKey }>;

/**
 * Top app bar. Keep high-frequency chrome compact; full preference controls
 * live on Settings so mobile headers do not accumulate wrapped controls.
 */
export function TopBar() {
  const { t } = useTranslation();
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
            className="touch-target rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground"
          >
            {THEME_CHOICES.map(({ choice: optionChoice, labelKey }) => (
              <option key={optionChoice} value={optionChoice}>
                {t(labelKey)}
              </option>
            ))}
          </select>
        </label>
      </div>
    </header>
  );
}
