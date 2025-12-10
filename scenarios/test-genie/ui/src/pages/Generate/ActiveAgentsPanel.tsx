import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import {
  Bot,
  Square,
  Clock,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  RefreshCw,
  ChevronDown,
  ChevronUp,
  Loader2,
  Lock,
  Trash2,
  StopCircle,
  Wifi,
  WifiOff,
  Search,
  Filter,
  X
} from "lucide-react";
import { Button } from "../../components/ui/button";
import {
  fetchActiveAgents,
  stopAgent,
  stopAllAgents,
  cleanupAgents,
  parseAgentStructuredOutput,
  hasStructuredOutput,
  type ActiveAgent,
  type ScopeLock,
  type AgentStructuredOutput
} from "../../lib/api";
import { cn } from "../../lib/utils";
import { useAgentUpdates } from "../../hooks/useAgentUpdates";
import { useWebSocket } from "../../contexts/WebSocketContext";

const STATUS_ICONS: Record<ActiveAgent["status"], React.ReactNode> = {
  pending: <Clock className="h-4 w-4 text-slate-400" />,
  running: <Loader2 className="h-4 w-4 text-cyan-400 animate-spin" />,
  completed: <CheckCircle2 className="h-4 w-4 text-emerald-400" />,
  failed: <XCircle className="h-4 w-4 text-rose-400" />,
  timeout: <AlertTriangle className="h-4 w-4 text-amber-400" />,
  stopped: <Square className="h-4 w-4 text-slate-400" />
};

const STATUS_COLORS: Record<ActiveAgent["status"], string> = {
  pending: "border-slate-400/50 bg-slate-400/10 text-slate-200",
  running: "border-cyan-400/50 bg-cyan-400/10 text-cyan-100",
  completed: "border-emerald-400/50 bg-emerald-400/10 text-emerald-100",
  failed: "border-rose-400/50 bg-rose-400/10 text-rose-100",
  timeout: "border-amber-400/50 bg-amber-400/10 text-amber-100",
  stopped: "border-slate-400/50 bg-slate-400/10 text-slate-200"
};

/**
 * Component to display structured agent output in a more readable format.
 * Falls back to raw output if no structured JSON is found.
 */
