import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

function themeChoiceLabel(choice: ThemeChoice) {
  switch (choice) {
    case "light":
      return strings.theme.choice.light;
    case "dark":
      return strings.theme.choice.dark;
    case "system":
      return strings.theme.choice.system;
  }
}

/**
 * Top app bar — title, locale switcher, theme toggle. Visible at every viewport
 * width. Replace the title with your real product surface; keep the locale and
 * theme controls (they're the canonical seams the template guarantees).
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
        className="min-w-0 text-lg font-semibold text-app-foreground"
      >
        {t(strings.app.title)}
      </h1>
      <p className="sr-only">
        {t(strings.app.eyebrow)}. {t(strings.app.description)}
      </p>
      <div className="flex shrink-0 items-center gap-3">
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
            className="min-h-11 rounded-control border border-app-border bg-app-surface px-3 text-app-foreground"
          >
            {THEME_CHOICES.map((c) => (
              <option key={c} value={c}>
                {t(themeChoiceLabel(c))}
              </option>
            ))}
          </select>
        </label>
      </div>
    </header>
  );
}
