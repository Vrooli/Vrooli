import { useCallback, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ExternalLink, MessagesSquare, Pencil, RotateCcw, Rows3 } from "lucide-react";
import { ChatThread } from "../chat/ChatThread";
import { MessageComposer, type MessageComposerHandle } from "../composer/MessageComposer";
import { ContextChipTray } from "../composer/ContextChipTray";
import { SessionPromptPreview } from "./SessionPromptPreview";
import { formatRelativeTime } from "../../lib/format-utils";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import { CHAT_DENSITIES, CHAT_DENSITY_DESCRIPTIONS, CHAT_DENSITY_LABELS, type ChatDensity } from "../../lib/chat-density";

/** Icon per density. Bubbles reads as a conversation; compact reads as rows. */
const CHAT_DENSITY_ICONS: Record<ChatDensity, typeof MessagesSquare> = {
  comfortable: MessagesSquare,
  compact: Rows3,
};
import { useAgentProfileUrl, useAgentRunUrl } from "../../services/external-links";
import type { CaptureAttachment } from "../../hooks/useIndexedDBAttachments";
import type { AgentSessionAttachment, AgentSessionContextType, AgentSessionKind, AgentSessionMessage, AgentSessionStatus } from "../../types";
import type { ChatMessageView } from "../chat/chat-types";
import type { SessionContextOption } from "./context/session-context-refs";
import { SessionContextPicker } from "./context/SessionContextPicker";
import { useStarterContextCounts } from "./context/useStarterContextCounts";
import { countForStarterCard, type StarterContextFilterKey } from "./context/starter-context-filters";
import {
  composerSeedForOpener,
  isUnfilledOpener,
  starterCardBadgeSpec,
  starterSuggestionsForKind,
  type StarterSuggestion,
  type SuggestionRequirement,
} from "./session-starter-suggestions";

interface SessionConversationProps {
  messages: AgentSessionMessage[];
  draft: string;
  onDraftChange: (value: string) => void;
  onSend: () => void;
  isMutating: boolean;
  isWaitingForAgent: boolean;
  sessionKind: AgentSessionKind;
  sessionStatus: AgentSessionStatus;
  sessionId: string;
  attachments: AgentSessionAttachment[];
  pendingAttachments: CaptureAttachment[];
  onAttachFiles: (files: File[]) => void;
  onRemovePendingAttachment: (id: string) => void;
  pendingContext: SessionContextOption[];
  onPendingContextChange: (items: SessionContextOption[]) => void;
  variant?: "desktop" | "mobile";
  desktopPresentation?: "card" | "pane";
  density: ChatDensity;
  onDensityChange: (density: ChatDensity) => void;
  /** Agent-manager profile driving this session's run, when one is attributed. */
  profileKey?: string;
  runId?: string;
  /** Locally-composed message awaiting (or failed) delivery. */
  pendingMessage?: ChatMessageView;
  onRetryPendingMessage?: () => void;
  onEditPendingMessage?: () => void;
  onStarterJobChange?: (jobId: string | undefined, opener?: string) => void;
  starterJobId?: string;
}

