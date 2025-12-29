/**
 * WorkflowPreviewPane Component
 *
 * Enhanced preview pane for viewing workflow details when selecting from the file tree.
 * Features:
 * - Header with name, type, and close button
 * - Primary action buttons (Run, Open Editor)
 * - Stats grid (steps, version, created, executions, success rate)
 * - Schedule section with view/add/edit
 * - Recent executions list (collapsible)
 * - JSON preview (collapsible) with Monaco editor
 */

import { useState, useEffect, useCallback, useMemo, lazy, Suspense } from "react";
import { useNavigate } from "react-router-dom";
import {
  X,
  Play,
  ExternalLink,
  Clock,
  Calendar,
  Hash,
  Activity,
  CheckCircle,
  XCircle,
  ChevronDown,
  ChevronRight,
  Plus,
  Pause,
  Trash2,
  Pencil,
  Timer,
  Code,
  Loader,
} from "lucide-react";
import type { WorkflowWithStats } from "./hooks/useProjectDetailStore";
import { useScheduleStore, type WorkflowSchedule, type CreateScheduleInput, type UpdateScheduleInput, describeCron, formatNextRun } from "@stores/scheduleStore";
import { useExecutionStore, type Execution } from "@/domains/executions";
import { useWorkflowStore } from "@stores/workflowStore";
import { ScheduleModal } from "@/views/SettingsView/sections/schedules/ScheduleModal";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";

// Lazy load Monaco editor to avoid large initial bundle
const Editor = lazy(() => import("@monaco-editor/react"));

// ============================================================================
// Types
// ============================================================================

interface WorkflowPreviewPaneProps {
  workflow: WorkflowWithStats;
  onClose: () => void;
  onOpenEditor: (workflowId: string) => Promise<void>;
}

// ============================================================================
// Collapsible Section Component
// ============================================================================

interface CollapsibleSectionProps {
  title: string;
  icon: React.ReactNode;
  defaultOpen?: boolean;
  children: React.ReactNode;
  badge?: React.ReactNode;
}

function CollapsibleSection({
  title,
  icon,
  defaultOpen = false,
  children,
  badge,
}: CollapsibleSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className="border border-gray-700 rounded-lg overflow-hidden">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center gap-2 px-3 py-2 bg-gray-800/50 hover:bg-gray-800 transition-colors text-left"
      >
        {isOpen ? (
          <ChevronDown size={16} className="text-gray-400 flex-shrink-0" />
        ) : (
          <ChevronRight size={16} className="text-gray-400 flex-shrink-0" />
        )}
        <span className="text-gray-400">{icon}</span>
        <span className="text-sm font-medium text-gray-300 flex-1">{title}</span>
        {badge}
      </button>
      {isOpen && <div className="p-3 border-t border-gray-700">{children}</div>}
    </div>
  );
}

// ============================================================================
// JSON Preview Section with Monaco Editor
// ============================================================================

interface JsonPreviewSectionProps {
  title: string;
  json: string | null;
  isLoading: boolean;
  error: string | null;
  onExpand: () => void;
}

