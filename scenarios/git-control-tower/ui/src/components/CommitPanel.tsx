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
import type { CommitCheckRun, PrecommitRunResult, RepoHistoryEntry } from "../lib/api";
import { XCircle } from "lucide-react";

export interface CommitPanelPrecommitProgress {
  running: boolean;
  command?: string;
  elapsedMs: number;
  tail: string[];
  onCancel?: () => void;
  // Persistent failure state (shown after the stream finishes with a non-passing status).
  failedResult?: PrecommitRunResult | null;
  onDismissFailure?: () => void;
  onCommitAnyway?: () => void;
  onRunAgain?: () => void;
  onDisable?: () => void;
  isCommittingAnyway?: boolean;
  isRunningAgain?: boolean;
  isDisablingChecks?: boolean;
}

function formatElapsedShort(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  const seconds = ms / 1000;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}m ${s.toString().padStart(2, "0")}s`;
}

interface CommitPanelProps {
  stagedCount: number;
  commitMessage: string;
  onCommitMessageChange: (value: string) => void;
  canUseApprovedMessage?: boolean;
  onUseApprovedMessage?: () => void;
  isUsingApprovedMessage?: boolean;
  onCommit: (
    message: string,
    options: { conventional: boolean; amend: boolean; skipHooks: boolean; authorName?: string; authorEmail?: string }
  ) => void;
  isCommitting: boolean;
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
  canPush?: boolean;
  aheadCount?: number;
  pushTarget?: string;
  sourceBranch?: string;
  // History mode - disables the panel
  isHistoryMode?: boolean;
  historyCommit?: Pick<RepoHistoryEntry, "hash" | "subject" | "checks"> | null;
  // Pre-commit live progress (driven by useStreamPrecommit in the parent)
  precommitProgress?: CommitPanelPrecommitProgress;
}

function CommitErrorDisplay({ error }: { error: string }) {
  const [copied, setCopied] = useState(false);
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
      <span className="break-words flex-1 max-h-32 overflow-y-auto">{error}</span>
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
  stagedCount,
  commitMessage,
  onCommitMessageChange,
  canUseApprovedMessage = false,
  onUseApprovedMessage,
  isUsingApprovedMessage = false,
  onCommit,
  isCommitting,
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
  canPush = false,
  aheadCount = 0,
  pushTarget,
  sourceBranch,
  isHistoryMode = false,
  historyCommit,
  precommitProgress,
  onRetryWithoutPrecommit,
  canRetryWithoutPrecommit = false
}: CommitPanelProps) {
  const [useConventional, setUseConventional] = useState(false);
  const [skipHooks, setSkipHooks] = useState(false);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [authorName, setAuthorName] = useState("");
  const [authorEmail, setAuthorEmail] = useState("");
  const [authorTouched, setAuthorTouched] = useState(false);
  const [amendLast, setAmendLast] = useState(false);

  const trimmedMessage = commitMessage.trim();
  const canCommit = stagedCount > 0 && !isCommitting && (trimmedMessage.length > 0 || amendLast);
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
      className={`flex flex-col min-w-0 ${fillHeight ? "h-full" : "h-auto"}`}
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
        <CardContent className="flex-1 min-h-0 min-w-0 pt-2 pb-3 overflow-y-auto overflow-x-hidden">
        {isHistoryMode ? (
          <CommitChecksContent commit={historyCommit} />
        ) : (
        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <textarea
              value={commitMessage}
              onChange={(e) => onCommitMessageChange(e.target.value)}
              placeholder={
                amendLast ? "Commit message (leave empty to keep previous)..." : "Commit message..."
              }
              className="w-full h-20 px-3 py-2 text-sm bg-slate-800/50 border border-slate-700 rounded-md resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder:text-slate-500"
              disabled={isCommitting}
              data-testid="commit-message-input"
            />
          </div>

          {canUseApprovedMessage && onUseApprovedMessage && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onUseApprovedMessage}
              disabled={isUsingApprovedMessage}
              className="h-7 px-2"
              data-testid="use-approved-message-button"
            >
              {isUsingApprovedMessage ? (
                <span className="flex items-center">
                  <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                  Loading...
                </span>
              ) : (
                "Use approved message"
              )}
            </Button>
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
                      <span className="truncate">Pushing...</span>
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
                {isCommitting ? (
                  <span className="flex items-center min-w-0">
                    <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                    <span className="truncate">Committing...</span>
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
                    className="w-full px-3 py-2 text-xs bg-slate-800/50 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder:text-slate-500"
                    disabled={isCommitting}
                  />
                  <input
                    value={authorEmail}
                    onChange={(e) => {
                      setAuthorEmail(e.target.value);
                      setAuthorTouched(true);
                    }}
                    placeholder="Author email"
                    className="w-full px-3 py-2 text-xs bg-slate-800/50 border border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder:text-slate-500"
                    disabled={isCommitting}
                  />
                </div>
              </div>
            </div>
          )}

          {/* Pre-commit live progress */}
          {precommitProgress?.running && (
            <div
              className="rounded-md border border-sky-800/60 bg-sky-950/30 p-3 text-xs text-sky-100"
              role="status"
              aria-live="polite"
              data-testid="commit-precommit-progress"
            >
              <div className="flex items-center justify-between gap-3">
                <span className="flex items-center gap-2 font-medium">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  Running pre-commit
                  {precommitProgress.command && (
                    <code className="text-sky-200 truncate max-w-[16rem]" title={precommitProgress.command}>
                      {precommitProgress.command}
                    </code>
                  )}
                </span>
                <span className="tabular-nums">{formatElapsedShort(precommitProgress.elapsedMs)}</span>
              </div>
              <p className="mt-2 text-[11px] text-sky-200/70">
                This is the pre-commit hook running configured checks before the commit. It may take a while…
              </p>
              {precommitProgress.tail.length > 0 && (
                <pre className="mt-3 max-h-32 overflow-auto whitespace-pre-wrap rounded bg-slate-950 p-2 text-[11px] text-slate-200">
                  {precommitProgress.tail.join("\n")}
                </pre>
              )}
              {(precommitProgress.onCancel || precommitProgress.onCommitAnyway) && (
                <div className="mt-2 flex justify-end gap-2">
                  {precommitProgress.onCommitAnyway && (
                    <button
                      type="button"
                      onClick={precommitProgress.onCommitAnyway}
                      disabled={precommitProgress.isCommittingAnyway}
                      className="inline-flex items-center gap-1 rounded border border-amber-700/70 bg-amber-950/40 px-2 py-1 text-[11px] text-amber-100 hover:bg-amber-900/40 disabled:opacity-60"
                      data-testid="commit-anyway-running"
                    >
                      <GitCommit className="h-3 w-3" />
                      Commit Anyway
                    </button>
                  )}
                  {precommitProgress.onCancel && (
                    <button
                      type="button"
                      onClick={precommitProgress.onCancel}
                      className="inline-flex items-center gap-1 rounded border border-red-700/60 px-2 py-1 text-[11px] text-red-200 hover:bg-red-900/40"
                    >
                      <XCircle className="h-3 w-3" />
                      Cancel
                    </button>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Pre-commit failure — persistent until user dismisses */}
          {!precommitProgress?.running && precommitProgress?.failedResult && (
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
                {precommitProgress.onDismissFailure && (
                  <button
                    type="button"
                    onClick={precommitProgress.onDismissFailure}
                    className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-red-800/60 text-red-200 hover:bg-red-900/40"
                    aria-label="Dismiss pre-commit failure"
                  >
                    <XCircle className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>
              <div className="mt-3 rounded border border-red-900/60 bg-red-950/50 p-2 text-[11px]">
                <div className="flex items-center justify-between gap-3">
                  <span className="font-medium text-red-100">{precommitProgress.failedResult.summary}</span>
                  <span className="text-red-200/70">exit {precommitProgress.failedResult.exit_code}</span>
                </div>
                {(precommitProgress.failedResult.stdout || precommitProgress.failedResult.stderr) && (
                  <pre className="mt-2 max-h-56 overflow-auto whitespace-pre-wrap rounded bg-slate-950 p-2 text-[11px] text-slate-200">
                    {[precommitProgress.failedResult.stdout, precommitProgress.failedResult.stderr].filter(Boolean).join("\n")}
                  </pre>
                )}
              </div>
              <div className="mt-3 flex flex-wrap justify-end gap-2">
                {precommitProgress.onRunAgain && (
                  <button
                    type="button"
                    onClick={precommitProgress.onRunAgain}
                    disabled={precommitProgress.isRunningAgain}
                    className="inline-flex items-center gap-1 rounded border border-slate-700 bg-slate-900 px-2 py-1 text-[11px] text-slate-100 hover:bg-slate-800 disabled:opacity-60"
                  >
                    Run Again
                  </button>
                )}
                {precommitProgress.onCommitAnyway && precommitProgress.failedResult.override_allowed && (
                  <button
                    type="button"
                    onClick={precommitProgress.onCommitAnyway}
                    disabled={precommitProgress.isCommittingAnyway}
                    className="inline-flex items-center gap-1 rounded border border-amber-700/70 bg-amber-950/40 px-2 py-1 text-[11px] text-amber-100 hover:bg-amber-900/40 disabled:opacity-60"
                  >
                    Commit Anyway
                  </button>
                )}
                {precommitProgress.onDisable && (
                  <button
                    type="button"
                    onClick={precommitProgress.onDisable}
                    disabled={precommitProgress.isDisablingChecks}
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
    </Card>
  );
}
