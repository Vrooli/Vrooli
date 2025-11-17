import { useState, useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";
import { Loader2, RefreshCw, CheckCircle, XCircle, AlertCircle } from "lucide-react";
import type { DesktopConnectionConfig } from "./types";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

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
      const payload: Record<string, unknown> = {
        scenario_name: scenarioName,
        template_type: 'universal'
      };
      if (connectionConfig?.server_url) {
        payload.server_url = connectionConfig.server_url;
      }
      if (connectionConfig?.api_url) {
        payload.api_url = connectionConfig.api_url;
      }
      if (connectionConfig?.deployment_mode) {
        payload.deployment_mode = connectionConfig.deployment_mode;
      }
      if (typeof connectionConfig?.auto_manage_tier1 === 'boolean') {
        payload.auto_manage_tier1 = connectionConfig.auto_manage_tier1;
      }
      if (connectionConfig?.vrooli_binary_path) {
        payload.vrooli_binary_path = connectionConfig.vrooli_binary_path;
      }

      const res = await fetch(buildUrl('/desktop/generate/quick'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to regenerate desktop app');
      }
      return res.json();
    },
    onSuccess: (data) => {
      setBuildId(data.build_id);
      setShowConfirm(false);
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['scenarios-desktop-status'] });
      }, 3000);
    }
  });

  // Poll build status if we have a buildId
  const { data: buildStatus } = useQuery({
    queryKey: ['regenerate-status', buildId],
    queryFn: async () => {
      if (!buildId) return null;
      const res = await fetch(buildUrl(`/desktop/status/${buildId}`));
      if (!res.ok) throw new Error('Failed to fetch build status');
      return res.json();
    },
    enabled: !!buildId,
    refetchInterval: (data) => {
      // Stop polling when build reaches any final state
      if (data?.status === 'ready' || data?.status === 'partial' || data?.status === 'failed') {
        return false;
      }
      return 2000;
    },
  });

  const isBuilding = regenerateMutation.isPending || (buildStatus && buildStatus.status === 'building');
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
