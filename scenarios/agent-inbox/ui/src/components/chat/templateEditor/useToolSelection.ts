/**
 * Hook for managing tool selection state in the template editor.
 * Handles tool ID normalization, scenario expansion, and selection counts.
 */

import { useCallback, useMemo, useState } from "react";
import { useTools } from "@/hooks/useTools";
import type { EffectiveTool } from "@/lib/api";

export function useToolSelection() {
  const [selectedToolIds, setSelectedToolIds] = useState<string[]>([]);
  const [initialToolIds, setInitialToolIds] = useState<string[]>([]);
  const [expandedScenarios, setExpandedScenarios] = useState<Set<string>>(new Set());

  const { toolsByScenario, isLoading: isLoadingTools } = useTools();

  const toggleToolSelection = useCallback((toolId: string) => {
    setSelectedToolIds((prev) =>
      prev.includes(toolId) ? prev.filter((id) => id !== toolId) : [...prev, toolId]
    );
  }, []);

  const toggleScenario = useCallback((scenario: string) => {
    setExpandedScenarios((prev) => {
      const next = new Set(prev);
      if (next.has(scenario)) { next.delete(scenario); } else { next.add(scenario); }
      return next;
    });
  }, []);

  const selectedCountByScenario = useMemo(() => {
    const counts = new Map<string, number>();
    for (const toolId of selectedToolIds) {
      const scenario = toolId.split(":")[0] ?? toolId;
      counts.set(scenario, (counts.get(scenario) ?? 0) + 1);
    }
    return counts;
  }, [selectedToolIds]);

  // Normalize tool IDs from old format (just tool name) to new format (scenario:toolName)
  const normalizeToolIds = useCallback((toolIds: string[], availableTools: Map<string, EffectiveTool[]>): string[] => {
    if (availableTools.size === 0) return toolIds;
    return toolIds.map(toolId => {
      if (toolId.includes(":")) return toolId;
      for (const [scenario, tools] of availableTools.entries()) {
        const found = tools.find(t => t.tool.name === toolId);
        if (found) return `${scenario}:${toolId}`;
      }
      return toolId;
    });
  }, []);

  const resetScenarios = useCallback(() => {
    setExpandedScenarios(new Set());
  }, []);

  return {
    selectedToolIds,
    setSelectedToolIds,
    initialToolIds,
    setInitialToolIds,
    expandedScenarios,
    toolsByScenario,
    isLoadingTools,
    toggleToolSelection,
    toggleScenario,
    selectedCountByScenario,
    normalizeToolIds,
    resetScenarios,
  };
}