export function SessionConversation({
  messages,
  draft,
  onDraftChange,
  onSend,
  isMutating,
  isWaitingForAgent,
  sessionKind,
  sessionStatus,
  sessionId,
  attachments,
  pendingAttachments,
  onAttachFiles,
  onRemovePendingAttachment,
  pendingContext,
  onPendingContextChange,
  variant = "desktop",
  desktopPresentation = "card",
  density,
  onDensityChange,
  profileKey,
  runId,
  pendingMessage,
  onRetryPendingMessage,
  onEditPendingMessage,
  onStarterJobChange,
  starterJobId,
}: SessionConversationProps) {
  const navigate = useNavigate();
  const composerRef = useRef<MessageComposerHandle>(null);
  const [contextPickerOpen, setContextPickerOpen] = useState(false);
  const [contextPickerInitialType, setContextPickerInitialType] = useState<AgentSessionContextType | null>(null);
  const [contextPickerFilterKey, setContextPickerFilterKey] = useState<StarterContextFilterKey | null>(null);
  const [imagePickerRequestKey, setImagePickerRequestKey] = useState(0);
  const isDraft = sessionStatus === "draft";
  const placeholder = isDraft
    ? (starterJobId ? "Add any specific context or direction, or send the selected job as-is..." : draftPlaceholderForKind(sessionKind))
    : "Continue this session...";
  const showStarterSuggestions = isDraft && messages.length === 0;

  const sortedMessages = useMemo(
    () => [...messages].sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()),
    [messages],
  );

  const chatMessages: ChatMessageView[] = useMemo(() => {
    const confirmed = sortedMessages.map((message) => ({
      id: message.id,
      role: message.role,
      content: message.content,
      createdAt: message.createdAt,
      attachmentIds: message.attachmentIds,
      context: message.context,
    }));
    // The optimistic message always tails the transcript: it is newer than
    // anything the server has confirmed, by construction.
    return pendingMessage ? [...confirmed, pendingMessage] : confirmed;
  }, [sortedMessages, pendingMessage]);

  /**
   * Choosing a starter card selects its server-owned job, and — when the card
   * needs the operator's own material — seeds the composer with its opener.
   *
   * The seed goes through `replaceText` rather than a controlled set so it
   * lands on the browser's undo stack: a mis-clicked card is one Ctrl+Z away,
   * and the caret ends up on the blank line below the invitation.
   *
   * The composer is only ours to overwrite while it is empty or still holds an
   * unfilled invitation. Once the operator has typed, switching cards changes
   * the job and leaves their words alone.
   */
  const handleChooseJob = (jobId: string, opener?: string) => {
    const clearing = jobId === starterJobId;
    onStarterJobChange?.(clearing ? undefined : jobId, clearing ? undefined : opener);
    if (draft.trim() !== "" && !isUnfilledOpener(sessionKind, draft)) return;
    composerRef.current?.replaceText(clearing || !opener ? "" : composerSeedForOpener(opener));
  };

  const openContextPicker = (
    initialType: AgentSessionContextType | null = null,
    filterKey: StarterContextFilterKey | null = null,
  ) => {
    setContextPickerInitialType(initialType);
    setContextPickerFilterKey(filterKey);
    setContextPickerOpen(true);
  };

  // ChatThread hands these to every bubble, and the bubbles are memoized, so an
  // inline arrow here would re-render the whole transcript on each 3s poll and
  // undo that memoization. `attachments` is the only value any of them close
  // over that changes during a session.
  const handleReferenceNavigate = useCallback((href: string) => navigate(href), [navigate]);

  const getMessageMeta = useCallback((message: ChatMessageView) => (
    <>
      <span>{message.role}</span>
      {message.createdAt && <span>{formatRelativeTime(message.createdAt)}</span>}
    </>
  ), []);

  const renderAttachmentPreview = useCallback((message: ChatMessageView) => (
    <SessionMessageExtras message={message} attachments={attachments} onOpenContext={(path) => navigate(path)} />
  ), [attachments, navigate]);

  const renderMessageActions = useCallback((message: ChatMessageView) => (
    message.delivery === "failed"
      ? <PendingMessageActions onRetry={onRetryPendingMessage} onEdit={onEditPendingMessage} />
      : null
  ), [onRetryPendingMessage, onEditPendingMessage]);

  return (
    <section
      className={cn(
        "flex min-h-0 flex-1 flex-col",
        variant === "desktop" && desktopPresentation === "card" && "rounded-lg border border-white/10 bg-slate-950/30",
        variant === "desktop" && desktopPresentation === "pane" && "h-full border-r border-white/10 bg-slate-950/30",
      )}
      data-testid="agent-session-conversation"
    >
      <ConversationToolbar
        profileKey={profileKey}
        runId={runId}
        density={density}
        onDensityChange={onDensityChange}
      />
      {showStarterSuggestions ? (
        <SessionStarterSuggestions
          sessionKind={sessionKind}
          pendingContext={pendingContext}
          pendingAttachmentCount={pendingAttachments.length}
          selectedJobId={starterJobId}
          // Re-clicking the selected card clears it. Without a way back out,
          // the only escape from a mis-clicked job was to abandon the draft.
          onChooseJob={handleChooseJob}
          onRequestContext={(type, filterKey) => openContextPicker(type, filterKey)}
          onRequestImage={() => setImagePickerRequestKey((value) => value + 1)}
          className={cn("flex-1 p-3", variant === "mobile" && "px-3 pb-40")}
        />
      ) : (
        <ChatThread
          messages={chatMessages}
          isWaiting={!isDraft && isWaitingForAgent}
          emptyLabel={isDraft ? "Start with the real context you want the agent to use." : "No messages recorded yet."}
          accent="cyan"
          density={density}
          className={cn(density === "comfortable" && "p-3", variant === "mobile" && "pb-40")}
          testId="agent-session-messages"
          onReferenceNavigate={handleReferenceNavigate}
          getMessageMeta={getMessageMeta}
          renderAttachmentPreview={renderAttachmentPreview}
          renderMessageActions={renderMessageActions}
        />
      )}
      <div
        className={cn(
          "border-t border-white/10 p-3",
          variant === "mobile" && "fixed inset-x-0 bottom-0 z-40 bg-slate-950/95 pb-[calc(0.75rem+env(safe-area-inset-bottom))] pl-[calc(1rem+env(safe-area-inset-left))] pr-[calc(1rem+env(safe-area-inset-right))] pt-2 backdrop-blur",
        )}
      >
        <MessageComposer
          ref={composerRef}
          value={draft}
          onChange={onDraftChange}
          onSubmit={onSend}
          disabled={isMutating}
          isSubmitting={isMutating}
          placeholder={placeholder}
          submitLabel="Send"
          testId={selectors.agentSessions.composer}
          attachTestId={selectors.agentSessions.composerImageAttach}
          contextTestId={selectors.agentSessions.composerContextAttach}
          attachments={pendingAttachments}
          onAttachFiles={onAttachFiles}
          onRemoveAttachment={onRemovePendingAttachment}
          contextItems={pendingContext}
          onOpenContextPicker={() => openContextPicker(null)}
          onRemoveContext={(type, ref) => onPendingContextChange(pendingContext.filter((item) => !(item.type === type && item.ref === ref)))}
          onOpenContext={(path) => navigate(path)}
          // A selected starter job is a complete message on its own — the
          // server composes its instruction into the job band. Omitting it
          // here left every job-only start with a disabled Send button, so
          // choosing a card appeared to do nothing at all.
          canSubmit={Boolean(draft.trim() || pendingAttachments.length > 0 || pendingContext.length > 0 || starterJobId)}
          imagePickerRequestKey={imagePickerRequestKey}
          onTranscript={(text) => onDraftChange((draft ? draft.trimEnd() + " " : "") + text)}
        />
        {/* Sits under the composer so the operator can read exactly what this
            message would send before sending it. */}
        <div className="mt-1 flex justify-end">
          <SessionPromptPreview
            sessionId={sessionId}
          message={draft}
          contextRefs={pendingContext.map((item) => ({ type: item.type, ref: item.ref }))}
            starterJobId={starterJobId}
            disabled={isMutating}
          />
        </div>
        <SessionContextPicker
          isOpen={contextPickerOpen}
          onClose={() => setContextPickerOpen(false)}
          sessionKind={sessionKind}
          selected={pendingContext}
          onApply={onPendingContextChange}
          currentSessionId={sessionId}
          initialType={contextPickerInitialType}
          initialFilterKey={contextPickerFilterKey}
        />
      </div>
    </section>
  );
}

