import { RotateCcw } from "lucide-react";

export interface SliderProps {
  /** Already-translated, user-facing label. */
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  /** Symbol shown after the value (e.g. "px", "°", "%"). Non-letter only. */
  unit?: string;
  /** The spec default; the Reset button appears only when value differs. */
  defaultValue: number;
  /** Already-translated reset aria-label. */
  resetLabel: string;
  onChange: (value: number) => void;
  "data-testid"?: string;
}

/**
 * A labelled range input with a live value + unit readout and a per-control
 * Reset button that only appears once the value drifts from its default. Copy
 * arrives pre-translated; only non-letter symbols (the unit) render inline.
 */
export function Slider({
  label,
  value,
  min,
  max,
  step = 1,
  unit,
  defaultValue,
  resetLabel,
  onChange,
  ...rest
}: SliderProps) {
  const dirty = value !== defaultValue;

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-app-muted-foreground">{label}</span>
        <span className="flex items-center gap-1 text-xs tabular-nums text-app-foreground">
          <span>
            {value}
            {unit ? <span aria-hidden="true">{unit}</span> : null}
          </span>
          {dirty ? (
            <button
              type="button"
              aria-label={resetLabel}
              title={resetLabel}
              onClick={() => onChange(defaultValue)}
              className="rounded-control p-0.5 text-app-muted-foreground hover:text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
            >
              <RotateCcw aria-hidden="true" className="h-3.5 w-3.5" />
            </button>
          ) : null}
        </span>
      </div>
      <input
        type="range"
        aria-label={label}
        data-testid={rest["data-testid"]}
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        style={{ accentColor: "var(--color-primary)" }}
        className="w-full"
      />
    </div>
  );
}
