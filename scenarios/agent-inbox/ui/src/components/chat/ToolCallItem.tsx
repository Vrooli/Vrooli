import { useState, type ComponentType, type SVGProps } from "react";
import {
  Loader2, CheckCircle2, XCircle, Play,
  ShieldAlert, ExternalLink, BookOpen,
} from "lucide-react";
import * as LucideIcons from "lucide-react";
import type { ToolCall, ToolCallRecord } from "../../lib/api";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import { Tooltip } from "../ui/tooltip";
import { Button } from "../ui/button";
import { ToolCallDetailModal } from "./ToolCallDetailModal";
import { parseToolInput, type SkillAttachment } from "../../lib/tool-utils";

// Type for Lucide icon components
type IconComponent = ComponentType<SVGProps<SVGSVGElement> & { className?: string }>;

function getIconComponent(name: string): IconComponent {
  const Icon = (LucideIcons as unknown as Record<string, IconComponent>)[name];
  return Icon || BookOpen;
}

/** Skill chip for inline display in tool calls */
function ToolCallSkillChip({
  skill,
  compact = false,
}: {
  skill: SkillAttachment;
  compact?: boolean;
}) {
  // Try to get a relevant icon based on tags or default to BookOpen
  const iconName = skill.tags?.[0] || "BookOpen";
  const IconComp = getIconComponent(
    iconName.charAt(0).toUpperCase() + iconName.slice(1).replace(/-/g, "")
  );

  return (
    <Tooltip content={`Skill: ${skill.label}`}>
      <span
        className={`inline-flex items-center gap-1 rounded-full font-medium
          bg-indigo-500/20 text-indigo-300 border border-indigo-500/30
          ${compact ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-0.5 text-xs"}`}
      >
        <IconComp className={compact ? "h-2.5 w-2.5" : "h-3 w-3"} />
        <span className="max-w-[80px] truncate">{skill.label}</span>
      </span>
    </Tooltip>
  );
}

/** Get status display info for tool calls */
function getToolCallStatusDisplay(status: string) {
  switch (status) {
    case "completed":
      return {
        icon: CheckCircle2,
        color: "text-green-400",
        bgColor: "bg-green-500/10",
        label: "completed",
      };
    case "failed":
    case "error":
    case "timeout":
      return {
        icon: XCircle,
        color: "text-red-400",
        bgColor: "bg-red-500/10",
        label: status,
      };
    case "rejected":
      return {
        icon: XCircle,
        color: "text-red-400",
        bgColor: "bg-red-500/10",
        label: "rejected",
      };
    case "cancelled":
      return {
        icon: XCircle,
        color: "text-slate-400",
        bgColor: "bg-slate-500/10",
        label: "cancelled",
      };
    case "running":
      return {
        icon: Loader2,
        color: "text-amber-400",
        bgColor: "bg-amber-500/10",
        label: "running",
        animate: true,
      };
    case "pending_approval":
      return {
        icon: ShieldAlert,
        color: "text-yellow-400",
        bgColor: "bg-yellow-500/10",
        label: "pending approval",
      };
    default:
      return {
        icon: Play,
        color: "text-amber-400",
        bgColor: "bg-amber-500/10",
        label: "pending",
      };
  }
}

// Component for displaying a single tool call with skill chips and details modal
export interface ToolCallItemProps {
  toolCall: ToolCall;
  record?: ToolCallRecord;
  asyncOperation?: AsyncStatusUpdate;
  variant: "compact" | "bubble";
  onOpenAsyncDrawer?: (operation: AsyncStatusUpdate) => void;
}

