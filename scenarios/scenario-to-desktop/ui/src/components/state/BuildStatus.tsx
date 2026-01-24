import { useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  getPipelineStatus,
  type BuildStatus as BuildStatusType,
} from "../../lib/api";
import {
  calculateBuildProgress,
  getBuildStageStatuses,
  pipelineStatusToBuildStatus,
} from "../../domain/build";
import { Badge } from "../ui/badge";
import { Progress } from "../ui/progress";
import { CheckCircle, Loader2 } from "lucide-react";

interface BuildStatusProps {
  buildId: string | null;
  onStatusChange?: (status: BuildStatusType) => void;
}

export function BuildStatus({ buildId, onStatusChange }: BuildStatusProps) {
  const { data: pipelineStatus } = useQuery({
    queryKey: ["build-status", buildId],
    queryFn: () => (buildId ? getPipelineStatus(buildId, { verbose: true }) : null),
    enabled: !!buildId,
    refetchInterval: (query) => {
      const pipelineData = query.state.data;
      // Stop polling when pipeline reaches any final state
      if (pipelineData?.status === "completed" || pipelineData?.status === "failed" || pipelineData?.status === "cancelled") {
        return false;
      }
      return 2000;
    }
  });

  // Map pipeline status to UI-friendly build status using domain function
  const data: BuildStatusType | null = useMemo(
    () => pipelineStatusToBuildStatus(pipelineStatus ?? null),
    [pipelineStatus]
  );

  useEffect(() => {
    if (data && onStatusChange) {
      onStatusChange(data);
    }
  }, [data, onStatusChange]);

  if (!buildId) {
    return null;
  }

  // Calculate progress and stage statuses using domain functions
  const progress = calculateBuildProgress(data);
  const stageStatuses = getBuildStageStatuses(data?.build_log ?? [], progress);

  return (
    <div className="space-y-4">
      <div>
        <div className="mb-1 flex justify-between text-sm">
          <span className="text-slate-400">Build ID:</span>
          <span className="font-mono text-slate-300">{buildId}</span>
        </div>
        {data && (
          <div className="mb-3 flex justify-between text-sm">
            <span className="text-slate-400">Status:</span>
            <Badge
              variant={
                data.status === "ready"
                  ? "success"
                  : data.status === "failed"
                    ? "error"
                    : "warning"
              }
            >
              {data.status}
            </Badge>
          </div>
        )}
        {data && (
          <>
            {data.template_type && (
              <div className="mb-1 flex justify-between text-sm">
                <span className="text-slate-400">Template:</span>
                <span className="text-slate-300">{data.template_type}</span>
              </div>
            )}
            {data.framework && (
              <div className="mb-1 flex justify-between text-sm">
                <span className="text-slate-400">Framework:</span>
                <span className="text-slate-300">{data.framework}</span>
              </div>
            )}
            {data.platforms?.length > 0 && (
              <div className="mb-3 flex justify-between text-sm">
                <span className="text-slate-400">Platforms:</span>
                <span className="text-slate-300">{data.platforms.join(", ")}</span>
              </div>
            )}
          </>
        )}
      </div>

          {/* Build Stages - uses domain-driven stage definitions */}
          {data && data.status === "building" && (
            <div className="space-y-2">
              <div className="text-sm font-medium text-slate-300 mb-3">Build Stages</div>
              {stageStatuses.map(({ stage, completed, active }, i) => (
                <div key={i} className="flex items-center gap-3">
                  {completed ? (
                    <CheckCircle className="h-4 w-4 text-green-400 flex-shrink-0" />
                  ) : active ? (
                    <Loader2 className="h-4 w-4 text-blue-400 animate-spin flex-shrink-0" />
                  ) : (
                    <div className="h-4 w-4 rounded-full border-2 border-slate-600 flex-shrink-0" />
                  )}
                  <span className={`text-sm ${completed ? "text-slate-300" : active ? "text-blue-300" : "text-slate-500"}`}>
                    {stage.name}
                  </span>
                </div>
              ))}
            </div>
          )}

          <div>
            <div className="mb-2 flex justify-between text-sm">
              <span className="text-slate-400">Progress</span>
              <span className="text-slate-300">{progress}%</span>
            </div>
            <Progress value={progress} />
          </div>

          {data?.build_log && data.build_log.length > 0 && (
            <div>
              <div className="mb-2 text-sm text-slate-400">Build Log</div>
              <div className="max-h-48 overflow-y-auto rounded-lg bg-slate-950 p-3 font-mono text-xs text-slate-300">
                {data.build_log.map((log, i) => (
                  <div key={i}>{log}</div>
                ))}
              </div>
            </div>
          )}

          {data?.error_log && data.error_log.length > 0 && (
            <div>
              <div className="mb-2 text-sm text-red-400">Errors</div>
              <div className="max-h-48 overflow-y-auto rounded-lg bg-red-950/30 p-3 font-mono text-xs text-red-300">
                {data.error_log.map((log, i) => (
                  <div key={i}>{log}</div>
                ))}
              </div>
            </div>
          )}

      {data?.status === "ready" && (
        <div className="rounded-lg bg-green-900/20 p-3 text-sm text-green-300">
          <div className="font-semibold">Build completed successfully!</div>
          <div className="mt-1 font-mono text-xs">
            Next steps:
            <br />
            cd {data.output_path}
            <br />
            npm install && npm run dev
          </div>
        </div>
      )}
    </div>
  );
}
