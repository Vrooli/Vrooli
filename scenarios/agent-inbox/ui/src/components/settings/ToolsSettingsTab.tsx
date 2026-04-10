import { Zap } from "lucide-react";
import { ToolConfiguration } from "./ToolConfiguration";
import { SettingsSection, SettingsSwitchRow } from "./SettingsControls";
import type { EffectiveTool, ScenarioStatus, ToolCategory, ApprovalOverride } from "../../lib/api";

interface ToolsSettingsTabProps {
  yoloMode: boolean;
  isLoadingYoloMode: boolean;
  isUpdatingYoloMode: boolean;
  onYoloModeToggle: (checked: boolean) => void;
  toolsByScenario: Map<string, EffectiveTool[]>;
  categories: ToolCategory[];
  scenarioStatuses?: ScenarioStatus[];
  isLoadingTools: boolean;
  isSyncingTools: boolean;
  isUpdatingTools: boolean;
  toolsError?: string;
  onToggleTool: (scenario: string, toolName: string, enabled: boolean) => void | Promise<void>;
  onSetApproval: (scenario: string, toolName: string, override: ApprovalOverride) => void;
  onSyncTools: () => void;
  onRunTool: (tool: EffectiveTool) => void;
  enabledCount: number;
  totalCount: number;
}

export function ToolsSettingsTab({
  yoloMode,
  isLoadingYoloMode,
  isUpdatingYoloMode,
  onYoloModeToggle,
  toolsByScenario,
  categories,
  scenarioStatuses,
  isLoadingTools,
  isSyncingTools,
  isUpdatingTools,
  toolsError,
  onToggleTool,
  onSetApproval,
  onSyncTools,
  onRunTool,
  enabledCount,
  totalCount,
}: ToolsSettingsTabProps) {
  return (
    <div className="space-y-4">
      <SettingsSection title="YOLO Mode">
        <SettingsSwitchRow
          title="Execute Without Approval"
          description="Execute all tools without asking for approval"
          checked={yoloMode}
          onCheckedChange={onYoloModeToggle}
          disabled={isLoadingYoloMode || isUpdatingYoloMode}
          tone="yellow"
          testId="yolo-mode-toggle"
        />
        {yoloMode && (
          <div className="mt-3 p-3 rounded-lg bg-yellow-500/10 border border-yellow-500/20">
            <p className="text-xs text-yellow-400 flex items-center gap-2">
              <Zap className="h-3.5 w-3.5" />
              Tools will execute automatically without confirmation
            </p>
          </div>
        )}
      </SettingsSection>

      <ToolConfiguration
        toolsByScenario={toolsByScenario}
        categories={categories}
        scenarioStatuses={scenarioStatuses}
        isLoading={isLoadingTools}
        isSyncing={isSyncingTools}
        isUpdating={isUpdatingTools}
        error={toolsError}
        onToggleTool={onToggleTool}
        onSetApproval={onSetApproval}
        onSyncTools={onSyncTools}
        yoloMode={yoloMode}
        onRunTool={onRunTool}
        enabledCount={enabledCount}
        totalCount={totalCount}
      />
    </div>
  );
}
