import {
  ChevronDown,
  ChevronRight,
  AlertCircle,
  CheckCircle2,
  XCircle,
} from "lucide-react";
import { Tooltip } from "../ui/tooltip";
import { Switch } from "../ui/switch";
import { ToolItem } from "./ToolItem";
import type { EffectiveTool, ScenarioStatus, ToolCategory, ApprovalOverride } from "../../lib/api";

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

interface ScenarioSectionProps {
  scenario: string;
  tools: EffectiveTool[];
  status?: ScenarioStatus;
  isExpanded: boolean;
  onToggleExpanded: (scenario: string) => void;
  onToggleScenario: (scenario: string, enableAll: boolean) => void;
  categoryById: Map<string, ToolCategory>;
  chatId?: string;
  isUpdating?: boolean;
  isBatchInProgress: boolean;
  yoloMode?: boolean;
  onToggleTool: (scenario: string, toolName: string, enabled: boolean) => void | Promise<void>;
  onSetApproval?: (scenario: string, toolName: string, override: ApprovalOverride) => void;
  onResetTool?: (scenario: string, toolName: string) => void;
  onRunTool?: (tool: EffectiveTool) => void;
}

export function ScenarioSection({
  scenario,
  tools,
  status,
  isExpanded,
  onToggleExpanded,
  onToggleScenario,
  categoryById,
  chatId,
  isUpdating,
  isBatchInProgress,
  yoloMode,
  onToggleTool,
  onSetApproval,
  onResetTool,
  onRunTool,
}: ScenarioSectionProps) {
  const scenarioEnabledCount = tools.filter((t) => t.enabled).length;
  const allEnabled = scenarioEnabledCount === tools.length;
  const someEnabled = scenarioEnabledCount > 0 && scenarioEnabledCount < tools.length;

  return (
    <div
      className="rounded-lg border border-white/10 bg-white/5 overflow-hidden"
      data-testid={`scenario-section-${scenario}`}
    >
      {/* Scenario header */}
      <div className="flex items-center gap-3 p-3 hover:bg-white/5 transition-colors">
        {/* Expand/collapse button */}
        <button
          onClick={() => onToggleExpanded(scenario)}
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
          <Switch
            checked={allEnabled}
            onCheckedChange={(checked) => {
              void onToggleScenario(scenario, checked);
            }}
            disabled={isUpdating || isBatchInProgress}
            className={!allEnabled && someEnabled ? "!bg-indigo-500/50" : undefined}
            data-testid={`scenario-toggle-all-${scenario}`}
            aria-label={`${scenario} toggle all tools`}
          />
        </Tooltip>
      </div>

      {/* Tools list */}
      {isExpanded && (
        <div className="border-t border-white/10">
          {tools.map((effectiveTool) => {
            const category = effectiveTool.tool.category
              ? categoryById.get(effectiveTool.tool.category) ?? null
              : null;

            return (
              <ToolItem
                key={effectiveTool.tool.name}
                effectiveTool={effectiveTool}
                scenario={scenario}
                category={category}
                chatId={chatId}
                isUpdating={isUpdating}
                yoloMode={yoloMode}
                onToggleTool={onToggleTool}
                onSetApproval={onSetApproval}
                onResetTool={onResetTool}
                onRunTool={onRunTool}
              />
            );
          })}
        </div>
      )}
    </div>
  );
}