export function ToolCallItem({ toolCall, record, asyncOperation, variant, onOpenAsyncDrawer }: ToolCallItemProps) {
  const [showDetailsModal, setShowDetailsModal] = useState(false);

  const status = record?.status || "pending";
  const statusDisplay = getToolCallStatusDisplay(status);
  const StatusIcon = statusDisplay.icon;
  const isFailed = ["failed", "rejected", "cancelled", "error", "timeout"].includes(status);
  const errorMessage = record?.error_message;

  const isCompact = variant === "compact";

  // Parse tool arguments to extract skills
  // IMPORTANT: Use record.arguments (enhanced with skills) when available,
  // fall back to toolCall.function.arguments (original from AI) if not.
  // The record contains the arguments that were actually sent to the tool,
  // including any injected _context_attachments (skills).
  const argumentsSource = record?.arguments || toolCall.function.arguments;
  const parsedInput = parseToolInput(argumentsSource);
  const skills = parsedInput.skills;
  const hasSkills = skills.length > 0;

  // Limit skills shown inline (show first 2-3, then "+N more")
  const maxInlineSkills = isCompact ? 2 : 3;
  const visibleSkills = skills.slice(0, maxInlineSkills);
  const hiddenSkillCount = skills.length - maxInlineSkills;

  return (
    <>
      <div
        className={`rounded-lg border transition-colors ${
          isCompact
            ? "bg-slate-100 dark:bg-slate-800/50 border-slate-200 dark:border-slate-700/50 p-2"
            : "bg-slate-800/30 border-slate-700/30 p-3"
        }`}
      >
        {/* Header row: status icon, tool name, status badge, details button */}
        <div className="flex items-center gap-2">
          <StatusIcon
            className={`h-4 w-4 ${statusDisplay.color} ${
              (statusDisplay as { animate?: boolean }).animate ? "animate-spin" : ""
            }`}
          />

          <code
            className={`font-mono ${
              isCompact
                ? "text-xs bg-slate-200 dark:bg-slate-700 px-1.5 py-0.5 rounded"
                : "text-sm bg-slate-700/50 px-2 py-0.5 rounded"
            }`}
          >
            {toolCall.function.name}
          </code>

          <span
            className={`text-xs px-1.5 py-0.5 rounded ${statusDisplay.bgColor} ${statusDisplay.color}`}
          >
            {statusDisplay.label}
          </span>

          <div className="flex-1" />

          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowDetailsModal(true)}
            className={`h-6 px-2 text-xs text-slate-400 hover:text-slate-200 ${
              isCompact ? "h-5 px-1.5 text-[10px]" : ""
            }`}
          >
            <ExternalLink className={isCompact ? "h-3 w-3 mr-0.5" : "h-3.5 w-3.5 mr-1"} />
            Details
          </Button>
        </div>

        {/* Skills row - shown if there are skills */}
        {hasSkills && (
          <div className="flex items-center gap-1.5 mt-2 flex-wrap">
            <span className={`text-slate-500 ${isCompact ? "text-[10px]" : "text-xs"}`}>
              Skills:
            </span>
            {visibleSkills.map((skill) => (
              <ToolCallSkillChip key={skill.key} skill={skill} compact={isCompact} />
            ))}
            {hiddenSkillCount > 0 && (
              <span
                className={`text-slate-500 ${isCompact ? "text-[10px]" : "text-xs"}`}
              >
                +{hiddenSkillCount} more
              </span>
            )}
          </div>
        )}

        {/* Error message - shown inline for failed status */}
        {isFailed && errorMessage && (
          <div
            className={`mt-2 text-red-400 bg-red-500/10 rounded px-2 py-1 break-words ${
              isCompact ? "text-[10px]" : "text-xs"
            }`}
          >
            {typeof errorMessage === "string"
              ? errorMessage.length > 100
                ? errorMessage.slice(0, 100) + "..."
                : errorMessage
              : "Error occurred"}
          </div>
        )}
      </div>

      {/* Details modal */}
      <ToolCallDetailModal
        open={showDetailsModal}
        onClose={() => setShowDetailsModal(false)}
        toolCall={toolCall}
        record={record}
        asyncOperation={asyncOperation}
        onOpenAsyncDrawer={asyncOperation && onOpenAsyncDrawer ? () => {
          setShowDetailsModal(false);
          onOpenAsyncDrawer(asyncOperation);
        } : undefined}
      />
    </>
  );
}
