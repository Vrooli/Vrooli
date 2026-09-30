import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";

export interface GraphFilterBarProps {
  /** All package paths available to filter, sorted. */
  packages: readonly string[];
  /** Currently-selected package paths. Empty set means "All". */
  selected: ReadonlySet<string>;
  onChange: (next: ReadonlySet<string>) => void;
}

export function GraphFilterBar({ packages, selected, onChange }: GraphFilterBarProps) {
  const { t } = useTranslation();

  if (packages.length === 0) {
    return (
      <p
        data-testid={selectors.features.explorer.filterBar.empty}
        className="text-xs text-app-muted-foreground"
      >
        {t(strings.explorer.filterBar.none)}
      </p>
    );
  }

  const toggle = (key: string): void => {
    const next = new Set(selected);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    onChange(next);
  };

  return (
    <div
      data-testid={selectors.features.explorer.filterBar.root}
      role="group"
      aria-label={t(strings.explorer.filterBar.label)}
      className="flex flex-wrap items-center gap-2"
    >
      <button
        type="button"
        data-testid={selectors.features.explorer.filterBar.allChip}
        onClick={() => onChange(new Set())}
        aria-pressed={selected.size === 0}
        className={cn(
          "rounded-pill border px-3 py-1 text-xs font-medium transition-colors",
          selected.size === 0
            ? "border-app-primary bg-app-primary text-app-primary-foreground"
            : "border-app-border bg-app-surface-muted text-app-foreground hover:bg-app-surface",
        )}
      >
        {t(strings.explorer.filterBar.all)}
      </button>
      {packages.map((pkg) => {
        const isOn = selected.has(pkg);
        return (
          <button
            type="button"
            key={pkg}
            data-testid={selectors.features.explorer.filterBar.chip({ key: pkg })}
            onClick={() => toggle(pkg)}
            aria-pressed={isOn}
            className={cn(
              "rounded-pill border px-3 py-1 text-xs font-medium transition-colors",
              isOn
                ? "border-app-primary bg-app-primary text-app-primary-foreground"
                : "border-app-border bg-app-surface-muted text-app-foreground hover:bg-app-surface",
            )}
          >
            {pkg.split("/").pop() ?? pkg}
          </button>
        );
      })}
    </div>
  );
}