/**
 * The conversation's own header: who you are talking to, and how the
 * transcript is displayed.
 *
 * The profile was previously only visible in the inspector's third tab, so
 * there was no way to tell which agent configuration a session was running
 * under while reading it. The profile key is what Swarm Manager actually
 * knows; the model itself is defined on the profile in agent-manager, which is
 * where the link goes.
 */
function ConversationToolbar({
  profileKey,
  runId,
  density,
  onDensityChange,
}: {
  profileKey?: string;
  runId?: string;
  density: ChatDensity;
  onDensityChange: (density: ChatDensity) => void;
}) {
  const profileUrl = useAgentProfileUrl(profileKey);
  const runUrl = useAgentRunUrl(runId);

  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-b border-white/10 px-3 py-2" data-testid="agent-session-conversation-toolbar">
      <div className="flex min-w-0 flex-wrap items-center gap-1.5 text-xs">
        <span className="font-medium text-slate-300">Conversation</span>
        {profileKey && (
          <ToolbarLink
            href={profileUrl}
            label="Profile"
            value={profileKey}
            testId="agent-session-conversation-profile"
          />
        )}
        {runId && (
          <ToolbarLink href={runUrl} label="Run" value={runId} testId="agent-session-conversation-run" />
        )}
      </div>
      <div
        className="flex shrink-0 items-center rounded-md border border-white/10 bg-slate-950/50 p-0.5"
        role="group"
        aria-label="Conversation display density"
      >
        {CHAT_DENSITIES.map((option) => {
          const DensityIcon = CHAT_DENSITY_ICONS[option];
          return (
            <button
              key={option}
              type="button"
              onClick={() => onDensityChange(option)}
              aria-pressed={density === option}
              aria-label={CHAT_DENSITY_LABELS[option]}
              title={CHAT_DENSITY_DESCRIPTIONS[option]}
              className={cn(
                "rounded p-1 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50",
                density === option ? "bg-cyan-500/15 text-cyan-200" : "text-slate-400 hover:text-slate-200",
              )}
              data-testid={`agent-session-density-${option}`}
            >
              <DensityIcon className="h-3.5 w-3.5" aria-hidden="true" />
            </button>
          );
        })}
      </div>
    </div>
  );
}

