/**
 * ToolConfiguration Component
 *
 * Displays available tools grouped by scenario with toggle switches.
 * Used in the Settings modal for global tool configuration and in
 * the ChatHeader for per-chat overrides.
 */

import { useState, useMemo, useRef, useCallback } from "react";
import {
  RefreshCw,
  AlertCircle,
  AlertTriangle,
  Loader2,
  Zap,
  Info,
  Search,
} from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { ScenarioSection } from "./ScenarioSection";
import type { EffectiveTool, ScenarioStatus, ToolCategory, ApprovalOverride } from "../../lib/api";

type SortOrder = "a-z" | "z-a" | "most-enabled" | "fewest-enabled";

interface ToolConfigurationProps {
  toolsByScenario: Map<string, EffectiveTool[]>;
  categories: ToolCategory[];
  scenarioStatuses?: ScenarioStatus[];
  chatId?: string;
  isLoading?: boolean;
  isSyncing?: boolean;
  isUpdating?: boolean;
  error?: string | null;
  yoloMode?: boolean;
  enabledCount?: number;
  totalCount?: number;
  onToggleTool: (scenario: string, toolName: string, enabled: boolean) => void | Promise<void>;
  onSetApproval?: (scenario: string, toolName: string, override: ApprovalOverride) => void;
  onResetTool?: (scenario: string, toolName: string) => void;
  onSyncTools?: () => void;
  onRunTool?: (tool: EffectiveTool) => void;
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
  const [expandedScenarios, setExpandedScenarios] = useState<Set<string>>(
    new Set(toolsByScenario.keys())
  );
  const [searchQuery, setSearchQuery] = useState("");
  const [sortOrder, setSortOrder] = useState<SortOrder>("a-z");

  const batchOperationInProgress = useRef(false);

  const handleToggleScenario = useCallback(async (scenarioName: string, enableAll: boolean) => {
    if (batchOperationInProgress.current || isUpdating) return;
    batchOperationInProgress.current = true;
    try {
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

  const toggleScenario = useCallback((scenario: string) => {
    setExpandedScenarios((prev) => {
      const next = new Set(prev);
      if (next.has(scenario)) next.delete(scenario);
      else next.add(scenario);
      return next;
    });
  }, []);

  // Build status lookup
  const statusByScenario = useMemo(() => {
    const map = new Map<string, ScenarioStatus>();
    for (const status of scenarioStatuses ?? []) {
      map.set(status.scenario, status);
    }
    return map;
  }, [scenarioStatuses]);

  // Build category lookup
  const categoryById = useMemo(() => {
    const map = new Map<string, ToolCategory>();
    for (const cat of categories) {
      map.set(cat.id, cat);
    }
    return map;
  }, [categories]);

  // Filter and sort scenarios
  const scenarioEntries = useMemo(() => {
    const entries = Array.from(toolsByScenario.entries());
    const query = searchQuery.toLowerCase().trim();

    const filtered = query
      ? entries.filter(([scenario, tools]) => {
          if (scenario.toLowerCase().includes(query)) return true;
          return tools.some(
            (t) =>
              t.tool.name.toLowerCase().includes(query) ||
              t.tool.description.toLowerCase().includes(query)
          );
        })
      : entries;

    return filtered.sort((a, b) => {
      const [scenarioA, toolsA] = a;
      const [scenarioB, toolsB] = b;
      const enabledA = toolsA.filter((t) => t.enabled).length;
      const enabledB = toolsB.filter((t) => t.enabled).length;

      switch (sortOrder) {
        case "a-z": return scenarioA.localeCompare(scenarioB);
        case "z-a": return scenarioB.localeCompare(scenarioA);
        case "most-enabled": return enabledB - enabledA || scenarioA.localeCompare(scenarioB);
        case "fewest-enabled": return enabledA - enabledB || scenarioA.localeCompare(scenarioB);
        default: return scenarioA.localeCompare(scenarioB);
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
      <div className="flex flex-col items-center justify-center py-8 text-center" data-testid="tools-error">
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
      <div className="flex flex-col items-center justify-center py-8 text-center" data-testid="tools-empty">
        <Zap className="h-8 w-8 text-slate-500 mb-2" />
        <p className="text-sm text-slate-400 mb-1">No tools available</p>
        <p className="text-xs text-slate-500">Start tool-providing scenarios to enable AI capabilities</p>
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
        <div className="flex items-center gap-1.5">
          <Tooltip content="Sort scenarios">
            <select
              value={sortOrder}
              onChange={(e) => setSortOrder(e.target.value as SortOrder)}
              className="text-xs bg-white/5 border border-white/10 rounded px-2 py-1.5 text-slate-300 focus:outline-none focus:ring-1 focus:ring-indigo-500/50 cursor-pointer"
              data-testid="tool-sort-select"
            >
              <option value="a-z">{`A \u2192 Z`}</option>
              <option value="z-a">{`Z \u2192 A`}</option>
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
            <Button variant="ghost" size="sm" onClick={onSyncTools} disabled={isSyncing} data-testid="sync-tools-button">
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
          <p className="text-sm text-slate-400">No tools or scenarios match &quot;{searchQuery}&quot;</p>
          <Button variant="ghost" size="sm" onClick={() => setSearchQuery("")} className="mt-2">
            Clear search
          </Button>
        </div>
      )}

      {/* Scenario sections */}
      {scenarioEntries.map(([scenario, tools]) => (
        <ScenarioSection
          key={scenario}
          scenario={scenario}
          tools={tools}
          status={statusByScenario.get(scenario)}
          isExpanded={expandedScenarios.has(scenario)}
          onToggleExpanded={toggleScenario}
          onToggleScenario={handleToggleScenario}
          categoryById={categoryById}
          chatId={chatId}
          isUpdating={isUpdating}
          isBatchInProgress={batchOperationInProgress.current}
          yoloMode={yoloMode}
          onToggleTool={onToggleTool}
          onSetApproval={onSetApproval}
          onResetTool={onResetTool}
          onRunTool={onRunTool}
        />
      ))}
    </div>
  );
}