function JsonPreviewSection({
  title,
  json,
  isLoading,
  error,
  onExpand,
}: JsonPreviewSectionProps) {
  const [isOpen, setIsOpen] = useState(false);

  const handleToggle = () => {
    const newIsOpen = !isOpen;
    setIsOpen(newIsOpen);
    if (newIsOpen && !json && !isLoading && !error) {
      onExpand();
    }
  };

  return (
    <div className="border border-gray-700 rounded-lg overflow-hidden">
      <button
        onClick={handleToggle}
        className="w-full flex items-center gap-2 px-3 py-2 bg-gray-800/50 hover:bg-gray-800 transition-colors text-left"
      >
        {isOpen ? (
          <ChevronDown size={16} className="text-gray-400 flex-shrink-0" />
        ) : (
          <ChevronRight size={16} className="text-gray-400 flex-shrink-0" />
        )}
        <span className="text-gray-400">
          <Code size={14} />
        </span>
        <span className="text-sm font-medium text-gray-300 flex-1">{title}</span>
      </button>
      {isOpen && (
        <div className="border-t border-gray-700">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader size={20} className="animate-spin text-gray-400" />
              <span className="ml-2 text-sm text-gray-400">Loading workflow...</span>
            </div>
          ) : error ? (
            <div className="p-3 text-sm text-red-400">{error}</div>
          ) : json ? (
            <div className="h-[300px]">
              <Suspense
                fallback={
                  <div className="flex items-center justify-center h-full">
                    <Loader size={20} className="animate-spin text-gray-400" />
                  </div>
                }
              >
                <Editor
                  height="100%"
                  defaultLanguage="json"
                  value={json}
                  theme="vs-dark"
                  options={{
                    readOnly: true,
                    minimap: { enabled: false },
                    fontSize: 12,
                    lineNumbers: "on",
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                    tabSize: 2,
                    wordWrap: "on",
                    folding: true,
                    scrollbar: {
                      verticalScrollbarSize: 8,
                      horizontalScrollbarSize: 8,
                    },
                  }}
                />
              </Suspense>
            </div>
          ) : (
            <div className="p-3 text-sm text-gray-500">No data available</div>
          )}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Stats Grid Component
// ============================================================================

interface StatItemProps {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
  subValue?: string;
}

function StatItem({ icon, label, value, subValue }: StatItemProps) {
  return (
    <div className="bg-gray-800/50 rounded-lg px-3 py-2">
      <div className="flex items-center gap-1.5 mb-1">
        <span className="text-gray-500">{icon}</span>
        <span className="text-xs text-gray-500">{label}</span>
      </div>
      <div className="text-lg font-semibold text-surface">{value}</div>
      {subValue && <div className="text-xs text-gray-500">{subValue}</div>}
    </div>
  );
}

// ============================================================================
// Schedule Card Component
// ============================================================================

interface ScheduleCardProps {
  schedule: WorkflowSchedule;
  onEdit: () => void;
  onToggle: () => void;
  onDelete: () => void;
  isLoading?: boolean;
}

function ScheduleCard({ schedule, onEdit, onToggle, onDelete, isLoading }: ScheduleCardProps) {
  return (
    <div className="bg-gray-800/50 rounded-lg p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span
              className={`w-2 h-2 rounded-full flex-shrink-0 ${
                schedule.is_active ? "bg-green-500" : "bg-gray-500"
              }`}
            />
            <span className="text-sm font-medium text-gray-200 truncate">
              {schedule.name}
            </span>
          </div>
          <p className="text-xs text-gray-400 mt-1">{describeCron(schedule.cron_expression)}</p>
          {schedule.is_active && schedule.next_run_at && (
            <p className="text-xs text-gray-500 mt-0.5 flex items-center gap-1">
              <Timer size={12} />
              {formatNextRun(schedule.next_run_at, schedule.next_run_human)}
            </p>
          )}
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={onEdit}
            disabled={isLoading}
            className="p-1.5 text-gray-500 hover:text-gray-300 hover:bg-gray-700 rounded transition-colors disabled:opacity-50"
            title="Edit schedule"
          >
            <Pencil size={14} />
          </button>
          <button
            onClick={onToggle}
            disabled={isLoading}
            className="p-1.5 text-gray-500 hover:text-gray-300 hover:bg-gray-700 rounded transition-colors disabled:opacity-50"
            title={schedule.is_active ? "Pause schedule" : "Resume schedule"}
          >
            {schedule.is_active ? <Pause size={14} /> : <Play size={14} />}
          </button>
          <button
            onClick={onDelete}
            disabled={isLoading}
            className="p-1.5 text-gray-500 hover:text-red-400 hover:bg-red-500/10 rounded transition-colors disabled:opacity-50"
            title="Delete schedule"
          >
            <Trash2 size={14} />
          </button>
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Execution Item Component
// ============================================================================

interface ExecutionItemProps {
  execution: Execution;
  onClick: () => void;
}

function ExecutionItem({ execution, onClick }: ExecutionItemProps) {
  const statusIcon =
    execution.status === "completed" ? (
      <CheckCircle size={14} className="text-green-400" />
    ) : execution.status === "failed" ? (
      <XCircle size={14} className="text-red-400" />
    ) : execution.status === "running" ? (
      <Loader size={14} className="text-blue-400 animate-spin" />
    ) : (
      <Clock size={14} className="text-gray-400" />
    );

  const duration = execution.completedAt
    ? Math.round((execution.completedAt.getTime() - execution.startedAt.getTime()) / 1000)
    : null;

  const formatTime = (date: Date) => {
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    if (diff < 60000) return "Just now";
    if (diff < 3600000) return `${Math.round(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.round(diff / 3600000)}h ago`;
    return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
  };

  return (
    <button
      onClick={onClick}
      className="w-full flex items-center gap-3 px-2 py-1.5 hover:bg-gray-800 rounded transition-colors text-left"
    >
      {statusIcon}
      <span className="flex-1 text-sm text-gray-300 capitalize">{execution.status}</span>
      {duration !== null && (
        <span className="text-xs text-gray-500">{duration}s</span>
      )}
      <span className="text-xs text-gray-500">{formatTime(execution.startedAt)}</span>
    </button>
  );
}

// ============================================================================
// Main Component
// ============================================================================

export function WorkflowPreviewPane({
  workflow,
  onClose,
  onOpenEditor,
}: WorkflowPreviewPaneProps) {
  const navigate = useNavigate();

  // Workflow store for loading full workflow definition
  const loadWorkflow = useWorkflowStore((s) => s.loadWorkflow);
  const currentWorkflow = useWorkflowStore((s) => s.currentWorkflow);
  const startExecution = useExecutionStore((s) => s.startExecution);

  // State for full workflow JSON
  const [fullWorkflowJson, setFullWorkflowJson] = useState<string | null>(null);
  const [isLoadingJson, setIsLoadingJson] = useState(false);
  const [jsonError, setJsonError] = useState<string | null>(null);

  // Schedule store
  const schedules = useScheduleStore((s) => s.schedules);
  const schedulesLoading = useScheduleStore((s) => s.isLoading);
  const fetchSchedulesByWorkflow = useScheduleStore((s) => s.fetchSchedulesByWorkflow);
  const createSchedule = useScheduleStore((s) => s.createSchedule);
  const updateSchedule = useScheduleStore((s) => s.updateSchedule);
  const deleteSchedule = useScheduleStore((s) => s.deleteSchedule);
  const toggleSchedule = useScheduleStore((s) => s.toggleSchedule);

  // Execution store
  const executions = useExecutionStore((s) => s.executions);
  const loadExecutions = useExecutionStore((s) => s.loadExecutions);
  const loadExecution = useExecutionStore((s) => s.loadExecution);

  // Local state
  const [showScheduleModal, setShowScheduleModal] = useState(false);
  const [editingSchedule, setEditingSchedule] = useState<WorkflowSchedule | null>(null);
  const [executionsLoading, setExecutionsLoading] = useState(false);

  // Filter schedules for this workflow
  const workflowSchedules = useMemo(
    () => schedules.filter((s) => s.workflow_id === workflow.id),
    [schedules, workflow.id]
  );

  // Filter executions for this workflow (last 5)
  const recentExecutions = useMemo(
    () =>
      executions
        .filter((e) => e.workflowId === workflow.id)
        .sort((a, b) => b.startedAt.getTime() - a.startedAt.getTime())
        .slice(0, 5),
    [executions, workflow.id]
  );

  // Load schedules and executions when workflow changes
  useEffect(() => {
    void fetchSchedulesByWorkflow(workflow.id);
  }, [workflow.id, fetchSchedulesByWorkflow]);

  useEffect(() => {
    const loadData = async () => {
      setExecutionsLoading(true);
      try {
        await loadExecutions(workflow.id);
      } catch (error) {
        logger.error(
          "Failed to load executions for preview",
          { component: "WorkflowPreviewPane", workflowId: workflow.id },
          error
        );
      } finally {
        setExecutionsLoading(false);
      }
    };
    void loadData();
  }, [workflow.id, loadExecutions]);

  // Format date helper
  const formatDate = useCallback((dateString: string | Date | undefined) => {
    if (!dateString) return "—";
    const date = typeof dateString === "string" ? new Date(dateString) : dateString;
    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  }, []);

  // Load full workflow JSON for preview
  const loadFullWorkflowJson = useCallback(async () => {
    if (fullWorkflowJson || isLoadingJson) return;
    setIsLoadingJson(true);
    setJsonError(null);
    try {
      await loadWorkflow(workflow.id);
    } catch (error) {
      logger.error(
        "Failed to load workflow JSON",
        { component: "WorkflowPreviewPane", workflowId: workflow.id },
        error
      );
      setJsonError(error instanceof Error ? error.message : "Failed to load workflow");
    } finally {
      setIsLoadingJson(false);
    }
  }, [workflow.id, fullWorkflowJson, isLoadingJson, loadWorkflow]);

  // Update full workflow JSON when currentWorkflow changes
  useEffect(() => {
    if (currentWorkflow && currentWorkflow.id === workflow.id) {
      try {
        // Build the full workflow object for display
        const fullWorkflow = {
          id: currentWorkflow.id,
          name: currentWorkflow.name,
          description: currentWorkflow.description,
          version: currentWorkflow.version,
          createdAt: currentWorkflow.createdAt,
          updatedAt: currentWorkflow.updatedAt,
          flowDefinition: currentWorkflow.flowDefinition,
        };
        setFullWorkflowJson(JSON.stringify(fullWorkflow, null, 2));
      } catch (error) {
        logger.error(
          "Failed to serialize workflow JSON",
          { component: "WorkflowPreviewPane", workflowId: workflow.id },
          error
        );
        setJsonError("Failed to serialize workflow");
      }
    }
  }, [currentWorkflow, workflow.id]);

  // Run workflow with live execution view in Record page
  const handleRunWithLiveView = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      try {
        // Start the execution to get an execution ID
        await startExecution(workflow.id);
        // Navigate to Record page in execution mode
        // The execution will be tracked via the execution store
        navigate(`/record/new?mode=execution`);
      } catch (error) {
        logger.error(
          "Failed to start execution",
          { component: "WorkflowPreviewPane", workflowId: workflow.id },
          error
        );
        toast.error("Failed to start execution");
      }
    },
    [workflow.id, startExecution, navigate]
  );

  // Calculate stats
  const stepCount = workflow.nodes?.length ?? 0;
  const version = workflow.version ?? 1;
  const createdAt = workflow.createdAt ?? workflow.created_at;
  const updatedAt = workflow.updatedAt ?? workflow.updated_at;
  const executionCount = workflow.stats?.execution_count ?? 0;
  const successRate = workflow.stats?.success_rate;

  // Get workflow type from path
  const getWorkflowType = (name: string): string => {
    const lower = name.toLowerCase();
    if (lower.endsWith(".action.json")) return "Action";
    if (lower.endsWith(".flow.json")) return "Flow";
    if (lower.endsWith(".case.json")) return "Case";
    return "Workflow";
  };

  const workflowType = getWorkflowType(workflow.name);

  // Schedule handlers
  const handleCreateSchedule = useCallback(() => {
    setEditingSchedule(null);
    setShowScheduleModal(true);
  }, []);

  const handleEditSchedule = useCallback((schedule: WorkflowSchedule) => {
    setEditingSchedule(schedule);
    setShowScheduleModal(true);
  }, []);

  const handleToggleSchedule = useCallback(
    async (schedule: WorkflowSchedule) => {
      const result = await toggleSchedule(schedule.id);
      if (result) {
        toast.success(result.is_active ? "Schedule resumed" : "Schedule paused");
      }
    },
    [toggleSchedule]
  );

  const handleDeleteSchedule = useCallback(
    async (schedule: WorkflowSchedule) => {
      if (!window.confirm(`Delete schedule "${schedule.name}"?`)) return;
      const success = await deleteSchedule(schedule.id);
      if (success) {
        toast.success("Schedule deleted");
      }
    },
    [deleteSchedule]
  );

  const handleSaveSchedule = useCallback(
    async (input: CreateScheduleInput | UpdateScheduleInput, workflowId: string) => {
      if (editingSchedule) {
        await updateSchedule(editingSchedule.id, input as UpdateScheduleInput);
        toast.success("Schedule updated");
      } else {
        await createSchedule(workflowId, input as CreateScheduleInput);
        toast.success("Schedule created");
      }
      setShowScheduleModal(false);
      setEditingSchedule(null);
    },
    [editingSchedule, createSchedule, updateSchedule]
  );

  // View execution
  const handleViewExecution = useCallback(
    async (execution: Execution) => {
      await loadExecution(execution.id);
    },
    [loadExecution]
  );

  return (
    <div className="bg-flow-node border border-gray-700 rounded-lg h-full flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-gray-700 flex items-start justify-between gap-4">
        <div className="flex items-center gap-3 min-w-0">
          <div className="flex-shrink-0 w-10 h-10 rounded-xl flex items-center justify-center bg-gradient-to-br from-green-500/20 to-emerald-500/10 border border-green-500/20">
            <Code size={18} className="text-green-400" />
          </div>
          <div className="min-w-0">
            <h3 className="text-lg font-semibold text-surface truncate">{workflow.name}</h3>
            <div className="flex items-center gap-2 mt-0.5">
              <span className="text-xs px-1.5 py-0.5 rounded bg-gray-800 text-gray-400">
                {workflowType}
              </span>
              <span className="text-xs text-gray-500">v{version}</span>
            </div>
          </div>
        </div>
        <button
          onClick={onClose}
          className="p-1.5 text-gray-500 hover:text-gray-300 hover:bg-gray-700 rounded-lg transition-colors"
          title="Close preview"
        >
          <X size={18} />
        </button>
      </div>

      {/* Primary Actions */}
      <div className="p-4 border-b border-gray-700 flex gap-3">
        <button
          onClick={handleRunWithLiveView}
          className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors font-medium"
        >
          <Play size={18} />
          <span>Run</span>
        </button>
        <button
          onClick={() => onOpenEditor(workflow.id)}
          className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors font-medium"
        >
          <ExternalLink size={18} />
          <span>Open Editor</span>
        </button>
      </div>

      {/* Scrollable Content */}
      <div className="flex-1 p-4 space-y-4 overflow-auto">
        {/* Description */}
        {workflow.description && (
          <div>
            <h4 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
              Description
            </h4>
            <p className="text-sm text-gray-300">{workflow.description}</p>
          </div>
        )}

        {/* Stats Grid */}
        <div>
          <h4 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
            Statistics
          </h4>
          <div className="grid grid-cols-2 gap-2">
            <StatItem
              icon={<Hash size={12} />}
              label="Steps"
              value={stepCount}
            />
            <StatItem
              icon={<Activity size={12} />}
              label="Executions"
              value={executionCount}
            />
            <StatItem
              icon={<CheckCircle size={12} />}
              label="Success Rate"
              value={
                successRate != null ? (
                  <span
                    className={
                      successRate >= 80
                        ? "text-green-400"
                        : successRate >= 50
                          ? "text-amber-400"
                          : "text-red-400"
                    }
                  >
                    {successRate}%
                  </span>
                ) : (
                  "—"
                )
              }
            />
            <StatItem
              icon={<Calendar size={12} />}
              label="Version"
              value={`v${version}`}
            />
          </div>
        </div>

        {/* Dates */}
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500">
          <span className="flex items-center gap-1">
            <Calendar size={12} />
            Created {formatDate(createdAt)}
          </span>
          <span className="flex items-center gap-1">
            <Clock size={12} />
            Updated {formatDate(updatedAt)}
          </span>
        </div>

        {/* Schedule Section */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <h4 className="text-xs font-medium text-gray-500 uppercase tracking-wider">
              Schedule
            </h4>
            <button
              onClick={handleCreateSchedule}
              className="text-xs text-flow-accent hover:text-blue-400 flex items-center gap-1 transition-colors"
            >
              <Plus size={12} />
              Add
            </button>
          </div>
          {schedulesLoading ? (
            <div className="flex items-center justify-center py-4">
              <Loader size={16} className="animate-spin text-gray-400" />
            </div>
          ) : workflowSchedules.length === 0 ? (
            <div className="bg-gray-800/30 rounded-lg p-3 text-center">
              <p className="text-sm text-gray-500">No schedules configured</p>
              <button
                onClick={handleCreateSchedule}
                className="mt-2 text-xs text-flow-accent hover:text-blue-400 transition-colors"
              >
                + Add a schedule
              </button>
            </div>
          ) : (
            <div className="space-y-2">
              {workflowSchedules.map((schedule) => (
                <ScheduleCard
                  key={schedule.id}
                  schedule={schedule}
                  onEdit={() => handleEditSchedule(schedule)}
                  onToggle={() => handleToggleSchedule(schedule)}
                  onDelete={() => handleDeleteSchedule(schedule)}
                  isLoading={schedulesLoading}
                />
              ))}
            </div>
          )}
        </div>

        {/* Recent Executions (Collapsible) */}
        <CollapsibleSection
          title="Recent Executions"
          icon={<Activity size={14} />}
          defaultOpen={false}
          badge={
            recentExecutions.length > 0 ? (
              <span className="text-xs px-1.5 py-0.5 rounded bg-gray-700 text-gray-400">
                {recentExecutions.length}
              </span>
            ) : null
          }
        >
          {executionsLoading ? (
            <div className="flex items-center justify-center py-4">
              <Loader size={16} className="animate-spin text-gray-400" />
            </div>
          ) : recentExecutions.length === 0 ? (
            <p className="text-sm text-gray-500 text-center py-2">No executions yet</p>
          ) : (
            <div className="space-y-1">
              {recentExecutions.map((execution) => (
                <ExecutionItem
                  key={execution.id}
                  execution={execution}
                  onClick={() => handleViewExecution(execution)}
                />
              ))}
            </div>
          )}
        </CollapsibleSection>

        {/* JSON Preview (Collapsible with Monaco Editor) */}
        <JsonPreviewSection
          title="Workflow JSON"
          json={fullWorkflowJson}
          isLoading={isLoadingJson}
          error={jsonError}
          onExpand={loadFullWorkflowJson}
        />
      </div>

      {/* Schedule Modal */}
      <ScheduleModal
        isOpen={showScheduleModal}
        onClose={() => {
          setShowScheduleModal(false);
          setEditingSchedule(null);
        }}
        onSave={handleSaveSchedule}
        schedule={editingSchedule}
        workflowId={workflow.id}
        workflowName={workflow.name}
      />
    </div>
  );
}

export default WorkflowPreviewPane;
