import { cn } from "../../lib/utils";

export interface FilterOption {
  value: string;
  /** Already-translated, user-facing label. */
  label: string;
  /** CSS `filter` value approximating the effect (e.g. "grayscale(1)"). */
  css: string;
}

export interface FilterThumbnailGridProps {
  /** Already-translated group label. */
  label: string;
  value: string;
  options: readonly FilterOption[];
  /** Preview image to show the effect on; falls back to labelled tiles. */
  previewUrl: string | null;
  onChange: (value: string) => void;
  "data-testid"?: string;
}

/**
 * A grid of filter tiles. Each tile previews the loaded image with a CSS-filter
 * approximation of the effect; when no image is loaded it falls back to a
 * labelled tile so the control still works headlessly + for keyboard users.
 */
export function FilterThumbnailGrid({
  label,
  value,
  options,
  previewUrl,
  onChange,
  ...rest
}: FilterThumbnailGridProps) {
  return (
    <div
      role="radiogroup"
      aria-label={label}
      data-testid={rest["data-testid"]}
      className="grid grid-cols-3 gap-2"
    >
      {options.map((option) => {
        const active = option.value === value;
        return (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={active}
            onClick={() => onChange(option.value)}
            className={cn(
              "flex flex-col items-center gap-1 rounded-control border p-1 text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50",
              active
                ? "border-app-primary bg-app-surface-muted text-app-foreground"
                : "border-app-border text-app-muted-foreground hover:text-app-foreground",
            )}
          >
            {previewUrl ? (
              <img
                src={previewUrl}
                alt={option.label}
                style={{ filter: option.css }}
                className="h-12 w-full rounded-control object-cover"
              />
            ) : (
              <span
                aria-hidden="true"
                style={{ filter: option.css }}
                className="flex h-12 w-full items-center justify-center rounded-control bg-app-surface-muted"
              >
                <span className="h-6 w-6 rounded-pill bg-app-border" />
              </span>
            )}
            <span>{option.label}</span>
          </button>
        );
      })}
    </div>
  );
}
