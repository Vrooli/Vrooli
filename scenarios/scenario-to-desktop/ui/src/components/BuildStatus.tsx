import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchBuildStatus, type BuildStatus as BuildStatusType } from "../lib/api";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Badge } from "./ui/badge";
import { Progress } from "./ui/progress";
import { Terminal, CheckCircle, Loader2 } from "lucide-react";

interface BuildStatusProps {
  buildId: string | null;
  onStatusChange?: (status: BuildStatusType) => void;
}

export function BuildStatus({ buildId, onStatusChange }: BuildStatusProps) {
  const { data } = useQuery({
    queryKey: ["build-status", buildId],
    queryFn: () => (buildId ? fetchBuildStatus(buildId) : null),
    enabled: !!buildId,
    refetchInterval: (query) => {
      const data = query.state.data as BuildStatusType | null;
      // Stop polling when build is complete or failed
      return data?.status === "ready" || data?.status === "failed" ? false : 2000;
    }
  });

  useEffect(() => {
    if (data && onStatusChange) {
      onStatusChange(data);
    }
  }, [data, onStatusChange]);

  if (!buildId) {
    return null;
  }

  // Calculate progress based on build log
  const calculateProgress = (status: BuildStatusType | undefined): number => {
    if (!status) return 0;
    if (status.status === "ready") return 100;
    if (status.status === "failed") return 0;

    // Estimate progress based on log entries
    const logs = status.build_log || [];
    if (logs.length === 0) return 10; // Just started

    const hasTemplateGeneration = logs.some(log => log.includes("Generating") || log.includes("Creating"));
    const hasDependencyInstall = logs.some(log => log.includes("npm install") || log.includes("Installing"));
    const hasCompilation = logs.some(log => log.includes("compiling") || log.includes("building"));
    const hasPackaging = logs.some(log => log.includes("packaging") || log.includes("electron-builder"));

    let progress = 10; // Starting
    if (hasTemplateGeneration) progress = 25;
    if (hasDependencyInstall) progress = 50;
    if (hasCompilation) progress = 75;
    if (hasPackaging) progress = 90;

    return progress;
  };

  const progress = calculateProgress(data);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span className="flex items-center gap-2">
            <Terminal className="h-5 w-5" />
            Build Status
          </span>
          {data && (
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
          )}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div>
            <div className="mb-1 flex justify-between text-sm">
              <span className="text-slate-400">Build ID:</span>
              <span className="font-mono text-slate-300">{buildId}</span>
            </div>
            {data && (
              <>
                <div className="mb-1 flex justify-between text-sm">
                  <span className="text-slate-400">Template:</span>
                  <span className="text-slate-300">{data.template_type}</span>
                </div>
                <div className="mb-1 flex justify-between text-sm">
                  <span className="text-slate-400">Framework:</span>
                  <span className="text-slate-300">{data.framework}</span>
                </div>
                <div className="mb-3 flex justify-between text-sm">
                  <span className="text-slate-400">Platforms:</span>
                  <span className="text-slate-300">{data.platforms.join(", ")}</span>
                </div>
              </>
            )}
          </div>

          {/* Build Stages */}
          {data && data.status === "building" && (
            <div className="space-y-2">
              <div className="text-sm font-medium text-slate-300 mb-3">Build Stages</div>
              {[
                { name: "Template Generation", key: ["Generating", "Creating"], progress: 25 },
                { name: "Installing Dependencies", key: ["npm install", "Installing"], progress: 50 },
                { name: "Compiling TypeScript", key: ["compiling", "building"], progress: 75 },
                { name: "Packaging Application", key: ["packaging", "electron-builder"], progress: 90 },
              ].map((stage, i) => {
                const completed = stage.key.some(k =>
                  data.build_log?.some(log => log.includes(k))
                );
                const isActive = progress >= stage.progress - 25 && progress < stage.progress;

                return (
                  <div key={i} className="flex items-center gap-3">
                    {completed ? (
                      <CheckCircle className="h-4 w-4 text-green-400 flex-shrink-0" />
                    ) : isActive ? (
                      <Loader2 className="h-4 w-4 text-blue-400 animate-spin flex-shrink-0" />
                    ) : (
                      <div className="h-4 w-4 rounded-full border-2 border-slate-600 flex-shrink-0" />
                    )}
                    <span className={`text-sm ${completed ? "text-slate-300" : isActive ? "text-blue-300" : "text-slate-500"}`}>
                      {stage.name}
                    </span>
                  </div>
                );
              })}
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
      </CardContent>
    </Card>
  );
}
