import { ClipboardCheck, Loader2 } from "lucide-react";
import { useCapabilities, useTriggerVisualCapture } from "../lib/hooks";

interface ReviewButtonProps {
  scenarioSlug: string;
  repoId?: string | null;
  onViewReport: () => void;
}

export function ReviewButton({ scenarioSlug, repoId, onViewReport }: ReviewButtonProps) {
  const capabilities = useCapabilities();
  const triggerCapture = useTriggerVisualCapture(repoId);

  const basAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "browser-automation-studio" && c.status === "available"
  ) ?? false;
  const testGenieAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "test-genie" && c.status === "available"
  ) ?? false;

  if (!basAvailable && !testGenieAvailable) return null;

  const isCapturing = triggerCapture.isPending;

  return (
    <button
      type="button"
      className="h-7 px-2 inline-flex items-center gap-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 transition-colors"
      onClick={(e) => {
        e.stopPropagation();
        onViewReport();
      }}
      onContextMenu={(e) => {
        e.preventDefault();
        e.stopPropagation();
        if (basAvailable) {
          triggerCapture.mutate(scenarioSlug);
        }
      }}
      title="Open scenario review"
      disabled={isCapturing}
    >
      {isCapturing ? (
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
      ) : (
        <ClipboardCheck className="h-3.5 w-3.5" />
      )}
    </button>
  );
}