function StructuredOutputDisplay({ output }: { output: string }) {
  const [showRaw, setShowRaw] = useState(false);
  const parsed = parseAgentStructuredOutput(output);

  // If no structured output, show raw
  if (!parsed) {
    return (
      <div className="rounded-lg border border-white/10 bg-white/5 p-2">
        <span className="text-xs text-slate-400">Output:</span>
        <pre className="mt-1 text-xs text-slate-300 whitespace-pre-wrap max-h-48 overflow-y-auto">
          {output.slice(0, 2000)}
          {output.length > 2000 && "..."}
        </pre>
      </div>
    );
  }

  // Structured output display
  const statusColors = {
    success: "text-emerald-400 bg-emerald-400/10 border-emerald-400/30",
    partial: "text-amber-400 bg-amber-400/10 border-amber-400/30",
    failed: "text-rose-400 bg-rose-400/10 border-rose-400/30",
  };

  return (
    <div className="space-y-3">
      {/* Status and Summary */}
      <div className={cn("rounded-lg border p-3", statusColors[parsed.status])}>
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold uppercase tracking-wider">
            {parsed.status === "success" ? "✓ Success" : parsed.status === "partial" ? "⚠ Partial" : "✗ Failed"}
          </span>
          <button
            onClick={() => setShowRaw(!showRaw)}
            className="text-[10px] text-slate-400 hover:text-white underline"
          >
            {showRaw ? "Show parsed" : "Show raw"}
          </button>
        </div>
        <p className="mt-2 text-sm">{parsed.summary}</p>
      </div>

      {showRaw ? (
        <pre className="rounded-lg border border-white/10 bg-white/5 p-2 text-xs text-slate-300 whitespace-pre-wrap max-h-64 overflow-y-auto">
          {output.slice(0, 4000)}
          {output.length > 4000 && "..."}
        </pre>
      ) : (
        <>
          {/* Files Changed */}
          {parsed.filesChanged.length > 0 && (
            <div className="rounded-lg border border-white/10 bg-white/5 p-2">
              <span className="text-xs text-slate-400">Files Changed ({parsed.filesChanged.length})</span>
              <div className="mt-2 space-y-1 max-h-32 overflow-y-auto">
                {parsed.filesChanged.map((file, idx) => (
                  <div key={idx} className="flex items-start gap-2 text-xs">
                    <span className={cn(
                      "shrink-0 px-1.5 py-0.5 rounded text-[10px] font-medium",
                      file.action === "created" ? "bg-emerald-400/20 text-emerald-300" :
                      file.action === "modified" ? "bg-cyan-400/20 text-cyan-300" :
                      "bg-rose-400/20 text-rose-300"
                    )}>
                      {file.action}
                    </span>
                    <span className="text-slate-300 font-mono truncate">{file.path}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Tests Added */}
          {parsed.testsAdded.count > 0 && (
            <div className="rounded-lg border border-white/10 bg-white/5 p-2">
              <span className="text-xs text-slate-400">Tests Added</span>
              <div className="mt-2 flex items-center gap-3">
                <span className="text-2xl font-bold text-emerald-400">{parsed.testsAdded.count}</span>
                <div className="flex flex-wrap gap-1">
                  {Object.entries(parsed.testsAdded.byPhase).map(([phase, count]) => (
                    <span key={phase} className="rounded-full border border-white/10 bg-white/5 px-2 py-0.5 text-[10px] text-slate-300">
                      {phase}: {count}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Coverage Impact */}
          {parsed.coverageImpact && (
            <div className="rounded-lg border border-white/10 bg-white/5 p-2">
              <span className="text-xs text-slate-400">Coverage Impact</span>
              <div className="mt-2 flex items-center gap-4 text-sm">
                <span className="text-slate-400">{parsed.coverageImpact.before.toFixed(1)}%</span>
                <span className="text-slate-500">→</span>
                <span className="text-white font-medium">{parsed.coverageImpact.after.toFixed(1)}%</span>
                <span className={cn(
                  "px-2 py-0.5 rounded text-xs font-medium",
                  parsed.coverageImpact.delta >= 0 ? "bg-emerald-400/20 text-emerald-300" : "bg-rose-400/20 text-rose-300"
                )}>
                  {parsed.coverageImpact.delta >= 0 ? "+" : ""}{parsed.coverageImpact.delta.toFixed(1)}%
                </span>
              </div>
            </div>
          )}

          {/* Blockers */}
          {parsed.blockers.length > 0 && (
            <div className="rounded-lg border border-rose-400/30 bg-rose-400/5 p-2">
              <span className="text-xs text-rose-400">Blockers ({parsed.blockers.length})</span>
              <div className="mt-2 space-y-2">
                {parsed.blockers.map((blocker, idx) => (
                  <div key={idx} className="text-xs">
                    <span className="text-rose-300">{blocker.description}</span>
                    {blocker.suggestedResolution && (
                      <p className="mt-0.5 text-slate-400 italic">→ {blocker.suggestedResolution}</p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Next Steps */}
          {parsed.nextSteps.length > 0 && (
            <div className="rounded-lg border border-white/10 bg-white/5 p-2">
              <span className="text-xs text-slate-400">Next Steps</span>
              <ul className="mt-2 text-xs text-slate-300 space-y-1">
                {parsed.nextSteps.map((step, idx) => (
                  <li key={idx} className="flex items-start gap-2">
                    <span className="text-slate-500">•</span>
                    <span>{step}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </>
      )}
    </div>
  );
}

interface ActiveAgentsPanelProps {
  scenario?: string;
  scope?: string[];
  onConflictDetected?: (conflictingAgentIds: string[]) => void;
  /** Callback when an agent's status changes to a terminal state */
  onAgentStatusChange?: (agentId: string, status: "completed" | "failed" | "timeout" | "stopped") => void;
}

// Status filter options
const STATUS_OPTIONS = [
  { value: "all", label: "All" },
  { value: "active", label: "Active" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
  { value: "stopped", label: "Stopped" },
] as const;

type StatusFilter = typeof STATUS_OPTIONS[number]["value"];

export function ActiveAgentsPanel({ scenario, scope, onConflictDetected, onAgentStatusChange }: ActiveAgentsPanelProps) {
  const [showAll, setShowAll] = useState(false);
  const [expandedAgents, setExpandedAgents] = useState<Set<string>>(new Set());
  const [streamingOutput, setStreamingOutput] = useState<Record<string, string>>({});
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("active");
  const [scenarioFilter, setScenarioFilter] = useState<string>("");
  const [searchQuery, setSearchQuery] = useState("");
  const queryClient = useQueryClient();
  const { isConnected } = useWebSocket();

  // Use WebSocket for real-time updates (invalidates queries automatically)
  const handleAgentUpdate = useCallback((agent: { id: string; status: string }) => {
    // Notify parent of terminal status changes
    if (onAgentStatusChange && agent.status) {
      const terminalStatuses = ["completed", "failed", "timeout", "stopped"];
      if (terminalStatuses.includes(agent.status)) {
        onAgentStatusChange(agent.id, agent.status as "completed" | "failed" | "timeout" | "stopped");
        // Clear streaming output when agent completes (final output will be in agent.output)
        setStreamingOutput((prev) => {
          const next = { ...prev };
          delete next[agent.id];
          return next;
        });
      }
    }
  }, [onAgentStatusChange]);

  // Handle streaming output from WebSocket
  const handleAgentOutput = useCallback((data: { agentId: string; output: string }) => {
    setStreamingOutput((prev) => ({
      ...prev,
      [data.agentId]: (prev[data.agentId] || "") + data.output,
    }));
  }, []);

  useAgentUpdates({
    onAgentUpdate: handleAgentUpdate,
    onAgentOutput: handleAgentOutput,
  });

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["active-agents", showAll],
    queryFn: () => fetchActiveAgents(showAll),
    // Use longer poll interval when WebSocket is connected (fallback only)
    refetchInterval: isConnected ? 30000 : 5000,
    staleTime: 2000
  });

  const stopMutation = useMutation({
    mutationFn: stopAgent,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["active-agents"] });
    }
  });

  const stopAllMutation = useMutation({
    mutationFn: stopAllAgents,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["active-agents"] });
    }
  });

  const cleanupMutation = useMutation({
    mutationFn: () => cleanupAgents(60),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["active-agents"] });
    }
  });

  const allAgents = data?.items ?? [];
  const locks = data?.activeLocks ?? [];
  const activeCount = allAgents.filter(
    (a) => a.status === "running" || a.status === "pending"
  ).length;

  // Get unique scenarios for filter dropdown
  const uniqueScenarios = Array.from(new Set(allAgents.map(a => a.scenario))).sort();

  // Apply filters
  const agents = allAgents.filter((agent) => {
    // Status filter
    if (statusFilter === "active" && agent.status !== "running" && agent.status !== "pending") {
      return false;
    }
    if (statusFilter === "completed" && agent.status !== "completed") {
      return false;
    }
    if (statusFilter === "failed" && agent.status !== "failed" && agent.status !== "timeout") {
      return false;
    }
    if (statusFilter === "stopped" && agent.status !== "stopped") {
      return false;
    }

    // Scenario filter
    if (scenarioFilter && agent.scenario !== scenarioFilter) {
      return false;
    }

    // Search query (searches ID, scenario, scope paths)
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      const matchesId = agent.id.toLowerCase().includes(query);
      const matchesScenario = agent.scenario.toLowerCase().includes(query);
      const matchesScope = agent.scope?.some(s => s.toLowerCase().includes(query));
      const matchesModel = agent.model?.toLowerCase().includes(query);
      if (!matchesId && !matchesScenario && !matchesScope && !matchesModel) {
        return false;
      }
    }

    return true;
  });

  // Check for conflicts with the current scenario/scope
  const conflictingAgents = scenario
    ? agents.filter((a) => {
        if (a.status !== "running" && a.status !== "pending") return false;
        if (a.scenario !== scenario) return false;
        if (!scope || scope.length === 0 || !a.scope || a.scope.length === 0) return false;
        // Check for path overlap
        return scope.some((s) =>
          a.scope.some(
            (as) =>
              s === as ||
              s.startsWith(as + "/") ||
              as.startsWith(s + "/")
          )
        );
      })
    : [];

  // Notify parent of conflicts
  if (onConflictDetected && conflictingAgents.length > 0) {
    onConflictDetected(conflictingAgents.map((a) => a.id));
  }

  const toggleExpanded = (id: string) => {
    setExpandedAgents((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

  const formatDuration = (startStr: string, endStr?: string) => {
    const start = new Date(startStr).getTime();
    const end = endStr ? new Date(endStr).getTime() : Date.now();
    const seconds = Math.floor((end - start) / 1000);
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}m ${remainingSeconds}s`;
  };

  return (
    <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <Bot className="h-5 w-5 text-cyan-400" />
          <h3 className="text-lg font-semibold text-white">Active Agents</h3>
          {activeCount > 0 && (
            <span className="rounded-full border border-cyan-400/50 bg-cyan-400/10 px-2 py-0.5 text-xs text-cyan-100">
              {activeCount} running
            </span>
          )}
          {/* WebSocket connection indicator */}
          <span
            className={cn(
              "flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px]",
              isConnected
                ? "border border-emerald-400/50 bg-emerald-400/10 text-emerald-200"
                : "border border-amber-400/50 bg-amber-400/10 text-amber-200"
            )}
            title={isConnected ? "Real-time updates active" : "Using polling fallback"}
          >
            {isConnected ? (
              <>
                <Wifi className="h-3 w-3" />
                Live
              </>
            ) : (
              <>
                <WifiOff className="h-3 w-3" />
                Polling
              </>
            )}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {/* Stop All button - only show when there are active agents */}
          {activeCount > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => stopAllMutation.mutate()}
              disabled={stopAllMutation.isPending}
              className="text-rose-400 border-rose-400/50 hover:bg-rose-400/10"
            >
              <StopCircle className={cn("h-4 w-4 mr-1", stopAllMutation.isPending && "animate-pulse")} />
              Stop All ({activeCount})
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => refetch()}
            disabled={isLoading}
          >
            <RefreshCw className={cn("h-4 w-4", isLoading && "animate-spin")} />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setShowAll(!showAll);
              // Reset status filter when toggling showAll
              if (!showAll) {
                setStatusFilter("all");
              } else {
                setStatusFilter("active");
              }
            }}
          >
            {showAll ? "Active only" : "Show all"}
          </Button>
          {showAll && allAgents.some((a) => a.status !== "running" && a.status !== "pending") && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => cleanupMutation.mutate()}
              disabled={cleanupMutation.isPending}
            >
              <Trash2 className="h-4 w-4 mr-1" />
              Cleanup
            </Button>
          )}
        </div>
      </div>

      {/* Filter controls */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        {/* Search input */}
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
          <input
            type="text"
            placeholder="Search by ID, scenario, scope, or model..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-lg border border-white/10 bg-white/5 pl-10 pr-8 py-2 text-sm text-white placeholder-slate-400 focus:border-cyan-400/50 focus:outline-none focus:ring-1 focus:ring-cyan-400/50"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery("")}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>

        {/* Status filter */}
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-slate-400" />
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
            className="rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-white focus:border-cyan-400/50 focus:outline-none"
          >
            {STATUS_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value} className="bg-slate-800">
                {opt.label}
              </option>
            ))}
          </select>
        </div>

        {/* Scenario filter */}
        {uniqueScenarios.length > 1 && (
          <select
            value={scenarioFilter}
            onChange={(e) => setScenarioFilter(e.target.value)}
            className="rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-white focus:border-cyan-400/50 focus:outline-none"
          >
            <option value="" className="bg-slate-800">All scenarios</option>
            {uniqueScenarios.map((s) => (
              <option key={s} value={s} className="bg-slate-800">
                {s}
              </option>
            ))}
          </select>
        )}

        {/* Clear filters button */}
        {(searchQuery || statusFilter !== "active" || scenarioFilter) && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setSearchQuery("");
              setStatusFilter("active");
              setScenarioFilter("");
            }}
            className="text-slate-400 hover:text-white"
          >
            <X className="h-4 w-4 mr-1" />
            Clear filters
          </Button>
        )}

        {/* Results count */}
        <span className="text-xs text-slate-400">
          {agents.length} of {allAgents.length} agents
        </span>
      </div>

      {error && (
        <div className="rounded-lg border border-rose-400/50 bg-rose-400/10 p-3 text-sm text-rose-100">
          Failed to load agents: {error instanceof Error ? error.message : "Unknown error"}
        </div>
      )}

      {conflictingAgents.length > 0 && (
        <div className="mb-4 rounded-lg border border-amber-400/50 bg-amber-400/10 p-3">
          <div className="flex items-center gap-2 text-amber-100">
            <AlertTriangle className="h-4 w-4" />
            <span className="text-sm font-medium">Scope Conflict Warning</span>
          </div>
          <p className="mt-1 text-xs text-amber-200/80">
            {conflictingAgents.length} agent{conflictingAgents.length > 1 ? "s are" : " is"} already
            working on overlapping paths. Spawning new agents for this scope may cause conflicts.
          </p>
        </div>
      )}

      {agents.length === 0 && !isLoading && (
        <div className="text-center py-8 text-slate-400 text-sm">
          No {showAll ? "" : "active "}agents found
        </div>
      )}

      <div className="space-y-2">
        {agents.map((agent) => {
          const isExpanded = expandedAgents.has(agent.id);
          const isActive = agent.status === "running" || agent.status === "pending";
          const isConflicting = conflictingAgents.some((c) => c.id === agent.id);

          return (
            <div
              key={agent.id}
              className={cn(
                "rounded-xl border p-4 transition-colors",
                isConflicting
                  ? "border-amber-400/50 bg-amber-400/5"
                  : "border-white/10 bg-white/[0.02]"
              )}
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  {STATUS_ICONS[agent.status]}
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-white">
                        {agent.scenario}
                      </span>
                      <span className="text-xs text-slate-400">#{agent.id}</span>
                      <span
                        className={cn(
                          "rounded-full border px-2 py-0.5 text-[10px]",
                          STATUS_COLORS[agent.status]
                        )}
                      >
                        {agent.status}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 mt-1 text-xs text-slate-400">
                      <span>{formatTime(agent.startedAt)}</span>
                      <span>|</span>
                      <span>{formatDuration(agent.startedAt, agent.completedAt)}</span>
                      <span>|</span>
                      <span className="truncate max-w-[200px]">{agent.model}</span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {isActive && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => stopMutation.mutate(agent.id)}
                      disabled={stopMutation.isPending}
                      className="text-rose-400 border-rose-400/50 hover:bg-rose-400/10"
                    >
                      <Square className="h-3 w-3 mr-1" />
                      Stop
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => toggleExpanded(agent.id)}
                  >
                    {isExpanded ? (
                      <ChevronUp className="h-4 w-4" />
                    ) : (
                      <ChevronDown className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              </div>

              {agent.scope && agent.scope.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1">
                  {agent.scope.map((path) => (
                    <span
                      key={path}
                      className="rounded-full border border-white/10 bg-white/5 px-2 py-0.5 text-[10px] text-slate-300"
                    >
                      {path}
                    </span>
                  ))}
                </div>
              )}

              {isExpanded && (
                <div className="mt-4 space-y-3">
                  {agent.phases && agent.phases.length > 0 && (
                    <div>
                      <span className="text-xs text-slate-400">Phases:</span>
                      <div className="flex gap-1 mt-1">
                        {agent.phases.map((phase) => (
                          <span
                            key={phase}
                            className="rounded-full border border-cyan-400/30 bg-cyan-400/10 px-2 py-0.5 text-[10px] text-cyan-200"
                          >
                            {phase}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  {agent.sessionId && (
                    <div>
                      <span className="text-xs text-slate-400">Session ID:</span>
                      <code className="ml-2 text-xs text-slate-300 bg-white/5 px-1 rounded">
                        {agent.sessionId}
                      </code>
                    </div>
                  )}

                  {agent.error && (
                    <div className="rounded-lg border border-rose-400/30 bg-rose-400/5 p-2">
                      <span className="text-xs text-rose-400">Error:</span>
                      <pre className="mt-1 text-xs text-rose-200/80 whitespace-pre-wrap max-h-32 overflow-y-auto">
                        {agent.error}
                      </pre>
                    </div>
                  )}

                  {/* Show streaming output for running agents */}
                  {isActive && streamingOutput[agent.id] && (
                    <div className="rounded-lg border border-cyan-400/30 bg-cyan-400/5 p-2">
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-cyan-400">Live Output:</span>
                        <Loader2 className="h-3 w-3 text-cyan-400 animate-spin" />
                      </div>
                      <pre className="mt-1 text-xs text-cyan-100 whitespace-pre-wrap max-h-48 overflow-y-auto font-mono">
                        {streamingOutput[agent.id].slice(-3000)}
                        {streamingOutput[agent.id].length > 3000 && "\n... (truncated, showing last 3000 chars)"}
                      </pre>
                    </div>
                  )}

                  {/* Show final output for completed agents */}
                  {agent.output && !agent.error && !isActive && (
                    <StructuredOutputDisplay output={agent.output} />
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>

      {locks.length > 0 && (
        <div className="mt-6 pt-4 border-t border-white/10">
          <div className="flex items-center gap-2 mb-3">
            <Lock className="h-4 w-4 text-slate-400" />
            <span className="text-sm font-medium text-slate-300">Active Scope Locks</span>
          </div>
          <div className="space-y-2">
            {locks.map((lock) => (
              <div
                key={`${lock.scenario}-${lock.agentId}`}
                className="rounded-lg border border-white/10 bg-white/[0.02] p-3"
              >
                <div className="flex items-center justify-between">
                  <span className="text-sm text-white">{lock.scenario}</span>
                  <span className="text-xs text-slate-400">Agent #{lock.agentId}</span>
                </div>
                <div className="mt-1 flex flex-wrap gap-1">
                  {lock.paths.map((path) => (
                    <span
                      key={path}
                      className="rounded-full border border-amber-400/30 bg-amber-400/10 px-2 py-0.5 text-[10px] text-amber-200"
                    >
                      {path}
                    </span>
                  ))}
                </div>
                <div className="mt-2 text-[10px] text-slate-500">
                  Expires: {new Date(lock.expiresAt).toLocaleTimeString()}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
