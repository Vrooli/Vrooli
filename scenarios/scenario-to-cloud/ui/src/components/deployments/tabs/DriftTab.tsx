import { useState } from "react";
import {
  RefreshCw,
  Loader2,
  AlertCircle,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  Server,
  HardDrive,
  Network,
  Shield,
  Play,
  Trash2,
} from "lucide-react";
import { useDrift, useRestartProcess, useKillProcess } from "../../../hooks/useLiveState";
import type { DriftCheck, DriftSummary } from "../../../lib/api";
import { cn } from "../../../lib/utils";

interface DriftTabProps {
  deploymentId: string;
}

export function DriftTab({ deploymentId }: DriftTabProps) {
  const { data: drift, isLoading, error, refetch, isFetching } = useDrift(deploymentId);
  const restartMutation = useRestartProcess(deploymentId);
  const killMutation = useKillProcess(deploymentId);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 rounded-lg border border-red-500/30 bg-red-500/10 text-red-400">
        <div className="flex items-center gap-2">
          <AlertCircle className="h-5 w-5" />
          <span>Failed to load drift report: {error.message}</span>
        </div>
      </div>
    );
  }

  if (!drift) {
    return (
      <div className="p-4 rounded-lg border border-amber-500/30 bg-amber-500/10 text-amber-400">
        <div className="flex items-center gap-2">
          <AlertCircle className="h-5 w-5" />
          <span>Unable to fetch drift report</span>
        </div>
      </div>
    );
  }

  const handleRestart = (type: "scenario" | "resource", id: string) => {
    restartMutation.mutate({ type, id });
  };

  const handleKill = (pid: number) => {
    killMutation.mutate({ pid, signal: "TERM" });
  };

  return (
    <div className="space-y-4">
      {/* Header with refresh */}
      <div className="flex items-center justify-between">
        <div className="text-sm text-slate-400">
          Comparing manifest expectations vs live state
        </div>
        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className={cn(
            "flex items-center gap-2 px-3 py-1.5 rounded-lg border border-white/10",
            "hover:bg-white/5 transition-colors text-sm font-medium",
            isFetching && "opacity-50 cursor-not-allowed"
          )}
        >
          {isFetching ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
          Re-check
        </button>
      </div>

      {/* Summary badges */}
      <DriftSummaryBadges summary={drift.summary} />

      {/* Group checks by category */}
      <div className="space-y-4">
        <DriftCategory
          title="Scenarios"
          icon={Server}
          checks={drift.checks.filter(c => c.category === "scenarios")}
          onRestart={(id) => handleRestart("scenario", id)}
          onKill={handleKill}
        />
        <DriftCategory
          title="Resources"
          icon={HardDrive}
          checks={drift.checks.filter(c => c.category === "resources")}
          onRestart={(id) => handleRestart("resource", id)}
          onKill={handleKill}
        />
        <DriftCategory
          title="Ports"
          icon={Network}
          checks={drift.checks.filter(c => c.category === "ports")}
          onRestart={() => {}}
          onKill={handleKill}
        />
        <DriftCategory
          title="Edge/TLS"
          icon={Shield}
          checks={drift.checks.filter(c => c.category === "edge")}
          onRestart={() => {}}
          onKill={handleKill}
        />
      </div>

      {/* Action feedback */}
      {restartMutation.isPending && (
        <div className="p-3 rounded-lg border border-blue-500/30 bg-blue-500/10 text-blue-400 text-sm flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          Restarting...
        </div>
      )}
      {killMutation.isPending && (
        <div className="p-3 rounded-lg border border-amber-500/30 bg-amber-500/10 text-amber-400 text-sm flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          Killing process...
        </div>
      )}
    </div>
  );
}

interface DriftSummaryBadgesProps {
  summary: DriftSummary;
}

function DriftSummaryBadges({ summary }: DriftSummaryBadgesProps) {
  return (
    <div className="flex items-center gap-3 p-4 border border-white/10 rounded-lg bg-slate-900/50">
      <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-emerald-500/20 text-emerald-400">
        <CheckCircle2 className="h-4 w-4" />
        <span className="text-sm font-medium">{summary.passed} passed</span>
      </div>
      {summary.warnings > 0 && (
        <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-amber-500/20 text-amber-400">
          <AlertTriangle className="h-4 w-4" />
          <span className="text-sm font-medium">{summary.warnings} warnings</span>
        </div>
      )}
      {summary.drifts > 0 && (
        <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-red-500/20 text-red-400">
          <XCircle className="h-4 w-4" />
          <span className="text-sm font-medium">{summary.drifts} drifts</span>
        </div>
      )}
    </div>
  );
}

