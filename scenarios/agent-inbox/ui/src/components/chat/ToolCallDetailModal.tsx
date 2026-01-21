/**
 * ToolCallDetailModal - Modal for viewing full tool call details.
 *
 * Shows:
 * - Tool name and status
 * - Source scenario (with "Open Scenario" button)
 * - Input arguments (cleaned JSON without _context_attachments)
 * - Skills that were injected as context
 * - Output result
 * - Error message (if failed)
 */

import { useState, useEffect } from "react";
import {
  Wrench,
  CheckCircle2,
  XCircle,
  Loader2,
  ShieldAlert,
  Play,
  BookOpen,
  Package,
  ExternalLink,
} from "lucide-react";
import * as LucideIcons from "lucide-react";
import { Dialog, DialogHeader, DialogBody } from "../ui/dialog";
import { CodeBlock } from "../markdown/components/CodeBlock";
import { SkillEditorModal } from "../settings/SkillEditorModal";
import { Button } from "../ui/button";
import {
  parseToolInput,
  formatToolResult,
  isFailedStatus,
  type SkillAttachment,
} from "../../lib/tool-utils";
import { fetchScenarioInfo, type ToolCall, type ToolCallRecord, type ScenarioInfo } from "../../lib/api";
import { openScenarioViewerInNewTab } from "../scenarios/ScenarioViewer";
import type { Skill } from "../../lib/types/templates";
import type { ComponentType, SVGProps } from "react";

type IconComponent = ComponentType<SVGProps<SVGSVGElement> & { className?: string }>;

function getIconComponent(name: string): IconComponent {
  const Icon = (LucideIcons as unknown as Record<string, IconComponent>)[name];
  return Icon || BookOpen;
}

interface ToolCallDetailModalProps {
  open: boolean;
  onClose: () => void;
  toolCall: ToolCall;
  record?: ToolCallRecord;
}

/** Get status display information */
function getStatusDisplay(status: string) {
  switch (status) {
    case "completed":
      return {
        icon: CheckCircle2,
        color: "text-green-400",
        bgColor: "bg-green-500/10",
        label: "Completed",
      };
    case "failed":
    case "error":
    case "timeout":
      return {
        icon: XCircle,
        color: "text-red-400",
        bgColor: "bg-red-500/10",
        label: status === "timeout" ? "Timed Out" : "Failed",
      };
    case "rejected":
      return {
        icon: XCircle,
        color: "text-red-400",
        bgColor: "bg-red-500/10",
        label: "Rejected",
      };
    case "cancelled":
      return {
        icon: XCircle,
        color: "text-slate-400",
        bgColor: "bg-slate-500/10",
        label: "Cancelled",
      };
    case "running":
      return {
        icon: Loader2,
        color: "text-amber-400",
        bgColor: "bg-amber-500/10",
        label: "Running",
        animate: true,
      };
    case "pending_approval":
      return {
        icon: ShieldAlert,
        color: "text-yellow-400",
        bgColor: "bg-yellow-500/10",
        label: "Pending Approval",
      };
    default:
      return {
        icon: Play,
        color: "text-amber-400",
        bgColor: "bg-amber-500/10",
        label: "Pending",
      };
  }
}

/** Format tool name for display */
function formatToolName(name: string): string {
  return name
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

/** Format scenario name for display */
function formatScenarioName(name: string): string {
  return name
    .split("-")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

/** Skill chip component */
function SkillChip({
  skill,
  onClick,
}: {
  skill: SkillAttachment;
  onClick: () => void;
}) {
  // Try to get a relevant icon based on tags or default to BookOpen
  const iconName = skill.tags?.[0] || "BookOpen";
  const IconComponent = getIconComponent(
    iconName.charAt(0).toUpperCase() + iconName.slice(1).replace(/-/g, "")
  );

  return (
    <button
      onClick={onClick}
      className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium
        bg-indigo-500/20 text-indigo-300 border border-indigo-500/30
        hover:bg-indigo-500/30 hover:border-indigo-500/50 transition-colors"
    >
      <IconComponent className="h-3 w-3" />
      <span>{skill.label}</span>
    </button>
  );
}

/** Section header component */
function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-2 text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
      {children}
    </div>
  );
}

