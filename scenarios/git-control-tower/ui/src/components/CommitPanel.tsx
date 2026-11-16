import { StagedSafetyNotice } from "./PushSafetyIndicators";
import { useState, useEffect } from "react";
import {
  GitCommit,
  Loader2,
  AlertCircle,
  ChevronDown,
  ChevronRight,
  Upload,
  History,
  Copy,
  Check
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Button } from "./ui/button";
import type { AuthorityStatus, CommitCheckRun, PrecommitRunResult, RepoHistoryEntry } from "../lib/api";
import { XCircle } from "lucide-react";
import { AuthenticatorSignIn } from "./AuthenticatorSignIn";
import { CommitMessageComposer } from "../features/commit-message/CommitMessageComposer";
import { mergeSuggestionTrailers, parseMessage, serializeMessage } from "../features/commit-message/message";
import type { QueuedStageSelection } from "../features/commit-message/stageQueue";
import { StagedMessageMode, type StagedMessageSuggestion } from "../features/commit-message/api";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { QueuedStageSuggestionStatus } from "./QueuedStageSuggestionStatus";

export interface CommitPanelPrecommitNotice {
  // Persistent failure state from identity-bound final admission.
  failedResult?: PrecommitRunResult | null;
  onDismissFailure?: () => void;
  onCommitAnyway?: () => void;
  onDisable?: () => void;
  isCommittingAnyway?: boolean;
  isDisablingChecks?: boolean;
}

interface CommitPanelProps {
  repoId?: string | null;
  statusVersion?: number;
  stagedCount: number;
  commitMessage: string;
  onCommitMessageChange: (value: string) => void;
  onCommit: (
    message: string,
    options: { conventional: boolean; amend: boolean; skipHooks: boolean; authorName?: string; authorEmail?: string }
  ) => void;
  isCommitting: boolean;
  isUpdatingIndex?: boolean;
  stagedPaths?: readonly string[];
  /** Narrows the index back to a commit suggestion's staged selection. */
  onUnstagePaths?: (paths: string[]) => Promise<void>;
  onQueueStagedExtras?: (paths: string[]) => Promise<boolean>;
  commitProgressLabel?: string;
  commitError?: string;
  // Reuse a passed pre-commit after a post-pass failure (e.g. index lock) — commit
  // again with --no-verify instead of re-streaming the ~1-minute pre-commit.
  onRetryWithoutPrecommit?: () => void;
  canRetryWithoutPrecommit?: boolean;
  defaultAuthorName?: string;
  defaultAuthorEmail?: string;
  canAmend?: boolean;
  amendDisabledReason?: string;
  collapsed?: boolean;
  onToggleCollapse?: () => void;
  fillHeight?: boolean;
  // Push functionality
  onPush?: () => void;
  isPushing?: boolean;
  /** Phase and elapsed time while a push is running, e.g. "Pushing 1m 12s". */
  pushProgressLabel?: string;
  canPush?: boolean;
  aheadCount?: number;
  pushTarget?: string;
  sourceBranch?: string;
  // History mode - disables the panel
  isHistoryMode?: boolean;
  historyCommit?: Pick<RepoHistoryEntry, "hash" | "subject" | "checks"> | null;
  // Identity-bound failure notice with retry and explicit override actions.
  precommitNotice?: CommitPanelPrecommitNotice;
  authorityStatus?: AuthorityStatus;
  onAuthoritySignedIn?: () => void;
  queuedStageSelections?: QueuedStageSelection[];
  onRemoveQueuedStageSelection?: (id: string) => void;
  onResolveQueuedCommit?: (id: string, committed: boolean) => void;
  onStageQueuedSelection?: () => void;
  onRetryQueuedSuggestion?: (selection: QueuedStageSelection) => void;
  onRefreshQueuedSuggestion?: (selection: QueuedStageSelection) => void;
  onStartQueuedSuggestion?: (selection: QueuedStageSelection, mode: StagedMessageMode) => void;
  onFinishQueuedUnstage?: (selection: QueuedStageSelection) => void;
}

