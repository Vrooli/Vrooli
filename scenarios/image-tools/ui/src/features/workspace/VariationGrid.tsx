import { Loader2 } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { CreateVariation } from "./useCreate";

export interface VariationGridProps {
  results: CreateVariation[];
  /** Variations the in-flight run requested; sizes the skeleton grid. */
  requestedCount: number;
  /** True while the job is running so empty slots show a pending state. */
  busy: boolean;
  onSendToCanvas: (variation: CreateVariation) => void;
  onSendToEnhance: (variation: CreateVariation) => void;
}

/**
 * The N-slot result grid. While a generation runs it shows `requestedCount`
 * pending slots; once the variations resolve each slot renders its image plus
 * the "send to canvas / Enhance / download" actions. Slots are 1-indexed in
 * test ids so automation can target a specific variation.
 */
export function VariationGrid({
  results,
  requestedCount,
  busy,
  onSendToCanvas,
  onSendToEnhance,
}: VariationGridProps) {
  const { t } = useTranslation();

  const slotCount = results.length > 0 ? results.length : busy ? Math.max(1, requestedCount) : 0;
  if (slotCount === 0) {
    return null;
  }

  const slots = Array.from({ length: slotCount }, (_, i) => results[i] ?? null);

  return (
    <div className="flex flex-col gap-2">
      <h4 className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
        {t(strings.workspace.create.result.heading)}
      </h4>
      <ul
        data-testid={selectors.workspace.create.results}
        className="grid grid-cols-2 gap-2"
      >
        {slots.map((variation, i) => {
          const index = i + 1;
          const slotLabel = t(strings.workspace.create.result.slot, { index });
          return (
            <li
              key={variation ? variation.result.url : `pending-${index}`}
              data-testid={selectors.workspace.createVariation({ index })}
              aria-label={slotLabel}
              className="flex flex-col gap-1 rounded-control border border-app-border bg-app-surface-muted p-1.5"
            >
              {variation ? (
                <>
                  <img
                    src={variation.result.url}
                    alt={slotLabel}
                    className="aspect-square w-full rounded-control border border-app-border object-cover"
                  />
                  <div className="flex flex-wrap gap-1">
                    <button
                      type="button"
                      data-testid={selectors.workspace.createSend({ index })}
                      onClick={() => onSendToCanvas(variation)}
                      className="rounded-control bg-app-primary px-2 py-1 text-xs font-medium text-app-primary-foreground hover:brightness-95"
                    >
                      {t(strings.workspace.create.result.sendToCanvas)}
                    </button>
                    <button
                      type="button"
                      onClick={() => onSendToEnhance(variation)}
                      className="rounded-control border border-app-border px-2 py-1 text-xs text-app-foreground hover:border-app-primary"
                    >
                      {t(strings.workspace.create.result.sendToEnhance)}
                    </button>
                    <a
                      href={variation.result.url}
                      download={`variation-${index}.${variation.result.format || "png"}`}
                      className="rounded-control px-2 py-1 text-xs text-app-primary underline"
                    >
                      {t(strings.workspace.create.result.download)}
                    </a>
                  </div>
                </>
              ) : (
                <div className="flex aspect-square w-full items-center justify-center gap-2 rounded-control border border-dashed border-app-border text-xs text-app-muted-foreground">
                  <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin text-app-brand" />
                  <span>{t(strings.workspace.create.result.pending)}</span>
                </div>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