/** Source Scenario section component */
function SourceScenarioSection({
  scenarioName,
  scenarioInfo,
  isLoading,
}: {
  scenarioName: string;
  scenarioInfo: ScenarioInfo | null;
  isLoading: boolean;
}) {
  const handleOpenScenario = () => {
    openScenarioViewerInNewTab(scenarioName);
  };

  return (
    <div>
      <SectionHeader>Source Scenario</SectionHeader>
      <div className="p-3 rounded-lg bg-slate-800/50 border border-slate-700/50">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-start gap-3 min-w-0">
            <div className="p-2 rounded-lg bg-indigo-500/10 shrink-0">
              <Package className="h-4 w-4 text-indigo-400" />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-sm font-medium text-white">
                  {formatScenarioName(scenarioName)}
                </span>
                {isLoading ? (
                  <Loader2 className="h-3 w-3 text-slate-400 animate-spin" />
                ) : scenarioInfo?.version ? (
                  <span className="text-xs px-1.5 py-0.5 rounded bg-slate-700 text-slate-300">
                    v{scenarioInfo.version}
                  </span>
                ) : null}
              </div>
              {scenarioInfo?.description && (
                <p className="text-xs text-slate-400 mt-1 line-clamp-2">
                  {scenarioInfo.description}
                </p>
              )}
              {!scenarioInfo && !isLoading && (
                <p className="text-xs text-slate-500 mt-1 italic">
                  Scenario info not available
                </p>
              )}
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleOpenScenario}
            className="gap-1.5 text-indigo-400 hover:text-indigo-300 hover:bg-indigo-500/10 shrink-0"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">Open Scenario</span>
          </Button>
        </div>
      </div>
    </div>
  );
}

