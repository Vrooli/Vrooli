import { memo, useState, forwardRef, useCallback, useEffect } from "react";
import {
  Loader2, Copy, Volume2, VolumeX, RefreshCw, Pencil, Trash2, GitBranch,
} from "lucide-react";
import type { Message, ToolCallRecord } from "../../lib/api";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import type { ViewMode } from "../settings/Settings";
import { useToast } from "../ui/toast";
import { VersionPicker } from "./VersionPicker";
import { getPreviousSibling, getNextSibling } from "../../lib/messageTree";
import { MarkdownRenderer } from "../markdown";
import { ActionButton } from "./ActionButton";
import { CompactMessageBubble } from "./CompactMessageBubble";
import { BubbleMessageBubble } from "./BubbleMessageBubble";

export interface MessageBubbleProps {
  message: Message;
  viewMode: ViewMode;
  allMessages: Message[];
  siblingInfo: { current: number; total: number; siblings: Message[] };
  toolCallRecordMap: Map<string, ToolCallRecord>;
  asyncOperationMap: Map<string, AsyncStatusUpdate>;
  onRegenerate?: (messageId: string) => void;
  onSelectBranch?: (messageId: string) => void;
  onFork?: (messageId: string) => void;
  onEdit?: (message: Message) => void;
  onOpenAsyncDrawer?: (operation: AsyncStatusUpdate) => void;
  isRegenerating?: boolean;
  isForking?: boolean;
  isHighlighted?: boolean;
}