function CommitErrorDisplay({ error }: { error: string }) {
  const [copied, setCopied] = useState(false);
  const normalized = error.toLowerCase();
  const reason = normalized.includes("expired")
    ? "Expired approval — review the exact change again."
    : normalized.includes("replay") || normalized.includes("already been consumed")
      ? "Replayed approval — review the exact change again."
      : normalized.includes("changed") || normalized.includes("stale")
        ? "The reviewed repository state is stale — review the exact change again."
        : undefined;
  const handleCopy = () => {
    void navigator.clipboard.writeText(error).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };
  return (
    <div
      className="flex items-start gap-2 px-3 py-2 bg-red-950/30 border border-red-800/50 rounded-md text-xs text-red-400"
      data-testid="commit-error"
    >
      <AlertCircle className="h-3.5 w-3.5 mt-0.5 flex-shrink-0" />
      <span className="break-words flex-1 max-h-32 overflow-y-auto">
        {reason && <span className="block font-medium mb-1" data-testid="commit-error-reason">{reason}</span>}
        {error}
      </span>
      <button type="button" onClick={handleCopy} className="hover:text-red-300 shrink-0" aria-label="Copy error" title="Copy error">
        {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
      </button>
    </div>
  );
}

function checkTone(status: CommitCheckRun["status"]) {
  switch (status) {
    case "passed":
      return "border-emerald-800/60 bg-emerald-950/30 text-emerald-200";
    case "failed":
      return "border-red-800/60 bg-red-950/30 text-red-200";
    case "timeout":
      return "border-amber-800/60 bg-amber-950/30 text-amber-200";
    default:
      return "border-slate-800 bg-slate-900/50 text-slate-300";
  }
}

function formatDuration(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
  return `${ms}ms`;
}

function CommitChecksContent({ commit }: { commit?: CommitPanelProps["historyCommit"] }) {
  const checks = commit?.checks ?? [];
  if (!commit) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <History className="h-8 w-8 text-amber-400/60 mb-3" />
        <p className="text-sm text-amber-200/80">Select a historical commit</p>
      </div>
    );
  }
  if (checks.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <History className="h-8 w-8 text-slate-500 mb-3" />
        <p className="text-sm text-slate-300">No commit checks recorded</p>
        <p className="text-xs text-slate-500 mt-1">This commit has no captured local check runs.</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="min-w-0">
        <p className="font-mono text-xs text-slate-500 truncate">{commit.hash}</p>
        <p className="text-xs text-slate-300 truncate" title={commit.subject}>{commit.subject}</p>
      </div>
      {checks.map((check, index) => (
        <div
          key={`${check.kind}-${check.timestamp}-${index}`}
          className={`rounded-md border p-3 space-y-2 ${checkTone(check.status)}`}
          data-testid="commit-check-run"
        >
          <div className="flex items-center justify-between gap-2 min-w-0">
            <span className="text-xs font-medium uppercase tracking-normal">{check.kind}</span>
            <span className="rounded border border-current/30 px-1.5 py-0.5 text-[10px]">
              {check.status}
            </span>
          </div>
          <div className="rounded bg-slate-950/40 px-2 py-1 font-mono text-xs text-slate-200 break-words">
            {check.command}
          </div>
          <div className="grid grid-cols-2 gap-2 text-[11px] text-slate-400">
            <span>Exit {check.exit_code}</span>
            <span>{formatDuration(check.duration_ms)}</span>
            <span className="col-span-2 truncate" title={check.timestamp}>
              {check.timestamp ? new Date(check.timestamp).toLocaleString() : ""}
            </span>
          </div>
          {check.summary && <p className="text-xs text-slate-300">{check.summary}</p>}
          {(check.stdout || check.stderr) && (
            <div className="space-y-2">
              {check.stdout && (
                <pre className="max-h-24 overflow-auto rounded bg-slate-950/50 p-2 text-[11px] text-slate-300 whitespace-pre-wrap">{check.stdout}</pre>
              )}
              {check.stderr && (
                <pre className="max-h-24 overflow-auto rounded bg-slate-950/50 p-2 text-[11px] text-red-200 whitespace-pre-wrap">{check.stderr}</pre>
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

export function CommitPanel({
  repoId,
  statusVersion = 0,
  stagedCount,
  commitMessage,
  onCommitMessageChange,
  onCommit,
  isCommitting,
  isUpdatingIndex = false,
  stagedPaths,
  onUnstagePaths,
  onQueueStagedExtras,
  commitProgressLabel,
  commitError,
  defaultAuthorName,
  defaultAuthorEmail,
  canAmend = false,
  amendDisabledReason,
  collapsed = false,
  onToggleCollapse,
  fillHeight = false,
  onPush,
  isPushing = false,
  pushProgressLabel,
  canPush = false,
  aheadCount = 0,
  pushTarget,
  sourceBranch,
  isHistoryMode = false,
  historyCommit,
  precommitNotice,
  onRetryWithoutPrecommit,
  canRetryWithoutPrecommit = false,
  authorityStatus,
  onAuthoritySignedIn,
  queuedStageSelections = [],
  onRemoveQueuedStageSelection,
  onResolveQueuedCommit,
  onStageQueuedSelection,
  onRetryQueuedSuggestion,
  onRefreshQueuedSuggestion,
  onStartQueuedSuggestion,
  onFinishQueuedUnstage,
}: CommitPanelProps) {
  const [signInOpen, setSignInOpen] = useState(false);
  const [useConventional, setUseConventional] = useState(false);
  const [skipHooks, setSkipHooks] = useState(false);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [authorName, setAuthorName] = useState("");
  const [authorEmail, setAuthorEmail] = useState("");
  const [authorTouched, setAuthorTouched] = useState(false);
  const [amendLast, setAmendLast] = useState(false);
  const [queuedSuggestionToApply, setQueuedSuggestionToApply] = useState<string>();

  const useQueuedSuggestion = (suggestion: StagedMessageSuggestion) => {
    const message = serializeMessage({ title: suggestion.title, description: suggestion.body, trailers: mergeSuggestionTrailers("", suggestion.trailers) });
    if (commitMessage.trim()) setQueuedSuggestionToApply(message);
    else onCommitMessageChange(message);
  };

  const trimmedMessage = commitMessage.trim();
  const hasTitle = Boolean(parseMessage(commitMessage).title.trim());
  const canCommit = stagedCount > 0 && !isCommitting && !isUpdatingIndex && (hasTitle || (amendLast && !trimmedMessage)) &&
    (!authorityStatus || authorityStatus.canMutate);
  const showPushAction = Boolean(onPush && aheadCount > 0);
  const pushDisabled = isPushing || !canPush;
  const handlePushClick = onPush ?? (() => {});
  const _pushTargetLabel = pushTarget ? `Target: ${pushTarget}` : undefined;
  const _sourceLabel =
    sourceBranch && pushTarget && !pushTarget.endsWith(`/${sourceBranch}`)
      ? `from ${sourceBranch}`
      : undefined;
  const handleToggleCollapse = onToggleCollapse ?? (() => {});

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (canCommit) {
      onCommit(trimmedMessage, {
        conventional: useConventional && trimmedMessage.length > 0,
        amend: amendLast,
        skipHooks,
        authorName: authorName.trim() || defaultAuthorName || undefined,
        authorEmail: authorEmail.trim() || defaultAuthorEmail || undefined
      });
    }
  };

  useEffect(() => {
    if (authorTouched) return;
    if (defaultAuthorName) {
      setAuthorName(defaultAuthorName);
    }
    if (defaultAuthorEmail) {
      setAuthorEmail(defaultAuthorEmail);
    }
  }, [authorTouched, defaultAuthorName, defaultAuthorEmail]);

  useEffect(() => {
    if (!canAmend && amendLast) {
      setAmendLast(false);
    }
  }, [amendLast, canAmend]);

  return (
    <Card
      className={`flex flex-col min-w-0 ${fillHeight ? "h-full" : "h-auto max-h-full"}`}
      data-testid="commit-panel"
    >
      <CardHeader className="py-3 min-w-0">
        <CardTitle className="flex items-center gap-2 text-sm min-w-0">
          <button
            className="p-1 rounded hover:bg-slate-800/70 transition-colors"
            onClick={handleToggleCollapse}
            aria-label={collapsed ? "Expand commit panel" : "Collapse commit panel"}
            type="button"
          >
            {collapsed ? (
              <ChevronRight className="h-3 w-3 text-slate-400" />
            ) : (
              <ChevronDown className="h-3 w-3 text-slate-400" />
            )}
          </button>
          <GitCommit className="h-4 w-4 text-slate-500" />
          <span className="truncate">{isHistoryMode ? "Commit Checks" : "Commit"}</span>
        </CardTitle>
      </CardHeader>

      {!collapsed && (
        <CardContent data-gct-scroll-owner="commit" className="flex-1 min-h-0 min-w-0 pt-2 pb-3 overflow-y-auto overflow-x-hidden">
        {isHistoryMode ? (
          <CommitChecksContent commit={historyCommit} />
        ) : (
        <form onSubmit={handleSubmit} className="space-y-3">
          {stagedCount > 0 && <StagedSafetyNotice />}
          <CommitMessageComposer
            key={repoId ?? "default"}
            repoId={repoId ?? ""}
            statusVersion={statusVersion}
            value={commitMessage}
            onChange={onCommitMessageChange}
            stagedCount={stagedCount}
            updatingIndex={isUpdatingIndex}
            disabled={isCommitting}
            amend={amendLast}
            stagedPaths={stagedPaths}
            onUnstage={onUnstagePaths}
            onQueueStagedExtras={onQueueStagedExtras}
          />

          {queuedStageSelections.length > 0 && <section className="space-y-2 rounded-lg border border-cyan-900/70 bg-cyan-950/20 p-3" aria-label="Queued commit selections" data-testid="queued-stage-selections">
            <div className="flex items-center justify-between gap-2"><p className="text-xs font-semibold text-cyan-100">{queuedStageSelections.length} next commit {queuedStageSelections.length === 1 ? "selection" : "selections"} queued</p>
              {onStageQueuedSelection && queuedStageSelections[0]?.state === "stage_failed" && <button type="button" disabled={isUpdatingIndex} className="min-h-9 rounded border border-cyan-800 px-2 text-[11px] text-cyan-100 disabled:opacity-50" onClick={onStageQueuedSelection}>Retry staging next</button>}
            </div>
            <p className="text-[11px] text-slate-400">Each selection has its own suggestion job and stays unstaged while the current commit is active. After this commit succeeds, the next unchanged selection is staged for review.{queuedStageSelections[0]?.state === "stage_failed" ? " The last staging attempt failed or was incomplete; retry it or remove this selection." : queuedStageSelections[0]?.state === "commit_pending" ? " The commit outcome needs confirmation before this queue can continue." : ""}</p>
            <ol className="space-y-1">{queuedStageSelections.map((selection, index) => <li key={selection.id} className="flex items-start gap-2 text-[11px] text-slate-300">
              <span className="shrink-0 font-mono text-cyan-300">{index + 1}.</span><div className="min-w-0 flex-1"><p className="break-all">{selection.paths.join(", ")}</p>{selection.stageError && <p role="alert" className="my-1 text-xs text-amber-200">{selection.stageError}</p>}{selection.state === "commit_pending" && onResolveQueuedCommit && <div className="mt-1 flex flex-wrap gap-2"><button type="button" disabled={isCommitting} className="text-cyan-200 underline disabled:opacity-50" onClick={() => onResolveQueuedCommit(selection.id, true)}>I committed these files</button><button type="button" disabled={isCommitting} className="text-slate-300 underline disabled:opacity-50" onClick={() => onResolveQueuedCommit(selection.id, false)}>The commit did not happen</button></div>}<QueuedStageSuggestionStatus repositoryId={repoId ?? ""} selection={selection} isStaged={selection.state === "staged"} onUse={useQueuedSuggestion} onRetry={onRetryQueuedSuggestion} onRefresh={onRefreshQueuedSuggestion} onStartSuggestion={onStartQueuedSuggestion} onFinishUnstage={onFinishQueuedUnstage} /></div>
              {onRemoveQueuedStageSelection && selection.state !== "commit_pending" && <button type="button" disabled={isUpdatingIndex} className="min-h-8 shrink-0 px-1 text-slate-400 hover:text-white disabled:opacity-50" onClick={() => onRemoveQueuedStageSelection(selection.id)} aria-label={`Remove queued selection ${index + 1}`}>Remove</button>}
            </li>)}</ol>
          </section>}

          {authorityStatus && !authorityStatus.canMutate && (
            <div
              className="rounded-md border border-amber-800/60 bg-amber-950/20 px-3 py-2 text-xs text-amber-200"
              role="status"
              data-testid="authority-status"
            >
              <span className="font-medium">
                {authorityStatus.authState === "expired"
                  ? "Cloudflare Access session expired"
                  : authorityStatus.authState === "service"
                    ? "Service authentication is read-only"
                    : authorityStatus.authState === "conflict"
                      ? "Conflicting authentication"
                      : "Read-only authority"}
              </span>
              {authorityStatus.authSource && (
                <span className="ml-2" data-testid="authority-source">
                  source: {authorityStatus.authSource}
                </span>
              )}
              {authorityStatus.email && <span className="ml-2">{authorityStatus.email}</span>}
              {authorityStatus.reason && <span className="ml-2">{authorityStatus.reason}</span>}
              {authorityStatus.recoveryUrl && (
                <a
                  className="ml-2 underline decoration-dotted underline-offset-2 hover:text-amber-100"
                  data-testid="auth-recovery"
                  href={authorityStatus.recoveryUrl}
                >
                  Reauthenticate with Cloudflare Access
                </a>
              )}
              {authorityStatus.authSource !== "cloudflare_access" && onAuthoritySignedIn && (
                <>
                  <button
                    className="ml-2 underline decoration-dotted underline-offset-2 hover:text-amber-100"
                    data-testid="open-authenticator-sign-in"
                    onClick={() => setSignInOpen((open) => !open)}
                    type="button"
                  >
                    {signInOpen ? "Hide sign-in" : "Sign in"}
                  </button>
                  {signInOpen && (
                    <AuthenticatorSignIn
                      onContinueReadOnly={() => setSignInOpen(false)}
                      onSignedIn={() => {
                        setSignInOpen(false);
                        onAuthoritySignedIn();
                      }}
                    />
                  )}
                </>
              )}
            </div>
          )}

          <div className="flex items-center justify-between gap-2 min-w-0">
            <button
              type="button"
              className="flex items-center gap-2 text-xs text-slate-400 hover:text-slate-200 transition-colors"
              onClick={() => setAdvancedOpen((prev) => !prev)}
            >
              {advancedOpen ? (
                <ChevronDown className="h-3 w-3" />
              ) : (
                <ChevronRight className="h-3 w-3" />
              )}
              Advanced
            </button>

            <div className="flex items-center gap-2 min-w-0">
              {showPushAction && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handlePushClick}
                  disabled={pushDisabled}
                  title={!canPush && aheadCount > 0 ? "Pull required first" : undefined}
                  data-testid="push-button"
                >
                  {isPushing ? (
                    <span className="flex items-center">
                      <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                      <span className="truncate">{pushProgressLabel ?? "Pushing…"}</span>
                    </span>
                  ) : (
                    <span className="flex items-center">
                      <Upload className="h-3 w-3 mr-1" />
                      <span className="truncate">
                        Push ({aheadCount})
                      </span>
                    </span>
                  )}
                </Button>
              )}
              <Button
                type="submit"
                variant="default"
                size="sm"
                disabled={!canCommit}
                className="min-w-0 max-w-full"
                data-testid="commit-button"
              >
                {isCommitting || isUpdatingIndex ? (
                  <span className="flex items-center min-w-0">
                    <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                    <span className="truncate">{isUpdatingIndex ? "Updating files…" : commitProgressLabel ?? "Committing…"}</span>
                  </span>
                ) : (
                  <span className="flex items-center min-w-0">
                    <GitCommit className="h-3 w-3 mr-1" />
                    <span className="truncate">
                      {amendLast ? "Amend" : "Commit"} ({stagedCount} file
                      {stagedCount !== 1 ? "s" : ""})
                    </span>
                  </span>
                )}
              </Button>
            </div>
          </div>

          {advancedOpen && (
            <div className="rounded-md border border-slate-800 bg-slate-900/40 p-3 space-y-3">
              <label className="flex items-center gap-2 text-xs text-slate-400 cursor-pointer">
                <input
                  type="checkbox"
                  checked={useConventional}
                  onChange={(e) => setUseConventional(e.target.checked)}
                  disabled={isCommitting}
                  className="w-3.5 h-3.5 rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500 focus:ring-offset-0"
                  data-testid="conventional-commit-checkbox"
                />
                Conventional commit format
              </label>

              <label className="flex items-center gap-2 text-xs text-slate-400 cursor-pointer">
                <input
                  type="checkbox"
                  checked={skipHooks}
                  onChange={(e) => setSkipHooks(e.target.checked)}
                  disabled={isCommitting}
                  className="w-3.5 h-3.5 rounded border-slate-600 bg-slate-800 text-amber-500 focus:ring-amber-500 focus:ring-offset-0"
                  data-testid="skip-hooks-checkbox"
                />
                Skip pre-commit hooks (commit anyway)
              </label>

              <label className="flex items-center gap-2 text-xs text-slate-400 cursor-pointer">
                <input
                  type="checkbox"
                  checked={amendLast}
                  onChange={(e) => setAmendLast(e.target.checked)}
                  disabled={isCommitting || !canAmend}
                  className="w-3.5 h-3.5 rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500 focus:ring-offset-0"
                  data-testid="amend-commit-checkbox"
                />
                Amend last commit (unpushed only)
              </label>
              {!canAmend && amendDisabledReason && (
                <p className="text-xs text-amber-400">{amendDisabledReason}</p>
              )}
              {amendLast && trimmedMessage.length === 0 && (
                <p className="text-xs text-slate-500">Using previous commit message</p>
              )}

              <div className="space-y-2">
                <p className="text-xs text-slate-500">Commit identity (overrides local git config)</p>
                <div className="grid gap-2">
                  <input
                    value={authorName}
                    onChange={(e) => {
                      setAuthorName(e.target.value);
                      setAuthorTouched(true);
                    }}
                    placeholder="Author name"
                    className="w-full px-3 py-2 gct-commit-input gct-commit-input-compact bg-slate-800/50 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder:text-slate-500"
                    disabled={isCommitting}
                  />
                  <input
                    value={authorEmail}
                    onChange={(e) => {
                      setAuthorEmail(e.target.value);
                      setAuthorTouched(true);
                    }}
                    placeholder="Author email"
                    className="w-full px-3 py-2 gct-commit-input gct-commit-input-compact bg-slate-800/50 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder:text-slate-500"
                    disabled={isCommitting}
                  />
                </div>
              </div>
            </div>
          )}

          {/* Pre-commit failure — persistent until user dismisses */}
          {precommitNotice?.failedResult && (
            <div
              className="rounded-md border border-red-800/60 bg-red-950/30 p-3 text-xs text-red-100"
              role="alert"
              data-testid="commit-precommit-failure"
            >
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="text-sm font-semibold text-red-50">
                    Pre-commit checks did not pass
                  </div>
                  <div className="mt-1 text-[11px] text-red-200/80">
                    The commit was not created. You can fix the issue and try again, or commit anyway if the checks allow it.
                  </div>
                </div>
                {precommitNotice.onDismissFailure && (
                  <button
                    type="button"
                    onClick={precommitNotice.onDismissFailure}
                    className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-red-800/60 text-red-200 hover:bg-red-900/40"
                    aria-label="Dismiss pre-commit failure"
                  >
                    <XCircle className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>
              <div className="mt-3 rounded border border-red-900/60 bg-red-950/50 p-2 text-[11px]">
                <div className="flex items-center justify-between gap-3">
                  <span className="font-medium text-red-100">{precommitNotice.failedResult.summary}</span>
                  <span className="text-red-200/70">exit {precommitNotice.failedResult.exit_code}</span>
                </div>
                {(precommitNotice.failedResult.stdout || precommitNotice.failedResult.stderr) && (
                  <pre className="mt-2 max-h-56 overflow-auto whitespace-pre-wrap rounded bg-slate-950 p-2 text-[11px] text-slate-200">
                    {[precommitNotice.failedResult.stdout, precommitNotice.failedResult.stderr].filter(Boolean).join("\n")}
                  </pre>
                )}
              </div>
              <div className="mt-3 flex flex-wrap justify-end gap-2">
                {precommitNotice.onCommitAnyway && precommitNotice.failedResult.override_allowed && (
                  <button
                    type="button"
                    onClick={precommitNotice.onCommitAnyway}
                    disabled={precommitNotice.isCommittingAnyway}
                    className="inline-flex items-center gap-1 rounded border border-amber-700/70 bg-amber-950/40 px-2 py-1 text-[11px] text-amber-100 hover:bg-amber-900/40 disabled:opacity-60"
                  >
                    Commit Anyway
                  </button>
                )}
                {precommitNotice.onDisable && (
                  <button
                    type="button"
                    onClick={precommitNotice.onDisable}
                    disabled={precommitNotice.isDisablingChecks}
                    className="inline-flex items-center gap-1 rounded border border-red-800/70 px-2 py-1 text-[11px] text-red-200 hover:bg-red-900/40 disabled:opacity-60"
                  >
                    Disable Checks
                  </button>
                )}
              </div>
            </div>
          )}

          {/* Error feedback */}
          {commitError && (
            <div className="space-y-2">
              <CommitErrorDisplay error={commitError} />
              {canRetryWithoutPrecommit && onRetryWithoutPrecommit && (
                <div className="flex justify-end">
                  <button
                    type="button"
                    onClick={onRetryWithoutPrecommit}
                    disabled={isCommitting}
                    className="inline-flex items-center gap-1 rounded border border-amber-700/70 bg-amber-950/40 px-2 py-1 text-[11px] text-amber-100 hover:bg-amber-900/40 disabled:opacity-60"
                    data-testid="retry-without-precommit"
                    title="Retry the commit without re-running the pre-commit checks that already passed"
                  >
                    <GitCommit className="h-3 w-3" />
                    Retry without pre-commit
                  </button>
                </div>
              )}
            </div>
          )}

          {/* Helper text when nothing staged */}
          {stagedCount === 0 && (
            <p className="text-xs text-slate-500 italic">
              Stage files to enable commit
            </p>
          )}
        </form>
        )}
        </CardContent>
      )}
      {queuedSuggestionToApply && <ResponsiveDialog
        open
        onClose={() => setQueuedSuggestionToApply(undefined)}
        title="Replace the current commit message?"
        ariaLabel="Compare current commit message with queued suggestion"
        closeLabel="Keep current message"
        size="lg"
        contentPadding="comfortable"
        testId="queued-suggestion-comparison"
        footer={<div className="flex justify-end gap-2"><Button type="button" variant="outline" onClick={() => setQueuedSuggestionToApply(undefined)}>Keep current</Button><Button type="button" onClick={() => { onCommitMessageChange(queuedSuggestionToApply); setQueuedSuggestionToApply(undefined); }}>Replace with suggestion</Button></div>}
      >
        <div className="grid gap-3 md:grid-cols-2"><section><h3 className="mb-2 text-xs font-semibold text-slate-300">Current message</h3><pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-slate-700 bg-slate-950 p-3 text-xs">{commitMessage}</pre></section><section><h3 className="mb-2 text-xs font-semibold text-cyan-200">Queued suggestion</h3><pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-cyan-900 bg-slate-950 p-3 text-xs">{queuedSuggestionToApply}</pre></section></div>
      </ResponsiveDialog>}
    </Card>
  );
}