/** A labelled metadata chip that becomes a deep link once its target URL
 * resolves; cross-scenario URLs load asynchronously and can be unavailable. */
function ToolbarLink({
  href,
  label,
  value,
  testId,
}: {
  href: string | null;
  label: string;
  value: string;
  testId: string;
}) {
  const content = (
    <>
      <span className="text-slate-500">{label}</span>
      <span className="min-w-0 truncate font-medium">{value}</span>
      {href && <ExternalLink className="h-3 w-3 shrink-0" aria-hidden />}
    </>
  );
  const className = "inline-flex min-w-0 max-w-[16rem] items-center gap-1 rounded border px-1.5 py-0.5 text-[11px]";
  if (!href) {
    return (
      <span className={cn(className, "border-white/10 bg-slate-900/60 text-slate-300")} data-testid={testId}>
        {content}
      </span>
    );
  }
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      aria-label={`Open ${label.toLowerCase()} ${value} in a new tab`}
      className={cn(
        className,
        "border-cyan-500/20 bg-cyan-500/5 text-cyan-300 transition-colors hover:border-cyan-400/40 hover:bg-cyan-500/10 hover:text-cyan-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50",
      )}
      data-testid={testId}
    >
      {content}
    </a>
  );
}

/** Recovery controls on a message the server never accepted. Retry re-sends it
 * verbatim; Edit returns it to the composer so it can be reworded. */
function PendingMessageActions({ onRetry, onEdit }: { onRetry?: () => void; onEdit?: () => void }) {
  if (!onRetry && !onEdit) return null;
  return (
    <span className="mt-1 inline-flex items-center gap-1">
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="inline-flex items-center gap-1 rounded border border-rose-400/30 px-1.5 py-0.5 text-[11px] font-medium text-rose-200 transition-colors hover:border-rose-300/50 hover:bg-rose-500/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-rose-500/50"
          data-testid="agent-session-message-retry"
        >
          <RotateCcw className="h-3 w-3" />
          Retry
        </button>
      )}
      {onEdit && (
        <button
          type="button"
          onClick={onEdit}
          className="inline-flex items-center gap-1 rounded border border-white/10 px-1.5 py-0.5 text-[11px] font-medium text-slate-300 transition-colors hover:border-white/20 hover:bg-white/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50"
          data-testid="agent-session-message-edit"
        >
          <Pencil className="h-3 w-3" />
          Edit
        </button>
      )}
    </span>
  );
}

function SessionMessageExtras({
  message,
  attachments,
  onOpenContext,
}: {
  message: ChatMessageView;
  attachments: AgentSessionAttachment[];
  onOpenContext: (path: string) => void;
}) {
  const messageAttachments = (message.attachmentIds ?? [])
    .map((id) => attachments.find((attachment) => attachment.id === id))
    .filter((attachment): attachment is AgentSessionAttachment => Boolean(attachment));

  if ((message.context?.length ?? 0) === 0 && messageAttachments.length === 0) return null;

  return (
    <div className="mt-2 space-y-2">
      {message.context && message.context.length > 0 && (
        <ContextChipTray
          items={message.context.map((item) => ({
            type: item.type,
            ref: item.ref,
            title: item.title,
            subtitle: item.summary,
            nodeId: item.nodeId,
          }))}
          onOpen={onOpenContext}
          constrainHeight={false}
          testId={selectors.agentSessions.messageContextChips}
        />
      )}
      {messageAttachments.length > 0 && (
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3" data-testid={selectors.agentSessions.messageImageThumbnails}>
          {messageAttachments.map((attachment) => (
            <a
              key={attachment.id}
              href={attachment.url}
              target="_blank"
              rel="noreferrer"
              className="overflow-hidden rounded border border-white/10 bg-slate-950/60"
              title={attachment.filename}
            >
              <img src={attachment.url} alt={attachment.filename} className="h-24 w-full object-cover" loading="lazy" />
            </a>
          ))}
        </div>
      )}
    </div>
  );
}