const MessageBubbleInner = forwardRef<HTMLDivElement, MessageBubbleProps>(function MessageBubbleInner(
  { message, viewMode, allMessages, siblingInfo, toolCallRecordMap, asyncOperationMap, onRegenerate, onSelectBranch, onFork, onEdit, onOpenAsyncDrawer, isRegenerating = false, isForking = false, isHighlighted = false },
  ref
) {
  const { addToast } = useToast();
  const isUser = message.role === "user";
  const isAssistant = message.role === "assistant";
  const isSystem = message.role === "system";
  const isTool = message.role === "tool";
  const hasToolCalls = message.role === "assistant" && message.tool_calls && message.tool_calls.length > 0;
  const isCompact = viewMode === "compact";
  const hasSiblings = siblingInfo.total > 1;

  const [isSpeaking, setIsSpeaking] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(message.content);
      addToast("Copied to clipboard", "success");
    } catch {
      addToast("Failed to copy", "error");
    }
  }, [message.content, addToast]);

  const handleReadAloud = useCallback(() => {
    if (isSpeaking) {
      window.speechSynthesis.cancel();
      setIsSpeaking(false);
      return;
    }
    if (typeof window.speechSynthesis === "undefined") {
      addToast("Text-to-speech not supported in this browser", "error");
      return;
    }
    const utterance = new SpeechSynthesisUtterance(message.content);
    utterance.onend = () => setIsSpeaking(false);
    utterance.onerror = () => {
      setIsSpeaking(false);
      addToast("Speech synthesis failed", "error");
    };
    window.speechSynthesis.speak(utterance);
    setIsSpeaking(true);
  }, [message.content, isSpeaking, addToast]);

  useEffect(() => {
    return () => {
      if (isSpeaking) window.speechSynthesis.cancel();
    };
  }, [isSpeaking]);

  const handleComingSoon = useCallback((feature: string) => {
    addToast(`${feature} coming soon`, "info");
  }, [addToast]);

  const handleRegenerate = useCallback(() => {
    if (onRegenerate && isAssistant) onRegenerate(message.id);
    else handleComingSoon("Regenerate");
  }, [onRegenerate, isAssistant, message.id, handleComingSoon]);

  const handleEdit = useCallback(() => {
    if (onEdit && isUser) onEdit(message);
    else handleComingSoon("Edit message");
  }, [onEdit, isUser, message, handleComingSoon]);

  const handleDelete = useCallback(() => handleComingSoon("Delete message"), [handleComingSoon]);

  const handleFork = useCallback(() => {
    if (onFork) onFork(message.id);
    else handleComingSoon("Fork conversation");
  }, [onFork, message.id, handleComingSoon]);

  const handlePreviousVersion = useCallback(() => {
    if (!onSelectBranch) return;
    const prevSibling = getPreviousSibling(allMessages, message.id);
    if (prevSibling) onSelectBranch(prevSibling.id);
  }, [onSelectBranch, allMessages, message.id]);

  const handleNextVersion = useCallback(() => {
    if (!onSelectBranch) return;
    const nextSibling = getNextSibling(allMessages, message.id);
    if (nextSibling) onSelectBranch(nextSibling.id);
  }, [onSelectBranch, allMessages, message.id]);

  const renderActions = (position: "user" | "assistant" | "tool") => {
    const iconSize = "h-3.5 w-3.5";

    if (position === "user") {
      return (
        <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
          <ActionButton icon={<Copy className={iconSize} />} tooltip="Copy" onClick={() => { void handleCopy(); }} />
          <ActionButton icon={<Pencil className={iconSize} />} tooltip="Edit" onClick={handleEdit} />
          <ActionButton icon={<Trash2 className={iconSize} />} tooltip="Delete" onClick={handleDelete} />
          <ActionButton
            icon={isForking ? <Loader2 className={`${iconSize} animate-spin`} /> : <GitBranch className={iconSize} />}
            tooltip={isForking ? "Forking..." : "Fork from here"}
            onClick={handleFork}
            className={isForking ? "cursor-not-allowed" : ""}
          />
        </div>
      );
    }

    if (position === "assistant") {
      return (
        <div className="flex items-center gap-0.5">
          {hasSiblings && (
            <VersionPicker
              current={siblingInfo.current}
              total={siblingInfo.total}
              onPrevious={handlePreviousVersion}
              onNext={handleNextVersion}
              disabled={isRegenerating}
              className="mr-1"
            />
          )}
          <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
            <ActionButton icon={<Copy className={iconSize} />} tooltip="Copy" onClick={() => { void handleCopy(); }} />
            <ActionButton
              icon={isSpeaking ? <VolumeX className={iconSize} /> : <Volume2 className={iconSize} />}
              tooltip={isSpeaking ? "Stop reading" : "Read aloud"}
              onClick={handleReadAloud}
              isActive={isSpeaking}
            />
            <ActionButton
              icon={isRegenerating ? <Loader2 className={`${iconSize} animate-spin`} /> : <RefreshCw className={iconSize} />}
              tooltip={isRegenerating ? "Regenerating..." : "Regenerate"}
              onClick={handleRegenerate}
              className={isRegenerating ? "cursor-not-allowed" : ""}
            />
            <ActionButton
              icon={isForking ? <Loader2 className={`${iconSize} animate-spin`} /> : <GitBranch className={iconSize} />}
              tooltip={isForking ? "Forking..." : "Fork from here"}
              onClick={handleFork}
              className={isForking ? "cursor-not-allowed" : ""}
            />
          </div>
        </div>
      );
    }

    return null;
  };

  const highlightClass = isHighlighted ? "ring-4 ring-yellow-400 rounded-xl bg-yellow-400/20 shadow-[0_0_15px_rgba(250,204,21,0.4)]" : "";

  // System messages - same in both modes
  if (isSystem) {
    return (
      <div ref={ref} className={`flex justify-center transition-all duration-300 ${highlightClass}`} data-testid={`message-${message.id}`}>
        <div className={`bg-slate-200/50 dark:bg-slate-800/50 rounded-lg px-4 py-2 text-sm text-slate-500 dark:text-slate-400 italic ${isCompact ? "w-full text-left" : "max-w-[80%]"}`}>
          <MarkdownRenderer content={message.content} />
        </div>
      </div>
    );
  }

  const sharedProps = {
    message,
    isUser,
    isTool,
    hasToolCalls: Boolean(hasToolCalls),
    toolCallRecordMap,
    asyncOperationMap,
    onOpenAsyncDrawer,
    highlightClass,
    renderActions,
    forwardedRef: ref,
  };

  if (isCompact) {
    return <CompactMessageBubble {...sharedProps} />;
  }

  return <BubbleMessageBubble {...sharedProps} />;
});

// Wrap MessageBubble with simple memo (no custom comparison)
export const MessageBubble = memo(MessageBubbleInner) as typeof MessageBubbleInner;
