import { selectors } from "../consts/selectors";
import type { ChangeEvent } from "react";
import { strings } from "../consts/strings";
import { Select } from "@vrooli/react-component-library/Select/1";
import { useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
const THEME_LABEL_KEYS = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
};

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
          <Select
            value={choice}
            onChange={(e: ChangeEvent<HTMLSelectElement>) => setTheme(e.target.value as ThemeChoice)}
            data-testid={selectors.theme.select}
            aria-label={t(strings.theme.switcherLabel)}
            className="min-h-11 rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground"
            options={THEME_CHOICES.map((c) => ({ value: c, label: t(THEME_LABEL_KEYS[c]) }))}
          />
        </label>
      </div>
    </header>
  );
}
