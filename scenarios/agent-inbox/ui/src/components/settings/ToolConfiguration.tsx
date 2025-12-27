/**
 * ToolConfiguration Component
 *
 * Displays available tools grouped by scenario with toggle switches.
 * Used in the Settings modal for global tool configuration and in
 * the ChatHeader for per-chat overrides.
 *
 * ARCHITECTURE:
 * - Receives tool data and callbacks from parent (separation of concerns)
 * - Pure presentational component with no direct API calls
 * - Supports both global and chat-specific configurations via props
 *
 * TESTING SEAMS:
 * - All data passed via props
 * - Callbacks for user actions
 * - Test IDs on interactive elements
 */

import { useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  RefreshCw,
  AlertCircle,
  CheckCircle2,
  XCircle,
  Loader2,
  RotateCcw,
  Zap,
  Clock,
  DollarSign,
  Shield,
  ShieldOff,
} from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import type { EffectiveTool, ScenarioStatus, ToolCategory, ApprovalOverride } from "../../lib/api";

interface ToolConfigurationProps {
  /** Tools grouped by scenario */
  toolsByScenario: Map<string, EffectiveTool[]>;
  /** Available categories for grouping */
  categories: ToolCategory[];
  /** Scenario availability statuses */
  scenarioStatuses?: ScenarioStatus[];
  /** Whether this is for a specific chat (shows "override" indicators) */
  chatId?: string;
  /** Loading state */
  isLoading?: boolean;
  /** Refreshing state */
  isRefreshing?: boolean;
  /** Updating state */
  isUpdating?: boolean;
  /** Error message */
  error?: string | null;
  /** Whether YOLO mode is enabled (hides approval selectors) */
  yoloMode?: boolean;
  /** Callback when tool is toggled */
  onToggleTool: (scenario: string, toolName: string, enabled: boolean) => void;
  /** Callback when tool approval is changed */
  onSetApproval?: (scenario: string, toolName: string, override: ApprovalOverride) => void;
  /** Callback when tool config is reset to default */
  onResetTool?: (scenario: string, toolName: string) => void;
  /** Callback to refresh tool registry */
  onRefresh?: () => void;
}

/**
 * Get icon for cost estimate
 */
function getCostIcon(cost?: string) {
  switch (cost) {
    case "low":
      return <DollarSign className="h-3 w-3 text-green-400" />;
    case "medium":
      return <DollarSign className="h-3 w-3 text-yellow-400" />;
    case "high":
      return <DollarSign className="h-3 w-3 text-red-400" />;
    default:
      return null;
  }
}

/**
 * Get scenario status icon and color
 */
function getStatusIndicator(status?: ScenarioStatus) {
  if (!status) {
    return <AlertCircle className="h-4 w-4 text-slate-500" />;
  }
  if (status.available) {
    return <CheckCircle2 className="h-4 w-4 text-green-400" />;
  }
  return <XCircle className="h-4 w-4 text-red-400" />;
}

