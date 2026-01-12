import { useState, useMemo } from "react";
import {
  Search,
  Loader2,
  AlertCircle,
  Bot,
  X,
  Check,
  Wrench,
  Zap,
  FileEdit,
  Shield,
  CheckSquare,
  FileText,
  GitBranch,
  Monitor,
  Package,
  ChevronDown,
} from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import { SelectableCard, CompactSelectableCard, type SelectableCardConfig } from "../ui/selectable-card";
import { useDeploymentInvestigation } from "../../hooks/useInvestigation";
import { createTask } from "../../lib/api";
import type { CreateTaskRequest } from "../../types/investigation";

// Task type configurations
const TASK_TYPES: SelectableCardConfig[] = [
  {
    id: "investigate",
    name: "Investigate",
    description: "Analyze the failure and report findings",
    icon: Search,
  },
  {
    id: "fix",
    name: "Fix",
    description: "Apply fixes through iterative loop",
    icon: Wrench,
  },
];

// Effort level configurations (for Investigate only)
const EFFORT_LEVELS: SelectableCardConfig[] = [
  {
    id: "checks",
    name: "Checks",
    description: "Quick health checks",
    icon: CheckSquare,
  },
  {
    id: "logs",
    name: "Logs",
    description: "Log analysis + diagnostics",
    icon: FileText,
  },
  {
    id: "trace",
    name: "Trace",
    description: "Full SSH tracing",
    icon: GitBranch,
  },
];

// Permission configurations (for Fix only)
const PERMISSIONS: SelectableCardConfig[] = [
  {
    id: "immediate",
    name: "Immediate",
    description: "Restart services, clear disk",
    icon: Zap,
  },
  {
    id: "permanent",
    name: "Permanent",
    description: "Code/config changes",
    icon: FileEdit,
  },
  {
    id: "prevention",
    name: "Prevention",
    description: "Add monitoring/alerts",
    icon: Shield,
  },
];

// Focus configurations (for both)
const FOCUS_AREAS: SelectableCardConfig[] = [
  {
    id: "harness",
    name: "Harness",
    description: "Deployment infrastructure",
    icon: Monitor,
  },
  {
    id: "subject",
    name: "Subject",
    description: "Target scenario",
    icon: Package,
  },
];

// Context items that can be included in investigations
const CONTEXT_OPTIONS = [
  {
    key: "error-info",
    label: "Error Information",
    description: "Failed step and error message",
    defaultChecked: true,
    recommended: true,
  },
  {
    key: "deployment-manifest",
    label: "Deployment Manifest",
    description: "Full dependency and target configuration",
    defaultChecked: true,
  },
  {
    key: "vps-connection",
    label: "VPS Connection Details",
    description: "SSH command, host, credentials",
    defaultChecked: true,
  },
  {
    key: "deployment-history",
    label: "Deployment History",
    description: "Timeline of deployment events",
    defaultChecked: true,
  },
  {
    key: "preflight-results",
    label: "Preflight Results",
    description: "VPS capability check results",
    defaultChecked: false,
  },
  {
    key: "setup-results",
    label: "Setup Results",
    description: "Vrooli installation phase output",
    defaultChecked: false,
  },
  {
    key: "deploy-results",
    label: "Deploy Results",
    description: "Deployment execution phase output",
    defaultChecked: false,
  },
  {
    key: "architecture-guide",
    label: "Architecture Guide",
    description: "How Vrooli deployments work",
    defaultChecked: true,
  },
] as const;

type ContextKey = (typeof CONTEXT_OPTIONS)[number]["key"];
type TaskType = "investigate" | "fix";
type EffortLevel = "checks" | "logs" | "trace";
type PermissionType = "immediate" | "permanent" | "prevention";
type FocusType = "harness" | "subject";

interface SpawnAgentButtonProps {
  deploymentId: string;
  disabled?: boolean;
  onTaskStarted?: (taskId: string) => void;
}

