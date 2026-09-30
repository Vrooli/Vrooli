/**
 * ExecutionOverviewTab — Metadata, failure reason, post-run status,
 * and action buttons for a single execution.
 */

import {
  ClipboardCheck,
  ExternalLink,
  Loader2,
  PauseCircle,
  PlayCircle,
  RefreshCw,
  RotateCcw,
  XCircle,
} from "lucide-react";
import { MarkdownRenderer } from "@vrooli/react-component-library/markdown-renderer/0";
import { Button } from "../ui/button";
import { EntityLink } from "../ui/entity-link";
import { IdentityBadge } from "../ui/identity-badge";
import { PostRunStatusBadge } from "./post-run-status-badge";
import { DetailSection } from "../detail/DetailSection";
import { formatRelativeTime, canRunPostRunChecks } from "../../lib";
import { ENTITY_TYPE_ICONS, formatExecutionMode } from "../../types";
import { selectors } from "../../consts/selectors";
import { buildAgentRunUrl } from "../../services/external-links";
import type { ExecutionRecord } from "../../types";

export interface ExecutionOverviewTabProps {
  execution: ExecutionRecord;
  isActive: boolean;
  isTerminal: boolean;
  actionBusy: boolean;
  postRunBadgeExecution: ExecutionRecord | null;
  agentManagerUiUrl: string | null;
  onFollowUp: () => void;
  onCancel: () => void;
  onRetry: () => void;
  onRunPostRunChecks: () => void;
  onHaltContinuation?: () => void;
  onResumeContinuation?: () => void;
}

