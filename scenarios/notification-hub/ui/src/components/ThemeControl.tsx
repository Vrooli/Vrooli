import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { usePreferenceStorageUnavailable, useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

export function ThemeControl() {
  const { t } = useTranslation();
  const { choice, setTheme } = useTheme();
  const preferenceStorageUnavailable = usePreferenceStorageUnavailable();
  return (
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
            aria-describedby={preferenceStorageUnavailable ? "theme-storage-warning" : undefined}
            className="min-h-11 rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground"
          >
            {THEME_CHOICES.map((c) => (
              <option key={c} value={c}>
                {t(strings.theme.choice[c])}
              </option>
            ))}
          </select>
          {preferenceStorageUnavailable && (
            <span id="theme-storage-warning" className="sr-only">{t(strings.preferenceStorage.body)}</span>
          )}
        </label>
  );
}
