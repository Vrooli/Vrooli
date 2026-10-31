import { cn } from "../../lib/utils";

export interface SegmentedOption<T extends string> {
  value: T;
  /** Already-translated, user-facing label. */
  label: string;
}

export interface SegmentedControlProps<T extends string> {
  /** Already-translated group label (drives `aria-label`). */
  label: string;
  value: T;
  options: ReadonlyArray<SegmentedOption<T>>;
  onChange: (value: T) => void;
  "data-testid"?: string;
  /** Per-option test id factory (e.g. for targeting a single segment). */
  optionTestId?: (value: T) => string;
}

/**
 * A small radiogroup of mutually-exclusive pills for a short closed set. All
 * copy arrives pre-translated as props so the primitive stays i18n-clean.
 */
export function SegmentedControl<T extends string>({
  label,
  value,
  options,
  onChange,
  optionTestId,
  ...rest
}: SegmentedControlProps<T>) {
  return (
    <div
      role="radiogroup"
      aria-label={label}
      data-testid={rest["data-testid"]}
      className="inline-flex flex-wrap gap-1 rounded-control border border-app-border bg-app-surface-muted p-1"
    >
      {options.map((option) => {
        const active = option.value === value;
        return (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={active}
            data-testid={optionTestId?.(option.value)}
            onClick={() => onChange(option.value)}
            className={cn(
              "rounded-control px-3 py-1.5 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50",
              active
                ? "bg-app-primary text-app-primary-foreground"
                : "text-app-muted-foreground hover:text-app-foreground",
            )}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}