export function ExecutionOverviewTab({
  execution,
  isActive,
  isTerminal,
  actionBusy,
  postRunBadgeExecution,
  agentManagerUiUrl,
  onFollowUp,
  onCancel,
  onRetry,
  onRunPostRunChecks,
  onHaltContinuation = () => undefined,
  onResumeContinuation = () => undefined,
}: ExecutionOverviewTabProps) {
  const showRunChecks = canRunPostRunChecks(execution);
  const runChecksLabel = execution.finalization ? "Rerun Post-Run Checks" : "Run Post-Run Checks";

  return (
    <div className="space-y-0" data-testid={selectors.executionDetails.page}>
      <DetailSection title="Overview" icon={ENTITY_TYPE_ICONS.execution} hideDivider>
        <div className="space-y-3">
          {/* Metadata grid */}
          <div
            className="grid grid-cols-2 gap-3 text-sm"
            data-testid={selectors.executionDetails.overviewMetadata}
          >
            <div>
              <p className="text-xs text-slate-500 uppercase tracking-wider">Backlog</p>
              <EntityLink
                entityType="backlog"
                kind={execution.backlogKind}
                name={execution.backlogName}
                label={`${execution.backlogKind}/${execution.backlogName}`}
              />
            </div>
            {execution.startedBy && (
              <div>
                <p className="text-xs text-slate-500 uppercase tracking-wider">Started by</p>
                <IdentityBadge value={execution.startedBy} agentManagerUiUrl={agentManagerUiUrl} />
              </div>
            )}
            <div>
              <p className="text-xs text-slate-500 uppercase tracking-wider">Mode</p>
              <p className="text-slate-200">{formatExecutionMode(execution.mode)}</p>
            </div>
            {execution.operation && (
              <div>
                <p className="text-xs text-slate-500 uppercase tracking-wider">Operation</p>
                <p className="text-slate-200 capitalize">{execution.operation}</p>
              </div>
            )}
            <div>
              <p className="text-xs text-slate-500 uppercase tracking-wider">Created</p>
              <p className="text-slate-200">{formatRelativeTime(execution.createdAt)}</p>
            </div>
            {execution.updatedAt && (
              <div>
                <p className="text-xs text-slate-500 uppercase tracking-wider">Updated</p>
                <p className="text-slate-200">{formatRelativeTime(execution.updatedAt)}</p>
              </div>
            )}
            {execution.parentExecutionId && (
              <div className="col-span-2">
                <p className="text-xs text-slate-500 uppercase tracking-wider">Parent Execution</p>
                <EntityLink
                  entityType="execution"
                  executionId={execution.parentExecutionId}
                  label={execution.parentExecutionId}
                />
              </div>
            )}
          </div>

          {(execution.executionPreferences || execution.actualRunner || execution.actualModel || execution.selectionReason) && (
            <div className="rounded-lg border border-cyan-300/15 bg-cyan-300/[0.04] p-3" data-testid="execution-run-selection">
              <p className="text-xs font-medium uppercase tracking-wider text-cyan-200">Runner selection</p>
              <div className="mt-2 grid grid-cols-2 gap-3 text-sm">
                <div><p className="text-xs text-slate-500">Requested runner</p><p className="text-slate-200">{execution.executionPreferences?.preferredRunner || "default"}</p></div>
                <div><p className="text-xs text-slate-500">Requested model</p><p className="text-slate-200">{execution.executionPreferences?.model || "default"}</p></div>
                <div><p className="text-xs text-slate-500">Actual runner</p><p className="text-slate-200">{execution.actualRunner || "not selected"}</p></div>
                <div><p className="text-xs text-slate-500">Actual model</p><p className="text-slate-200">{execution.actualModel || "not selected"}</p></div>
              </div>
              {execution.selectionReason && <p className="mt-2 text-xs leading-5 text-slate-400">Selection reason: {execution.selectionReason}</p>}
            </div>
          )}

          {(execution.continuationOf || execution.continuationChildIds?.length || execution.scopeExtensions?.length) && (
            <div className="rounded-lg border border-violet-300/15 bg-violet-300/[0.04] p-3" data-testid="execution-continuation-chain">
              <p className="text-xs font-medium uppercase tracking-wider text-violet-200">Continuation and scope</p>
              {execution.continuationOf && <p className="mt-2 text-sm text-slate-300">Continues <EntityLink entityType="execution" executionId={execution.continuationOf} label={execution.continuationOf} /></p>}
              {!!execution.continuationChildIds?.length && <p className="mt-2 text-sm text-slate-300">Child runs: {execution.continuationChildIds.join(", ")}</p>}
              {!!execution.scopeExtensions?.length && <div className="mt-2 text-xs text-slate-400"><p>Recorded scope extensions:</p><ul className="mt-1 list-disc pl-5">{execution.scopeExtensions.map((extension, index) => <li key={`${extension.recordedAt}-${index}`}>{extension.paths.join(", ")} — {extension.reason} ({extension.author})</li>)}</ul></div>}
            </div>
          )}

          {/* Agent Manager run link */}
          {execution.runId && agentManagerUiUrl && (
            <a
              href={buildAgentRunUrl(agentManagerUiUrl, execution.runId) ?? "#"}
              target="_blank"
              rel="noopener noreferrer"
              data-testid={selectors.executionDetails.viewRunButton}
            >
              <Button
                size="sm"
                variant="outline"
                className="border-slate-600/40 text-slate-400 hover:text-slate-200"
              >
                <ExternalLink className="mr-1.5 h-3 w-3" />
                View Run
              </Button>
            </a>
          )}

          {/* Failure reason */}
          {execution.failureReason && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3">
              <p className="text-xs text-red-400 font-medium uppercase tracking-wider mb-1">Failure Reason</p>
              <MarkdownRenderer content={execution.failureReason} className="prose-sm-slate text-sm text-red-200" />
            </div>
          )}

          {/* Post-run status badge */}
          {postRunBadgeExecution && (
            <div className="space-y-2">
              <p className="text-xs text-slate-500 uppercase tracking-wider">Post-Run Checks</p>
              <PostRunStatusBadge execution={postRunBadgeExecution} />
            </div>
          )}
        </div>
      </DetailSection>

      {/* Actions */}
      <DetailSection title="Actions">
        <div
          className="flex flex-wrap gap-2"
          data-testid={selectors.executionDetails.overviewActions}
        >
          {isActive && (
            <Button
              variant="destructive"
              size="sm"
              disabled={actionBusy}
              onClick={onCancel}
              data-testid={selectors.executionDetails.cancelButton}
            >
              {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <XCircle className="mr-1 h-3.5 w-3.5" />}
              Cancel
            </Button>
          )}
          {isTerminal && execution.status === "failed" && (
            <Button
              variant="outline"
              size="sm"
              disabled={actionBusy}
              onClick={onRetry}
              data-testid={selectors.executionDetails.retryButton}
            >
              {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <RotateCcw className="mr-1 h-3.5 w-3.5" />}
              Retry
            </Button>
          )}
          {isTerminal && (
            <Button
              variant="outline"
              size="sm"
              disabled={actionBusy}
              onClick={onFollowUp}
              data-testid={selectors.executionDetails.followUpButton}
            >
              {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="mr-1 h-3.5 w-3.5" />}
              Follow-up
            </Button>
          )}
          {(execution.continuationOf || execution.continuationChildIds?.length) && (
            <>
              <Button variant="outline" size="sm" disabled={actionBusy} onClick={onHaltContinuation} data-testid="halt-continuation-button">
                {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <PauseCircle className="mr-1 h-3.5 w-3.5" />}Halt continuation
              </Button>
              <Button variant="outline" size="sm" disabled={actionBusy} onClick={onResumeContinuation} data-testid="resume-continuation-button">
                {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <PlayCircle className="mr-1 h-3.5 w-3.5" />}Resume continuation
              </Button>
            </>
          )}
          {showRunChecks && (
            <Button
              variant="outline"
              size="sm"
              disabled={actionBusy}
              onClick={onRunPostRunChecks}
              data-testid={selectors.executionDetails.runChecksButton}
            >
              {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <ClipboardCheck className="mr-1 h-3.5 w-3.5" />}
              {runChecksLabel}
            </Button>
          )}
        </div>
      </DetailSection>
    </div>
  );
}
