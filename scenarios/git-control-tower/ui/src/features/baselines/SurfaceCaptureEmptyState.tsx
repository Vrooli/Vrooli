// SurfaceCaptureEmptyState (Plan C Phase 2) — the shared "nothing captured yet"
// state for every baseline-includable surface tab. It offers two intents
// (Decision 2):
//   • Capture <surface>  — run the tool, show the result, create NO manifest.
//                          For mid-change progress checks.
//   • Capture baseline   — open SetBaselineModal scoped to this surface.
// When the owning service is unavailable, capture is disabled with an
// explanation rather than hidden, so the affordance stays discoverable.

import type { ReactNode } from "react";
import { Anchor, Camera, Loader2 } from "lucide-react";
import { Button } from "../../components/ui/button";

export function SurfaceCaptureEmptyState({
  label,
  hasService,
  onCaptureLoose,
  onCaptureBaseline,
  captureLabel,
  isCapturing = false,
  serviceMessage,
  icon,
}: {
  label: string;
  hasService: boolean;
  onCaptureLoose: () => void;
  onCaptureBaseline: () => void;
  /** Verb shown on the loose-capture button, e.g. "Capture screenshots". */
  captureLabel?: string;
  isCapturing?: boolean;
  /** Shown when hasService is false (e.g. "Start test-genie to run tests"). */
  serviceMessage?: string;
  icon?: ReactNode;
}) {
  const normalizedLabel = label.toLowerCase();
  const looseLabel = captureLabel ?? `Capture ${normalizedLabel}`;

  return (
    <div className="flex flex-col items-center justify-center py-12 text-slate-500">
      {icon}
      <p className="text-sm">No {normalizedLabel} captured yet</p>
      {hasService ? (
        <>
          <p className="text-xs mt-1 mb-3 text-slate-600 text-center max-w-xs">
            Capture the current state to check progress, or capture a baseline to
            track changes against it.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={onCaptureLoose}
              disabled={isCapturing}
              className="h-7 text-xs gap-1.5"
            >
              {isCapturing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Camera className="h-3.5 w-3.5" />}
              {looseLabel}
            </Button>
            <Button
              size="sm"
              onClick={onCaptureBaseline}
              disabled={isCapturing}
              className="h-7 text-xs gap-1.5"
            >
              <Anchor className="h-3.5 w-3.5" />
              Capture baseline
            </Button>
          </div>
        </>
      ) : (
        <p className="text-xs mt-1 text-slate-600 text-center max-w-xs">
          {serviceMessage ?? `Start the owning service to capture ${normalizedLabel}.`}
        </p>
      )}
    </div>
  );
}
