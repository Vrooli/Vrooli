import { X } from "lucide-react";

import { composeColor, parseColor } from "./colorMath";

export interface ColorFieldProps {
  /** Already-translated label. */
  label: string;
  /** "" (backend default), "#rrggbb", or "#rrggbbaa". */
  value: string;
  onChange: (value: string) => void;
  /** Already-translated Clear-button label. */
  clearLabel: string;
  /** Already-translated alpha-slider label. */
  alphaLabel: string;
  "data-testid"?: string;
}

/**
 * A native color swatch plus an alpha range and a Clear button. Emits "" when
 * cleared (backend default), a 6-digit hex at full opacity, or an 8-digit hex
 * with alpha. The native `<input type=color>` is the keyboard-accessible core.
 */
export function ColorField({
  label,
  value,
  onChange,
  clearLabel,
  alphaLabel,
  ...rest
}: ColorFieldProps) {
  const { base, alpha } = parseColor(value);
  const hasColor = value !== "";

  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs text-app-muted-foreground">{label}</span>
      <div className="flex items-center gap-2">
        <input
          type="color"
          aria-label={label}
          data-testid={rest["data-testid"]}
          value={base}
          onChange={(e) => onChange(composeColor(e.target.value, alpha))}
          className="h-8 w-10 cursor-pointer rounded-control border border-app-border bg-app-surface-muted p-0.5"
        />
        <input
          type="range"
          aria-label={alphaLabel}
          min={0}
          max={100}
          value={hasColor ? alpha : 100}
          disabled={!hasColor}
          onChange={(e) => onChange(composeColor(base, Number(e.target.value)))}
          style={{ accentColor: "var(--color-primary)" }}
          className="flex-1"
        />
        {hasColor ? (
          <button
            type="button"
            aria-label={clearLabel}
            title={clearLabel}
            onClick={() => onChange("")}
            className="rounded-control p-1 text-app-muted-foreground hover:text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
          >
            <X aria-hidden="true" className="h-4 w-4" />
          </button>
        ) : null}
      </div>
    </div>
  );
}
