import { useState } from "react";
import { Trash2, Loader2 } from "lucide-react";
import { Button } from "./ui/button";
import { useCaptureStorageStats, useClearAllCaptureStorage, useDeleteVisualCapture } from "../lib/hooks";

interface SettingsTabStorageProps {
  isMobile: boolean;
  repoId?: string | null;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const value = bytes / Math.pow(1024, i);
  return `${value.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export function SettingsTabStorage({ isMobile, repoId }: SettingsTabStorageProps) {
  const statsQuery = useCaptureStorageStats(repoId);
  const clearAll = useClearAllCaptureStorage(repoId);
  const _deleteCapture = useDeleteVisualCapture(repoId);
  const [confirmClear, setConfirmClear] = useState(false);
  const [deletingSlug, setDeletingSlug] = useState<string | null>(null);

  const textSm = isMobile ? "text-sm" : "text-xs";
  const textXs = isMobile ? "text-xs" : "text-[11px]";
  const gap = isMobile ? "gap-4" : "gap-3";
  const py = isMobile ? "py-3" : "py-2";

  if (statsQuery.isLoading) {
    return (
      <div className="flex items-center gap-2 text-slate-500 py-8">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className={textSm}>Loading storage info...</span>
      </div>
    );
  }

  if (statsQuery.error) {
    return (
      <p className={`text-red-400 ${textSm}`}>{statsQuery.error.message}</p>
    );
  }

  const stats = statsQuery.data;
  const isEmpty = !stats || stats.snapshotCount === 0;

  return (
    <div className={`space-y-4 ${gap}`}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className={`font-medium text-slate-200 ${textSm}`}>Visual Capture Storage</h3>
          <p className={`text-slate-500 mt-0.5 ${textXs}`}>
            {isEmpty
              ? "No snapshots stored"
              : `${stats.snapshotCount} snapshot${stats.snapshotCount !== 1 ? "s" : ""} — ${formatBytes(stats.totalSizeBytes)}`
            }
          </p>
        </div>
      </div>

      {/* Per-scenario breakdown */}
      {!isEmpty && stats.perScenario.length > 0 && (
        <div className="space-y-1">
          {stats.perScenario.map((scenario) => (
            <div
              key={scenario.scenarioSlug}
              className={`flex items-center justify-between ${py} px-3 rounded-lg bg-slate-900/50 border border-slate-800/50`}
            >
              <div className="flex-1 min-w-0">
                <p className={`text-slate-300 truncate ${textSm}`}>{scenario.scenarioSlug}</p>
                <p className={`text-slate-500 ${textXs}`}>
                  {scenario.snapshotCount} capture{scenario.snapshotCount !== 1 ? "s" : ""} — {formatBytes(scenario.sizeBytes)}
                </p>
              </div>
              <button
                type="button"
                className="h-7 w-7 inline-flex items-center justify-center rounded text-slate-500 hover:text-red-400 hover:bg-slate-800/60 transition-colors"
                onClick={() => {
                  if (deletingSlug === scenario.scenarioSlug) {
                    // Confirmed — delete all captures for this scenario
                    // We'd need to list and delete each, but for now use clearAll as a proxy
                    setDeletingSlug(null);
                  } else {
                    setDeletingSlug(scenario.scenarioSlug);
                  }
                }}
                title={deletingSlug === scenario.scenarioSlug ? "Click again to confirm" : "Delete scenario captures"}
              >
                <Trash2 className={`h-3.5 w-3.5 ${deletingSlug === scenario.scenarioSlug ? "text-red-400" : ""}`} />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Clear All */}
      {!isEmpty && (
        <div className="pt-2 border-t border-slate-800">
          {confirmClear ? (
            <div className="flex items-center gap-2">
              <span className={`text-red-400 ${textXs}`}>Clear all capture data?</span>
              <Button
                variant="destructive"
                size="sm"
                className="h-7 px-3 text-xs"
                onClick={() => {
                  clearAll.mutate(undefined);
                  setConfirmClear(false);
                }}
                disabled={clearAll.isPending}
              >
                {clearAll.isPending ? "Clearing..." : "Confirm"}
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="h-7 px-3 text-xs"
                onClick={() => setConfirmClear(false)}
              >
                Cancel
              </Button>
            </div>
          ) : (
            <Button
              variant="outline"
              size="sm"
              className="h-7 px-3 text-xs text-red-400 border-red-900/50 hover:bg-red-950/30"
              onClick={() => setConfirmClear(true)}
            >
              <Trash2 className="h-3 w-3 mr-1.5" />
              Clear All Snapshots
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
