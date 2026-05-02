/**
 * OrientationBanner
 *
 * One-time inline banner shown after a successful operating-mode switch.
 * Names the new mode and points the operator at the next action they need
 * to take. Component-local state only — no global toast primitive.
 */

import { Sparkles, X } from "lucide-react";
import { selectors } from "../../../consts/selectors";
import { Button } from "../../ui/button";

export interface OrientationBannerProps {
  title: string;
  description?: string;
  ctaLabel?: string;
  onCta?: () => void;
  onDismiss: () => void;
}

export function OrientationBanner({
  title,
  description,
  ctaLabel,
  onCta,
  onDismiss,
}: OrientationBannerProps) {
  return (
    <div
      role="status"
      data-testid={selectors.initiativeDetails.orientationBanner}
      className="flex items-start gap-3 rounded-lg border border-cyan-500/40 bg-cyan-500/10 px-4 py-3 text-sm text-cyan-100"
    >
      <Sparkles className="mt-0.5 h-4 w-4 shrink-0 text-cyan-300" aria-hidden="true" />
      <div className="flex-1 space-y-1">
        <p className="font-medium text-cyan-100">{title}</p>
        {description ? (
          <p className="text-cyan-200/90">{description}</p>
        ) : null}
        {ctaLabel && onCta ? (
          <div className="pt-1">
            <Button type="button" size="sm" variant="outline" onClick={onCta}>
              {ctaLabel}
            </Button>
          </div>
        ) : null}
      </div>
      <button
        type="button"
        onClick={onDismiss}
        className="rounded p-1 text-cyan-300 transition-colors hover:bg-white/5 hover:text-cyan-100"
        aria-label="Dismiss orientation banner"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
