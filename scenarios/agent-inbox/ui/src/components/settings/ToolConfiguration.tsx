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

import { useState, useMemo, useRef, useCallback } from "react";
import {
  ChevronDown,
  ChevronRight,
  RefreshCw,
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Loader2,
  RotateCcw,
  Zap,
  Clock,
  DollarSign,
  Shield,
  ShieldOff,
  Play,
  Info,
  Search,
} from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import type { EffectiveTool, ScenarioStatus, ToolCategory, ApprovalOverride } from "../../lib/api";

type SortOrder = "a-z" | "z-a" | "most-enabled" | "fewest-enabled";

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
  /** Syncing state (discovering tools from running scenarios) */
  isSyncing?: boolean;
  /** Updating state */
  isUpdating?: boolean;
  /** Error message */
  error?: string | null;
  /** Whether YOLO mode is enabled (hides approval selectors) */
  yoloMode?: boolean;
  /** Total count of enabled tools (for token cost warning) */
  enabledCount?: number;
  /** Total count of all tools */
  totalCount?: number;
  /** Callback when tool is toggled (can be async for sequential batch operations) */
  onToggleTool: (scenario: string, toolName: string, enabled: boolean) => void | Promise<void>;
  /** Callback when tool approval is changed */
  onSetApproval?: (scenario: string, toolName: string, override: ApprovalOverride) => void;
  /** Callback when tool config is reset to default */
  onResetTool?: (scenario: string, toolName: string) => void;
  /** Callback to discover tools from all running scenarios */
  onSyncTools?: () => void;
  /** Callback when user wants to run a tool manually */
  onRunTool?: (tool: EffectiveTool) => void;
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
  isSyncing,
  isUpdating,
  error,
  yoloMode,
  enabledCount,
  totalCount,
  onToggleTool,
  onSetApproval,
  onResetTool,
  onSyncTools,
  onRunTool,
}: ToolConfigurationProps) {
  // Track which scenarios are expanded
  const [expandedScenarios, setExpandedScenarios] = useState<Set<string>>(
    new Set(toolsByScenario.keys())
  );
  // Search query
  const [searchQuery, setSearchQuery] = useState("");
  // Sort order (default A-Z)
  const [sortOrder, setSortOrder] = useState<SortOrder>("a-z");

  // Ref to prevent concurrent batch operations
  const batchOperationInProgress = useRef(false);

  // Stable callback for toggling all tools in a scenario
  // Uses current toolsByScenario from props to avoid stale closures
  const handleToggleScenario = useCallback(async (scenarioName: string, enableAll: boolean) => {
    if (batchOperationInProgress.current || isUpdating) return;

    batchOperationInProgress.current = true;
    try {
      // Get CURRENT tools from props (not from closure)
      const currentTools = toolsByScenario.get(scenarioName) ?? [];
      for (const tool of currentTools) {
        if (tool.enabled !== enableAll) {
          await onToggleTool(scenarioName, tool.tool.name, enableAll);
        }
      }
    } finally {
      batchOperationInProgress.current = false;
    }
  }, [toolsByScenario, isUpdating, onToggleTool]);

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

  // Filter and sort scenarios
  // NOTE: This useMemo MUST be called before any early returns to comply with Rules of Hooks
  const scenarioEntries = useMemo(() => {
    const entries = Array.from(toolsByScenario.entries());
    const query = searchQuery.toLowerCase().trim();

    // Filter by search query (match scenario name or tool names/descriptions)
    const filtered = query
      ? entries.filter(([scenario, tools]) => {
          // Match scenario name
          if (scenario.toLowerCase().includes(query)) return true;
          // Match any tool name or description
          return tools.some(
            (t) =>
              t.tool.name.toLowerCase().includes(query) ||
              t.tool.description.toLowerCase().includes(query)
          );
        })
      : entries;

    // Sort scenarios
    return filtered.sort((a, b) => {
      const [scenarioA, toolsA] = a;
      const [scenarioB, toolsB] = b;
      const enabledA = toolsA.filter((t) => t.enabled).length;
      const enabledB = toolsB.filter((t) => t.enabled).length;

      switch (sortOrder) {
        case "a-z":
          return scenarioA.localeCompare(scenarioB);
        case "z-a":
          return scenarioB.localeCompare(scenarioA);
        case "most-enabled":
          return enabledB - enabledA || scenarioA.localeCompare(scenarioB);
        case "fewest-enabled":
          return enabledA - enabledB || scenarioA.localeCompare(scenarioB);
        default:
          return scenarioA.localeCompare(scenarioB);
      }
    });
  }, [toolsByScenario, searchQuery, sortOrder]);

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
        {onSyncTools && (
          <Button variant="secondary" size="sm" onClick={onSyncTools} disabled={isSyncing}>
            <RefreshCw className={`h-4 w-4 mr-2 ${isSyncing ? "animate-spin" : ""}`} />
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
        {onSyncTools && (
          <Button variant="secondary" size="sm" onClick={onSyncTools} disabled={isSyncing} className="mt-4">
            <RefreshCw className={`h-4 w-4 mr-2 ${isSyncing ? "animate-spin" : ""}`} />
            Sync Tools
          </Button>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4" data-testid="tool-configuration">
      {/* Token cost explanation */}
      <div className="p-3 rounded-lg bg-slate-800/50 border border-white/10">
        <div className="flex items-start gap-2">
          <Info className="h-4 w-4 text-slate-400 mt-0.5 shrink-0" />
          <div className="text-xs text-slate-400">
            <p>Each enabled tool adds ~50-200 tokens to every AI request, increasing costs and latency.</p>
            <p className="mt-1">Enable only the tools you need for your current workflow.</p>
          </div>
        </div>
      </div>

      {/* Warning when >10 tools enabled */}
      {enabledCount !== undefined && enabledCount > 10 && (
        <div className="p-3 rounded-lg bg-amber-500/10 border border-amber-500/30">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-4 w-4 text-amber-400 shrink-0" />
            <span className="text-xs text-amber-300">
              {enabledCount} tools enabled — this adds ~{enabledCount * 100}-{enabledCount * 200} tokens per request
            </span>
          </div>
        </div>
      )}

      {/* Search and sort controls */}
      <div className="flex items-center gap-3">
        {/* Search input */}
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
          <input
            type="text"
            placeholder="Search tools or scenarios..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-8 pr-3 py-1.5 text-sm bg-white/5 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-indigo-500/50"
            data-testid="tool-search-input"
          />
        </div>

        {/* Sort dropdown */}
        <div className="flex items-center gap-1.5">
          <Tooltip content="Sort scenarios">
            <select
              value={sortOrder}
              onChange={(e) => setSortOrder(e.target.value as SortOrder)}
              className="text-xs bg-white/5 border border-white/10 rounded px-2 py-1.5 text-slate-300 focus:outline-none focus:ring-1 focus:ring-indigo-500/50 cursor-pointer"
              data-testid="tool-sort-select"
            >
              <option value="a-z">A → Z</option>
              <option value="z-a">Z → A</option>
              <option value="most-enabled">Most Enabled</option>
              <option value="fewest-enabled">Fewest Enabled</option>
            </select>
          </Tooltip>
        </div>
      </div>

      {/* Header with sync button */}
      {onSyncTools && (
        <div className="flex items-center justify-between">
          <p className="text-xs text-slate-500">
            {chatId ? "Override tool settings for this chat" : "Configure default tools for all chats"}
            {totalCount !== undefined && ` (${enabledCount ?? 0}/${totalCount} enabled)`}
          </p>
          <Tooltip content="Discover tools from all running scenarios">
            <Button
              variant="ghost"
              size="sm"
              onClick={onSyncTools}
              disabled={isSyncing}
              data-testid="sync-tools-button"
            >
              <RefreshCw className={`h-4 w-4 ${isSyncing ? "animate-spin" : ""}`} />
              <span className="ml-1 text-xs">{isSyncing ? "Syncing..." : "Sync"}</span>
            </Button>
          </Tooltip>
        </div>
      )}

      {/* No results message */}
      {searchQuery && scenarioEntries.length === 0 && (
        <div className="flex flex-col items-center justify-center py-8 text-center" data-testid="tools-no-results">
          <Search className="h-8 w-8 text-slate-500 mb-2" />
          <p className="text-sm text-slate-400">No tools or scenarios match "{searchQuery}"</p>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setSearchQuery("")}
            className="mt-2"
          >
            Clear search
          </Button>
        </div>
      )}

      {/* Scenario sections */}
      {scenarioEntries.map(([scenario, tools]) => {
        const status = statusByScenario.get(scenario);
        const isExpanded = expandedScenarios.has(scenario);
        const scenarioEnabledCount = tools.filter((t) => t.enabled).length;
        const allEnabled = scenarioEnabledCount === tools.length;
        const someEnabled = scenarioEnabledCount > 0 && scenarioEnabledCount < tools.length;

        return (
          <div
            key={scenario}
            className="rounded-lg border border-white/10 bg-white/5 overflow-hidden"
            data-testid={`scenario-section-${scenario}`}
          >
            {/* Scenario header */}
            <div className="flex items-center gap-3 p-3 hover:bg-white/5 transition-colors">
              {/* Expand/collapse button */}
              <button
                onClick={() => toggleScenario(scenario)}
                className="flex items-center gap-3 flex-1 min-w-0"
                data-testid={`scenario-toggle-${scenario}`}
              >
                {isExpanded ? (
                  <ChevronDown className="h-4 w-4 text-slate-400 shrink-0" />
                ) : (
                  <ChevronRight className="h-4 w-4 text-slate-400 shrink-0" />
                )}
                <div className="flex-1 flex items-center gap-2 min-w-0">
                  {getStatusIndicator(status)}
                  <span className="font-medium text-white truncate">{scenario}</span>
                  <span className="text-xs text-slate-500">
                    {scenarioEnabledCount}/{tools.length} enabled
                  </span>
                </div>
              </button>

              {/* Toggle all switch */}
              <Tooltip content={allEnabled ? "Disable all tools in this scenario" : "Enable all tools in this scenario"}>
                <button
                  type="button"
                  onClick={() => handleToggleScenario(scenario, !allEnabled)}
                  disabled={isUpdating || batchOperationInProgress.current}
                  className={`relative inline-flex items-center shrink-0 w-9 h-5 rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500/50 ${
                    isUpdating ? "cursor-not-allowed opacity-50" : "cursor-pointer"
                  } ${
                    allEnabled
                      ? "bg-indigo-500"
                      : someEnabled
                        ? "bg-indigo-500/50"
                        : "bg-slate-700"
                  }`}
                  data-testid={`scenario-toggle-all-${scenario}`}
                >
                  <span
                    className={`absolute top-[2px] left-[2px] bg-white rounded-full h-4 w-4 transition-transform ${
                      allEnabled ? "translate-x-full" : ""
                    }`}
                  />
                </button>
              </Tooltip>
            </div>

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

                      {/* Action buttons */}
                      <div className="flex items-center gap-1 shrink-0">
                        {/* Run button (manual tool execution) */}
                        {onRunTool && (
                          <Tooltip content="Run manually">
                            <Button
                              variant="ghost"
                              size="icon"
                              onClick={() => onRunTool(effectiveTool)}
                              disabled={isUpdating}
                              data-testid={`tool-run-${scenario}-${tool.name}`}
                            >
                              <Play className="h-4 w-4 text-indigo-400" />
                            </Button>
                          </Tooltip>
                        )}
                        {/* Reset button (for chat-specific overrides) */}
                        {hasOverride && onResetTool && (
                          <Tooltip content="Reset to default">
                            <Button
                              variant="ghost"
                              size="icon"
                              onClick={() => onResetTool(scenario, tool.name)}
                              disabled={isUpdating}
                              data-testid={`tool-reset-${scenario}-${tool.name}`}
                            >
                              <RotateCcw className="h-4 w-4" />
                            </Button>
                          </Tooltip>
                        )}
                      </div>
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
