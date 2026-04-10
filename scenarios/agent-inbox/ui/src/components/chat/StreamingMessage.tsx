import {
  Loader2, Bot, Wrench, CheckCircle2, XCircle, Play,
  ShieldAlert,
} from "lucide-react";
import { resolveAttachmentUrl } from "../../lib/api";
import type { ActiveToolCall, PendingApproval } from "../../hooks/useCompletion";
import { PendingApprovalCard } from "./PendingApprovalCard";
import { MarkdownRenderer } from "../markdown";

interface ActiveToolCallsDisplayProps {
  activeToolCalls: ActiveToolCall[];
  isCompact: boolean;
}

export function ActiveToolCallsDisplay({ activeToolCalls, isCompact }: ActiveToolCallsDisplayProps) {
  if (activeToolCalls.length === 0) return null;

  if (isCompact) {
    return (
      <div className="border-l-2 border-l-amber-500 pl-3 py-1" data-testid="active-tool-calls">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-xs font-medium text-amber-400">Tool</span>
        </div>
        <div className="space-y-1">
          {activeToolCalls.map((tc) => (
            <div key={tc.id} className="flex items-center gap-2 text-sm">
              {tc.status === "running" ? (
                <>
                  <Play className="h-3 w-3 text-amber-500 dark:text-amber-400 animate-pulse" />
                  <span className="text-slate-700 dark:text-slate-300">Running <code className="bg-slate-200 dark:bg-slate-700 px-1 py-0.5 rounded text-xs">{tc.name}</code>...</span>
                </>
              ) : tc.status === "completed" ? (
                <>
                  <CheckCircle2 className="h-3 w-3 text-green-500 dark:text-green-400" />
                  <span className="text-green-600 dark:text-green-400">{tc.name} completed</span>
                </>
              ) : (
                <>
                  <XCircle className="h-3 w-3 text-red-500 dark:text-red-400" />
                  <span className="text-red-600 dark:text-red-400">{tc.name} failed</span>
                </>
              )}
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="flex justify-start" data-testid="active-tool-calls">
      <div className="flex gap-3 max-w-[85%] min-w-0">
        <div className="w-8 h-8 rounded-full bg-amber-500/20 flex items-center justify-center shrink-0">
          <Wrench className="h-4 w-4 text-amber-400" />
        </div>
        <div className="bg-slate-100 dark:bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-700 dark:text-slate-200 space-y-2 min-w-0">
          {activeToolCalls.map((tc) => (
            <div key={tc.id} className="flex items-center gap-2">
              {tc.status === "running" ? (
                <>
                  <Play className="h-4 w-4 text-amber-500 dark:text-amber-400 animate-pulse" />
                  <span className="text-sm">Running <code className="bg-slate-200 dark:bg-slate-700 px-1.5 py-0.5 rounded">{tc.name}</code>...</span>
                </>
              ) : tc.status === "completed" ? (
                <>
                  <CheckCircle2 className="h-4 w-4 text-green-500 dark:text-green-400" />
                  <span className="text-sm text-green-600 dark:text-green-400">{tc.name} completed</span>
                </>
              ) : (
                <>
                  <XCircle className="h-4 w-4 text-red-500 dark:text-red-400" />
                  <span className="text-sm text-red-600 dark:text-red-400">{tc.name} failed</span>
                </>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

interface PendingApprovalsDisplayProps {
  pendingApprovals: PendingApproval[];
  awaitingApprovals: boolean;
  isProcessingApproval: boolean;
  onApproveTool: (toolCallId: string) => void;
  onRejectTool: (toolCallId: string, reason?: string) => void;
}

export function PendingApprovalsDisplay({
  pendingApprovals,
  awaitingApprovals,
  isProcessingApproval,
  onApproveTool,
  onRejectTool,
}: PendingApprovalsDisplayProps) {
  if (pendingApprovals.length === 0) return null;

  return (
    <div className="space-y-2" data-testid="pending-approvals">
      {awaitingApprovals && (
        <div className="flex items-center gap-2 py-2">
          <ShieldAlert className="h-4 w-4 text-yellow-400" />
          <span className="text-sm text-yellow-400">Awaiting approval to continue</span>
        </div>
      )}
      {pendingApprovals.map((approval) => (
        <PendingApprovalCard
          key={approval.id}
          approval={approval}
          onApprove={onApproveTool}
          onReject={onRejectTool}
          isProcessing={isProcessingApproval}
        />
      ))}
    </div>
  );
}

interface StreamingMessageDisplayProps {
  streamingContent: string;
  generatedImages: string[];
  isCompact: boolean;
}

export function StreamingMessageDisplay({ streamingContent, generatedImages, isCompact }: StreamingMessageDisplayProps) {
  if (isCompact) {
    return (
      <div className="border-l-2 border-l-emerald-500 pl-3 py-1" data-testid="streaming-message">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-xs font-medium text-emerald-400">Assistant</span>
          {!streamingContent && generatedImages.length === 0 && (
            <Loader2 className="h-3 w-3 animate-spin text-emerald-400" />
          )}
        </div>
        {generatedImages.length > 0 && (
          <div className="flex flex-wrap gap-2 mb-2">
            {generatedImages.map((imgUrl, idx) => (
              <img
                key={idx}
                src={resolveAttachmentUrl(imgUrl)}
                alt={`Generated image ${idx + 1}`}
                className="max-w-[150px] max-h-[150px] rounded-lg object-contain border border-slate-200 dark:border-slate-700"
              />
            ))}
          </div>
        )}
        {streamingContent ? (
          <div className="text-sm text-slate-700 dark:text-slate-200">
            <MarkdownRenderer content={streamingContent} isStreaming />
          </div>
        ) : generatedImages.length === 0 ? (
          <span className="text-sm text-slate-500 dark:text-slate-400">Thinking...</span>
        ) : null}
      </div>
    );
  }

  return (
    <div className="flex justify-start" data-testid="streaming-message">
      <div className="flex gap-3 max-w-[85%] min-w-0">
        <div className="w-8 h-8 rounded-full bg-indigo-500/20 flex items-center justify-center shrink-0">
          <Bot className="h-4 w-4 text-indigo-400" />
        </div>
        <div className="bg-slate-100 dark:bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-700 dark:text-slate-200 min-w-0">
          {generatedImages.length > 0 && (
            <div className="flex flex-wrap gap-2 mb-3">
              {generatedImages.map((imgUrl, idx) => (
                <img
                  key={idx}
                  src={resolveAttachmentUrl(imgUrl)}
                  alt={`Generated image ${idx + 1}`}
                  className="max-w-[300px] max-h-[300px] rounded-lg object-contain border border-slate-200 dark:border-slate-700"
                />
              ))}
            </div>
          )}
          {streamingContent ? (
            <MarkdownRenderer content={streamingContent} isStreaming />
          ) : generatedImages.length === 0 ? (
            <div className="flex items-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin text-indigo-500 dark:text-indigo-400" />
              <span className="text-sm text-slate-500 dark:text-slate-400">Thinking...</span>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