export function ToolConfiguration({
  toolsByScenario,
  categories,
  scenarioStatuses,
  chatId,
  isLoading,
  isRefreshing,
  isUpdating,
  error,
  yoloMode,
  onToggleTool,
  onSetApproval,
  onResetTool,
  onRefresh,
}: ToolConfigurationProps) {
  // Track which scenarios are expanded
  const [expandedScenarios, setExpandedScenarios] = useState<Set<string>>(
    new Set(toolsByScenario.keys())
  );

  const toggleScenario = (scenario: string) => {
    setExpandedScenarios((prev) => {
      const next = new Set(prev);
      if (next.has(scenario)) {
        next.delete(scenario);
      } else {
        next.add(scenario);
      }
      return next;
    });
  };

  // Build status lookup
  const statusByScenario = new Map<string, ScenarioStatus>();
  for (const status of scenarioStatuses ?? []) {
    statusByScenario.set(status.scenario, status);
  }

  // Build category lookup
  const categoryById = new Map<string, ToolCategory>();
  for (const cat of categories) {
    categoryById.set(cat.id, cat);
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8" data-testid="tools-loading">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-400" />
        <span className="ml-2 text-sm text-slate-400">Loading tools...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex flex-col items-center justify-center py-8 text-center"
        data-testid="tools-error"
      >
        <AlertCircle className="h-8 w-8 text-red-400 mb-2" />
        <p className="text-sm text-red-400 mb-3">{error}</p>
        {onRefresh && (
          <Button variant="secondary" size="sm" onClick={onRefresh}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Retry
          </Button>
        )}
      </div>
    );
  }

  if (toolsByScenario.size === 0) {
    return (
      <div
        className="flex flex-col items-center justify-center py-8 text-center"
        data-testid="tools-empty"
      >
        <Zap className="h-8 w-8 text-slate-500 mb-2" />
        <p className="text-sm text-slate-400 mb-1">No tools available</p>
        <p className="text-xs text-slate-500">
          Start tool-providing scenarios to enable AI capabilities
        </p>
        {onRefresh && (
          <Button variant="secondary" size="sm" onClick={onRefresh} className="mt-4">
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        )}
      </div>
    );
  }

  const scenarioEntries = Array.from(toolsByScenario.entries());

  return (
    <div className="space-y-4" data-testid="tool-configuration">
      {/* Header with refresh button */}
      {onRefresh && (
        <div className="flex items-center justify-between">
          <p className="text-xs text-slate-500">
            {chatId ? "Override tool settings for this chat" : "Configure default tools for all chats"}
          </p>
          <Tooltip content="Refresh tool registry">
            <Button
              variant="ghost"
              size="sm"
              onClick={onRefresh}
              disabled={isRefreshing}
              data-testid="refresh-tools-button"
            >
              <RefreshCw className={`h-4 w-4 ${isRefreshing ? "animate-spin" : ""}`} />
            </Button>
          </Tooltip>
        </div>
      )}

      {/* Scenario sections */}
      {scenarioEntries.map(([scenario, tools]) => {
        const status = statusByScenario.get(scenario);
        const isExpanded = expandedScenarios.has(scenario);
        const enabledCount = tools.filter((t) => t.enabled).length;

        return (
          <div
            key={scenario}
            className="rounded-lg border border-white/10 bg-white/5 overflow-hidden"
            data-testid={`scenario-section-${scenario}`}
          >
            {/* Scenario header */}
            <button
              onClick={() => toggleScenario(scenario)}
              className="w-full flex items-center gap-3 p-3 hover:bg-white/5 transition-colors"
              data-testid={`scenario-toggle-${scenario}`}
            >
              {isExpanded ? (
                <ChevronDown className="h-4 w-4 text-slate-400" />
              ) : (
                <ChevronRight className="h-4 w-4 text-slate-400" />
              )}
              <div className="flex-1 flex items-center gap-2 min-w-0">
                {getStatusIndicator(status)}
                <span className="font-medium text-white truncate">{scenario}</span>
                <span className="text-xs text-slate-500">
                  {enabledCount}/{tools.length} enabled
                </span>
              </div>
            </button>

            {/* Tools list */}
            {isExpanded && (
              <div className="border-t border-white/10">
                {tools.map((effectiveTool) => {
                  const { tool, enabled, source, requires_approval, approval_override, approval_source } = effectiveTool;
                  const hasOverride = chatId && source === "chat";
                  const hasApprovalOverride = chatId && approval_source === "chat";
                  const category = tool.category ? categoryById.get(tool.category) : null;

                  // Determine current approval state for display
                  const currentApprovalOverride = approval_override ?? "";
                  const defaultRequiresApproval = tool.metadata.requires_approval;

                  return (
                    <div
                      key={tool.name}
                      className="flex items-start gap-3 p-3 border-b border-white/5 last:border-b-0 hover:bg-white/5 transition-colors"
                      data-testid={`tool-item-${scenario}-${tool.name}`}
                    >
                      {/* Toggle switch */}
                      <label className="relative inline-flex items-center cursor-pointer mt-0.5">
                        <input
                          type="checkbox"
                          checked={enabled}
                          onChange={(e) => onToggleTool(scenario, tool.name, e.target.checked)}
                          disabled={isUpdating}
                          className="sr-only peer"
                          data-testid={`tool-toggle-${scenario}-${tool.name}`}
                        />
                        <div className="w-9 h-5 bg-slate-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-indigo-500/50 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-500" />
                      </label>

                      {/* Tool info */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-sm font-medium text-white">{tool.name}</span>
                          {category && (
                            <span className="text-xs px-1.5 py-0.5 rounded bg-white/10 text-slate-400">
                              {category.name}
                            </span>
                          )}
                          {tool.metadata.long_running && (
                            <Tooltip content="Long-running operation">
                              <Clock className="h-3 w-3 text-slate-500" />
                            </Tooltip>
                          )}
                          {getCostIcon(tool.metadata.cost_estimate)}
                          {hasOverride && (
                            <span className="text-xs px-1.5 py-0.5 rounded bg-indigo-500/20 text-indigo-300">
                              override
                            </span>
                          )}
                          {/* Approval indicator (when not in YOLO mode) */}
                          {!yoloMode && enabled && (
                            <Tooltip content={requires_approval ? "Requires approval" : "Auto-executes"}>
                              {requires_approval ? (
                                <Shield className="h-3 w-3 text-yellow-400" />
                              ) : (
                                <ShieldOff className="h-3 w-3 text-green-400" />
                              )}
                            </Tooltip>
                          )}
                        </div>
                        <p className="text-xs text-slate-400 mt-0.5 line-clamp-2">
                          {tool.description}
                        </p>

                        {/* Approval selector (only shown when enabled and not in YOLO mode) */}
                        {enabled && !yoloMode && onSetApproval && (
                          <div className="flex items-center gap-2 mt-2">
                            <span className="text-xs text-slate-500">Approval:</span>
                            <select
                              value={currentApprovalOverride}
                              onChange={(e) => onSetApproval(scenario, tool.name, e.target.value as ApprovalOverride)}
                              disabled={isUpdating}
                              className="text-xs bg-white/5 border border-white/10 rounded px-2 py-1 text-slate-300 focus:outline-none focus:ring-1 focus:ring-indigo-500/50"
                              data-testid={`tool-approval-${scenario}-${tool.name}`}
                            >
                              <option value="">
                                Default ({defaultRequiresApproval ? "Require" : "Auto"})
                              </option>
                              <option value="require">Require Approval</option>
                              <option value="skip">Auto-execute</option>
                            </select>
                            {hasApprovalOverride && (
                              <span className="text-xs px-1.5 py-0.5 rounded bg-yellow-500/20 text-yellow-300">
                                custom
                              </span>
                            )}
                          </div>
                        )}

                        {tool.metadata.tags && tool.metadata.tags.length > 0 && (
                          <div className="flex gap-1 mt-1.5 flex-wrap">
                            {tool.metadata.tags.slice(0, 3).map((tag) => (
                              <span
                                key={tag}
                                className="text-xs px-1 py-0.5 rounded bg-white/5 text-slate-500"
                              >
                                {tag}
                              </span>
                            ))}
                          </div>
                        )}
                      </div>

                      {/* Reset button (for chat-specific overrides) */}
                      {hasOverride && onResetTool && (
                        <Tooltip content="Reset to default">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => onResetTool(scenario, tool.name)}
                            disabled={isUpdating}
                            className="shrink-0"
                            data-testid={`tool-reset-${scenario}-${tool.name}`}
                          >
                            <RotateCcw className="h-4 w-4" />
                          </Button>
                        </Tooltip>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
