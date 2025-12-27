import { useMemo } from "react";
import { Loader2 } from "lucide-react";
import { ChatHeader } from "./ChatHeader";
import { MessageList } from "./MessageList";
import { MessageInput } from "./MessageInput";
import type { ChatWithMessages, Model, Label } from "../../lib/api";
import type { ActiveToolCall } from "../../hooks/useChats";
import type { ViewMode } from "../settings/Settings";
import { computeVisibleMessages } from "../../lib/messageTree";

interface ChatViewProps {
  chatData: ChatWithMessages | null;
  models: Model[];
  labels: Label[];
  isLoading: boolean;
  isGenerating: boolean;
  streamingContent: string;
  activeToolCalls?: ActiveToolCall[];
  scrollToMessageId?: string | null;
  onScrollComplete?: () => void;
  onSendMessage: (content: string) => void;
  onUpdateChat: (data: { name?: string; model?: string }) => void;
  onToggleRead: () => void;
  onToggleStar: () => void;
  onToggleArchive: () => void;
  onDeleteChat: () => void;
  onAssignLabel: (labelId: string) => void;
  onRemoveLabel: (labelId: string) => void;
  viewMode?: ViewMode;
  // Branching operations
  onRegenerateMessage?: (messageId: string) => void;
  onSelectBranch?: (messageId: string) => void;
  isRegenerating?: boolean;
}

export function ChatView({
  chatData,
  models,
  labels,
  isLoading,
  isGenerating,
  streamingContent,
  activeToolCalls = [],
  scrollToMessageId,
  onScrollComplete,
  onSendMessage,
  onUpdateChat,
  onToggleRead,
  onToggleStar,
  onToggleArchive,
  onDeleteChat,
  onAssignLabel,
  onRemoveLabel,
  viewMode,
  onRegenerateMessage,
  onSelectBranch,
  isRegenerating = false,
}: ChatViewProps) {
  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center bg-slate-950" data-testid="chat-view-loading">
        <div className="text-center">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-400 mx-auto mb-3" />
          <p className="text-sm text-slate-500">Loading conversation...</p>
        </div>
      </div>
    );
  }

  if (!chatData) {
    return null;
  }

  // Compute visible messages based on the active branch
  // This filters the full message tree to only show the active path
  const allMessages = chatData.messages || [];
  const visibleMessages = useMemo(() => {
    console.log("[ChatView] Computing visible messages", {
      allMessagesCount: allMessages.length,
      activeLeafId: chatData.chat.active_leaf_message_id,
      allMessages: allMessages.map(m => ({
        id: m.id.slice(0, 8),
        role: m.role,
        parent: m.parent_message_id?.slice(0, 8) ?? null,
        siblingIndex: m.sibling_index,
      })),
    });
    const result = computeVisibleMessages(allMessages, chatData.chat.active_leaf_message_id);
    console.log("[ChatView] Visible messages result:", result.map(m => ({
      id: m.id.slice(0, 8),
      role: m.role,
      siblingIndex: m.sibling_index,
    })));
    return result;
  }, [allMessages, chatData.chat.active_leaf_message_id]);

  return (
    <div className="flex-1 flex flex-col bg-slate-950" data-testid="chat-view">
      <ChatHeader
        chat={chatData.chat}
        models={models}
        labels={labels}
        onUpdateChat={(data) => onUpdateChat(data)}
        onToggleRead={onToggleRead}
        onToggleStar={onToggleStar}
        onToggleArchive={onToggleArchive}
        onDelete={onDeleteChat}
        onAssignLabel={onAssignLabel}
        onRemoveLabel={onRemoveLabel}
      />

      <MessageList
        messages={visibleMessages}
        allMessages={allMessages}
        isGenerating={isGenerating}
        streamingContent={streamingContent}
        activeToolCalls={activeToolCalls}
        scrollToMessageId={scrollToMessageId}
        onScrollComplete={onScrollComplete}
        viewMode={viewMode}
        onRegenerateMessage={onRegenerateMessage}
        onSelectBranch={onSelectBranch}
        isRegenerating={isRegenerating}
      />

      <MessageInput onSend={onSendMessage} isGenerating={isGenerating} />
    </div>
  );
}
