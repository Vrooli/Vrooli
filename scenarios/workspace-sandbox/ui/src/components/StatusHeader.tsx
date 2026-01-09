import {
  Box,
  RefreshCw,
  Plus,
  HardDrive,
  Activity,
  CheckCircle,
  XCircle,
  AlertCircle,
  Loader2,
  Settings,
  GitCommit,
} from "lucide-react";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import type { HealthResponse, SandboxStats } from "../lib/api";
import { formatBytes } from "../lib/api";
import { SELECTORS } from "../consts/selectors";

interface StatusHeaderProps {
  health?: HealthResponse;
  stats?: SandboxStats;
  isLoading: boolean;
  onRefresh: () => void;
  onCreateClick: () => void;
  onSettingsClick: () => void;
  onCommitClick: () => void;
}

export function StatusHeader({
  health,
  stats,
  isLoading,
  onRefresh,
  onCreateClick,
  onSettingsClick,
  onCommitClick,
}: StatusHeaderProps) {
  const isHealthy = health?.status === "healthy";
  const driverAvailable = health?.dependencies?.driver === "available";
  const dbConnected = health?.dependencies?.database === "connected";

  return (
    <header
      className="flex items-center justify-between px-4 py-3 border-b border-slate-800 bg-slate-900/50"
      data-testid={SELECTORS.statusHeader}
    >
      {/* Left: Title and Health */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <Box className="h-5 w-5 text-slate-400" />
          <h1 className="text-lg font-semibold text-slate-100">Workspace Sandbox</h1>
        </div>

        {/* Health Indicator */}
        <div
          className="flex items-center gap-2"
          data-testid={SELECTORS.healthIndicator}
        >
          {isHealthy ? (
            <Badge variant="success" className="flex items-center gap-1">
              <CheckCircle className="h-3 w-3" />
              Healthy
            </Badge>
          ) : health ? (
            <Badge variant="error" className="flex items-center gap-1">
              <AlertCircle className="h-3 w-3" />
              Unhealthy
            </Badge>
          ) : isLoading ? (
            <Badge variant="default" className="flex items-center gap-1">
              <Loader2 className="h-3 w-3 animate-spin" />
              Checking...
            </Badge>
          ) : (
            <Badge variant="warning" className="flex items-center gap-1">
              <AlertCircle className="h-3 w-3" />
              Unknown
            </Badge>
          )}

          {/* Driver status */}
          {health && (
            <Badge
              variant={driverAvailable ? "success" : "error"}
              className="flex items-center gap-1 text-[10px]"
            >
              <Activity className="h-2.5 w-2.5" />
              {driverAvailable ? "Driver OK" : "Driver N/A"}
            </Badge>
          )}
        </div>
      </div>

      {/* Center: Stats */}
      {stats && (
        <div
          className="flex items-center gap-6 text-sm"
          data-testid={SELECTORS.statsPanel}
        >
          <div className="flex items-center gap-2 text-slate-400">
            <Box className="h-4 w-4" />
            <span className="text-slate-300 font-medium">{stats.total}</span>
            <span>total</span>
          </div>

          {stats.active > 0 && (
            <div className="flex items-center gap-1.5 text-emerald-400">
              <Activity className="h-3.5 w-3.5" />
              <span className="font-medium">{stats.active}</span>
              <span className="text-emerald-500">active</span>
            </div>
          )}

          {stats.stopped > 0 && (
            <div className="flex items-center gap-1.5 text-amber-400">
              <Box className="h-3.5 w-3.5" />
              <span className="font-medium">{stats.stopped}</span>
              <span className="text-amber-500">stopped</span>
            </div>
          )}

          {stats.error > 0 && (
            <div className="flex items-center gap-1.5 text-red-400">
              <XCircle className="h-3.5 w-3.5" />
              <span className="font-medium">{stats.error}</span>
              <span className="text-red-500">error</span>
            </div>
          )}

          <div className="flex items-center gap-1.5 text-slate-400">
            <HardDrive className="h-3.5 w-3.5" />
            <span className="text-slate-300">{formatBytes(stats.totalSizeBytes)}</span>
          </div>
        </div>
      )}

      {/* Right: Actions */}
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="icon"
          onClick={onCommitClick}
          title="Commit pending changes"
        >
          <GitCommit className="h-4 w-4" />
        </Button>

        <Button
          variant="ghost"
          size="icon"
          onClick={onSettingsClick}
          title="Settings"
        >
          <Settings className="h-4 w-4" />
        </Button>

        <Button
          variant="ghost"
          size="icon"
          onClick={onRefresh}
          disabled={isLoading}
          data-testid={SELECTORS.refreshButton}
          title="Refresh"
        >
          <RefreshCw
            className={`h-4 w-4 ${isLoading ? "animate-spin" : ""}`}
          />
        </Button>

        <Button
          onClick={onCreateClick}
          size="sm"
          data-testid={SELECTORS.createButton}
        >
          <Plus className="h-4 w-4 mr-1.5" />
          Create Sandbox
        </Button>
      </div>
    </header>
  );
}
