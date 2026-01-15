import { useState, useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";
import { Loader2, RefreshCw, CheckCircle, XCircle, AlertCircle } from "lucide-react";
import { runPipeline, getPipelineStatus, type PipelineConfig } from "../../lib/api";
import type { DesktopConnectionConfig } from "./types";

/** Maps pipeline status to UI-friendly status */
function mapPipelineStatus(status: string): "building" | "ready" | "partial" | "failed" {
  switch (status) {
    case "pending":
    case "running":
      return "building";
    case "completed":
      return "ready";
    case "failed":
    case "cancelled":
      return "failed";
    default:
      return "building";
  }
}

interface RegenerateButtonProps {
  scenarioName: string;
  connectionConfig?: DesktopConnectionConfig;
}

export function RegenerateButton({ scenarioName, connectionConfig }: RegenerateButtonProps) {
  const queryClient = useQueryClient();
  const [showConfirm, setShowConfirm] = useState(false);
  const [buildId, setBuildId] = useState<string | null>(null);

  const regenerateMutation = useMutation({
    mutationFn: async () => {
      const config: PipelineConfig = {
        scenario_name: scenarioName,
        template_type: 'universal',
        stop_after_stage: 'generate',
      };
      if (connectionConfig?.proxy_url || connectionConfig?.server_url) {
        config.proxy_url = connectionConfig.proxy_url || connectionConfig.server_url;
      }
      if (connectionConfig?.deployment_mode) {
        config.deployment_mode = connectionConfig.deployment_mode as "bundled" | "proxy";
      }
      return runPipeline(config);
    },
    onSuccess: (data) => {
      setBuildId(data.pipeline_id);
      setShowConfirm(false);
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['scenarios-desktop-status'] });
      }, 3000);
    }
  });

  // Poll pipeline status if we have a buildId
  const { data: pipelineStatus } = useQuery({
    queryKey: ['regenerate-status', buildId],
    queryFn: async () => (buildId ? getPipelineStatus(buildId) : null),
    enabled: !!buildId,
    refetchInterval: (query) => {
      // Stop polling when pipeline reaches any final state
      const data = query.state.data;
      if (data?.status === 'completed' || data?.status === 'failed' || data?.status === 'cancelled') {
        return false;
      }
      return 2000;
    },
  });

  // Map pipeline status to UI-friendly status
  const buildStatus = pipelineStatus ? { status: mapPipelineStatus(pipelineStatus.status) } : null;

  const isBuilding = regenerateMutation.isPending || buildStatus?.status === 'building';
  // Consider both 'ready' and 'partial' as successful (at least generated the desktop wrapper)
  const isComplete = (buildStatus?.status === 'ready' || buildStatus?.status === 'partial') && !regenerateMutation.isPending;
  const isFailed = buildStatus?.status === 'failed' && !regenerateMutation.isPending;

  // Clear buildId after showing success for 3 seconds
  useEffect(() => {
    if (isComplete) {
      const timer = setTimeout(() => {
        setBuildId(null);
      }, 3000);
      return () => clearTimeout(timer);
    }
  }, [isComplete]);

  if (isComplete) {
    return (
      <Badge variant="success" className="gap-1">
        <CheckCircle className="h-3 w-3" />
        Regenerated!
      </Badge>
    );
  }

  if (isFailed) {
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
            setBuildId(null);
            regenerateMutation.reset();
            setShowConfirm(true);
          }}
        >
          Retry
        </Button>
      </div>
    );
  }

  if (isBuilding) {
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
            <p className="text-yellow-300/80 mt-1">This will overwrite existing files. Make sure you've saved any custom changes.</p>
          </div>
        </div>
        <div className="flex gap-2 justify-end">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowConfirm(false)}
            disabled={isBuilding}
          >
            Cancel
          </Button>
          <Button
            variant="default"
            size="sm"
            onClick={() => regenerateMutation.mutate()}
            disabled={isBuilding}
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
      disabled={isBuilding}
      className="gap-1"
    >
      <RefreshCw className="h-4 w-4" />
      Regenerate
    </Button>
  );
}
