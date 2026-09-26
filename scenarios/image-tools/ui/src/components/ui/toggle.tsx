import { cn } from "../../lib/utils";

export interface ToggleProps {
  /** Already-translated, user-facing label. */
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  "data-testid"?: string;
}

/**
 * An on/off switch (replaces a bare checkbox). The label is clickable and the
 * track exposes `role=switch` + `aria-checked` for assistive tech.
 */
export function Toggle({ label, checked, onChange, ...rest }: ToggleProps) {
  return (
    <label className="flex items-center justify-between gap-3 text-sm text-app-foreground">
      <span>{label}</span>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        data-testid={rest["data-testid"]}
        onClick={() => onChange(!checked)}
        className={cn(
          "relative inline-flex h-6 w-11 shrink-0 items-center rounded-pill border border-app-border transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50",
          checked ? "bg-app-primary" : "bg-app-surface-muted",
        )}
      >
        <span
          aria-hidden="true"
          className={cn(
            "inline-block h-4 w-4 rounded-pill bg-app-surface shadow transition-transform",
            checked ? "translate-x-6" : "translate-x-1",
          )}
        />
      </button>
    </label>
  );
}
