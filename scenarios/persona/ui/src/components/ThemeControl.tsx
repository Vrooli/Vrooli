import { Select } from "@vrooli/react-component-library/Select/1";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

export function ThemeControl() {
  const { t } = useTranslation();
  const { choice, setTheme } = useTheme();
  return (
        <label
          data-testid={selectors.theme.switcher}
          className="flex items-center gap-2 text-xs text-app-muted-foreground"
        >
          <span className="sr-only">{t(strings.theme.switcherLabel)}</span>
          <Select
            value={choice}
            onValueChange={(value) => setTheme(value as ThemeChoice)}
            options={THEME_CHOICES.map((value) => ({ value, label: t(strings.theme.choice[value]) }))}
            data-testid={selectors.theme.select}
            aria-label={t(strings.theme.switcherLabel)}
            className="min-h-11 rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground"
          />
        </label>
  );
}
