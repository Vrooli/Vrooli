import { useState, useEffect } from "react";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";
import { Loader2, RefreshCw, CheckCircle, XCircle, AlertCircle } from "lucide-react";
import type { PipelineConfig } from "../../lib/api";
import { usePipelineMutation, usePipelineStatus } from "../../hooks";
import type { DesktopConnectionConfig } from "./types";

interface RegenerateButtonProps {
  scenarioName: string;
  connectionConfig?: DesktopConnectionConfig;
}

export function RegenerateButton({ scenarioName, connectionConfig }: RegenerateButtonProps) {
  const [showConfirm, setShowConfirm] = useState(false);

  const {
    state: { buildId },
    mutation,
    runPipelineWithConfig,
    reset,
    clearBuildId,
  } = usePipelineMutation({
    invalidateOnSuccess: ["scenarios-desktop-status"],
    onSuccess: () => setShowConfirm(false),
  });

  const { isBuilding, isComplete, isFailed } = usePipelineStatus({
    buildId,
    queryKeyPrefix: "regenerate-status",
  });

  const handleRegenerate = () => {
    const config: PipelineConfig = {
      scenario_name: scenarioName,
      template_type: "universal",
      stop_after_stage: "generate",
    };
    if (connectionConfig?.proxy_url || connectionConfig?.server_url) {
      config.proxy_url = connectionConfig.proxy_url || connectionConfig.server_url;
    }
    if (connectionConfig?.deployment_mode) {
      config.deployment_mode = connectionConfig.deployment_mode as "bundled" | "proxy";
    }
    runPipelineWithConfig(config);
  };

  // Clear buildId after showing success for 3 seconds
  useEffect(() => {
    if (isComplete) {
      const timer = setTimeout(clearBuildId, 3000);
      return () => clearTimeout(timer);
    }
  }, [isComplete, clearBuildId]);

  const combinedIsBuilding = mutation.isPending || isBuilding;

  if (isComplete && !mutation.isPending) {
    return (
      <Badge variant="success" className="gap-1">
        <CheckCircle className="h-3 w-3" />
        Regenerated!
      </Badge>
    );
  }

  if (isFailed && !mutation.isPending) {
    return (
      <div className="flex gap-2">
        <Badge variant="destructive" className="gap-1">
          <XCircle className="h-3 w-3" />
          Failed
        </Badge>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            reset();
            setShowConfirm(true);
          }}
        >
          Retry
        </Button>
      </div>
    );
  }

  if (combinedIsBuilding) {
    return (
      <div className="flex items-center gap-2">
        <Loader2 className="h-4 w-4 animate-spin text-blue-400" />
        <span className="text-sm text-slate-400">Regenerating...</span>
      </div>
    );
  }

  if (showConfirm) {
    return (
      <div className="flex flex-col gap-2 p-3 bg-yellow-950/20 border border-yellow-800/30 rounded">
        <div className="flex items-start gap-2">
          <AlertCircle className="h-4 w-4 text-yellow-400 flex-shrink-0 mt-0.5" />
          <div className="text-xs text-yellow-200">
            <p className="font-semibold">Regenerate desktop app?</p>
            <p className="text-yellow-300/80 mt-1">
              This will overwrite existing files. Make sure you've saved any custom changes.
            </p>
          </div>
        </div>
        <div className="flex gap-2 justify-end">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowConfirm(false)}
            disabled={combinedIsBuilding}
          >
            Cancel
          </Button>
          <Button
            variant="default"
            size="sm"
            onClick={handleRegenerate}
            disabled={combinedIsBuilding}
          >
            <RefreshCw className="h-3 w-3 mr-1" />
            Confirm Regenerate
          </Button>
        </div>
      </div>
    );
  }

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => setShowConfirm(true)}
      disabled={combinedIsBuilding}
      className="gap-1"
    >
      <RefreshCw className="h-4 w-4" />
      Regenerate
    </Button>
  );
}
