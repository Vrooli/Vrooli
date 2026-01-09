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
  Play,
  Square,
  Settings,
} from "lucide-react";
import { useRestartProcess, useKillProcess, useProcessControl, formatUptime } from "../../../hooks/useLiveState";
import type { ProcessState, ScenarioProcess, ResourceProcess, UnexpectedProcess, ExpectedProcess } from "../../../lib/api";
import { cn } from "../../../lib/utils";

interface ProcessCardsProps {
  deploymentId: string;
  processes: ProcessState;
  expected?: ExpectedProcess[];
}

export function ProcessCards({ deploymentId, processes, expected }: ProcessCardsProps) {
  const restartMutation = useRestartProcess(deploymentId);
  const killMutation = useKillProcess(deploymentId);
  const controlMutation = useProcessControl(deploymentId);

  // Find expected processes that are not running
  const missingProcesses = expected?.filter((exp) => exp.state !== "running") || [];

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
                deploymentId={deploymentId}
                scenario={scenario}
                controlMutation={controlMutation}
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
                deploymentId={deploymentId}
                resource={resource}
                controlMutation={controlMutation}
              />
            ))}
          </div>
        )}
      </div>

      {/* Missing Dependencies */}
      {missingProcesses.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-amber-400 mb-3 flex items-center gap-2">
            <AlertTriangle className="h-4 w-4" />
            Missing Dependencies ({missingProcesses.length})
          </h3>
          <div className="grid gap-3">
            {missingProcesses.map((proc) => (
              <MissingProcessCard
                key={`${proc.type}-${proc.id}`}
                deploymentId={deploymentId}
                process={proc}
                controlMutation={controlMutation}
              />
            ))}
          </div>
        </div>
      )}

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
  deploymentId: string;
  scenario: ScenarioProcess;
  controlMutation: ReturnType<typeof useProcessControl>;
}

function ScenarioCard({ deploymentId, scenario, controlMutation }: ScenarioCardProps) {
  const isRunning = scenario.status === "running";
  const isPending = controlMutation.isPending;

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

        <div className="flex items-center gap-2">
          {isRunning ? (
            <>
              <button
                onClick={() => controlMutation.mutate({ action: "restart", type: "scenario", id: scenario.id })}
                disabled={isPending}
                className={cn(
                  "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
                  "border border-blue-500/30 bg-blue-500/10 text-blue-400 hover:bg-blue-500/20 transition-colors",
                  isPending && "opacity-50 cursor-not-allowed"
                )}
              >
                {isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />}
                Restart
              </button>
              <button
                onClick={() => controlMutation.mutate({ action: "stop", type: "scenario", id: scenario.id })}
                disabled={isPending}
                className={cn(
                  "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
                  "border border-red-500/30 bg-red-500/10 text-red-400 hover:bg-red-500/20 transition-colors",
                  isPending && "opacity-50 cursor-not-allowed"
                )}
              >
                {isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <Square className="h-3 w-3" />}
                Stop
              </button>
            </>
          ) : (
            <button
              onClick={() => controlMutation.mutate({ action: "start", type: "scenario", id: scenario.id })}
              disabled={isPending}
              className={cn(
                "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
                "border border-emerald-500/30 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 transition-colors",
                isPending && "opacity-50 cursor-not-allowed"
              )}
            >
              {isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}
              Start
            </button>
          )}
        </div>
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
  deploymentId: string;
  resource: ResourceProcess;
  controlMutation: ReturnType<typeof useProcessControl>;
}

function ResourceCard({ deploymentId, resource, controlMutation }: ResourceCardProps) {
  const isRunning = resource.status === "running";
  const isPending = controlMutation.isPending;

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

        <div className="flex items-center gap-2">
          {isRunning ? (
            <>
              <button
                onClick={() => controlMutation.mutate({ action: "restart", type: "resource", id: resource.id })}
                disabled={isPending}
                className={cn(
                  "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
                  "border border-blue-500/30 bg-blue-500/10 text-blue-400 hover:bg-blue-500/20 transition-colors",
                  isPending && "opacity-50 cursor-not-allowed"
                )}
              >
                {isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />}
                Restart
              </button>
              <button
                onClick={() => controlMutation.mutate({ action: "stop", type: "resource", id: resource.id })}
                disabled={isPending}
                className={cn(
                  "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
                  "border border-red-500/30 bg-red-500/10 text-red-400 hover:bg-red-500/20 transition-colors",
                  isPending && "opacity-50 cursor-not-allowed"
                )}
              >
                {isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <Square className="h-3 w-3" />}
                Stop
              </button>
            </>
          ) : (
            <button
              onClick={() => controlMutation.mutate({ action: "start", type: "resource", id: resource.id })}
              disabled={isPending}
              className={cn(
                "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
                "border border-emerald-500/30 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 transition-colors",
                isPending && "opacity-50 cursor-not-allowed"
              )}
            >
              {isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}
              Start
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

interface MissingProcessCardProps {
  deploymentId: string;
  process: ExpectedProcess;
  controlMutation: ReturnType<typeof useProcessControl>;
}

function MissingProcessCard({ deploymentId, process, controlMutation }: MissingProcessCardProps) {
  const isPending = controlMutation.isPending;
  const needsSetup = process.state === "needs_setup";
  const isResource = process.type === "resource";

  return (
    <div className="border border-amber-500/30 bg-amber-500/5 rounded-lg p-4">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-amber-500/20">
            {process.type === "scenario" ? (
              <Activity className="h-5 w-5 text-amber-400" />
            ) : (
              <Database className="h-5 w-5 text-amber-400" />
            )}
          </div>
          <div>
            <h4 className="font-medium text-white capitalize">{process.id}</h4>
            <div className="flex items-center gap-2 mt-1">
              <span
                className={cn(
                  "inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium",
                  needsSetup
                    ? "bg-amber-500/20 text-amber-400"
                    : "bg-slate-500/20 text-slate-400"
                )}
              >
                <AlertTriangle className="h-3 w-3" />
                {needsSetup ? "needs setup" : "stopped"}
              </span>
              <span className="text-xs text-slate-500 capitalize">{process.type}</span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {needsSetup && isResource && (
            <button
              onClick={() => controlMutation.mutate({ action: "setup", type: "resource", id: process.id })}
              disabled={isPending}
              className={cn(
                "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
                "border border-amber-500/30 bg-amber-500/10 text-amber-400 hover:bg-amber-500/20 transition-colors",
                isPending && "opacity-50 cursor-not-allowed"
              )}
            >
              {isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <Settings className="h-3 w-3" />}
              Setup
            </button>
          )}
          <button
            onClick={() => controlMutation.mutate({ action: "start", type: process.type, id: process.id })}
            disabled={isPending}
            className={cn(
              "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
              "border border-emerald-500/30 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 transition-colors",
              isPending && "opacity-50 cursor-not-allowed"
            )}
          >
            {isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}
            Start
          </button>
        </div>
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
