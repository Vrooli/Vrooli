/**
 * ToolCallDetailModal - Modal for viewing full tool call details.
 *
 * Shows tool name, status, source scenario, input arguments,
 * injected skills, output result, and error messages.
 */

import { useState } from "react";
import { Wrench, Zap, ExternalLink, Loader2 } from "lucide-react";
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
import type { ToolCall, ToolCallRecord } from "../../lib/api";
import type { Skill } from "../../lib/types/templates";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import {
  AsyncProgressFromOperation,
  getAsyncStatusDisplay,
} from "./AsyncProgressDisplay";
import {
  getStatusDisplay,
  formatToolName,
  SkillChip,
  SectionHeader,
  SourceScenarioSection,
} from "./ToolCallDetailHelpers";

interface ToolCallDetailModalProps {
  open: boolean;
  onClose: () => void;
  toolCall: ToolCall;
  record?: ToolCallRecord;
  asyncOperation?: AsyncStatusUpdate;
  onOpenAsyncDrawer?: () => void;
}

export function ToolCallDetailModal({
  open,
  onClose,
  toolCall,
  record,
  asyncOperation,
  onOpenAsyncDrawer,
}: ToolCallDetailModalProps) {
  const [previewSkill, setPreviewSkill] = useState<SkillAttachment | null>(null);

  const status = record?.status || "pending";
  const statusDisplay = getStatusDisplay(status);
  const StatusIcon = statusDisplay.icon;
  const scenarioName = record?.scenario_name;

  const isAsyncTool = !!record?.external_run_id || !!asyncOperation;
  const asyncStatusDisplay = asyncOperation
    ? getAsyncStatusDisplay(asyncOperation.status, asyncOperation.is_terminal)
    : null;

  const argumentsSource = record?.arguments || toolCall.function.arguments;
  const parsedInput = parseToolInput(argumentsSource);
  const hasArguments = Object.keys(parsedInput.arguments).length > 0;
  const hasSkills = parsedInput.skills.length > 0;

  const result = record?.result;
  const hasResult = status === "completed" && result;
  const isFailed = isFailedStatus(status);
  const errorMessage = record?.error_message;

  const skillAttachmentToSkill = (attachment: SkillAttachment): Skill => ({
    id: attachment.key, name: attachment.label, description: "", content: attachment.content, tags: attachment.tags,
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
              <div className="p-2 rounded-lg bg-slate-700/50"><Wrench className="h-5 w-5 text-slate-400" /></div>
              <div>
                <div className="text-sm font-medium text-white">{formatToolName(toolCall.function.name)}</div>
                <div className="text-xs text-slate-500 font-mono">{toolCall.function.name}</div>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {isAsyncTool && (
                <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-indigo-500/20 border border-indigo-500/30">
                  <Zap className="h-3.5 w-3.5 text-indigo-400" /><span className="text-xs font-medium text-indigo-400">Async</span>
                </div>
              )}
              <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full ${statusDisplay.bgColor}`}>
                <StatusIcon className={`h-4 w-4 ${statusDisplay.color} ${(statusDisplay as { animate?: boolean }).animate ? "animate-spin" : ""}`} />
                <span className={`text-sm font-medium ${statusDisplay.color}`}>{statusDisplay.label}</span>
              </div>
            </div>
          </div>

          {scenarioName && <SourceScenarioSection scenarioName={scenarioName} scenarioInfo={null} isLoading={false} />}

          {isAsyncTool && asyncOperation && (
            <div>
              <SectionHeader><Zap className="h-3.5 w-3.5 text-indigo-400" />Async Operation Status</SectionHeader>
              <div className={`p-3 rounded-lg border ${asyncStatusDisplay?.borderColor || "border-slate-700/50"} ${asyncStatusDisplay?.bgColor || "bg-slate-800/50"}`}>
                <AsyncProgressFromOperation operation={asyncOperation} variant="modal" />
                {asyncOperation.phase && (
                  <div className="mt-2 flex items-center gap-2">
                    <span className="text-xs text-slate-500">Current Phase:</span>
                    <span className="text-sm text-white font-medium">{asyncOperation.phase}</span>
                  </div>
                )}
                {onOpenAsyncDrawer && (
                  <div className="mt-3 pt-3 border-t border-slate-700/50">
                    <Button variant="ghost" size="sm" onClick={onOpenAsyncDrawer} className="text-indigo-400 hover:text-indigo-300 hover:bg-indigo-500/10">
                      <ExternalLink className="h-3.5 w-3.5 mr-1.5" />View in Async Panel
                    </Button>
                  </div>
                )}
              </div>
            </div>
          )}

          {isAsyncTool && !asyncOperation && (
            <div>
              <SectionHeader><Zap className="h-3.5 w-3.5 text-indigo-400" />Async Operation</SectionHeader>
              <div className="p-3 rounded-lg border border-slate-700/50 bg-slate-800/50">
                <p className="text-sm text-slate-400">This is an async tool call. Real-time status updates are available in the Async Status panel.</p>
                {record?.external_run_id && <p className="text-xs text-slate-500 mt-2 font-mono">Run ID: {record.external_run_id}</p>}
              </div>
            </div>
          )}

          <div>
            <SectionHeader>Input</SectionHeader>
            {hasArguments && (
              <div className="mb-3">
                <div className="text-xs text-slate-500 mb-1">Arguments</div>
                <div className="max-h-48 overflow-y-auto rounded-lg"><CodeBlock code={parsedInput.cleanedArgumentsJson} language="json" /></div>
              </div>
            )}
            {!hasArguments && !hasSkills && <div className="text-sm text-slate-500 italic">No input provided</div>}
            {hasSkills && (
              <div>
                <div className="text-xs text-slate-500 mb-2">Skills</div>
                <div className="flex flex-wrap gap-2">
                  {parsedInput.skills.map((skill) => <SkillChip key={skill.key} skill={skill} onClick={() => setPreviewSkill(skill)} />)}
                </div>
              </div>
            )}
          </div>

          <div>
            <SectionHeader>Output</SectionHeader>
            {(() => {
              const asyncResult = asyncOperation?.result;
              const asyncError = asyncOperation?.error;
              const displayResult = asyncResult ?? result;
              const displayFormattedResult = displayResult ? formatToolResult(displayResult) : null;
              const asyncTerminal = asyncOperation?.is_terminal ?? false;
              const asyncFailed = asyncOperation && (asyncOperation.status === "failed" || asyncOperation.status === "error" || asyncOperation.status === "timeout");

              if (isAsyncTool && asyncOperation && !asyncTerminal) {
                return (
                  <div className="p-3 rounded-lg bg-yellow-500/10 border border-yellow-500/20">
                    <div className="flex items-center gap-2"><Loader2 className="h-4 w-4 text-yellow-400 animate-spin" /><span className="text-sm text-yellow-400">Operation in progress...</span></div>
                    {asyncOperation.message && <p className="text-xs text-slate-400 mt-2">{asyncOperation.message}</p>}
                  </div>
                );
              }
              if (displayFormattedResult && (hasResult || (asyncTerminal && !asyncFailed))) {
                return <div className="max-h-64 overflow-y-auto rounded-lg"><CodeBlock code={displayFormattedResult} language="json" /></div>;
              }
              if (asyncFailed && asyncError) {
                return (<div className="mt-2"><div className="text-xs text-slate-500 mb-1">Error</div><div className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-sm text-red-300 break-words">{asyncError}</div></div>);
              }
              if (isFailed && errorMessage) {
                return (<div className="mt-2"><div className="text-xs text-slate-500 mb-1">Error</div><div className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-sm text-red-300 break-words">{typeof errorMessage === "string" ? errorMessage : JSON.stringify(errorMessage, null, 2)}</div></div>);
              }
              if (isFailed || asyncFailed) return <div className="text-sm text-red-400 italic">Tool execution failed</div>;
              return <div className="text-sm text-slate-500 italic">{status === "running" || status === "pending" || status === "pending_approval" ? "Waiting for result..." : "No output"}</div>;
            })()}
          </div>
        </DialogBody>
      </Dialog>

      <SkillEditorModal
        open={!!previewSkill}
        onClose={() => setPreviewSkill(null)}
        skill={previewSkill ? skillAttachmentToSkill(previewSkill) : undefined}
        readOnly={true}
      />
    </>
  );
}
