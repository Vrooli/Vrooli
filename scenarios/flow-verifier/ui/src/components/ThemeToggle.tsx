import { Moon, Sun, SunMoon } from "lucide-react";

import { type Theme } from "./theme/theme-context";
import { useTheme } from "./theme/useTheme";
import { useTranslation } from "../i18n";

const ORDER: Theme[] = ["light", "dark", "system"];

export function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  const { t } = useTranslation();
  const Icon = theme === "dark" ? Moon : theme === "system" ? SunMoon : Sun;
  const label = t(`theme.${theme}`, {
    defaultValue: theme === "system" ? "System theme" : theme === "dark" ? "Dark theme" : "Light theme",
  });

  return (
    <button
      type="button"
      data-testid="theme-toggle"
      onClick={() => {
        const idx = ORDER.indexOf(theme);
        const next = ORDER[(idx + 1) % ORDER.length] ?? "system";
        setTheme(next);
      }}
      aria-label={label}
      title={label}
      className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
    >
      <Icon aria-hidden className="h-5 w-5" />
    </button>
  );
}