interface DriftCategoryProps {
  title: string;
  icon: React.ElementType;
  checks: DriftCheck[];
  onRestart: (id: string) => void;
  onKill: (pid: number) => void;
}

function DriftCategory({ title, icon: Icon, checks, onRestart, onKill }: DriftCategoryProps) {
  if (checks.length === 0) {
    return null;
  }

  return (
    <div className="border border-white/10 rounded-lg bg-slate-900/50">
      <div className="p-4 border-b border-white/10 flex items-center gap-2">
        <Icon className="h-4 w-4 text-slate-400" />
        <h3 className="text-sm font-medium text-white">{title}</h3>
      </div>
      <div className="divide-y divide-white/10">
        {checks.map((check, index) => (
          <DriftCheckRow
            key={`${check.category}-${check.name}-${index}`}
            check={check}
            onRestart={onRestart}
            onKill={onKill}
          />
        ))}
      </div>
    </div>
  );
}

interface DriftCheckRowProps {
  check: DriftCheck;
  onRestart: (id: string) => void;
  onKill: (pid: number) => void;
}

function DriftCheckRow({ check, onRestart, onKill }: DriftCheckRowProps) {
  const [expanded, setExpanded] = useState(check.status !== "pass");

  const statusColors = {
    pass: "text-emerald-400",
    warning: "text-amber-400",
    drift: "text-red-400",
  };

  const statusBgColors = {
    pass: "bg-emerald-500/10",
    warning: "bg-amber-500/10",
    drift: "bg-red-500/10",
  };

  const StatusIcon = {
    pass: CheckCircle2,
    warning: AlertTriangle,
    drift: XCircle,
  }[check.status];

  // Extract PID from actual string for kill action
  const pidMatch = check.actual.match(/PID\s*(\d+)/i);
  const pid = pidMatch ? parseInt(pidMatch[1], 10) : null;

  return (
    <div className={cn("p-4", statusBgColors[check.status])}>
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 flex-1">
          <StatusIcon className={cn("h-5 w-5 mt-0.5", statusColors[check.status])} />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium text-white">{check.name}</span>
            </div>
            <div className="mt-1 text-sm text-slate-400">
              <div>Expected: <span className="text-slate-300">{check.expected}</span></div>
              <div>Actual: <span className="text-slate-300">{check.actual}</span></div>
            </div>
            {check.message && (
              <p className="mt-2 text-sm text-slate-400">{check.message}</p>
            )}
          </div>
        </div>

        {/* Actions */}
        {check.actions && check.actions.length > 0 && (
          <div className="flex items-center gap-2 flex-shrink-0">
            {check.actions.includes("restart_scenario") && (
              <button
                onClick={() => onRestart(check.name)}
                className="flex items-center gap-1 px-2 py-1 rounded text-xs bg-blue-500/20 text-blue-400 hover:bg-blue-500/30 transition-colors"
              >
                <Play className="h-3 w-3" />
                Restart
              </button>
            )}
            {check.actions.includes("restart_resource") && (
              <button
                onClick={() => onRestart(check.name)}
                className="flex items-center gap-1 px-2 py-1 rounded text-xs bg-blue-500/20 text-blue-400 hover:bg-blue-500/30 transition-colors"
              >
                <Play className="h-3 w-3" />
                Restart
              </button>
            )}
            {check.actions.includes("stop") && (
              <button
                onClick={() => onRestart(check.name)}
                className="flex items-center gap-1 px-2 py-1 rounded text-xs bg-amber-500/20 text-amber-400 hover:bg-amber-500/30 transition-colors"
              >
                Stop
              </button>
            )}
            {check.actions.includes("kill_pid") && pid && (
              <button
                onClick={() => onKill(pid)}
                className="flex items-center gap-1 px-2 py-1 rounded text-xs bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors"
              >
                <Trash2 className="h-3 w-3" />
                Kill
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