export function SessionStarterSuggestions({
  sessionKind,
  pendingContext,
  pendingAttachmentCount,
  selectedJobId,
  onChooseJob,
  onRequestContext,
  onRequestImage,
  className,
}: {
  sessionKind: AgentSessionKind;
  pendingContext: SessionContextOption[];
  pendingAttachmentCount: number;
  selectedJobId?: string;
  onChooseJob: (jobId: string, opener?: string) => void;
  onRequestContext: (type: AgentSessionContextType, filterKey?: StarterContextFilterKey) => void;
  onRequestImage: () => void;
  className?: string;
}) {
  const suggestions = starterSuggestionsForKind(sessionKind);
  const selectedSuggestion = suggestions.find((item) => (item.jobId ?? item.id) === selectedJobId);
  const { optionsByType, executions, backlogItems, loading } = useStarterContextCounts(sessionKind);

  return (
    <div className={cn("min-h-0 overflow-y-auto", className)} data-testid={selectors.agentSessions.starterSuggestions}>
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-2">
        <div className="px-1 pb-1">
          <h3 className="text-sm font-medium text-slate-200">Start this session</h3>
          <p className="mt-0.5 text-xs leading-5 text-slate-400">
            {!selectedJobId
              ? "Pick a starting job, or just type your own message below."
              : selectedSuggestion?.opener
                ? "Job selected. Finish the line in the composer below."
                : "Job selected. Send it as-is, or add your own context first."}
          </p>
        </div>
        {suggestions.map((suggestion) => {
          const missing = firstMissingRequirement(suggestion, pendingContext, pendingAttachmentCount);
          const badge = starterCardBadgeSpec(suggestion);
          const badgeLoading = badge ? Boolean(loading[badge.type]) : false;
          const badgeCount = badge && !badgeLoading
            ? countForStarterCard({ optionsByType, executions, backlogItems, type: badge.type, filterKey: badge.filterKey })
            : null;
          // A required-context card whose backing set resolved to zero is a
          // dead-end click (empty picker) — disable it. Never disable while the
          // count is still loading, and never for soft/optional/text cards.
          const disabled = Boolean(badge?.gating && !badgeLoading && badgeCount === 0);
          const selected = (suggestion.jobId ?? suggestion.id) === selectedJobId;
          const Icon = suggestion.icon;
          return (
            <button
              key={suggestion.id}
              type="button"
              disabled={disabled}
              aria-disabled={disabled}
              aria-pressed={selected}
              onClick={() => {
                if (disabled) return;
                if (missing?.kind === "context") {
                  onRequestContext(missing.type, missing.filterKey);
                  return;
                }
                if (missing?.kind === "image") {
                  onRequestImage();
                  return;
                }
                onChooseJob(suggestion.jobId ?? suggestion.id, suggestion.opener);
              }}
              className={cn(
                "group flex w-full items-start gap-3 rounded-md border px-3 py-3 text-left transition-colors",
                disabled
                  ? "cursor-not-allowed border-slate-800/60 bg-slate-950/25 opacity-55"
                  : selected
                    ? "border-cyan-500/70 bg-cyan-500/10"
                    : "border-slate-800 bg-slate-950/45 hover:border-cyan-500/45 hover:bg-slate-900/70",
              )}
              data-testid={selectors.agentSessions.starterSuggestion}
            >
              <span
                className={cn(
                  "mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-md border transition-colors",
                  selected
                    ? "border-cyan-400/60 bg-cyan-500/20 text-cyan-100"
                    : "border-cyan-500/20 bg-cyan-500/10 text-cyan-200 group-hover:border-cyan-400/45 group-hover:text-cyan-100",
                )}
              >
                <Icon className="h-4 w-4" />
              </span>
              <span className="min-w-0 flex-1">
                <span className="flex items-center gap-2">
                  <span className="min-w-0 flex-1 text-sm font-medium leading-5 text-slate-100">{suggestion.label}</span>
                  {selected && (
                    <span className="shrink-0 rounded border border-cyan-500/40 bg-cyan-500/15 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-cyan-200">
                      Selected
                    </span>
                  )}
                </span>
                {suggestion.detail && <span className="mt-0.5 block text-xs leading-5 text-slate-400">{suggestion.detail}</span>}
                <span className="mt-2 flex flex-wrap items-center gap-1.5">
                  {badge && <StarterCountChip type={badge.type} count={badgeCount} loading={badgeLoading} />}
                  {requirementChips(suggestion, pendingContext, pendingAttachmentCount)}
                </span>
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function firstMissingRequirement(
  suggestion: StarterSuggestion,
  pendingContext: SessionContextOption[],
  pendingAttachmentCount: number,
): SuggestionRequirement | undefined {
  return (suggestion.requirements ?? []).find((requirement) => {
    if (requirement.optional) return false;
    return !isRequirementSatisfied(requirement, pendingContext, pendingAttachmentCount);
  });
}

function requirementChips(
  suggestion: StarterSuggestion,
  pendingContext: SessionContextOption[],
  pendingAttachmentCount: number,
) {
  const requirements = suggestion.requirements ?? [];
  if (requirements.length === 0) {
    return [<RequirementChip key="ready" label="Ready" state="ready" />];
  }
  return requirements.map((requirement) => {
    const satisfied = isRequirementSatisfied(requirement, pendingContext, pendingAttachmentCount);
    return (
      <RequirementChip
        key={requirement.kind === "context" ? requirement.type : "image"}
        label={requirementLabel(requirement, satisfied)}
        state={satisfied ? "ready" : requirement.optional ? "optional" : "missing"}
      />
    );
  });
}

function isRequirementSatisfied(
  requirement: SuggestionRequirement,
  pendingContext: SessionContextOption[],
  pendingAttachmentCount: number,
): boolean {
  if (requirement.kind === "image") return pendingAttachmentCount > 0;
  return pendingContext.some((item) => item.type === requirement.type);
}

function requirementLabel(requirement: SuggestionRequirement, satisfied: boolean): string {
  if (requirement.kind === "image") return satisfied ? "Image attached" : "Attach image";
  const label = contextRequirementLabel(requirement.type);
  if (satisfied) return `${label} attached`;
  return requirement.optional ? `${label} optional` : `Select ${label.toLowerCase()}`;
}

function contextRequirementLabel(type: AgentSessionContextType): string {
  switch (type) {
    case "agent_activity":
      return "Activity";
    case "backlog_item":
      return "Backlog";
    case "operating_mode":
      return "Archived workflow";
    default:
      return type.replace(/_/g, " ").replace(/\b\w/g, (char) => char.toUpperCase());
  }
}

function RequirementChip({ label, state }: { label: string; state: "ready" | "missing" | "optional" }) {
  return (
    <span
      className={cn(
        "rounded border px-1.5 py-0.5 text-[11px] font-medium leading-4",
        state === "ready" && "border-cyan-400/30 bg-cyan-400/10 text-cyan-100",
        state === "missing" && "border-amber-300/30 bg-amber-300/10 text-amber-100",
        state === "optional" && "border-slate-600 bg-slate-800/55 text-slate-400",
      )}
    >
      {label}
    </span>
  );
}

function StarterCountChip({ type, count, loading }: { type: AgentSessionContextType; count: number | null; loading: boolean }) {
  if (loading) {
    return (
      <span
        className="inline-flex h-[18px] w-14 animate-pulse rounded border border-slate-700 bg-slate-800/60"
        data-testid={selectors.agentSessions.starterSuggestionCount}
        data-loading="true"
        aria-hidden="true"
      />
    );
  }
  const value = count ?? 0;
  const zero = value === 0;
  return (
    <span
      className={cn(
        "rounded border px-1.5 py-0.5 text-[11px] font-medium leading-4 tabular-nums",
        zero ? "border-slate-700 bg-slate-800/50 text-slate-500" : "border-cyan-400/30 bg-cyan-400/10 text-cyan-100",
      )}
      data-testid={selectors.agentSessions.starterSuggestionCount}
      data-count={value}
    >
      {value} {countNoun(type, value)}
    </span>
  );
}

function countNoun(type: AgentSessionContextType, count: number): string {
  const plural = count !== 1;
  switch (type) {
    case "backlog_item":
      return plural ? "backlog items" : "backlog item";
    case "execution":
      return plural ? "runs" : "run";
    case "goal":
      return plural ? "goals" : "goal";
    case "operating_mode":
      return plural ? "archived workflows" : "archived workflow";
    case "scenario":
      return plural ? "scenarios" : "scenario";
    default:
      return type.replace(/_/g, " ");
  }
}

function draftPlaceholderForKind(kind: AgentSessionKind): string {
  switch (kind) {
    case "operating_mode_authoring":
      return "This archived session is read-only.";
    case "swarm_operations":
      return "Ask what to review, unblock, decide, or move forward in Swarm Manager...";
	case "workflow_authoring":
		return "Describe how you want to work with coding agents...";
    case "meta_orchestration":
    default:
      return "Describe what you want to plan...";
  }
}
