/**
 * ThemeToggle — cycle through light → dark → system.
 *
 * Compact icon button suitable for the top bar. The accessible label
 * announces the current theme (the next click cycles to the next state).
 */
import { Moon, Sun, SunMoon } from "lucide-react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const ORDER: readonly ThemeChoice[] = ["light", "dark", "system"];

export function ThemeToggle() {
  const { choice, setTheme } = useTheme();
  const { t } = useTranslation();
  const Icon = choice === "dark" ? Moon : choice === "system" ? SunMoon : Sun;
  const label =
    choice === "dark"
      ? t(strings.theme.dark)
      : choice === "light"
        ? t(strings.theme.light)
        : t(strings.theme.system);

  return (
    <button
      type="button"
      data-testid={selectors.theme.toggle}
      onClick={() => {
        const idx = ORDER.indexOf(choice);
        const next = ORDER[(idx + 1) % ORDER.length] ?? "system";
        setTheme(next);
      }}
      aria-label={label}
      title={label}
      className="inline-flex h-touch w-touch min-w-touch items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
    >
      <Icon aria-hidden className="h-5 w-5" />
    </button>
  );
}
