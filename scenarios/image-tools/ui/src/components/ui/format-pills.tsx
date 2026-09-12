import { cn } from "../../lib/utils";

export interface FormatPillsProps {
  /** Already-translated group label. */
  label: string;
  value: string;
  /** Format tokens (e.g. "png", "jpeg"); the pill shows the uppercased token. */
  options: readonly string[];
  onChange: (value: string) => void;
  "data-testid"?: string;
}

/**
 * Encode-format chooser. Each pill renders the format token uppercased — a
 * derived, non-translated technical token — so it stays i18n-clean.
 */
export function FormatPills({ label, value, options, onChange, ...rest }: FormatPillsProps) {
  return (
    <div
      role="radiogroup"
      aria-label={label}
      data-testid={rest["data-testid"]}
      className="flex flex-wrap gap-1"
    >
      {options.map((format) => {
        const active = format === value;
        return (
          <button
            key={format}
            type="button"
            role="radio"
            aria-checked={active}
            onClick={() => onChange(format)}
            className={cn(
              "rounded-pill border px-3 py-1 text-xs font-medium tabular-nums transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50",
              active
                ? "border-app-primary bg-app-primary text-app-primary-foreground"
                : "border-app-border text-app-muted-foreground hover:text-app-foreground",
            )}
          >
            {format.toUpperCase()}
          </button>
        );
      })}
    </div>
  );
}