export function ToolCallDetailModal({
  open,
  onClose,
  toolCall,
  record,
}: ToolCallDetailModalProps) {
  const [previewSkill, setPreviewSkill] = useState<SkillAttachment | null>(null);
  const [scenarioInfo, setScenarioInfo] = useState<ScenarioInfo | null>(null);
  const [scenarioLoading, setScenarioLoading] = useState(false);

  const status = record?.status || "pending";
  const statusDisplay = getStatusDisplay(status);
  const StatusIcon = statusDisplay.icon;
  const scenarioName = record?.scenario_name;

  // Fetch scenario info when modal opens and we have a scenario name
  useEffect(() => {
    if (!open || !scenarioName) {
      setScenarioInfo(null);
      return;
    }

    setScenarioLoading(true);
    fetchScenarioInfo(scenarioName)
      .then((info) => {
        setScenarioInfo(info);
      })
      .catch((err) => {
        console.warn("Failed to fetch scenario info:", err);
        setScenarioInfo(null);
      })
      .finally(() => {
        setScenarioLoading(false);
      });
  }, [open, scenarioName]);

  // Parse the arguments to extract skills
  // Use record.arguments (enhanced with skills) when available
  const argumentsSource = record?.arguments || toolCall.function.arguments;
  const parsedInput = parseToolInput(argumentsSource);
  const hasArguments = Object.keys(parsedInput.arguments).length > 0;
  const hasSkills = parsedInput.skills.length > 0;

  // Format the result
  const result = record?.result;
  const hasResult = status === "completed" && result;
  const formattedResult = hasResult ? formatToolResult(result) : null;

  // Error handling
  const isFailed = isFailedStatus(status);
  const errorMessage = record?.error_message;

  // Convert SkillAttachment to Skill for the modal
  const skillAttachmentToSkill = (attachment: SkillAttachment): Skill => ({
    id: attachment.key,
    name: attachment.label,
    description: "",
    content: attachment.content,
    tags: attachment.tags,
  });

  return (
    <>
      <Dialog open={open} onClose={onClose} className="max-w-2xl">
        <DialogHeader onClose={onClose}>
          <span className="flex items-center gap-2">
            <Wrench className="h-5 w-5 text-slate-400" />
            Tool Call Details
          </span>
        </DialogHeader>

        <DialogBody className="space-y-4">
          {/* Tool name and status header */}
          <div className="flex items-center justify-between p-3 rounded-lg bg-slate-800/50 border border-slate-700/50">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-slate-700/50">
                <Wrench className="h-5 w-5 text-slate-400" />
              </div>
              <div>
                <div className="text-sm font-medium text-white">
                  {formatToolName(toolCall.function.name)}
                </div>
                <div className="text-xs text-slate-500 font-mono">
                  {toolCall.function.name}
                </div>
              </div>
            </div>

            <div
              className={`flex items-center gap-2 px-3 py-1.5 rounded-full ${statusDisplay.bgColor}`}
            >
              <StatusIcon
                className={`h-4 w-4 ${statusDisplay.color} ${
                  (statusDisplay as { animate?: boolean }).animate ? "animate-spin" : ""
                }`}
              />
              <span className={`text-sm font-medium ${statusDisplay.color}`}>
                {statusDisplay.label}
              </span>
            </div>
          </div>

          {/* Source Scenario Section */}
          {scenarioName && (
            <SourceScenarioSection
              scenarioName={scenarioName}
              scenarioInfo={scenarioInfo}
              isLoading={scenarioLoading}
            />
          )}

          {/* Input Section */}
          <div>
            <SectionHeader>Input</SectionHeader>

            {/* Arguments */}
            {hasArguments && (
              <div className="mb-3">
                <div className="text-xs text-slate-500 mb-1">Arguments</div>
                <div className="max-h-48 overflow-y-auto rounded-lg">
                  <CodeBlock code={parsedInput.cleanedArgumentsJson} language="json" />
                </div>
              </div>
            )}

            {!hasArguments && !hasSkills && (
              <div className="text-sm text-slate-500 italic">No input provided</div>
            )}

            {/* Skills */}
            {hasSkills && (
              <div>
                <div className="text-xs text-slate-500 mb-2">Skills</div>
                <div className="flex flex-wrap gap-2">
                  {parsedInput.skills.map((skill) => (
                    <SkillChip
                      key={skill.key}
                      skill={skill}
                      onClick={() => setPreviewSkill(skill)}
                    />
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Output Section */}
          <div>
            <SectionHeader>Output</SectionHeader>

            {hasResult && formattedResult && (
              <div className="max-h-64 overflow-y-auto rounded-lg">
                <CodeBlock code={formattedResult} language="json" />
              </div>
            )}

            {!hasResult && !isFailed && (
              <div className="text-sm text-slate-500 italic">
                {status === "running" || status === "pending" || status === "pending_approval"
                  ? "Waiting for result..."
                  : "No output"}
              </div>
            )}

            {isFailed && errorMessage && (
              <div className="mt-2">
                <div className="text-xs text-slate-500 mb-1">Error</div>
                <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-sm text-red-300 break-words">
                  {typeof errorMessage === "string"
                    ? errorMessage
                    : JSON.stringify(errorMessage, null, 2)}
                </div>
              </div>
            )}

            {isFailed && !errorMessage && (
              <div className="text-sm text-red-400 italic">Tool execution failed</div>
            )}
          </div>
        </DialogBody>
      </Dialog>

      {/* Skill preview modal */}
      <SkillEditorModal
        open={!!previewSkill}
        onClose={() => setPreviewSkill(null)}
        skill={previewSkill ? skillAttachmentToSkill(previewSkill) : undefined}
        readOnly={true}
      />
    </>
  );
}
