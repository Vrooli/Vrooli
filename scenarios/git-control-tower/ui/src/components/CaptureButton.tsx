import { Camera, Loader2 } from "lucide-react";
import { useCapabilities, useTriggerVisualCapture, useVisualCaptures } from "../lib/hooks";

interface CaptureButtonProps {
  scenarioSlug: string;
  repoId?: string | null;
  onViewReport: () => void;
}

export function CaptureButton({ scenarioSlug, repoId, onViewReport }: CaptureButtonProps) {
  const capabilities = useCapabilities();
  const triggerCapture = useTriggerVisualCapture(repoId);
  const captures = useVisualCaptures(scenarioSlug, true, repoId);

  const basAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "browser-automation-studio" && c.status === "available"
  ) ?? false;

  if (!basAvailable) return null;

  const isCapturing = triggerCapture.isPending;
  const hasCaptures = (captures.data?.total ?? 0) > 0;

  return (
    <button
      type="button"
      className="h-7 px-2 inline-flex items-center gap-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 transition-colors"
      onClick={(e) => {
        e.stopPropagation();
        if (hasCaptures) {
          onViewReport();
        } else {
          triggerCapture.mutate(scenarioSlug);
        }
      }}
      onContextMenu={(e) => {
        e.preventDefault();
        e.stopPropagation();
        triggerCapture.mutate(scenarioSlug);
      }}
      title={hasCaptures ? "View visual report" : "Capture screenshots"}
      disabled={isCapturing}
    >
      {isCapturing ? (
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
      ) : (
        <Camera className="h-3.5 w-3.5" />
      )}
    </button>
  );
}
