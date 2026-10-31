import { useState } from "react";
import { useTranslation } from "react-i18next";
import { AlertTriangle, MessageSquare, PlugZap, RefreshCw, Terminal } from "lucide-react";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import type { ConversationViewState } from "../stores/useConversationStore";

/**
 * The Messages pane's non-message states.
 *
 * There used to be exactly one: a full-width dashed rectangle reading "No
 * conversation events yet for this session.", pinned to the top-left of an
 * otherwise empty pane. It was shown while the first request was still in
 * flight, when that request failed, when the session was unknown, and when the
 * server could not read the session's transcript at all — so the single case it
 * described accurately was the rarest one.
 *
 * Each state here gets a glyph, a title, one line of explanation and at most
 * one action, and only the genuinely-empty case keeps the dashed border: dashes
 * read as "nothing here yet", which is misleading on a fault.
 */

interface MessagesPaneStateProps {
  view: Exclude<ConversationViewState, { kind: "messages" }>;
  onRetry: () => void;
  /** Present only where the pane can offer a repair path. */
  onOpenSettings?: () => void;
}

export default function MessagesPaneState({ view, onRetry, onOpenSettings }: MessagesPaneStateProps) {
  const { t } = useTranslation();

  if (view.kind === "loading") {
    return <MessagesPaneSkeleton />;
  }

  if (view.kind === "failed") {
    return (
      <StateCard
        tone="alert"
        testId="messages-state-failed"
        icon={<PlugZap className="h-6 w-6" aria-hidden="true" />}
        title={t(strings.messagesPane.state.failedTitle)}
        body={view.error.message}
        action={
          view.error.retryable
            ? { label: t(strings.messagesPane.state.retry), onClick: onRetry, icon: <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" /> }
            : undefined
        }
      />
    );
  }

  if (view.kind === "not-applicable") {
    return (
      <StateCard
        tone="quiet"
        testId="messages-state-not-applicable"
        icon={<Terminal className="h-6 w-6" aria-hidden="true" />}
        title={t(strings.messagesPane.state.notApplicableTitle)}
        body={view.capture.summary || t(strings.messagesPane.state.notApplicableBody)}
        detail={view.capture.detail}
      />
    );
  }

  if (view.kind === "unavailable") {
    return (
      <StateCard
        tone="alert"
        testId="messages-state-unavailable"
        icon={<AlertTriangle className="h-6 w-6" aria-hidden="true" />}
        title={t(strings.messagesPane.state.unavailableTitle)}
        body={view.capture.summary || t(strings.messagesPane.state.unavailableBody)}
        detail={joinDetail(view.capture.detail, view.capture.remediation, view.capture.transcriptPath)}
        action={
          onOpenSettings
            ? { label: t(strings.messagesPane.state.openSettings), onClick: onOpenSettings }
            : { label: t(strings.messagesPane.state.retry), onClick: onRetry, icon: <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" /> }
        }
        secondaryAction={onOpenSettings ? { label: t(strings.messagesPane.state.retry), onClick: onRetry } : undefined}
      />
    );
  }

  // Genuinely empty. When the server has told us capture is still starting up
  // we say so, because "no messages yet" and "not listening yet" call for
  // different amounts of patience.
  const pending = view.capture.state === "pending";
  return (
    <StateCard
      tone="empty"
      testId="messages-state-empty"
      icon={<MessageSquare className="h-6 w-6" aria-hidden="true" />}
      title={pending ? t(strings.messagesPane.state.pendingTitle) : t(strings.messagesPane.state.emptyTitle)}
      body={view.capture.summary || t(strings.messagesPane.state.emptyBody)}
      detail={view.capture.detail}
    />
  );
}

function joinDetail(...parts: Array<string | undefined>): string {
  return parts.map((part) => part?.trim()).filter((part): part is string => Boolean(part)).join("\n\n");
}

// ---------------------------------------------------------------------------
// Presentation
// ---------------------------------------------------------------------------

interface StateAction {
  label: string;
  onClick: () => void;
  icon?: React.ReactNode;
}

interface StateCardProps {
  tone: "empty" | "quiet" | "alert";
  testId: string;
  icon: React.ReactNode;
  title: string;
  body: string;
  detail?: string;
  action?: StateAction;
  secondaryAction?: StateAction;
}

function StateCard({ tone, testId, icon, title, body, detail, action, secondaryAction }: StateCardProps) {
  const { t } = useTranslation();
  const [detailOpen, setDetailOpen] = useState(false);

  return (
    // Centering matters: the old state sat in the top-left corner of a tall
    // empty pane, which reads as a rendering failure rather than a message.
    <div className="flex h-full w-full items-center justify-center p-6" data-testid={testId}>
      <div
        className={cn(
          "flex w-full max-w-sm flex-col items-center gap-2 rounded-xl px-6 py-8 text-center",
          tone === "empty" && "border border-dashed border-wc-default bg-wc-surface",
          tone === "quiet" && "border border-wc-default bg-wc-surface",
          tone === "alert" && "border border-wc-error bg-wc-error-surface",
        )}
      >
        <span className={cn("mb-1", tone === "alert" ? "text-wc-error-text" : "text-wc-text-muted")}>{icon}</span>
        <h2 className={cn("text-sm font-semibold", tone === "alert" ? "text-wc-error-text" : "text-wc-text-primary")}>{title}</h2>
        <p className="text-xs leading-relaxed text-wc-text-secondary">{body}</p>

        {(action || secondaryAction) && (
          <div className="mt-2 flex flex-wrap items-center justify-center gap-2">
            {action && (
              <button
                type="button"
                onClick={action.onClick}
                data-testid={`${testId}-action`}
                className="inline-flex items-center gap-1.5 rounded-md border border-wc-default bg-wc-surface-raised px-3 py-1.5 text-xs font-medium text-wc-text-primary transition-colors hover:bg-wc-surface-input focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2"
              >
                {action.icon}
                {action.label}
              </button>
            )}
            {secondaryAction && (
              <button
                type="button"
                onClick={secondaryAction.onClick}
                data-testid={`${testId}-secondary`}
                className="inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium text-wc-text-secondary underline-offset-2 transition-colors hover:text-wc-text-primary hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2"
              >
                {secondaryAction.label}
              </button>
            )}
          </div>
        )}

        {detail && (
          <div className="mt-2 w-full">
            <button
              type="button"
              onClick={() => { setDetailOpen((open) => !open); }}
              aria-expanded={detailOpen}
              data-testid={`${testId}-details-toggle`}
              className="text-[11px] font-medium text-wc-text-muted underline-offset-2 hover:text-wc-text-secondary hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2"
            >
              {detailOpen ? t(strings.messagesPane.state.hideDetails) : t(strings.messagesPane.state.showDetails)}
            </button>
            {detailOpen && (
              <pre
                data-testid={`${testId}-details`}
                className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-md border border-wc-default bg-wc-surface-input p-2 text-start text-[11px] leading-relaxed text-wc-text-muted"
              >
                {detail}
              </pre>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * A skeleton shaped like messages rather than a spinner, so the pane never
 * flashes an empty state on the way to content.
 */
export function MessagesPaneSkeleton() {
  const { t } = useTranslation();
  return (
    <div
      className="flex flex-col gap-5 p-4"
      data-testid="messages-state-loading"
      role="status"
      aria-live="polite"
      aria-label={t(strings.messagesPane.state.loading)}
    >
      {[0, 1, 2].map((row) => (
        <div key={row} className="flex flex-col gap-2">
          <div className="h-2.5 w-20 animate-pulse rounded bg-wc-surface-input" />
          <div className="h-2.5 w-full animate-pulse rounded bg-wc-surface-input" />
          <div className="h-2.5 w-11/12 animate-pulse rounded bg-wc-surface-input" />
          <div className="h-2.5 w-3/5 animate-pulse rounded bg-wc-surface-input" />
        </div>
      ))}
      <span className="sr-only">{t(strings.messagesPane.state.loading)}</span>
    </div>
  );
}
