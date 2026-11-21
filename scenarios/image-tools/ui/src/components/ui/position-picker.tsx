import { cn } from "../../lib/utils";

/** The 9 gravity/position tokens the backend accepts (row-major). */
export const POSITION_TOKENS = [
  "top-left",
  "top",
  "top-right",
  "left",
  "center",
  "right",
  "bottom-left",
  "bottom",
  "bottom-right",
] as const;

export type PositionToken = (typeof POSITION_TOKENS)[number];

export interface PositionPickerProps {
  /** Already-translated group label. */
  label: string;
  /** Current token; "" means the backend default (treated as center here). */
  value: string;
  onChange: (value: PositionToken) => void;
  /** Already-translated cell label for a given token. */
  cellLabel: (token: PositionToken) => string;
  "data-testid"?: string;
}

/**
 * A 3×3 grid mapped to the nine gravity tokens. Each cell is a radio; the
 * selected cell shows a filled dot. An empty value selects nothing (the
 * backend then applies its own default).
 */
export function PositionPicker({
  label,
  value,
  onChange,
  cellLabel,
  ...rest
}: PositionPickerProps) {
  return (
    <div
      role="radiogroup"
      aria-label={label}
      data-testid={rest["data-testid"]}
      className="grid w-fit grid-cols-3 gap-1 rounded-control border border-app-border bg-app-surface-muted p-1"
    >
      {POSITION_TOKENS.map((token) => {
        const active = token === value;
        return (
          <button
            key={token}
            type="button"
            role="radio"
            aria-checked={active}
            aria-label={cellLabel(token)}
            title={cellLabel(token)}
            onClick={() => onChange(token)}
            className={cn(
              "flex h-7 w-7 items-center justify-center rounded-control transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50",
              active ? "bg-app-surface" : "hover:bg-app-surface",
            )}
          >
            <span
              aria-hidden="true"
              className={cn(
                "h-2 w-2 rounded-pill",
                active ? "bg-app-primary" : "bg-app-border",
              )}
            />
          </button>
        );
      })}
    </div>
  );
}
