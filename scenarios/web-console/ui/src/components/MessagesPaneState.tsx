import { useTranslation } from "react-i18next";
import { AlertTriangle, MessageSquare, PlugZap, RefreshCw, Terminal } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Skeleton } from "@vrooli/react-component-library/Skeleton/1";
import { strings } from "../consts/strings";
import type { ConversationViewState } from "../stores/useConversationStore";
import { PaneEmptyState } from "./PaneEmptyState";

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
  const emptyTitle = pending ? t(strings.messagesPane.state.pendingTitle) : t(strings.messagesPane.state.emptyTitle);
  const summary = view.capture.summary?.trim() ?? "";
  return (
    <StateCard
      tone="empty"
      testId="messages-state-empty"
      icon={<MessageSquare className="h-6 w-6" aria-hidden="true" />}
      title={emptyTitle}
      // A capture summary that only restates the title is not an explanation.
      body={summary && summary !== emptyTitle && !summary.startsWith(emptyTitle) ? summary : t(strings.messagesPane.state.emptyBody)}
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
  const actions = action || secondaryAction ? (
    <>
      {action && (
        <Button
          type="button"
          variant={tone === "alert" ? "primary" : "secondary"}
          size="sm"
          data-testid={`${testId}-action`}
          onClick={action.onClick}
        >
          {action.icon}
          {action.label}
        </Button>
      )}
      {secondaryAction && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          data-testid={`${testId}-secondary`}
          onClick={secondaryAction.onClick}
        >
          {secondaryAction.label}
        </Button>
      )}
    </>
  ) : undefined;

  return (
    <PaneEmptyState
      testId={testId}
      tone={tone === "alert" ? "alert" : "neutral"}
      icon={icon}
      title={title}
      description={body}
      detail={detail}
      detailShowLabel={t(strings.messagesPane.state.showDetails)}
      detailHideLabel={t(strings.messagesPane.state.hideDetails)}
      actions={actions}
    />
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
          <Skeleton aria-hidden="true" className="min-h-0 animate-pulse motion-reduce:animate-none h-2.5 w-20 rounded" />
          <Skeleton aria-hidden="true" className="min-h-0 animate-pulse motion-reduce:animate-none h-2.5 w-full rounded" />
          <Skeleton aria-hidden="true" className="min-h-0 animate-pulse motion-reduce:animate-none h-2.5 w-11/12 rounded" />
          <Skeleton aria-hidden="true" className="min-h-0 animate-pulse motion-reduce:animate-none h-2.5 w-3/5 rounded" />
        </div>
      ))}
      <span className="sr-only">{t(strings.messagesPane.state.loading)}</span>
    </div>
  );
}
