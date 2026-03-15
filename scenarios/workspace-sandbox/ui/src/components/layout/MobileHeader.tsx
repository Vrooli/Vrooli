import { useState } from "react";
import {
  Box,
  RefreshCw,
  Plus,
  MoreVertical,
  Settings,
  GitCommitVertical,
  Activity,
  Server,
  Database,
  HardDrive,
} from "lucide-react";
import { BottomSheet, BottomSheetAction } from "../ui/bottom-sheet";
import { Badge } from "../ui/badge";
import { SELECTORS } from "../../consts/selectors";
import type { HealthResponse, SandboxStats } from "../../lib/api";
import { formatBytes } from "../../lib/api";

interface MobileHeaderProps {
  health?: HealthResponse;
  stats?: SandboxStats;
  isLoading: boolean;
  onRefresh: () => void;
  onCreateClick: () => void;
  onSettingsClick: () => void;
  onCommitClick: () => void;
}

export function MobileHeader({
  health,
  stats,
  isLoading,
  onRefresh,
  onCreateClick,
  onSettingsClick,
  onCommitClick,
}: MobileHeaderProps) {
  const [showMore, setShowMore] = useState(false);

  const isHealthy = health?.status === "healthy";
  const driverAvailable = health?.dependencies?.driver === "available";
  const dbConnected = health?.dependencies?.database === "connected";

  return (
    <>
      <header
        className="flex items-center justify-between px-4 py-2 border-b border-slate-800 bg-slate-950/95 pt-safe"
        data-testid={SELECTORS.mobileHeader}
      >
        {/* Left: Logo + title */}
        <div className="flex items-center gap-2">
          <Box className="h-5 w-5 text-slate-400" />
          <span className="text-sm font-semibold text-slate-100">WSB</span>
          {/* Health dot */}
          <span
            className={`h-2 w-2 rounded-full ${isHealthy ? "bg-emerald-400" : "bg-red-400"}`}
            title={isHealthy ? "Healthy" : "Unhealthy"}
          />
        </div>

        {/* Right: Action buttons */}
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={onRefresh}
            className="p-2 rounded-lg text-slate-400 hover:text-slate-200 active:bg-slate-800 touch-target"
            aria-label="Refresh"
            disabled={isLoading}
            data-testid={SELECTORS.refreshButton}
          >
            <RefreshCw className={`h-4.5 w-4.5 ${isLoading ? "animate-spin" : ""}`} />
          </button>
          <button
            type="button"
            onClick={onCreateClick}
            className="p-2 rounded-lg text-slate-400 hover:text-slate-200 active:bg-slate-800 touch-target"
            aria-label="Create sandbox"
            data-testid={SELECTORS.createButton}
          >
            <Plus className="h-4.5 w-4.5" />
          </button>
          <button
            type="button"
            onClick={() => setShowMore(true)}
            className="p-2 rounded-lg text-slate-400 hover:text-slate-200 active:bg-slate-800 touch-target"
            aria-label="More options"
            data-testid={SELECTORS.mobileHeaderMore}
          >
            <MoreVertical className="h-4.5 w-4.5" />
          </button>
        </div>
      </header>

      {/* More menu bottom sheet */}
      <BottomSheet
        isOpen={showMore}
        onClose={() => setShowMore(false)}
        title="Options"
      >
        {/* Stats summary */}
        {stats && (
          <div className="mb-4 p-3 rounded-xl bg-slate-900/60 border border-slate-800">
            <div className="grid grid-cols-3 gap-3 text-center">
              <div>
                <div className="text-lg font-bold text-slate-100">{stats.total}</div>
                <div className="text-[10px] text-slate-500 uppercase tracking-wider">Total</div>
              </div>
              <div>
                <div className="text-lg font-bold text-emerald-400">{stats.active}</div>
                <div className="text-[10px] text-slate-500 uppercase tracking-wider">Active</div>
              </div>
              <div>
                <div className="text-lg font-bold text-red-400">{stats.error}</div>
                <div className="text-[10px] text-slate-500 uppercase tracking-wider">Error</div>
              </div>
            </div>
            {stats.totalSizeBytes > 0 && (
              <div className="mt-2 pt-2 border-t border-slate-800 text-center">
                <span className="text-xs text-slate-500">
                  <HardDrive className="h-3 w-3 inline mr-1" />
                  {formatBytes(stats.totalSizeBytes)} total
                </span>
              </div>
            )}
          </div>
        )}

        {/* Status indicators */}
        <div className="mb-4 px-4 flex items-center gap-4 text-xs text-slate-500">
          <span className="flex items-center gap-1.5">
            <Server className="h-3.5 w-3.5" />
            Driver: {" "}
            <Badge variant={driverAvailable ? "success" : "error"} className="text-[10px] px-1.5 py-0">
              {driverAvailable ? "OK" : "N/A"}
            </Badge>
          </span>
          <span className="flex items-center gap-1.5">
            <Database className="h-3.5 w-3.5" />
            DB: {" "}
            <Badge variant={dbConnected ? "success" : "error"} className="text-[10px] px-1.5 py-0">
              {dbConnected ? "OK" : "N/A"}
            </Badge>
          </span>
        </div>

        {/* Actions */}
        <BottomSheetAction
          icon={<GitCommitVertical className="h-5 w-5 text-slate-400" />}
          label="Commit Pending Changes"
          description="Review and commit pending workspace changes"
          onClick={() => {
            setShowMore(false);
            onCommitClick();
          }}
        />
        <BottomSheetAction
          icon={<Settings className="h-5 w-5 text-slate-400" />}
          label="Settings"
          description="Configure workspace sandbox settings"
          onClick={() => {
            setShowMore(false);
            onSettingsClick();
          }}
        />
        <BottomSheetAction
          icon={<Activity className="h-5 w-5 text-slate-400" />}
          label="Refresh"
          description="Reload sandbox data and health status"
          onClick={() => {
            setShowMore(false);
            onRefresh();
          }}
        />
      </BottomSheet>
    </>
  );
}
