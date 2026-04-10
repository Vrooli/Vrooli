/**
 * Tool selection component for the template editor.
 * Displays available tools grouped by scenario with checkboxes.
 */

import { ChevronDown, ChevronRight, Wrench } from "lucide-react";
import type { EffectiveTool } from "@/lib/api";

interface SuggestedToolsEditorProps {
  selectedToolIds: string[];
  toolsByScenario: Map<string, EffectiveTool[]>;
  isLoadingTools: boolean;
  expandedScenarios: Set<string>;
  selectedCountByScenario: Map<string, number>;
  onToggleToolSelection: (toolId: string) => void;
  onToggleScenario: (scenario: string) => void;
}

export function SuggestedToolsEditor({
  selectedToolIds,
  toolsByScenario,
  isLoadingTools,
  expandedScenarios,
  selectedCountByScenario,
  onToggleToolSelection,
  onToggleScenario,
}: SuggestedToolsEditorProps) {
  return (
    <div>
      <div className="flex items-center gap-2 mb-2">
        <Wrench className="h-4 w-4 text-slate-400" />
        <label className="text-sm font-medium text-slate-300">
          Suggested Tools
        </label>
        {selectedToolIds.length > 0 && (
          <span className="text-xs bg-indigo-600/30 text-indigo-300 px-2 py-0.5 rounded-full">
            {selectedToolIds.length} selected
          </span>
        )}
      </div>
      <p className="text-xs text-slate-500 mb-3">
        These tools will be auto-enabled when this template is selected
      </p>

      {isLoadingTools ? (
        <p className="text-sm text-slate-500 italic">Loading tools...</p>
      ) : toolsByScenario.size === 0 ? (
        <p className="text-sm text-slate-500 italic">No tools available</p>
      ) : (
        <div className="space-y-2">
          {Array.from(toolsByScenario.entries()).map(([scenario, tools]) => {
            const isExpanded = expandedScenarios.has(scenario);
            const selectedCount = selectedCountByScenario.get(scenario) || 0;

            return (
              <div
                key={scenario}
                className="bg-slate-800/50 border border-white/5 rounded-lg overflow-hidden"
              >
                <button
                  type="button"
                  onClick={() => onToggleScenario(scenario)}
                  className="w-full flex items-center justify-between px-3 py-2 hover:bg-white/5 transition-colors"
                >
                  <div className="flex items-center gap-2">
                    {isExpanded ? (
                      <ChevronDown className="h-4 w-4 text-slate-400" />
                    ) : (
                      <ChevronRight className="h-4 w-4 text-slate-400" />
                    )}
                    <span className="text-sm text-slate-300">{scenario}</span>
                  </div>
                  <span className="text-xs text-slate-500">
                    {selectedCount}/{tools.length} selected
                  </span>
                </button>

                {isExpanded && (
                  <div className="border-t border-white/5 px-3 py-2 space-y-1.5">
                    {tools.map((tool) => {
                      const toolId = `${scenario}:${tool.tool.name}`;
                      const isSelected = selectedToolIds.includes(toolId);

                      return (
                        <label
                          key={tool.tool.name}
                          className="flex items-start gap-2 cursor-pointer group"
                        >
                          <input
                            type="checkbox"
                            checked={isSelected}
                            onChange={() => onToggleToolSelection(toolId)}
                            className="mt-0.5 rounded bg-slate-700 border-white/20 text-indigo-500 focus:ring-indigo-500"
                          />
                          <div className="flex-1 min-w-0">
                            <span className="text-sm text-white group-hover:text-indigo-300 transition-colors">
                              {tool.tool.name}
                            </span>
                            {tool.tool.description && (
                              <p className="text-xs text-slate-500 truncate">
                                {tool.tool.description}
                              </p>
                            )}
                          </div>
                        </label>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
