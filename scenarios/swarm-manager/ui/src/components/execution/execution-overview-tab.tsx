/**
 * ExecutionOverviewTab — Metadata, failure reason, post-run status,
 * and action buttons for a single execution.
 */

import {
  ClipboardCheck,
  ExternalLink,
  Loader2,
  RefreshCw,
  RotateCcw,
  XCircle,
} from "lucide-react";
import { renderMarkdown } from "../../lib/render-markdown";
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
              <div className="prose-sm-slate text-sm text-red-200" dangerouslySetInnerHTML={{ __html: renderMarkdown(execution.failureReason) }} />
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
