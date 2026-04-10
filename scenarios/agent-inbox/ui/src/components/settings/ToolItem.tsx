import {
  Clock,
  DollarSign,
  Shield,
  ShieldOff,
  Play,
  RotateCcw,
} from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { Switch } from "../ui/switch";
import type { EffectiveTool, ToolCategory, ApprovalOverride } from "../../lib/api";

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

interface ToolItemProps {
  effectiveTool: EffectiveTool;
  scenario: string;
  category: ToolCategory | null;
  chatId?: string;
  isUpdating?: boolean;
  yoloMode?: boolean;
  onToggleTool: (scenario: string, toolName: string, enabled: boolean) => void | Promise<void>;
  onSetApproval?: (scenario: string, toolName: string, override: ApprovalOverride) => void;
  onResetTool?: (scenario: string, toolName: string) => void;
  onRunTool?: (tool: EffectiveTool) => void;
}

export function ToolItem({
  effectiveTool,
  scenario,
  category,
  chatId,
  isUpdating,
  yoloMode,
  onToggleTool,
  onSetApproval,
  onResetTool,
  onRunTool,
}: ToolItemProps) {
  const { tool, enabled, source, requires_approval, approval_override, approval_source } = effectiveTool;
  const hasOverride = chatId && source === "chat";
  const hasApprovalOverride = chatId && approval_source === "chat";

  // Determine current approval state for display
  const currentApprovalOverride = approval_override ?? "";
  const defaultRequiresApproval = tool.metadata.requires_approval;

  return (
    <div
      className="flex items-start gap-3 p-3 border-b border-white/5 last:border-b-0 hover:bg-white/5 transition-colors"
      data-testid={`tool-item-${scenario}-${tool.name}`}
    >
      {/* Toggle switch */}
      <Switch
        checked={enabled}
        onCheckedChange={(checked) => {
          void onToggleTool(scenario, tool.name, checked);
        }}
        disabled={isUpdating}
        className="mt-0.5"
        data-testid={`tool-toggle-${scenario}-${tool.name}`}
        aria-label={`${scenario} ${tool.name} toggle`}
      />

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
}
