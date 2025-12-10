import { AlertTriangle, Clock, StopCircle, Cpu, Layers } from "lucide-react";
import { Button } from "../../components/ui/button";
import { ConflictDetail, stopAgent } from "../../lib/api";
import { cn } from "../../lib/utils";
import { useState } from "react";

interface ConflictPreviewProps {
  conflicts: ConflictDetail[];
  onStopAgent?: (agentId: string) => void;
  onRefresh?: () => void;
}

function formatTimeRemaining(expiresAt: string): string {
  const expires = new Date(expiresAt);
  const now = new Date();
  const diffMs = expires.getTime() - now.getTime();

  if (diffMs <= 0) return "expired";

  const minutes = Math.floor(diffMs / (1000 * 60));
  const seconds = Math.floor((diffMs % (1000 * 60)) / 1000);

  if (minutes > 60) {
    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;
    return `${hours}h ${remainingMinutes}m`;
  }

  return `${minutes}m ${seconds}s`;
}

function formatDuration(startedAt: string): string {
  const started = new Date(startedAt);
  const now = new Date();
  const diffMs = now.getTime() - started.getTime();

  const minutes = Math.floor(diffMs / (1000 * 60));
  const seconds = Math.floor((diffMs % (1000 * 60)) / 1000);

  if (minutes > 60) {
    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;
    return `${hours}h ${remainingMinutes}m`;
  }

  if (minutes === 0) {
    return `${seconds}s`;
  }

  return `${minutes}m ${seconds}s`;
}

function formatModelName(model: string): string {
  // Extract readable model name from full ID (e.g., "anthropic/claude-3-opus" -> "claude-3-opus")
  const parts = model.split("/");
  return parts[parts.length - 1] || model;
}

export function ConflictPreview({ conflicts, onStopAgent, onRefresh }: ConflictPreviewProps) {
  const [stoppingAgents, setStoppingAgents] = useState<Set<string>>(new Set());

  if (conflicts.length === 0) {
    return null;
  }

  // Group conflicts by agent with enhanced details
  const byAgent = conflicts.reduce<Record<string, {
    agentId: string;
    paths: string[];
    startedAt: string;
    expiresAt: string;
    model?: string;
    phases?: string[];
    status?: string;
    runningSeconds?: number;
  }>>((acc, conflict) => {
    const { agentId, startedAt, expiresAt, model, phases, status, runningSeconds } = conflict.lockedBy;
    if (!acc[agentId]) {
      acc[agentId] = { agentId, paths: [], startedAt, expiresAt, model, phases, status, runningSeconds };
    }
    acc[agentId].paths.push(conflict.path);
    return acc;
  }, {});

  const agentConflicts = Object.values(byAgent);

  const handleStopAgent = async (agentId: string) => {
    setStoppingAgents(prev => new Set(prev).add(agentId));
    try {
      await stopAgent(agentId);
      onStopAgent?.(agentId);
      // Wait a moment then refresh
      setTimeout(() => onRefresh?.(), 500);
    } catch (err) {
      console.error("Failed to stop agent:", err);
    } finally {
      setStoppingAgents(prev => {
        const next = new Set(prev);
        next.delete(agentId);
        return next;
      });
    }
  };

  return (
    <div className="rounded-2xl border border-amber-400/50 bg-amber-950/20 p-5">
      <div className="flex items-start gap-3">
        <AlertTriangle className="h-6 w-6 text-amber-400 flex-shrink-0 mt-0.5" />
        <div className="flex-1 min-w-0">
          <h3 className="text-lg font-semibold text-amber-100">
            Scope Conflicts Detected - Spawning Blocked
          </h3>
          <p className="mt-1 text-sm text-amber-200/80">
            The following paths are currently locked by other agents.
            Wait for them to complete or stop them to proceed.
          </p>

          <div className="mt-4 space-y-3">
            {agentConflicts.map(({ agentId, paths, startedAt, expiresAt, model, phases, status }) => (
              <div
                key={agentId}
                className={cn(
                  "rounded-xl border border-amber-400/30 bg-amber-950/30 p-4",
                  "flex flex-col gap-3"
                )}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2 text-sm flex-wrap">
                      <span className="font-mono text-amber-100">
                        Agent {agentId}
                      </span>
                      <span className="text-amber-400/60">•</span>
                      <span className={cn(
                        "text-amber-200/70",
                        status === "running" && "text-green-300/80"
                      )}>
                        {status === "running" ? "running" : status || "running"} {formatDuration(startedAt)}
                      </span>
                    </div>

                    {/* Model and phases info */}
                    <div className="mt-2 flex items-center gap-3 flex-wrap text-xs">
                      {model && (
                        <span className="flex items-center gap-1 text-amber-200/60">
                          <Cpu className="h-3 w-3" />
                          <span className="font-mono">{formatModelName(model)}</span>
                        </span>
                      )}
                      {phases && phases.length > 0 && (
                        <span className="flex items-center gap-1 text-amber-200/60">
                          <Layers className="h-3 w-3" />
                          <span>{phases.join(", ")}</span>
                        </span>
                      )}
                    </div>

                    {/* Locked paths */}
                    <div className="mt-2 flex flex-wrap gap-2">
                      {paths.slice(0, 5).map(path => (
                        <span
                          key={path}
                          className="rounded-full border border-amber-400/30 bg-amber-400/10 px-2 py-0.5 text-xs text-amber-100 font-mono"
                        >
                          {path}
                        </span>
                      ))}
                      {paths.length > 5 && (
                        <span className="text-xs text-amber-400/60">
                          +{paths.length - 5} more
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-3 flex-shrink-0">
                    <div className="flex items-center gap-1 text-xs text-amber-200/70">
                      <Clock className="h-3 w-3" />
                      <span>Expires in {formatTimeRemaining(expiresAt)}</span>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleStopAgent(agentId)}
                      disabled={stoppingAgents.has(agentId)}
                      className="border-amber-400/40 text-amber-100 hover:bg-amber-400/10"
                    >
                      {stoppingAgents.has(agentId) ? (
                        <span className="flex items-center gap-1">
                          <span className="animate-pulse">Stopping...</span>
                        </span>
                      ) : (
                        <span className="flex items-center gap-1">
                          <StopCircle className="h-4 w-4" />
                          Stop
                        </span>
                      )}
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div className="mt-4 flex items-center gap-4">
            <p className="text-xs text-amber-200/60">
              {conflicts.length} path{conflicts.length !== 1 ? "s" : ""} locked by{" "}
              {agentConflicts.length} agent{agentConflicts.length !== 1 ? "s" : ""}
            </p>
            {onRefresh && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onRefresh}
                className="text-xs text-amber-200/70 hover:text-amber-100"
              >
                Refresh status
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
