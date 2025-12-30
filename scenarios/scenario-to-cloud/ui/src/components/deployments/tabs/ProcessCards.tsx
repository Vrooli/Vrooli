import {
  Activity,
  Database,
  AlertTriangle,
  RefreshCw,
  XCircle,
  Cpu,
  HardDrive,
  Loader2,
  CheckCircle2,
} from "lucide-react";
import { useRestartProcess, useKillProcess, formatUptime } from "../../../hooks/useLiveState";
import type { ProcessState, ScenarioProcess, ResourceProcess, UnexpectedProcess } from "../../../lib/api";
import { cn } from "../../../lib/utils";

interface ProcessCardsProps {
  deploymentId: string;
  processes: ProcessState;
}

export function ProcessCards({ deploymentId, processes }: ProcessCardsProps) {
  const restartMutation = useRestartProcess(deploymentId);
  const killMutation = useKillProcess(deploymentId);

  return (
    <div className="space-y-6">
      {/* Scenarios */}
      <div>
        <h3 className="text-sm font-medium text-slate-400 mb-3 flex items-center gap-2">
          <Activity className="h-4 w-4" />
          Scenarios ({processes.scenarios.length})
        </h3>
        {processes.scenarios.length === 0 ? (
          <p className="text-sm text-slate-500">No scenarios running</p>
        ) : (
          <div className="grid gap-3">
            {processes.scenarios.map((scenario) => (
              <ScenarioCard
                key={scenario.id}
                scenario={scenario}
                onRestart={() =>
                  restartMutation.mutate({ type: "scenario", id: scenario.id })
                }
                isRestarting={restartMutation.isPending}
              />
            ))}
          </div>
        )}
      </div>

      {/* Resources */}
      <div>
        <h3 className="text-sm font-medium text-slate-400 mb-3 flex items-center gap-2">
          <Database className="h-4 w-4" />
          Resources ({processes.resources.length})
        </h3>
        {processes.resources.length === 0 ? (
          <p className="text-sm text-slate-500">No resources running</p>
        ) : (
          <div className="grid gap-3">
            {processes.resources.map((resource) => (
              <ResourceCard
                key={resource.id}
                resource={resource}
                onRestart={() =>
                  restartMutation.mutate({ type: "resource", id: resource.id })
                }
                isRestarting={restartMutation.isPending}
              />
            ))}
          </div>
        )}
      </div>

      {/* Unexpected Processes */}
      {processes.unexpected.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-amber-400 mb-3 flex items-center gap-2">
            <AlertTriangle className="h-4 w-4" />
            Unexpected Processes ({processes.unexpected.length})
          </h3>
          <div className="grid gap-3">
            {processes.unexpected.map((proc) => (
              <UnexpectedCard
                key={proc.pid}
                process={proc}
                onKill={() => killMutation.mutate({ pid: proc.pid })}
                isKilling={killMutation.isPending}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

interface ScenarioCardProps {
  scenario: ScenarioProcess;
  onRestart: () => void;
  isRestarting: boolean;
}

function ScenarioCard({ scenario, onRestart, isRestarting }: ScenarioCardProps) {
  const isRunning = scenario.status === "running";

  return (
    <div
      className={cn(
        "border rounded-lg p-4",
        isRunning
          ? "border-emerald-500/30 bg-emerald-500/5"
          : "border-red-500/30 bg-red-500/5"
      )}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div
            className={cn(
              "p-2 rounded-lg",
              isRunning ? "bg-emerald-500/20" : "bg-red-500/20"
            )}
          >
            <Activity
              className={cn(
                "h-5 w-5",
                isRunning ? "text-emerald-400" : "text-red-400"
              )}
            />
          </div>
          <div>
            <h4 className="font-medium text-white">{scenario.id}</h4>
            <div className="flex items-center gap-2 mt-1">
              <StatusBadge status={scenario.status} />
              <span className="text-xs text-slate-500">PID: {scenario.pid}</span>
              {scenario.uptime_seconds > 0 && (
                <span className="text-xs text-slate-500">
                  Uptime: {formatUptime(scenario.uptime_seconds)}
                </span>
              )}
            </div>
          </div>
        </div>

        <button
          onClick={onRestart}
          disabled={isRestarting}
          className={cn(
            "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
            "border border-white/10 hover:bg-white/5 transition-colors",
            isRestarting && "opacity-50 cursor-not-allowed"
          )}
        >
          {isRestarting ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <RefreshCw className="h-3 w-3" />
          )}
          Restart
        </button>
      </div>

      {/* Resource usage */}
      <div className="flex items-center gap-4 mt-3 text-xs text-slate-400">
        <div className="flex items-center gap-1.5">
          <Cpu className="h-3 w-3" />
          <span>CPU: {scenario.resources.cpu_percent.toFixed(1)}%</span>
        </div>
        <div className="flex items-center gap-1.5">
          <HardDrive className="h-3 w-3" />
          <span>RAM: {scenario.resources.memory_mb}MB ({scenario.resources.memory_percent.toFixed(1)}%)</span>
        </div>
      </div>

      {/* Ports */}
      {scenario.ports && scenario.ports.length > 0 && (
        <div className="flex items-center gap-2 mt-2 text-xs text-slate-400">
          <span>Ports:</span>
          {scenario.ports.map((port) => (
            <span
              key={port.port}
              className={cn(
                "px-1.5 py-0.5 rounded",
                port.responding
                  ? "bg-emerald-500/20 text-emerald-400"
                  : "bg-slate-500/20 text-slate-400"
              )}
            >
              {port.port}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

interface ResourceCardProps {
  resource: ResourceProcess;
  onRestart: () => void;
  isRestarting: boolean;
}

function ResourceCard({ resource, onRestart, isRestarting }: ResourceCardProps) {
  const isRunning = resource.status === "running";

  return (
    <div
      className={cn(
        "border rounded-lg p-4",
        isRunning
          ? "border-purple-500/30 bg-purple-500/5"
          : "border-red-500/30 bg-red-500/5"
      )}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div
            className={cn(
              "p-2 rounded-lg",
              isRunning ? "bg-purple-500/20" : "bg-red-500/20"
            )}
          >
            <Database
              className={cn(
                "h-5 w-5",
                isRunning ? "text-purple-400" : "text-red-400"
              )}
            />
          </div>
          <div>
            <h4 className="font-medium text-white capitalize">{resource.id}</h4>
            <div className="flex items-center gap-2 mt-1">
              <StatusBadge status={resource.status} />
              <span className="text-xs text-slate-500">PID: {resource.pid}</span>
              {resource.port && (
                <span className="text-xs text-slate-500">Port: {resource.port}</span>
              )}
              {resource.uptime_seconds > 0 && (
                <span className="text-xs text-slate-500">
                  Uptime: {formatUptime(resource.uptime_seconds)}
                </span>
              )}
            </div>
          </div>
        </div>

        <button
          onClick={onRestart}
          disabled={isRestarting}
          className={cn(
            "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
            "border border-white/10 hover:bg-white/5 transition-colors",
            isRestarting && "opacity-50 cursor-not-allowed"
          )}
        >
          {isRestarting ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <RefreshCw className="h-3 w-3" />
          )}
          Restart
        </button>
      </div>
    </div>
  );
}

interface UnexpectedCardProps {
  process: UnexpectedProcess;
  onKill: () => void;
  isKilling: boolean;
}

function UnexpectedCard({ process, onKill, isKilling }: UnexpectedCardProps) {
  return (
    <div className="border border-amber-500/30 bg-amber-500/5 rounded-lg p-4">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-amber-500/20">
            <AlertTriangle className="h-5 w-5 text-amber-400" />
          </div>
          <div>
            <h4 className="font-medium text-white font-mono text-sm truncate max-w-md">
              {process.command}
            </h4>
            <div className="flex items-center gap-2 mt-1">
              <span className="text-xs text-slate-500">PID: {process.pid}</span>
              <span className="text-xs text-slate-500">User: {process.user}</span>
              {process.port && (
                <span className="text-xs text-amber-400">Port: {process.port}</span>
              )}
            </div>
          </div>
        </div>

        <button
          onClick={onKill}
          disabled={isKilling}
          className={cn(
            "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
            "border border-red-500/30 text-red-400 hover:bg-red-500/10 transition-colors",
            isKilling && "opacity-50 cursor-not-allowed"
          )}
        >
          {isKilling ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <XCircle className="h-3 w-3" />
          )}
          Kill
        </button>
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const isRunning = status === "running";
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium",
        isRunning
          ? "bg-emerald-500/20 text-emerald-400"
          : "bg-red-500/20 text-red-400"
      )}
    >
      {isRunning ? (
        <CheckCircle2 className="h-3 w-3" />
      ) : (
        <XCircle className="h-3 w-3" />
      )}
      {status}
    </span>
  );
}
