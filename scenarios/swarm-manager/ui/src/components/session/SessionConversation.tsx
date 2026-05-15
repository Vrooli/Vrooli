import { useMemo, useState } from "react";
import { ChatThread } from "../chat/ChatThread";
import { MessageComposer } from "../composer/MessageComposer";
import { formatRelativeTime } from "../../lib/format-utils";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import type { CaptureAttachment } from "../../hooks/useIndexedDBAttachments";
import type { AgentSessionAttachment, AgentSessionContextType, AgentSessionKind, AgentSessionMessage, AgentSessionStatus } from "../../types";
import type { ChatMessageView } from "../chat/chat-types";
import type { SessionContextOption } from "./context/session-context-refs";
import { SessionContextPicker } from "./context/SessionContextPicker";

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
}: SessionConversationProps) {
  const [contextPickerOpen, setContextPickerOpen] = useState(false);
  const isDraft = sessionStatus === "draft";
  const placeholder = isDraft
    ? draftPlaceholderForKind(sessionKind)
    : "Continue this session...";

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

  return (
    <section
      className={cn(
        "flex min-h-0 flex-1 flex-col",
        variant === "desktop" && "rounded-lg border border-white/10 bg-slate-950/30",
      )}
      data-testid="agent-session-conversation"
    >
      {variant === "desktop" && <div className="border-b border-white/10 px-3 py-2 text-xs font-medium text-slate-300">Conversation</div>}
      <ChatThread
        messages={chatMessages}
        isWaiting={!isDraft && isWaitingForAgent}
        emptyLabel={isDraft ? "Start with the real context you want the agent to use." : "No messages recorded yet."}
        accent="cyan"
        className={cn("p-3", variant === "mobile" && "px-3 pb-40")}
        testId="agent-session-messages"
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
      <div
        className={cn(
          "border-t border-white/10 p-3",
          variant === "mobile" && "fixed inset-x-0 bottom-0 z-40 bg-slate-950/95 pb-[calc(0.75rem+env(safe-area-inset-bottom))] pl-[calc(1rem+env(safe-area-inset-left))] pr-[calc(1rem+env(safe-area-inset-right))] pt-2 backdrop-blur",
        )}
      >
        <MessageComposer
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
          onOpenContextPicker={() => setContextPickerOpen(true)}
          onRemoveContext={(type, ref) => onPendingContextChange(pendingContext.filter((item) => !(item.type === type && item.ref === ref)))}
          canSubmit={Boolean(draft.trim() || pendingAttachments.length > 0 || pendingContext.length > 0)}
        />
        <SessionContextPicker
          isOpen={contextPickerOpen}
          onClose={() => setContextPickerOpen(false)}
          sessionKind={sessionKind}
          selected={pendingContext}
          onApply={onPendingContextChange}
          currentSessionId={sessionId}
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

function contextLabel(type: AgentSessionContextType): string {
  return type.replace(/_/g, " ");
}

function draftPlaceholderForKind(kind: AgentSessionKind): string {
  switch (kind) {
    case "operating_mode_authoring":
      return "Describe the recurring agent workflow you want to author...";
    case "swarm_operations":
      return "Ask what to review, unblock, decide, or move forward in Swarm Manager...";
    case "meta_orchestration":
    default:
      return "Describe what you want to plan...";
  }
}
