import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Activity,
  Gauge,
  GitPullRequestArrow,
  Image,
  Layers,
  ListTodo,
  Search,
  Workflow,
  type LucideIcon,
} from "lucide-react";
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
  const [contextPickerOpen, setContextPickerOpen] = useState(false);
  const [contextPickerInitialType, setContextPickerInitialType] = useState<AgentSessionContextType | null>(null);
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

  const openContextPicker = (initialType: AgentSessionContextType | null = null) => {
    setContextPickerInitialType(initialType);
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
          onChooseText={onDraftChange}
          onRequestContext={(type) => openContextPicker(type)}
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

type SuggestionRequirement =
  | { kind: "context"; type: AgentSessionContextType; optional?: boolean }
  | { kind: "image"; optional?: boolean };

interface StarterSuggestion {
  id: string;
  icon: LucideIcon;
  text: string;
  detail?: string;
  requirements?: SuggestionRequirement[];
}

function SessionStarterSuggestions({
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
  onRequestContext: (type: AgentSessionContextType) => void;
  onRequestImage: () => void;
  className?: string;
}) {
  const suggestions = starterSuggestionsForKind(sessionKind);

  return (
    <div className={cn("min-h-0 overflow-y-auto", className)} data-testid={selectors.agentSessions.starterSuggestions}>
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-2">
        <div className="px-1 pb-1">
          <h3 className="text-sm font-medium text-slate-200">Start this session</h3>
        </div>
        {suggestions.map((suggestion) => {
          const missing = firstMissingRequirement(suggestion, pendingContext, pendingAttachmentCount);
          const Icon = suggestion.icon;
          return (
            <button
              key={suggestion.id}
              type="button"
              onClick={() => {
                if (missing?.kind === "context") {
                  onRequestContext(missing.type);
                  return;
                }
                if (missing?.kind === "image") {
                  onRequestImage();
                  return;
                }
                onChooseText(suggestion.text);
              }}
              className="group flex w-full items-start gap-3 rounded-md border border-slate-800 bg-slate-950/45 px-3 py-3 text-left transition-colors hover:border-cyan-500/45 hover:bg-slate-900/70"
              data-testid={selectors.agentSessions.starterSuggestion}
            >
              <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-cyan-500/20 bg-cyan-500/10 text-cyan-200 transition-colors group-hover:border-cyan-400/45 group-hover:text-cyan-100">
                <Icon className="h-4 w-4" />
              </span>
              <span className="min-w-0 flex-1">
                <span className="block text-sm font-medium leading-5 text-slate-100">{suggestion.text}</span>
                {suggestion.detail && <span className="mt-0.5 block text-xs leading-5 text-slate-400">{suggestion.detail}</span>}
                <span className="mt-2 flex flex-wrap gap-1.5">
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

function starterSuggestionsForKind(kind: AgentSessionKind): StarterSuggestion[] {
  switch (kind) {
    case "swarm_operations":
      return [
        {
          id: "operations-review",
          icon: Gauge,
          text: "Review active initiatives and recommend the top next action.",
        },
        {
          id: "operations-decisions",
          icon: ListTodo,
          text: "Help me drain workshop decisions for a backlog item.",
          requirements: [{ kind: "context", type: "backlog_item" }],
        },
        {
          id: "operations-run",
          icon: Activity,
          text: "Review a failed or stale run and recommend recovery.",
          requirements: [{ kind: "context", type: "execution" }, { kind: "context", type: "agent_activity", optional: true }],
        },
        {
          id: "operations-initiative",
          icon: Layers,
          text: "Assess an initiative and recommend the best operating mode.",
          requirements: [{ kind: "context", type: "initiative" }],
        },
      ];
    case "operating_mode_authoring":
      return [
        {
          id: "mode-classify",
          icon: Search,
          text: "Classify whether this workflow deserves a new operating mode.",
        },
        {
          id: "mode-draft",
          icon: GitPullRequestArrow,
          text: "Draft a mode proposal with phases, artifacts, metrics, and tests.",
        },
        {
          id: "mode-compare",
          icon: Gauge,
          text: "Compare this workflow against an existing operating mode.",
          requirements: [{ kind: "context", type: "operating_mode" }],
        },
        {
          id: "mode-initiative",
          icon: Layers,
          text: "Design an operating mode for this initiative's workflow.",
          requirements: [{ kind: "context", type: "initiative" }],
        },
      ];
    case "meta_orchestration":
    default:
      return [
        {
          id: "meta-plan",
          icon: Workflow,
          text: "Turn this idea into initiatives and backlog items.",
        },
        {
          id: "meta-existing",
          icon: Search,
          text: "Inspect existing Swarm context first, then propose a plan.",
          requirements: [{ kind: "context", type: "initiative", optional: true }, { kind: "context", type: "scenario", optional: true }],
        },
        {
          id: "meta-backlog",
          icon: ListTodo,
          text: "Plan follow-up work for a backlog item.",
          requirements: [{ kind: "context", type: "backlog_item" }],
        },
        {
          id: "meta-image",
          icon: Image,
          text: "Use an image or whiteboard as source material for backlog candidates.",
          requirements: [{ kind: "image" }],
        },
      ];
  }
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
      return "Mode";
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
