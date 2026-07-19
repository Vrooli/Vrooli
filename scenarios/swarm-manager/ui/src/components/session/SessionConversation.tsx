import { useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ChatThread } from "../chat/ChatThread";
import { MessageComposer, type MessageComposerHandle } from "../composer/MessageComposer";
import { formatRelativeTime } from "../../lib/format-utils";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import type { CaptureAttachment } from "../../hooks/useIndexedDBAttachments";
import type { AgentSessionAttachment, AgentSessionContextType, AgentSessionKind, AgentSessionMessage, AgentSessionStatus } from "../../types";
import type { ChatMessageView } from "../chat/chat-types";
import type { SessionContextOption } from "./context/session-context-refs";
import { SessionContextPicker } from "./context/SessionContextPicker";
import { useStarterContextCounts } from "./context/useStarterContextCounts";
import { countForStarterCard, type StarterContextFilterKey } from "./context/starter-context-filters";
import {
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
}: SessionConversationProps) {
  const navigate = useNavigate();
  const composerRef = useRef<MessageComposerHandle>(null);
  const [contextPickerOpen, setContextPickerOpen] = useState(false);
  const [contextPickerInitialType, setContextPickerInitialType] = useState<AgentSessionContextType | null>(null);
  const [contextPickerFilterKey, setContextPickerFilterKey] = useState<StarterContextFilterKey | null>(null);
  const [imagePickerRequestKey, setImagePickerRequestKey] = useState(0);
  const isDraft = sessionStatus === "draft";
  const placeholder = isDraft
    ? draftPlaceholderForKind(sessionKind)
    : "Continue this session...";
  const showStarterSuggestions = isDraft && messages.length === 0;

  const sortedMessages = useMemo(
    () => [...messages].sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()),
    [messages],
  );

  const chatMessages: ChatMessageView[] = useMemo(
    () =>
      sortedMessages.map((message) => ({
        id: message.id,
        role: message.role,
        content: message.content,
        createdAt: message.createdAt,
        attachmentIds: message.attachmentIds,
        context: message.context,
      })),
    [sortedMessages],
  );

  const openContextPicker = (
    initialType: AgentSessionContextType | null = null,
    filterKey: StarterContextFilterKey | null = null,
  ) => {
    setContextPickerInitialType(initialType);
    setContextPickerFilterKey(filterKey);
    setContextPickerOpen(true);
  };

  return (
    <section
      className={cn(
        "flex min-h-0 flex-1 flex-col",
        variant === "desktop" && desktopPresentation === "card" && "rounded-lg border border-white/10 bg-slate-950/30",
        variant === "desktop" && desktopPresentation === "pane" && "h-full border-r border-white/10 bg-slate-950/30",
      )}
      data-testid="agent-session-conversation"
    >
      {variant === "desktop" && <div className="border-b border-white/10 px-3 py-2 text-xs font-medium text-slate-300">Conversation</div>}
      {showStarterSuggestions ? (
        <SessionStarterSuggestions
          sessionKind={sessionKind}
          pendingContext={pendingContext}
          pendingAttachmentCount={pendingAttachments.length}
          onChooseText={(text) => {
            // Through the composer handle so the user's undo history survives
            // the swap; falls back to a plain state set before first render.
            if (composerRef.current) composerRef.current.replaceText(text);
            else onDraftChange(text);
          }}
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
          className={cn("p-3", variant === "mobile" && "px-3 pb-40")}
          testId="agent-session-messages"
          onReferenceNavigate={(href) => navigate(href)}
          getMessageMeta={(message) => (
            <>
              <span>{message.role}</span>
              {message.createdAt && <span>{formatRelativeTime(message.createdAt)}</span>}
            </>
          )}
          renderAttachmentPreview={(message) => (
            <SessionMessageExtras message={message} attachments={attachments} />
          )}
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
          canSubmit={Boolean(draft.trim() || pendingAttachments.length > 0 || pendingContext.length > 0)}
          imagePickerRequestKey={imagePickerRequestKey}
          onTranscript={(text) => onDraftChange((draft ? draft.trimEnd() + " " : "") + text)}
        />
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

function SessionMessageExtras({ message, attachments }: { message: ChatMessageView; attachments: AgentSessionAttachment[] }) {
  const messageAttachments = (message.attachmentIds ?? [])
    .map((id) => attachments.find((attachment) => attachment.id === id))
    .filter((attachment): attachment is AgentSessionAttachment => Boolean(attachment));

  if ((message.context?.length ?? 0) === 0 && messageAttachments.length === 0) return null;

  return (
    <div className="mt-2 space-y-2">
      {message.context && message.context.length > 0 && (
        <div className="flex flex-wrap gap-1.5" data-testid={selectors.agentSessions.messageContextChips}>
          {message.context.map((item) => (
            <span
              key={`${item.type}:${item.ref}`}
              className="max-w-full truncate rounded border border-cyan-500/25 bg-cyan-500/10 px-2 py-1 text-[11px] text-cyan-100"
              title={item.summary || item.ref}
            >
              {contextLabel(item.type)} · {item.title}
            </span>
          ))}
        </div>
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
  onChooseText,
  onRequestContext,
  onRequestImage,
  className,
}: {
  sessionKind: AgentSessionKind;
  pendingContext: SessionContextOption[];
  pendingAttachmentCount: number;
  onChooseText: (value: string) => void;
  onRequestContext: (type: AgentSessionContextType, filterKey?: StarterContextFilterKey) => void;
  onRequestImage: () => void;
  className?: string;
}) {
  const suggestions = starterSuggestionsForKind(sessionKind);
  const { optionsByType, executions, loading } = useStarterContextCounts(sessionKind);

  return (
    <div className={cn("min-h-0 overflow-y-auto", className)} data-testid={selectors.agentSessions.starterSuggestions}>
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-2">
        <div className="px-1 pb-1">
          <h3 className="text-sm font-medium text-slate-200">Start this session</h3>
        </div>
        {suggestions.map((suggestion) => {
          const missing = firstMissingRequirement(suggestion, pendingContext, pendingAttachmentCount);
          const badge = starterCardBadgeSpec(suggestion);
          const badgeLoading = badge ? Boolean(loading[badge.type]) : false;
          const badgeCount = badge && !badgeLoading
            ? countForStarterCard({ optionsByType, executions, type: badge.type, filterKey: badge.filterKey })
            : null;
          // A required-context card whose backing set resolved to zero is a
          // dead-end click (empty picker) — disable it. Never disable while the
          // count is still loading, and never for soft/optional/text cards.
          const disabled = Boolean(badge?.gating && !badgeLoading && badgeCount === 0);
          const Icon = suggestion.icon;
          return (
            <button
              key={suggestion.id}
              type="button"
              disabled={disabled}
              aria-disabled={disabled}
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
                onChooseText(suggestion.text);
              }}
              className={cn(
                "group flex w-full items-start gap-3 rounded-md border px-3 py-3 text-left transition-colors",
                disabled
                  ? "cursor-not-allowed border-slate-800/60 bg-slate-950/25 opacity-55"
                  : "border-slate-800 bg-slate-950/45 hover:border-cyan-500/45 hover:bg-slate-900/70",
              )}
              data-testid={selectors.agentSessions.starterSuggestion}
            >
              <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-cyan-500/20 bg-cyan-500/10 text-cyan-200 transition-colors group-hover:border-cyan-400/45 group-hover:text-cyan-100">
                <Icon className="h-4 w-4" />
              </span>
              <span className="min-w-0 flex-1">
                <span className="block text-sm font-medium leading-5 text-slate-100">{suggestion.text}</span>
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
    case "initiative":
      return plural ? "initiatives" : "initiative";
    case "operating_mode":
      return plural ? "archived workflows" : "archived workflow";
    case "scenario":
      return plural ? "scenarios" : "scenario";
    default:
      return type.replace(/_/g, " ");
  }
}

function contextLabel(type: AgentSessionContextType): string {
  return type.replace(/_/g, " ");
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
