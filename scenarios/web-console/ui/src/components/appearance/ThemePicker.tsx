import { Check } from "lucide-react";
import { useTranslation } from "react-i18next";
import { TERMINAL_THEMES } from "../../consts/config";
import { cn } from "../../lib/classnames";
import { strings } from "../../consts/strings";

interface ThemePickerProps {
  currentThemeId: string;
  onSelectTheme: (themeId: string) => void;
  testIdPrefix?: string;
}

export default function ThemePicker({
  currentThemeId,
  onSelectTheme,
  testIdPrefix = "appearance",
}: ThemePickerProps) {
  const { t } = useTranslation();
  return (
    <section>
      <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
        {t(strings.appearance.terminalThemeHeading)}
      </h3>
      <div className="grid grid-cols-2 gap-2">
        {Object.values(TERMINAL_THEMES).map((theme) => {
          const selected = currentThemeId === theme.id;
          return (
            <button
              key={theme.id}
              type="button"
              data-testid={`${testIdPrefix}-theme-${theme.id}`}
              className={cn(
                "rounded-lg border p-2 text-start transition-colors",
                selected
                  ? "border-wc-accent ring-1 ring-wc-accent"
                  : "border-wc-default hover:border-wc-text-faint",
              )}
              onClick={() => onSelectTheme(theme.id)}
              aria-pressed={selected}
            >
              <div
                className="rounded px-2 py-1.5 mb-1.5 font-mono text-[10px] leading-tight"
                style={{ backgroundColor: theme.colors.background, color: theme.colors.foreground }}
              >
                <span>$ hello world</span>
                <span
                  className="inline-block ms-0.5 h-2.5 w-1 align-middle rounded-sm"
                  style={{ backgroundColor: theme.colors.cursor }}
                />
              </div>
              <span className="flex items-center gap-1 text-xs text-wc-text-secondary">
                {selected && <Check className="h-3 w-3 text-wc-accent" aria-hidden="true" />}
                {theme.label}
              </span>
            </button>
          );
        })}
      </div>
    </section>
  );
}