export function SpawnAgentButton({
  deploymentId,
  disabled,
  onTaskStarted,
}: SpawnAgentButtonProps) {
  const [showOptions, setShowOptions] = useState(false);
  const [taskType, setTaskType] = useState<TaskType>("investigate");
  const [effortLevel, setEffortLevel] = useState<EffortLevel>("logs");
  const [permissions, setPermissions] = useState<Set<PermissionType>>(
    () => new Set(["immediate"])
  );
  const [focus, setFocus] = useState<Set<FocusType>>(
    () => new Set(["harness", "subject"])
  );
  const [note, setNote] = useState("");
  const [selectedContexts, setSelectedContexts] = useState<Set<ContextKey>>(
    () => new Set(CONTEXT_OPTIONS.filter((o) => o.defaultChecked).map((o) => o.key))
  );
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | null>(null);

  const {
    isAgentAvailable,
    isAgentEnabled,
    isAgentLoading,
    isRunning,
    activeInvestigationId,
    refresh,
  } = useDeploymentInvestigation(deploymentId);

  // Combined triggering state (from hook or local)
  const isTriggering = isCreating;
  const triggerError = createError;

  // Validation
  const isFormValid = useMemo(() => {
    if (focus.size === 0) return false;
    if (taskType === "fix" && permissions.size === 0) return false;
    return true;
  }, [taskType, permissions, focus]);

  const handleClick = () => {
    if (isRunning) {
      if (activeInvestigationId) {
        onTaskStarted?.(activeInvestigationId);
      }
      return;
    }
    setShowOptions(true);
  };

  const handleTrigger = async () => {
    setIsCreating(true);
    setCreateError(null);
    try {
      // Build the task request with all new fields
      const request: CreateTaskRequest = {
        task_type: taskType,
        focus: {
          harness: focus.has("harness"),
          subject: focus.has("subject"),
        },
        note: note.trim() || undefined,
        include_contexts: Array.from(selectedContexts),
      };

      // Add task-type-specific fields
      if (taskType === "investigate") {
        request.effort = effortLevel;
      } else if (taskType === "fix") {
        request.permissions = {
          immediate: permissions.has("immediate"),
          permanent: permissions.has("permanent"),
          prevention: permissions.has("prevention"),
        };
      }

      const response = await createTask(deploymentId, request);
      if (response.task) {
        onTaskStarted?.(response.task.id);
        refresh(); // Refresh the investigations list
        handleReset();
      }
    } catch (err) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setIsCreating(false);
    }
  };

  const handleReset = () => {
    setShowOptions(false);
    setTaskType("investigate");
    setEffortLevel("logs");
    setPermissions(new Set(["immediate"]));
    setFocus(new Set(["harness", "subject"]));
    setNote("");
    setSelectedContexts(
      new Set(CONTEXT_OPTIONS.filter((o) => o.defaultChecked).map((o) => o.key))
    );
    setAdvancedOpen(false);
  };

  const togglePermission = (perm: PermissionType) => {
    setPermissions((prev) => {
      const next = new Set(prev);
      if (next.has(perm)) {
        next.delete(perm);
      } else {
        next.add(perm);
      }
      return next;
    });
  };

  const toggleFocus = (f: FocusType) => {
    setFocus((prev) => {
      const next = new Set(prev);
      if (next.has(f)) {
        next.delete(f);
      } else {
        next.add(f);
      }
      return next;
    });
  };

  const toggleContext = (key: ContextKey) => {
    setSelectedContexts((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  // Agent not configured
  if (!isAgentEnabled) {
    return (
      <Button variant="outline" disabled title="Agent manager integration is not enabled">
        <Bot className="h-4 w-4 mr-1.5 opacity-50" />
        <span className="opacity-50">Spawn Agent</span>
      </Button>
    );
  }

  return (
    <>
      {/* Main button */}
      <Button
        variant="outline"
        onClick={handleClick}
        disabled={disabled || isAgentLoading || isTriggering}
      >
        {isAgentLoading ? (
          <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
        ) : isRunning ? (
          <Loader2 className="h-4 w-4 mr-1.5 animate-spin text-blue-400" />
        ) : !isAgentAvailable ? (
          <AlertCircle className="h-4 w-4 mr-1.5 text-amber-400" />
        ) : (
          <Bot className="h-4 w-4 mr-1.5" />
        )}
        {isRunning ? "View Task" : "Spawn Agent"}
      </Button>

      {/* Options modal */}
      {showOptions && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-slate-900 border border-slate-700 rounded-lg shadow-xl w-[95vw] max-w-2xl mx-4">
            {/* Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
              <div className="flex items-center gap-2">
                <Bot className="h-5 w-5 text-blue-400" />
                <h2 className="text-lg font-semibold text-white">Spawn Agent</h2>
              </div>
              <Button variant="ghost" size="sm" onClick={handleReset}>
                <X className="h-4 w-4" />
              </Button>
            </div>

            {/* Content */}
            <div className="px-6 py-4 space-y-6 max-h-[70vh] overflow-y-auto">
              {!isAgentAvailable && (
                <Alert variant="warning" title="Agent Manager Unavailable">
                  The agent-manager service is not running. Start it to enable tasks.
                </Alert>
              )}

              {triggerError && (
                <Alert variant="error" title="Task Failed">
                  {triggerError instanceof Error ? triggerError.message : String(triggerError)}
                </Alert>
              )}

              {/* Task Type Selection */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-3">
                  What should the agent do?
                </label>
                <div className="grid grid-cols-2 gap-3">
                  {TASK_TYPES.map((config) => (
                    <SelectableCard
                      key={config.id}
                      config={config}
                      selected={taskType === config.id}
                      onSelect={() => setTaskType(config.id as TaskType)}
                      selectionMode="radio"
                    />
                  ))}
                </div>
              </div>

              {/* Permissions (Fix only) */}
              {taskType === "fix" && (
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-3">
                    What permissions does the agent have?
                  </label>
                  <div className="space-y-2">
                    {PERMISSIONS.map((config) => (
                      <CompactSelectableCard
                        key={config.id}
                        config={config}
                        selected={permissions.has(config.id as PermissionType)}
                        onSelect={() => togglePermission(config.id as PermissionType)}
                        selectionMode="checkbox"
                      />
                    ))}
                  </div>
                  {permissions.size === 0 && (
                    <p className="text-xs text-amber-400 mt-2">
                      Select at least one permission
                    </p>
                  )}
                </div>
              )}

              {/* Effort Level (Investigate only) */}
              {taskType === "investigate" && (
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-3">
                    Investigation depth
                  </label>
                  <div className="space-y-2">
                    {EFFORT_LEVELS.map((config) => (
                      <CompactSelectableCard
                        key={config.id}
                        config={config}
                        selected={effortLevel === config.id}
                        onSelect={() => setEffortLevel(config.id as EffortLevel)}
                        selectionMode="radio"
                      />
                    ))}
                  </div>
                </div>
              )}

              {/* Focus Areas */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-3">
                  Where should the agent focus?
                </label>
                <div className="grid grid-cols-2 gap-3">
                  {FOCUS_AREAS.map((config) => (
                    <SelectableCard
                      key={config.id}
                      config={config}
                      selected={focus.has(config.id as FocusType)}
                      onSelect={() => toggleFocus(config.id as FocusType)}
                      selectionMode="checkbox"
                    />
                  ))}
                </div>
                {focus.size === 0 && (
                  <p className="text-xs text-amber-400 mt-2">
                    Select at least one focus area
                  </p>
                )}
              </div>

              {/* Note */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Additional Notes
                </label>
                <textarea
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                  placeholder="E.g., 'This started failing after updating the database schema...'"
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-sm text-slate-200 placeholder:text-slate-500 resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  rows={3}
                />
              </div>

              {/* Advanced Options (Collapsible) */}
              <div className="border border-slate-700 rounded-lg">
                <button
                  type="button"
                  onClick={() => setAdvancedOpen(!advancedOpen)}
                  className="w-full flex items-center justify-between px-4 py-3 text-sm text-slate-300 hover:bg-slate-800/50 transition-colors"
                >
                  <span>
                    Advanced Options ({selectedContexts.size} contexts selected)
                  </span>
                  <ChevronDown
                    className={`h-4 w-4 transition-transform ${
                      advancedOpen ? "rotate-180" : ""
                    }`}
                  />
                </button>
                {advancedOpen && (
                  <div className="px-4 pb-4 space-y-2 border-t border-slate-700 pt-3">
                    <label className="block text-xs text-slate-400 mb-2">
                      Context to Include
                    </label>
                    {CONTEXT_OPTIONS.map((option) => (
                      <label
                        key={option.key}
                        className={`flex items-center gap-3 p-2 rounded-md cursor-pointer border transition-colors ${
                          selectedContexts.has(option.key)
                            ? "bg-slate-800/70 border-blue-500/60"
                            : "bg-slate-900 border-slate-800 hover:bg-slate-800/50"
                        }`}
                      >
                        <div className="relative flex items-center justify-center">
                          <input
                            type="checkbox"
                            checked={selectedContexts.has(option.key)}
                            onChange={() => toggleContext(option.key)}
                            className="sr-only"
                          />
                          <div
                            className={`w-4 h-4 rounded border flex items-center justify-center transition-colors ${
                              selectedContexts.has(option.key)
                                ? "bg-blue-500 border-blue-500"
                                : "bg-slate-800 border-slate-600"
                            }`}
                          >
                            {selectedContexts.has(option.key) && (
                              <Check className="h-3 w-3 text-white" />
                            )}
                          </div>
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="text-sm text-slate-200">
                              {option.label}
                            </span>
                            {"recommended" in option && option.recommended && (
                              <span className="text-[10px] px-1.5 py-0.5 bg-blue-500/20 text-blue-400 rounded">
                                recommended
                              </span>
                            )}
                          </div>
                          <p className="text-xs text-slate-500">
                            {option.description}
                          </p>
                        </div>
                      </label>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Footer */}
            <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-slate-700">
              <Button variant="ghost" onClick={handleReset}>
                Cancel
              </Button>
              <Button
                onClick={handleTrigger}
                disabled={!isAgentAvailable || isTriggering || !isFormValid}
              >
                {isTriggering ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
                    Starting...
                  </>
                ) : (
                  <>
                    <Bot className="h-4 w-4 mr-1.5" />
                    Spawn Agent
                  </>
                )}
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
